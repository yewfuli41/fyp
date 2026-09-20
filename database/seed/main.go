// Command seed wipes and repopulates the local dev database with test data:
// three businesses, each with three staff (plus an owner-managed bucket) and
// three services, ~4 months of history through 2 weeks ahead worth of slots
// and bookings with varied statuses/types, and a pool of customers with
// different booking patterns — enough spread for the analytics dashboard to
// have something to show. Run from the repo root:
//
//	go run ./database/seed
package main

import (
	"database/sql"
	"log"
	"math/rand"
	"strings"
	"time"

	"fyp/database"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

const testPassword = "Password123!"

// staffSpec is one staff member's recurring weekly pattern: which weekdays
// they work, their working hours on those days, and the two fixed
// appointment slots (time + service) generated for every one of those days.
type staffSpec struct {
	staffID         int64
	weekdays        []string
	workStart       string
	workEnd         string
	recurringByDay  map[string]int64
	slot1Start      string
	slot1End        string
	slot1ServiceOpt int64
	slot2Start      string
	slot2End        string
	slot2ServiceOpt int64
}

// slotRef is a generated (and slot-optioned) service slot, kept around so
// the booking pass can decide whether/how to book it.
type slotRef struct {
	slotOptionID int64
	date         time.Time
}

type weightedCustomer struct {
	id     int64
	weight int
}

// serviceSeed is one service + its single default option and that option's
// items — the shape every seeded business's catalogue uses.
type serviceSeed struct {
	name        string
	description string
	optionName  string
	optionDesc  string
	items       []string
}

// staffSeed is one staff member to create for a business: their login, their
// weekly pattern, and the two daily slots generated for them. slot1Svc /
// slot2Svc index into the owning businessSeed's services.
type staffSeed struct {
	name       string
	email      string
	contact    string
	position   string
	weekdays   []string
	workStart  string
	workEnd    string
	slot1Start string
	slot1End   string
	slot1Svc   int
	slot2Start string
	slot2End   string
	slot2Svc   int
}

// businessSeed is a whole business to create — its owner's login, profile,
// catalogue and staff. ownerSlotSvc indexes the service offered by the
// owner-managed (unassigned) Saturday slots.
type businessSeed struct {
	ownerName    string
	ownerEmail   string
	ownerContact string
	name         string
	description  string
	address      string
	contact      string
	email        string
	services     []serviceSeed
	staff        []staffSeed
	ownerSlotSvc int
}

// leavePlan is one leave application, as an offset in days from today —
// cycled across each business's staff so every status is represented.
type leavePlan struct {
	fromDay int
	toDay   int
	reason  string
	status  string
}

var leavePlans = []leavePlan{
	{-30, -28, "Family trip", "approved"},
	{10, 12, "Medical appointment", "pending"},
	{-10, -9, "Personal matters", "rejected"},
}

func main() {
	if err := godotenv.Load(".env.local"); err != nil {
		log.Println("No .env.local found, using environment variables")
	}

	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Database error: ", err)
	}
	defer db.Close()

	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	hashedPassword := string(hash)

	log.Println("Truncating existing data...")
	if _, err := db.Exec(`TRUNCATE TABLE
		fyp_fuli_bookings,
		fyp_fuli_service_slot_options,
		fyp_fuli_service_slots,
		fyp_fuli_recurring_schedules,
		fyp_fuli_leave_applications,
		fyp_fuli_staff_working_hours,
		fyp_fuli_staff,
		fyp_fuli_service_option_items,
		fyp_fuli_service_options,
		fyp_fuli_services,
		fyp_fuli_business_working_hours,
		fyp_fuli_business_profiles,
		fyp_fuli_users
		RESTART IDENTITY CASCADE`); err != nil {
		log.Fatal("truncate: ", err)
	}
	if _, err := db.Exec(`ALTER SEQUENCE fyp_fuli_booking_group_seq RESTART WITH 1`); err != nil {
		log.Fatal("reset booking group seq: ", err)
	}

	rng := rand.New(rand.NewSource(42))

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	businessCreatedAt := today.AddDate(0, -4, 0)
	slotsUntil := today.AddDate(0, 0, 14)

	// ── Customers ────────────────────────────────────────────────────────
	// Generic filler accounts only — every real (UAT) login is an owner or a
	// staff member instead, since those are the roles being tested. These
	// exist purely so the generated bookings have someone to belong to.
	// Weighted so a few come back repeatedly (returning) while others show up
	// once or twice (new), so customerRetention has something to show
	// regardless of which date range is selected.
	customerSpecs := []struct {
		name    string
		email   string
		contact string
		weight  int
	}{
		{"Dan Customer", "customer1@test.com", "0123456701", 6},
		{"Eve Customer", "customer2@test.com", "0123456702", 1},
		{"Faiz Customer", "customer3@test.com", "0123456703", 3},
		{"Grace Customer", "customer4@test.com", "0123456704", 1},
		{"Hana Customer", "customer5@test.com", "0123456705", 2},
		{"Ivan Customer", "customer6@test.com", "0123456706", 2},
	}
	customers := make([]weightedCustomer, len(customerSpecs))
	for i, c := range customerSpecs {
		customers[i] = weightedCustomer{insertUser(db, c.name, c.email, c.contact, hashedPassword), c.weight}
	}

	// ── Businesses ───────────────────────────────────────────────────────
	businesses := []businessSeed{
		{
			ownerName: "Alice Owner", ownerEmail: "owner@test.com", ownerContact: "0123456789",
			name:        "Fuli's Grooming Studio",
			description: "A friendly neighbourhood salon for FYP testing.",
			address:     "123 Test Street, Cyberjaya",
			contact:     "0387654321", email: "contact@fuligrooming.test",
			services: []serviceSeed{
				{"Haircut", "Classic haircut service", "Standard Haircut", "Wash, cut & style", []string{"Hair Wash", "Cut & Style"}},
				{"Manicure", "Nail care service", "Classic Manicure", "Basic manicure treatment", []string{"Nail Trim", "Polish"}},
				{"Facial", "Skin care treatment", "Classic Facial", "Cleanse and hydrate", []string{"Cleanse", "Mask"}},
			},
			staff: []staffSeed{
				{
					name: "Ben Barber", email: "staff1@test.com", contact: "0123456781", position: "Senior Stylist",
					weekdays: []string{"monday", "wednesday", "friday"}, workStart: "09:00", workEnd: "17:00",
					slot1Start: "09:00", slot1End: "10:00", slot1Svc: 0,
					slot2Start: "11:00", slot2End: "12:00", slot2Svc: 1,
				},
				{
					name: "Cara Stylist", email: "staff2@test.com", contact: "0123456782", position: "Junior Stylist",
					weekdays: []string{"tuesday", "thursday"}, workStart: "10:00", workEnd: "18:00",
					slot1Start: "10:00", slot1End: "11:00", slot1Svc: 2,
					slot2Start: "12:00", slot2End: "13:00", slot2Svc: 0,
				},
				{
					name: "Diana Nails", email: "staff3@test.com", contact: "0123456783", position: "Nail Technician",
					weekdays: []string{"monday", "thursday", "saturday"}, workStart: "09:00", workEnd: "15:00",
					slot1Start: "09:00", slot1End: "10:00", slot1Svc: 1,
					slot2Start: "11:00", slot2End: "12:00", slot2Svc: 2,
				},
			},
			ownerSlotSvc: 0,
		},
		{
			ownerName: "Adam Hashim", ownerEmail: "adam.hashim@paynet.my", ownerContact: "0123456790",
			name:        "PayNet Wellness Spa",
			description: "Massage and wellness treatments for the Sentral crowd.",
			address:     "1 Sentral, Jalan Rakyat, Kuala Lumpur",
			contact:     "0322648000", email: "hello@paynetwellness.test",
			services: []serviceSeed{
				{"Massage", "Full body massage", "Aromatherapy Massage", "60-minute full body massage", []string{"Warm Towel", "Essential Oils"}},
				{"Reflexology", "Foot and hand reflexology", "Foot Reflexology", "45-minute foot reflexology", []string{"Foot Soak", "Pressure Point Massage"}},
				{"Body Scrub", "Exfoliating body treatment", "Signature Body Scrub", "Exfoliate and moisturise", []string{"Sea Salt Scrub", "Body Lotion"}},
			},
			staff: []staffSeed{
				{
					name: "Eng Hong Ng", email: "enghong.ng@paynet.my", contact: "0123456791", position: "Senior Therapist",
					weekdays: []string{"monday", "tuesday", "thursday"}, workStart: "09:00", workEnd: "17:00",
					slot1Start: "09:30", slot1End: "10:30", slot1Svc: 0,
					slot2Start: "14:00", slot2End: "15:00", slot2Svc: 2,
				},
				{
					name: "Lim Tze Yi", email: "lim.tzeyi@paynet.my", contact: "0123456792", position: "Therapist",
					weekdays: []string{"wednesday", "friday", "saturday"}, workStart: "10:00", workEnd: "13:00",
					slot1Start: "10:30", slot1End: "11:30", slot1Svc: 1,
					slot2Start: "11:30", slot2End: "12:30", slot2Svc: 0,
				},
			},
			ownerSlotSvc: 1,
		},
		{
			ownerName: "Amira Iryanti", ownerEmail: "amira.iryanti@paynet.my", ownerContact: "0123456793",
			name:        "Sentral Barber & Co",
			description: "Walk-in friendly barbershop for cuts, beards and colour.",
			address:     "88 Jalan Tun Sambanthan, Kuala Lumpur",
			contact:     "0322745500", email: "hello@sentralbarber.test",
			services: []serviceSeed{
				{"Haircut", "Barbershop haircut", "Signature Cut", "Consultation, cut & finish", []string{"Consultation", "Cut & Finish"}},
				{"Beard Grooming", "Beard trim and shaping", "Beard Trim", "Trim, shape and hot towel", []string{"Hot Towel", "Beard Oil"}},
				{"Hair Colouring", "Colour and highlights", "Full Colour", "Single-process full colour", []string{"Colour Application", "Wash & Style"}},
			},
			staff: []staffSeed{
				{
					name: "Wei Xin Low", email: "weixin.low@paynet.my", contact: "0123456794", position: "Master Barber",
					weekdays: []string{"monday", "wednesday", "friday"}, workStart: "10:00", workEnd: "18:00",
					slot1Start: "10:00", slot1End: "11:00", slot1Svc: 0,
					slot2Start: "13:00", slot2End: "14:00", slot2Svc: 1,
				},
				{
					name: "Aqib Haqmi", email: "aqibhaqmi@paynet.my", contact: "0123456795", position: "Colour Specialist",
					weekdays: []string{"tuesday", "thursday"}, workStart: "09:00", workEnd: "17:00",
					slot1Start: "09:00", slot1End: "10:00", slot1Svc: 2,
					slot2Start: "11:30", slot2End: "12:30", slot2Svc: 0,
				},
			},
			ownerSlotSvc: 0,
		},
		{
			ownerName: "Zi Xi Hee", ownerEmail: "zixi.hee@paynet.my", ownerContact: "0123456796",
			name:        "Bangsar Nail Bar",
			description: "Nail art, gel and spa manicures in Bangsar.",
			address:     "12 Jalan Telawi 3, Bangsar, Kuala Lumpur",
			contact:     "0322820900", email: "hello@bangsarnail.test",
			services: []serviceSeed{
				{"Manicure", "Hand and nail care", "Gel Manicure", "Shape, cuticle care & gel polish", []string{"Nail Shaping", "Gel Polish"}},
				{"Pedicure", "Foot and nail care", "Spa Pedicure", "Soak, scrub & polish", []string{"Foot Soak", "Callus Care"}},
				{"Nail Art", "Custom nail designs", "Custom Nail Art", "Hand-painted designs per nail", []string{"Design Consultation", "Hand Painting"}},
			},
			staff: []staffSeed{
				{
					name: "Yi Hern Wong", email: "yihern.wong@paynet.my", contact: "0123456797", position: "Senior Nail Technician",
					weekdays: []string{"monday", "tuesday", "thursday"}, workStart: "09:00", workEnd: "17:00",
					slot1Start: "09:00", slot1End: "10:00", slot1Svc: 0,
					slot2Start: "14:30", slot2End: "15:30", slot2Svc: 2,
				},
				{
					name: "Leong Chien Koh", email: "leongchien.koh@paynet.my", contact: "0123456798", position: "Nail Technician",
					weekdays: []string{"wednesday", "friday", "saturday"}, workStart: "09:00", workEnd: "13:00",
					slot1Start: "09:30", slot1End: "10:30", slot1Svc: 1,
					slot2Start: "11:00", slot2End: "12:00", slot2Svc: 0,
				},
			},
			ownerSlotSvc: 1,
		},
		{
			ownerName: "Amirul Zaidi", ownerEmail: "amirul.zaidi@paynet.my", ownerContact: "0123456806",
			name:        "Glow Skin Studio",
			description: "Facials and skin treatments tailored to your skin type.",
			address:     "5 Jalan SS15/4, Subang Jaya, Selangor",
			contact:     "0356321200", email: "hello@glowskin.test",
			services: []serviceSeed{
				{"Facial", "Deep cleansing facial", "Hydrating Facial", "Cleanse, exfoliate & hydrate", []string{"Deep Cleanse", "Hydrating Mask"}},
				{"Peeling", "Chemical exfoliation", "Gentle Peel", "Mild acid peel for brightening", []string{"Skin Prep", "Peel Application"}},
				{"Eye Treatment", "Under-eye care", "Brightening Eye Care", "Targets dark circles & puffiness", []string{"Eye Massage", "Eye Mask"}},
			},
			staff: []staffSeed{
				{
					name: "Brandon Lai", email: "brandon.lai@paynet.my", contact: "0123456800", position: "Senior Aesthetician",
					weekdays: []string{"tuesday", "wednesday", "friday"}, workStart: "10:00", workEnd: "18:00",
					slot1Start: "10:00", slot1End: "11:00", slot1Svc: 0,
					slot2Start: "15:00", slot2End: "16:00", slot2Svc: 1,
				},
				{
					name: "Tian Xin Lee", email: "tianxin.lee@paynet.my", contact: "0123456801", position: "Aesthetician",
					weekdays: []string{"monday", "thursday", "saturday"}, workStart: "09:00", workEnd: "13:00",
					slot1Start: "09:00", slot1End: "10:00", slot1Svc: 2,
					slot2Start: "11:00", slot2End: "12:00", slot2Svc: 0,
				},
			},
			ownerSlotSvc: 0,
		},
		{
			ownerName: "Mei Lin Lee", ownerEmail: "meilin.lee@paynet.my", ownerContact: "0123456802",
			name:        "Mont Kiara Hair Lounge",
			description: "Cuts, colour and treatments in a relaxed lounge setting.",
			address:     "1 Jalan Kiara, Mont Kiara, Kuala Lumpur",
			contact:     "0362013300", email: "hello@mkhairlounge.test",
			services: []serviceSeed{
				{"Haircut", "Cut and blow dry", "Cut & Blow Dry", "Consultation, cut & blow dry", []string{"Consultation", "Blow Dry"}},
				{"Hair Treatment", "Scalp and hair repair", "Keratin Treatment", "Smoothing and repair treatment", []string{"Scalp Analysis", "Keratin Application"}},
				{"Hair Colouring", "Colour and highlights", "Balayage", "Hand-painted highlights", []string{"Colour Consultation", "Toning"}},
			},
			staff: []staffSeed{
				{
					name: "Sam Lee", email: "sam.lee@paynet.my", contact: "0123456803", position: "Senior Stylist",
					weekdays: []string{"monday", "wednesday", "friday"}, workStart: "10:00", workEnd: "18:00",
					slot1Start: "10:30", slot1End: "11:30", slot1Svc: 0,
					slot2Start: "14:00", slot2End: "15:00", slot2Svc: 2,
				},
				{
					name: "Xin Yi Chong", email: "xinyi.chong@paynet.my", contact: "0123456804", position: "Stylist",
					weekdays: []string{"tuesday", "thursday", "saturday"}, workStart: "09:00", workEnd: "13:00",
					slot1Start: "09:00", slot1End: "10:00", slot1Svc: 1,
					slot2Start: "11:30", slot2End: "12:30", slot2Svc: 0,
				},
			},
			ownerSlotSvc: 0,
		},
		{
			ownerName: "Anamallah Samshul", ownerEmail: "anamallah.samshul@paynet.my", ownerContact: "0123456805",
			name:        "KL Physio & Recovery",
			description: "Physiotherapy and sports recovery sessions by appointment.",
			address:     "20 Jalan Ampang, Kuala Lumpur",
			contact:     "0321616700", email: "hello@klphysio.test",
			services: []serviceSeed{
				{"Physiotherapy", "Assessment and treatment", "Physio Session", "Assessment and guided treatment", []string{"Assessment", "Manual Therapy"}},
				{"Sports Massage", "Deep tissue recovery", "Sports Recovery Massage", "Deep tissue massage for athletes", []string{"Warm Up", "Deep Tissue Work"}},
				{"Rehab Consult", "Injury rehab planning", "Rehab Consultation", "Personalised rehab plan", []string{"Movement Screening", "Exercise Plan"}},
			},
			staff: []staffSeed{
				{
					name: "Jeyashiri Sanglimuthu", email: "jeyashiri.sanglimuthu@paynet.my", contact: "0123456799", position: "Senior Physiotherapist",
					weekdays: []string{"monday", "tuesday", "thursday"}, workStart: "09:00", workEnd: "17:00",
					slot1Start: "09:00", slot1End: "10:00", slot1Svc: 0,
					slot2Start: "13:30", slot2End: "14:30", slot2Svc: 2,
				},
				{
					name: "Jing Wey Yew", email: "jingwey.yew@paynet.my", contact: "0123456807", position: "Physiotherapist",
					weekdays: []string{"wednesday", "friday", "saturday"}, workStart: "09:00", workEnd: "13:00",
					slot1Start: "09:30", slot1End: "10:30", slot1Svc: 1,
					slot2Start: "11:00", slot2End: "12:00", slot2Svc: 0,
				},
			},
			ownerSlotSvc: 0,
		},
		{
			ownerName: "Kenny Chong", ownerEmail: "kenny.chong@paynet.my", ownerContact: "0123456808",
			name:        "Kenny's Fitness Studio",
			description: "Personal training and small-group fitness sessions by appointment.",
			address:     "8 Jalan Tun Razak, Kuala Lumpur",
			contact:     "0322930100", email: "hello@kennyfitness.test",
			services: []serviceSeed{
				{"Personal Training", "One-to-one coaching session", "1-to-1 Training Session", "60-minute personal training session", []string{"Warm Up", "Strength Circuit"}},
				{"Group Class", "Small-group fitness class", "Small Group Class", "45-minute small group class", []string{"Mobility Drill", "Circuit Set"}},
				{"Fitness Assessment", "Body and movement assessment", "Full Fitness Assessment", "Body composition and movement screening", []string{"Body Composition", "Movement Screening"}},
			},
			staff: []staffSeed{
				{
					name: "Farah Trainer", email: "staff7@test.com", contact: "0123456809", position: "Senior Personal Trainer",
					weekdays: []string{"monday", "wednesday", "friday"}, workStart: "09:00", workEnd: "17:00",
					slot1Start: "09:00", slot1End: "10:00", slot1Svc: 0,
					slot2Start: "11:00", slot2End: "12:00", slot2Svc: 2,
				},
				{
					name: "Gavin Coach", email: "staff8@test.com", contact: "0123456810", position: "Personal Trainer",
					weekdays: []string{"tuesday", "thursday"}, workStart: "10:00", workEnd: "18:00",
					slot1Start: "10:00", slot1End: "11:00", slot1Svc: 1,
					slot2Start: "14:00", slot2End: "15:00", slot2Svc: 0,
				},
				{
					// Works Saturdays, so their hours stay inside the business's
					// shorter Saturday opening (09:00-13:00), not the weekday one.
					name: "Ivy Fitness", email: "staff9@test.com", contact: "0123456811", position: "Fitness Coach",
					weekdays: []string{"wednesday", "saturday"}, workStart: "09:00", workEnd: "13:00",
					slot1Start: "09:30", slot1End: "10:30", slot1Svc: 2,
					slot2Start: "11:00", slot2End: "12:00", slot2Svc: 1,
				},
			},
			ownerSlotSvc: 1,
		},
	}

	totalSlots, totalBookings := 0, 0
	for i, cfg := range businesses {
		slots, bookings := seedBusiness(db, rng, cfg, hashedPassword, customers, businessCreatedAt, today, slotsUntil, i)
		totalSlots += slots
		totalBookings += bookings
	}

	log.Println("Seed complete.")
	log.Printf("Generated %d businesses, %d slots, %d bookings, from %s to %s",
		len(businesses), totalSlots, totalBookings,
		businessCreatedAt.Format("2006-01-02"), slotsUntil.Format("2006-01-02"))
	log.Printf("Test users (all password: %s)", testPassword)
	for _, cfg := range businesses {
		log.Printf("  %-32s - owner of %s", cfg.ownerEmail, cfg.name)
		for _, s := range cfg.staff {
			log.Printf("  %-32s - staff (%s, %s)", s.email, s.name, cfg.name)
		}
	}
	for _, c := range customerSpecs {
		log.Printf("  %-32s - customer (%s)", c.email, c.name)
	}
}

