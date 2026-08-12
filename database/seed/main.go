// Command seed wipes and repopulates the local dev database with test data:
// one business, three staff (plus an owner-managed bucket), three services,
// ~4 months of history through 2 weeks ahead worth of slots and bookings
// with varied statuses/types, and a handful of customers with different
// booking patterns — enough spread for the analytics dashboard to have
// something to show. Run from the repo root:
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

	// ── Users ────────────────────────────────────────────────────────────
	ownerID := insertUser(db, "Alice Owner", "owner@test.com", "0123456789", hashedPassword)
	staff1UserID := insertUser(db, "Ben Barber", "staff1@test.com", "0123456781", hashedPassword)
	staff2UserID := insertUser(db, "Cara Stylist", "staff2@test.com", "0123456782", hashedPassword)
	staff3UserID := insertUser(db, "Diana Nails", "staff3@test.com", "0123456783", hashedPassword)
	customer1ID := insertUser(db, "Dan Customer", "customer1@test.com", "0123456701", hashedPassword)
	customer2ID := insertUser(db, "Eve Customer", "customer2@test.com", "0123456702", hashedPassword)
	customer3ID := insertUser(db, "Faiz Customer", "customer3@test.com", "0123456703", hashedPassword)
	customer4ID := insertUser(db, "Grace Customer", "customer4@test.com", "0123456704", hashedPassword)
	customer5ID := insertUser(db, "Hana Customer", "customer5@test.com", "0123456705", hashedPassword)
	customer6ID := insertUser(db, "Ivan Customer", "customer6@test.com", "0123456706", hashedPassword)

	// ── Business ─────────────────────────────────────────────────────────
	businessID := insertBusinessWithCreatedAt(db, ownerID, "Fuli's Grooming Studio",
		"A friendly neighbourhood salon for FYP testing.", "123 Test Street, Cyberjaya",
		"0387654321", "contact@fuligrooming.test", businessCreatedAt)
	insertBusinessWorkingHours(db, businessID)

	// ── Services ─────────────────────────────────────────────────────────
	haircutID := insertService(db, businessID, "Haircut", "Classic haircut service")
	haircutOptionID := insertServiceOption(db, haircutID, "Standard Haircut", "Wash, cut & style")
	insertServiceOptionItem(db, haircutOptionID, "Hair Wash")
	insertServiceOptionItem(db, haircutOptionID, "Cut & Style")

	manicureID := insertService(db, businessID, "Manicure", "Nail care service")
	manicureOptionID := insertServiceOption(db, manicureID, "Classic Manicure", "Basic manicure treatment")
	insertServiceOptionItem(db, manicureOptionID, "Nail Trim")
	insertServiceOptionItem(db, manicureOptionID, "Polish")

	facialID := insertService(db, businessID, "Facial", "Skin care treatment")
	facialOptionID := insertServiceOption(db, facialID, "Classic Facial", "Cleanse and hydrate")
	insertServiceOptionItem(db, facialOptionID, "Cleanse")
	insertServiceOptionItem(db, facialOptionID, "Mask")

	// ── Staff ────────────────────────────────────────────────────────────
	staff1ID := insertStaff(db, staff1UserID, businessID, "Ben Barber", "0123456781", "Senior Stylist")
	staff2ID := insertStaff(db, staff2UserID, businessID, "Cara Stylist", "0123456782", "Junior Stylist")
	staff3ID := insertStaff(db, staff3UserID, businessID, "Diana Nails", "0123456783", "Nail Technician")

	specs := []*staffSpec{
		{
			staffID: staff1ID, weekdays: []string{"monday", "wednesday", "friday"},
			workStart: "09:00", workEnd: "17:00",
			slot1Start: "09:00", slot1End: "10:00", slot1ServiceOpt: haircutOptionID,
			slot2Start: "11:00", slot2End: "12:00", slot2ServiceOpt: manicureOptionID,
		},
		{
			staffID: staff2ID, weekdays: []string{"tuesday", "thursday"},
			workStart: "10:00", workEnd: "18:00",
			slot1Start: "10:00", slot1End: "11:00", slot1ServiceOpt: facialOptionID,
			slot2Start: "12:00", slot2End: "13:00", slot2ServiceOpt: haircutOptionID,
		},
		{
			staffID: staff3ID, weekdays: []string{"monday", "thursday", "saturday"},
			workStart: "09:00", workEnd: "15:00",
			slot1Start: "09:00", slot1End: "10:00", slot1ServiceOpt: manicureOptionID,
			slot2Start: "11:00", slot2End: "12:00", slot2ServiceOpt: facialOptionID,
		},
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
			allSlots = append(allSlots, slotRef{slotOptionID: insertSlotOption(db, haircutOptionID, s), date: d})
		}
	}

	// ── Bookings ─────────────────────────────────────────────────────────
	// Weighted so a few customers come back repeatedly (returning) while
	// others show up once or twice (new) — customerRetention has something
	// to show regardless of which date range is selected.
	customers := []weightedCustomer{
		{customer1ID, 6}, {customer2ID, 1}, {customer3ID, 3},
		{customer4ID, 1}, {customer5ID, 2}, {customer6ID, 2},
	}

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

		createdAt := s.date.AddDate(0, 0, -(rng.Intn(10) + 1))
		if createdAt.Before(businessCreatedAt) {
			createdAt = businessCreatedAt
		}
		createdAt = createdAt.Add(time.Duration(rng.Intn(10)+8) * time.Hour)
		decidedAt := createdAt.Add(time.Duration(rng.Intn(6)+1) * time.Hour)

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
	insertLeave(db, staff1ID, today.AddDate(0, 0, -30), today.AddDate(0, 0, -28), "Family trip", "approved")
	insertLeave(db, staff2ID, today.AddDate(0, 0, 10), today.AddDate(0, 0, 12), "Medical appointment", "pending")
	insertLeave(db, staff3ID, today.AddDate(0, 0, -10), today.AddDate(0, 0, -9), "Personal matters", "rejected")

	log.Println("Seed complete.")
	log.Printf("Generated %d slots, %d bookings, from %s to %s", len(allSlots), bookedCount,
		businessCreatedAt.Format("2006-01-02"), slotsUntil.Format("2006-01-02"))
	log.Printf("Test users (all password: %s)", testPassword)
	log.Println("  owner@test.com     - business owner")
	log.Println("  staff1@test.com    - staff (Ben Barber)")
	log.Println("  staff2@test.com    - staff (Cara Stylist)")
	log.Println("  staff3@test.com    - staff (Diana Nails)")
	log.Println("  customer1@test.com .. customer6@test.com - customers")
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