// seedBusiness creates one whole business — owner, profile, working hours,
// catalogue, staff, their recurring schedules and generated slots, the
// bookings placed against those slots, and a leave application per staff.
// Returns how many slots and bookings it generated.
//
// index is the business's position in the seed list; it only shifts which
// leave status each staff member gets, so that every status still shows up
// across the whole dataset even though a business has fewer staff than
// there are statuses.
func seedBusiness(
	db *sql.DB, rng *rand.Rand, cfg businessSeed, hashedPassword string,
	customers []weightedCustomer, businessCreatedAt, today, slotsUntil time.Time, index int,
) (int, int) {
	ownerID := insertUser(db, cfg.ownerName, cfg.ownerEmail, cfg.ownerContact, hashedPassword)
	businessID := insertBusinessWithCreatedAt(db, ownerID, cfg.name, cfg.description, cfg.address,
		cfg.contact, cfg.email, businessCreatedAt)
	insertBusinessWorkingHours(db, businessID)

	// ── Services ─────────────────────────────────────────────────────────
	optionIDs := make([]int64, len(cfg.services))
	for i, svc := range cfg.services {
		serviceID := insertService(db, businessID, svc.name, svc.description)
		optionID := insertServiceOption(db, serviceID, svc.optionName, svc.optionDesc)
		for _, item := range svc.items {
			insertServiceOptionItem(db, optionID, item)
		}
		optionIDs[i] = optionID
	}

	// ── Staff ────────────────────────────────────────────────────────────
	specs := make([]*staffSpec, len(cfg.staff))
	for i, s := range cfg.staff {
		userID := insertUser(db, s.name, s.email, s.contact, hashedPassword)
		specs[i] = &staffSpec{
			staffID:  insertStaff(db, userID, businessID, s.name, s.contact, s.position),
			weekdays: s.weekdays, workStart: s.workStart, workEnd: s.workEnd,
			slot1Start: s.slot1Start, slot1End: s.slot1End, slot1ServiceOpt: optionIDs[s.slot1Svc],
			slot2Start: s.slot2Start, slot2End: s.slot2End, slot2ServiceOpt: optionIDs[s.slot2Svc],
		}
	}
	for _, spec := range specs {
		spec.recurringByDay = make(map[string]int64)
		for _, day := range spec.weekdays {
			insertStaffWorkingHours(db, spec.staffID, day, spec.workStart, spec.workEnd)
			sid := spec.staffID
			spec.recurringByDay[day] = insertRecurringSchedule(db, businessID, &sid, day, spec.workStart, spec.workEnd, ownerID)
		}
	}

	// Owner-managed (unassigned) Saturday coverage — populates the
	// "Owner-managed" bucket in staff utilization/cancellation analytics.
	ownerRSID := insertRecurringSchedule(db, businessID, nil, "saturday", "09:00", "13:00", ownerID)

	// ── Slots ────────────────────────────────────────────────────────────
	var allSlots []slotRef
	for d := businessCreatedAt; !d.After(slotsUntil); d = d.AddDate(0, 0, 1) {
		weekday := strings.ToLower(d.Weekday().String())

		for _, spec := range specs {
			rsID, works := spec.recurringByDay[weekday]
			if !works {
				continue
			}
			sid := spec.staffID
			s1 := insertServiceSlot(db, &sid, &rsID, d, spec.slot1Start, spec.slot1End, ownerID)
			allSlots = append(allSlots, slotRef{slotOptionID: insertSlotOption(db, spec.slot1ServiceOpt, s1), date: d})
			s2 := insertServiceSlot(db, &sid, &rsID, d, spec.slot2Start, spec.slot2End, ownerID)
			allSlots = append(allSlots, slotRef{slotOptionID: insertSlotOption(db, spec.slot2ServiceOpt, s2), date: d})
		}

		if weekday == "saturday" && rng.Intn(2) == 0 {
			s := insertServiceSlot(db, nil, &ownerRSID, d, "09:00", "10:00", ownerID)
			allSlots = append(allSlots, slotRef{slotOptionID: insertSlotOption(db, optionIDs[cfg.ownerSlotSvc], s), date: d})
		}
	}

	// ── Bookings ─────────────────────────────────────────────────────────
	bookedCount := 0
	for _, s := range allSlots {
		isPast := s.date.Before(today)
		bookProb := 0.55
		if isPast {
			bookProb = 0.75
		}
		if rng.Float64() > bookProb {
			continue
		}
		bookedCount++

		bookingType := "online"
		if rng.Float64() < 0.15 {
			bookingType = "walk_in"
		}

		// A walk-in is recorded in person by the business itself, not
		// booked by the customer online — see the comment on
		// StaffCancellation in schema.analytics.graphqls.
		userID := ownerID
		if bookingType != "walk_in" {
			userID = pickCustomer(rng, customers)
		}

		var status string
		if isPast {
			status = weightedChoice(rng, []string{"past", "cancelled", "rejected"}, []int{70, 18, 12})
		} else {
			status = weightedChoice(rng, []string{"accepted", "pending", "cancelled"}, []int{55, 35, 10})
		}

		// Booked a few days before the appointment itself — realistic for a
		// past slot, but slots run two weeks ahead, so a future slot would
		// otherwise claim to have been booked in the future. A real booking
		// made while demoing carries created_at = now and would then sort
		// below these on any most-recently-booked list. Pull anything that
		// overshoots back into the last ten days, staggered rather than
		// pinned to the same instant so their relative order still means
		// something.
		seedRunAt := time.Now().UTC()
		createdAt := s.date.AddDate(0, 0, -(rng.Intn(10) + 1))
		if createdAt.Before(businessCreatedAt) {
			createdAt = businessCreatedAt
		}
		createdAt = createdAt.Add(time.Duration(rng.Intn(10)+8) * time.Hour)
		if createdAt.After(seedRunAt) {
			createdAt = seedRunAt.Add(-time.Duration(rng.Intn(14400)) * time.Minute)
		}
		// Decided an hour or so after being booked, on the same footing: a
		// decision that hasn't happened yet would be its own small lie.
		decidedAt := createdAt.Add(time.Duration(rng.Intn(6)+1) * time.Hour)
		if decidedAt.After(seedRunAt) {
			decidedAt = seedRunAt
		}

		var decidedBy *int64
		switch status {
		case "past", "accepted", "rejected":
			ob := ownerID
			decidedBy = &ob
		case "cancelled":
			if bookingType != "walk_in" && rng.Float64() < 0.5 {
				u := userID
				decidedBy = &u
			} else {
				ob := ownerID
				decidedBy = &ob
			}
		}
		var decidedAtPtr *time.Time
		if decidedBy != nil {
			decidedAtPtr = &decidedAt
		}

		insertBookingFull(db, userID, s.slotOptionID, status, bookingType, "", decidedBy, decidedAtPtr, createdAt)
	}

	// ── Leave applications ───────────────────────────────────────────────
	for i, spec := range specs {
		p := leavePlans[(index+i)%len(leavePlans)]
		insertLeave(db, spec.staffID, today.AddDate(0, 0, p.fromDay), today.AddDate(0, 0, p.toDay), p.reason, p.status)
	}

	return len(allSlots), bookedCount
}

func pickCustomer(rng *rand.Rand, customers []weightedCustomer) int64 {
	total := 0
	for _, c := range customers {
		total += c.weight
	}
	r := rng.Intn(total)
	for _, c := range customers {
		if r < c.weight {
			return c.id
		}
		r -= c.weight
	}
	return customers[len(customers)-1].id
}

func weightedChoice(rng *rand.Rand, options []string, weights []int) string {
	total := 0
	for _, w := range weights {
		total += w
	}
	r := rng.Intn(total)
	for i, w := range weights {
		if r < w {
			return options[i]
		}
		r -= w
	}
	return options[len(options)-1]
}

func insertUser(db *sql.DB, username, email, contact, hashedPassword string) int64 {
	var id int64
	err := db.QueryRow(
		`INSERT INTO fyp_fuli_users (username, email, contact_number, password)
		 VALUES ($1,$2,$3,$4) RETURNING user_id`,
		username, email, contact, hashedPassword).Scan(&id)
	if err != nil {
		log.Fatalf("insert user %s: %v", email, err)
	}
	return id
}

func insertBusinessWithCreatedAt(db *sql.DB, ownerID int64, name, description, address, contact, email string, createdAt time.Time) int64 {
	var id int64
	err := db.QueryRow(
		`INSERT INTO fyp_fuli_business_profiles
		 (owner_user_id, business_name, description, address, business_contact_number, business_email, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING business_id`,
		ownerID, name, description, address, contact, email, createdAt).Scan(&id)
	if err != nil {
		log.Fatal("insert business: ", err)
	}
	return id
}

func insertBusinessWorkingHours(db *sql.DB, businessID int64) {
	weekdays := []string{"monday", "tuesday", "wednesday", "thursday", "friday"}
	for _, day := range weekdays {
		if _, err := db.Exec(
			`INSERT INTO fyp_fuli_business_working_hours (business_id, day, start_time, end_time)
			 VALUES ($1,$2,$3,$4)`,
			businessID, day, "09:00", "18:00"); err != nil {
			log.Fatal("insert business working hours: ", err)
		}
	}
	if _, err := db.Exec(
		`INSERT INTO fyp_fuli_business_working_hours (business_id, day, start_time, end_time)
		 VALUES ($1,'saturday',$2,$3)`,
		businessID, "09:00", "13:00"); err != nil {
		log.Fatal("insert business working hours (saturday): ", err)
	}
}

func insertService(db *sql.DB, businessID int64, name, description string) int64 {
	var id int64
	err := db.QueryRow(
		`INSERT INTO fyp_fuli_services (business_id, service_name, description)
		 VALUES ($1,$2,$3) RETURNING service_id`,
		businessID, name, description).Scan(&id)
	if err != nil {
		log.Fatal("insert service: ", err)
	}
	return id
}

func insertServiceOption(db *sql.DB, serviceID int64, name, description string) int64 {
	var id int64
	err := db.QueryRow(
		`INSERT INTO fyp_fuli_service_options (service_id, service_option_name, description, is_default)
		 VALUES ($1,$2,$3,TRUE) RETURNING service_option_id`,
		serviceID, name, description).Scan(&id)
	if err != nil {
		log.Fatal("insert service option: ", err)
	}
	return id
}

func insertServiceOptionItem(db *sql.DB, optionID int64, name string) int64 {
	var id int64
	err := db.QueryRow(
		`INSERT INTO fyp_fuli_service_option_items (service_option_id, service_option_item_name)
		 VALUES ($1,$2) RETURNING service_option_item_id`,
		optionID, name).Scan(&id)
	if err != nil {
		log.Fatal("insert service option item: ", err)
	}
	return id
}

func insertStaff(db *sql.DB, userID, businessID int64, name, contact, position string) int64 {
	var id int64
	err := db.QueryRow(
		`INSERT INTO fyp_fuli_staff (user_id, business_id, staff_name, staff_contact_number, position)
		 VALUES ($1,$2,$3,$4,$5) RETURNING staff_id`,
		userID, businessID, name, contact, position).Scan(&id)
	if err != nil {
		log.Fatal("insert staff: ", err)
	}
	return id
}

func insertStaffWorkingHours(db *sql.DB, staffID int64, day, start, end string) {
	if _, err := db.Exec(
		`INSERT INTO fyp_fuli_staff_working_hours (staff_id, day, start_time, end_time)
		 VALUES ($1,$2,$3,$4)`,
		staffID, day, start, end); err != nil {
		log.Fatal("insert staff working hours: ", err)
	}
}

func insertRecurringSchedule(db *sql.DB, businessID int64, staffID *int64, day, start, end string, createdBy int64) int64 {
	var id int64
	err := db.QueryRow(
		`INSERT INTO fyp_fuli_recurring_schedules (business_id, staff_id, day, start_time, end_time, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING recurring_schedule_id`,
		businessID, staffID, day, start, end, createdBy).Scan(&id)
	if err != nil {
		log.Fatal("insert recurring schedule: ", err)
	}
	return id
}

func insertServiceSlot(db *sql.DB, staffID, recurringScheduleID *int64, date time.Time, start, end string, createdBy int64) int64 {
	var id int64
	err := db.QueryRow(
		`INSERT INTO fyp_fuli_service_slots (staff_id, recurring_schedule_id, date, start_time, end_time, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING service_slot_id`,
		staffID, recurringScheduleID, date.Format("2006-01-02"), start, end, createdBy).Scan(&id)
	if err != nil {
		log.Fatal("insert service slot: ", err)
	}
	return id
}

func insertSlotOption(db *sql.DB, serviceOptionID, slotID int64) int64 {
	var id int64
	err := db.QueryRow(
		`INSERT INTO fyp_fuli_service_slot_options (service_option_id, service_slot_id)
		 VALUES ($1,$2) RETURNING slot_option_id`,
		serviceOptionID, slotID).Scan(&id)
	if err != nil {
		log.Fatal("insert slot option: ", err)
	}
	return id
}

func insertBookingFull(
	db *sql.DB, userID, slotOptionID int64, status, bookingType, description string,
	decidedBy *int64, decidedAt *time.Time, createdAt time.Time,
) int64 {
	var id int64
	err := db.QueryRow(
		`INSERT INTO fyp_fuli_bookings
		 (user_id, slot_option_id, booking_group_id, status, booking_type, description, decided_by, decided_at, created_at)
		 VALUES ($1,$2,nextval('fyp_fuli_booking_group_seq'),$3,$4,$5,$6,$7,$8) RETURNING booking_id`,
		userID, slotOptionID, status, bookingType, description, decidedBy, decidedAt, createdAt).Scan(&id)
	if err != nil {
		log.Fatal("insert booking: ", err)
	}
	return id
}

// A "pending" leave has no decided_at yet — anything else is treated as
// already decided, at its own end date.
func insertLeave(db *sql.DB, staffID int64, start, end time.Time, justification, status string) int64 {
	var id int64
	var decidedAtArg any
	if status != "pending" {
		decidedAtArg = end
	}
	var remark any
	if status == "rejected" {
		remark = "Short-staffed that week"
	}
	err := db.QueryRow(
		`INSERT INTO fyp_fuli_leave_applications
		 (staff_id, start_date, end_date, justification, status, decided_at, remark)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING leave_id`,
		staffID, start.Format("2006-01-02"), end.Format("2006-01-02"), justification, status, decidedAtArg, remark).Scan(&id)
	if err != nil {
		log.Fatal("insert leave application: ", err)
	}
	return id
}
