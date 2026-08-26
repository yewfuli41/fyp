package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"fyp/config"
	"fyp/database"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"strings"
	"time"
)

type serviceSlotService struct {
	serviceSlotRepo   interfaces.IServiceSlotRepo
	serviceRepo       interfaces.IServiceRepo
	leaveRepo         interfaces.ILeaveRepo
	emailService      interfaces.IEmailService
	tx                *database.Transaction
	serviceSlotConfig config.ServiceSlotConfig
}

func NewServiceSlotService(db *sql.DB, serviceSlotRepo interfaces.IServiceSlotRepo, serviceRepo interfaces.IServiceRepo, leaveRepo interfaces.ILeaveRepo, serviceSlotConfig config.ServiceSlotConfig, emailService ...interfaces.IEmailService) interfaces.IServiceSlotService {
	var emails interfaces.IEmailService
	if len(emailService) > 0 {
		emails = emailService[0]
	}
	return &serviceSlotService{
		serviceSlotRepo:   serviceSlotRepo,
		serviceRepo:       serviceRepo,
		leaveRepo:         leaveRepo,
		emailService:      emails,
		tx:                database.NewTransaction(db),
		serviceSlotConfig: serviceSlotConfig,
	}
}

func weekdayOf(date string) (string, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", err
	}
	return strings.ToLower(t.Weekday().String()), nil
}

var weekdayIndex = map[string]time.Weekday{
	"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
	"wednesday": time.Wednesday, "thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday,
}

// weekdayOccurrences returns the next `count` dates (from today, inclusive)
// that fall on the given weekday, formatted as YYYY-MM-DD.
func weekdayOccurrences(weekday string, count int) []string {
	target, ok := weekdayIndex[strings.ToLower(weekday)]
	if !ok {
		return nil
	}
	today := time.Now()
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	offset := (int(target) - int(start.Weekday()) + 7) % 7
	first := start.AddDate(0, 0, offset)

	dates := make([]string, 0, count)
	for i := 0; i < count; i++ {
		dates = append(dates, first.AddDate(0, 0, 7*i).Format("2006-01-02"))
	}
	return dates
}

// weekdayOccurrencesUntil returns every date on the given weekday from today
// through endDate inclusive. The owner picks the end date per series, so this
// is what actually bounds a series — weekdayOccurrences' fixed count only
// supplies the default when none was given.
func weekdayOccurrencesUntil(weekday string, endDate string) []string {
	target, ok := weekdayIndex[strings.ToLower(weekday)]
	if !ok {
		return nil
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil
	}
	today := time.Now()
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	offset := (int(target) - int(start.Weekday()) + 7) % 7
	first := start.AddDate(0, 0, offset)

	var dates []string
	for d := first; !d.After(end); d = d.AddDate(0, 0, 7) {
		dates = append(dates, d.Format("2006-01-02"))
	}
	return dates
}

func slotNotFound() error {
	return errs.ValidationErrors{{Field: "serviceSlotId", Message: "Service slot not found."}}
}

// scheduledDate pairs a concrete date with the weekday it falls on, so a
// slot inserted for that date can be attached to the right weekday's
// recurring_schedule_id.
type scheduledDate struct {
	Date    string
	Weekday string
}

// resolveSchedule expands a slot param into the concrete dates to create and
// the weekdays they cover — a single date, or every occurrence of each
// selected weekday over the next recurringHorizonWeeks. This is the only time
// a series' occurrences are ever generated: what's created here is what the
// series will ever have.
func (s *serviceSlotService) resolveSchedule(p param.ServiceSlotParam) ([]scheduledDate, []string, error) {
	if len(p.DaysOfWeek) > 0 {
		// An explicit end date wins; without one the series runs for the
		// configured default number of weeks.
		endDate := strings.TrimSpace(p.RecurringEndDate)
		var scheduled []scheduledDate
		for _, wd := range p.DaysOfWeek {
			var dates []string
			if endDate != "" {
				dates = weekdayOccurrencesUntil(wd, endDate)
			} else {
				dates = weekdayOccurrences(wd, s.serviceSlotConfig.RecurringHorizonWeeks)
			}
			for _, date := range dates {
				scheduled = append(scheduled, scheduledDate{Date: date, Weekday: wd})
			}
		}
		// Only when an end date was actually supplied: an end date too close
		// to today can select none of the chosen weekdays, and saying so is
		// far clearer than the generic "no valid dates" further down. Without
		// one, zero dates can only mean an unrecognised weekday, which keeps
		// its existing handling.
		if endDate != "" && len(scheduled) == 0 {
			return nil, nil, errs.ValidationErrors{{
				Field:   "recurringEndDate",
				Message: "The end date is too early. Please select a later date.",
			}}
		}
		return scheduled, p.DaysOfWeek, nil
	}
	wd, err := weekdayOf(p.Date)
	if err != nil {
		return nil, nil, errs.ValidationErrors{{Field: "date", Message: "Invalid date"}}
	}
	return []scheduledDate{{Date: p.Date, Weekday: wd}}, []string{wd}, nil
}

// validateCoverage checks that the requested time is actually worked on every
// given weekday — by the assigned staff if one is set, otherwise by the
// business's own working hours (owner-managed). Without this, an
// owner-managed slot could be created on a weekday nobody works, since
// there'd be no staff record to check against at all.
func (s *serviceSlotService) validateCoverage(ctx context.Context, staffID *int64, businessID int64, weekdays []string, startTime, endTime time.Time) error {
	if staffID == nil {
		for _, wd := range weekdays {
			covers, err := s.serviceSlotRepo.BusinessCoversTime(ctx, businessID, wd, startTime, endTime)
			if err != nil {
				return err
			}
			if !covers {
				return errs.ValidationErrors{{Field: "startTime", Message: "The business is not working on " + wd + " during the selected time."}}
			}
		}
		return nil
	}

	staffOK, err := s.serviceSlotRepo.StaffBelongsToBusiness(ctx, *staffID, businessID)
	if err != nil {
		return err
	}
	if !staffOK {
		return errs.ValidationErrors{{Field: "staffId", Message: "Selected staff was not found."}}
	}
	for _, wd := range weekdays {
		covers, err := s.serviceSlotRepo.StaffCoversTime(ctx, *staffID, wd, startTime, endTime)
		if err != nil {
			return err
		}
		if !covers {
			return errs.ValidationErrors{{Field: "startTime", Message: "Staff is not working on " + wd + " during the selected time."}}
		}
	}
	return nil
}

// validateNoLeaveConflict checks that the assigned staff (if any) isn't on
// approved leave on any of the scheduled dates. Leave is date-only (no
// time-of-day component — see leave_applications), so unlike validateCoverage
// this needs the concrete dates, not just the weekdays: a weekday-recurring
// series' occurrences fall on many different calendar dates, any of which
// could individually be covered by an approved leave even though the weekday
// itself is normally worked. Owner-managed slots (staffID nil) have no staff
// to be on leave, so there's nothing to check.
//
// Returned as a plain (non-field) error, unlike validateCoverage's errors —
// it isn't really about the time picker specifically, so it's shown as a
// general form message rather than pinned to one field's feedback text.
func (s *serviceSlotService) validateNoLeaveConflict(ctx context.Context, staffID *int64, scheduled []scheduledDate) error {
	if staffID == nil {
		return nil
	}
	for _, sd := range scheduled {
		onLeave, err := s.leaveRepo.IsStaffOnLeave(ctx, *staffID, sd.Date)
		if err != nil {
			return err
		}
		if onLeave {
			return fmt.Errorf("This staff member is on approved leave on %s — pick a different date or staff member.", sd.Date)
		}
	}
	return nil
}

// insertSlotsForSchedule inserts one service_slots row (plus its packages)
// per scheduled date and returns the first inserted ID. recurringScheduleIDs
// maps weekday -> recurring_schedule_id (nil map for a non-recurring slot);
// each inserted slot is attached to its weekday's series, if any.
//
// isRecurring distinguishes a weekday series (spanning many weeks) from a
// single-date slot. For a recurring occurrence whose date no selected option
// actually covers — before an "upcoming" option's effectiveFrom, or after one
// with an effectiveUntil — the occurrence is skipped entirely rather than
// creating an empty, unbookable slot for every remaining week in the
// horizon. A single-date slot keeps the old behaviour (insert with whatever
// options remain, even zero) since its one date was already chosen deliberately.
func (s *serviceSlotService) insertSlotsForSchedule(
	ctx context.Context, tx *sql.Tx, p param.ServiceSlotParam, scheduled []scheduledDate, recurringScheduleIDs map[string]int64, isRecurring bool,
) (int64, error) {
	var createdID int64
	for _, sd := range scheduled {
		// A backdated walk-in resolves against its own date like anything
		// else: it's a record of what was actually sold that day, so the
		// option must have been in effect then — not merely still on the
		// menu now. Judging by today's catalog got this wrong both ways,
		// admitting options that hadn't started yet and rejecting ones that
		// had since ended.
		optionIDs, err := s.resolveOptionsForDate(ctx, p.ServiceOptionIDs, sd.Date)
		if err != nil {
			return 0, err
		}
		// A recurring occurrence with no effective option is simply skipped —
		// the series continues on later dates where an option does apply.
		// Everything else (a single slot, walk-in or not) must never be left
		// with zero bookable options: fail loudly instead of creating an
		// optionless slot the caller can't book against. Inside the
		// transaction, so nothing is left behind either way.
		if len(optionIDs) == 0 {
			if isRecurring {
				continue
			}
			if p.AllowPast {
				return 0, errs.ValidationErrors{{
					Field:   "serviceOptionIds",
					Message: "That service option wasn't offered on " + sd.Date + ". Pick one that was available then.",
				}}
			}
			return 0, errs.ValidationErrors{{
				Field:   "serviceOptionIds",
				Message: "None of the selected options are offered on " + sd.Date + ".",
			}}
		}

		slot := p
		slot.Date = sd.Date
		slot.RecurringScheduleID = nil
		if id, ok := recurringScheduleIDs[sd.Weekday]; ok {
			slot.RecurringScheduleID = &id
		}

		id, err := s.serviceSlotRepo.InsertServiceSlot(ctx, tx, slot)
		if err != nil {
			return 0, err
		}
		if err := s.insertSlotPackages(ctx, tx, id, optionIDs); err != nil {
			return 0, err
		}
		if createdID == 0 {
			createdID = id
		}
	}
	return createdID, nil
}

// resolveOptionsForDate drops any submitted option id whose own validity
// window doesn't cover the SLOT'S OWN DATE. The frontend's checkbox list
// reflects what's effective around "now" (or the single date it knows
// about), which can diverge from a specific occurrence date — especially for
// weekday-recurring slots spanning weeks. Without this, a slot could end up
// offering an option that isn't actually valid on its date (see
// bookingService.GetAvailableSlots, which checks the same window at booking
// time), making the slot silently unbookable for that option. An option not
// covering the date is simply dropped for that occurrence rather than
// failing the whole slot.
func (s *serviceSlotService) resolveOptionsForDate(ctx context.Context, optionIDs []int64, date string) ([]int64, error) {
	resolved := make([]int64, 0, len(optionIDs))
	for _, id := range optionIDs {
		ok, err := s.serviceRepo.IsOptionEffectiveOn(ctx, id, date)
		if err != nil {
			return nil, err
		}
		if ok {
			resolved = append(resolved, id)
		}
	}
	return resolved, nil
}

// insertRecurringSchedules records one (staff-or-owner-managed, day) series
// per selected weekday and returns a weekday -> recurring_schedule_id lookup
// for insertSlotsForSchedule to attach to each generated slot — including
// owner-managed slots (p.StaffID nil), so their series is renewable too (see
// deletable-as-a-series (see DeleteServiceSlot's deleteFutureRecurring).
func (s *serviceSlotService) insertRecurringSchedules(ctx context.Context, tx *sql.Tx, p param.ServiceSlotParam, weekdays []string) (map[string]int64, error) {
	ids := make(map[string]int64, len(weekdays))
	for _, wd := range weekdays {
		id, err := s.serviceSlotRepo.InsertRecurringSchedule(ctx, tx, p, wd)
		if err != nil {
			return nil, err
		}
		ids[wd] = id
	}
	return ids, nil
}

func (s *serviceSlotService) CreateServiceSlot(ctx context.Context, p param.ServiceSlotParam) (*param.ServiceSlotParam, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	packagesOK, err := s.serviceSlotRepo.OptionsBelongToBusiness(ctx, p.BusinessID, p.ServiceOptionIDs)
	if err != nil {
		return nil, err
	}
	if !packagesOK {
		return nil, errs.ValidationErrors{{Field: "serviceOptionIds", Message: "One or more selected options are invalid."}}
	}

	scheduled, weekdays, err := s.resolveSchedule(p)
	if err != nil {
		return nil, err
	}

	if err := s.validateCoverage(ctx, p.StaffID, p.BusinessID, weekdays, p.StartTime, p.EndTime); err != nil {
		return nil, err
	}

	if err := s.validateNoLeaveConflict(ctx, p.StaffID, scheduled); err != nil {
		return nil, err
	}

	var createdID int64
	err = s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		var recurringIDs map[string]int64
		if len(p.DaysOfWeek) > 0 {
			recurringIDs, err = s.insertRecurringSchedules(ctx, tx, p, weekdays)
			if err != nil {
				return err
			}
		}

		id, err := s.insertSlotsForSchedule(ctx, tx, p, scheduled, recurringIDs, len(p.DaysOfWeek) > 0)
		if err != nil {
			return err
		}
		createdID = id
		return nil
	})
	if err != nil {
		return nil, err
	}

	if createdID == 0 {
		return nil, errs.ValidationErrors{{Field: "schedule", Message: "No valid dates were resolved for the selected schedule."}}
	}

	return s.serviceSlotRepo.GetServiceSlotByID(ctx, createdID, p.BusinessID)
}

func (s *serviceSlotService) insertSlotPackages(ctx context.Context, tx *sql.Tx, slotID int64, packageIDs []int64) error {
	for _, pkgID := range packageIDs {
		if err := s.serviceSlotRepo.InsertServiceSlotOption(ctx, tx, slotID, pkgID); err != nil {
			return err
		}
	}
	return nil
}

// UpdateServiceSlot edits a single occurrence's date, time, staff and
// packages. Once it has a booking, nothing here can change anymore — only
// staff reassignment (ReassignServiceSlotStaff) remains available for it.
//
// staffScope, when set (a staff member rather than the owner is acting),
// requires the slot to currently belong to that staff ID and forces the
// edited slot to stay assigned to them — a staff member can edit their own
// slots but can't reassign them away or touch anyone else's.
func (s *serviceSlotService) UpdateServiceSlot(ctx context.Context, p param.ServiceSlotParam, staffScope *int64) (*param.ServiceSlotParam, error) {
	existing, err := s.serviceSlotRepo.GetServiceSlotByID(ctx, p.ServiceSlotID, p.BusinessID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, slotNotFound()
		}
		return nil, err
	}

	if staffScope != nil {
		if existing.StaffID == nil || *existing.StaffID != *staffScope {
			return nil, slotNotFound()
		}
		p.StaffID = staffScope
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	hasBooking, err := s.serviceSlotRepo.HasBookingForServiceSlot(ctx, p.ServiceSlotID)
	if err != nil {
		return nil, err
	}
	if hasBooking {
		return nil, errs.ValidationErrors{{Field: "serviceSlotId", Message: "This occurrence already has a booking — only staff reassignment is allowed for it."}}
	}

	packagesOK, err := s.serviceSlotRepo.OptionsBelongToBusiness(ctx, p.BusinessID, p.ServiceOptionIDs)
	if err != nil {
		return nil, err
	}
	if !packagesOK {
		return nil, errs.ValidationErrors{{Field: "serviceOptionIds", Message: "One or more selected options are invalid."}}
	}

	scheduled, weekdays, err := s.resolveSchedule(p)
	if err != nil {
		return nil, err
	}

	if err := s.validateCoverage(ctx, p.StaffID, p.BusinessID, weekdays, p.StartTime, p.EndTime); err != nil {
		return nil, err
	}

	if err := s.validateNoLeaveConflict(ctx, p.StaffID, scheduled); err != nil {
		return nil, err
	}

	var newID int64
	err = s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		if err := s.serviceSlotRepo.SoftDeleteServiceSlots(ctx, tx, []int64{p.ServiceSlotID}, p.BusinessID); err != nil {
			return err
		}

		id, err := s.insertSlotsForSchedule(ctx, tx, p, scheduled, nil, false)
		if err != nil {
			return err
		}
		newID = id
		return nil
	})
	if err != nil {
		return nil, err
	}

	if newID == 0 {
		return nil, errs.ValidationErrors{{Field: "schedule", Message: "No valid dates were resolved for the selected schedule."}}
	}

	return s.serviceSlotRepo.GetServiceSlotByID(ctx, newID, p.BusinessID)
}

func (s *serviceSlotService) ReassignServiceSlotStaff(ctx context.Context, serviceSlotID int64, businessID int64, staffID *int64) (*param.ServiceSlotParam, error) {
	slot, err := s.serviceSlotRepo.GetServiceSlotByID(ctx, serviceSlotID, businessID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, slotNotFound()
		}
		return nil, err
	}

	weekday, err := weekdayOf(slot.Date)
	if err != nil {
		return nil, err
	}

	// staffID nil => unassign (owner-managed); otherwise validate the new staff.
	if staffID != nil {
		staffOK, err := s.serviceSlotRepo.StaffBelongsToBusiness(ctx, *staffID, businessID)
		if err != nil {
			return nil, err
		}
		if !staffOK {
			return nil, errs.ValidationErrors{{Field: "staffId", Message: "Selected staff was not found."}}
		}

		covers, err := s.serviceSlotRepo.StaffCoversTime(ctx, *staffID, weekday, slot.StartTime, slot.EndTime)
		if err != nil {
			return nil, err
		}
		if !covers {
			return nil, errs.ValidationErrors{{Field: "staffId", Message: "Staff is not working during the selected time."}}
		}

		onLeave, err := s.leaveRepo.IsStaffOnLeave(ctx, *staffID, slot.Date)
		if err != nil {
			return nil, err
		}
		if onLeave {
			return nil, errs.ValidationErrors{{Field: "staffId", Message: "This staff member is on approved leave on " + slot.Date + " — pick a different staff member."}}
		}
	}

	err = s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		return s.serviceSlotRepo.ReassignStaff(ctx, tx, serviceSlotID, businessID, staffID)
	})
	if err != nil {
		return nil, err
	}

	return s.serviceSlotRepo.GetServiceSlotByID(ctx, serviceSlotID, businessID)
}

// staffScope, when set (a staff member rather than the owner is acting),
// requires the slot to currently belong to that staff ID — anything else is
// treated as not found rather than leaking its existence.
func (s *serviceSlotService) DeleteServiceSlot(ctx context.Context, serviceSlotID int64, businessID int64, deleteFutureRecurring bool, staffScope *int64) error {
	slot, err := s.serviceSlotRepo.GetServiceSlotByID(ctx, serviceSlotID, businessID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return slotNotFound()
		}
		return err
	}

	if staffScope != nil {
		if slot.StaffID == nil || *slot.StaffID != *staffScope {
			return slotNotFound()
		}
	}

	affectedIDs := []int64{serviceSlotID}
	if deleteFutureRecurring && slot.RecurringScheduleID != nil {
		// "Future" is relative to the specific occurrence the owner opened,
		// not today — deleting "this and future" from an occurrence three
		// weeks out only removes that date onward, leaving anything between
		// now and then untouched.
		ids, err := s.serviceSlotRepo.GetFutureRecurringSlotIDs(ctx, businessID, *slot.RecurringScheduleID, slot.Date)
		if err != nil {
			return err
		}
		if len(ids) > 0 {
			affectedIDs = ids
		}
	}

	selectedHasBooking, err := s.serviceSlotRepo.HasBookingForServiceSlot(ctx, serviceSlotID)
	if err != nil {
		return err
	}
	if selectedHasBooking {
		return errs.ValidationErrors{{Field: "serviceSlotId", Message: "This slot already has a booking and cannot be deleted."}}
	}

	deletableIDs := make([]int64, 0, len(affectedIDs))
	var skippedIDs []int64
	for _, id := range affectedIDs {
		if id == serviceSlotID {
			deletableIDs = append(deletableIDs, id)
			continue
		}
		hasBooking, err := s.serviceSlotRepo.HasBookingForServiceSlot(ctx, id)
		if err != nil {
			return err
		}
		if hasBooking {
			skippedIDs = append(skippedIDs, id)
			continue
		}
		deletableIDs = append(deletableIDs, id)
	}

	// deletableIDs is by construction never a slot with an active (pending,
	// accepted, or rescheduled) booking — HasBookingForServiceSlot above
	// already routed any such slot into skippedIDs instead.
	err = s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		if !deleteFutureRecurring {
			return s.serviceSlotRepo.SoftDeleteServiceSlot(ctx, tx, serviceSlotID, businessID)
		}

		if err := s.serviceSlotRepo.SoftDeleteServiceSlots(ctx, tx, deletableIDs, businessID); err != nil {
			return err
		}
		if slot.RecurringScheduleID != nil {
			// Retire the series here regardless of any booked occurrence(s)
			// left in place, so nothing keeps pointing at a series the owner
			// has deleted.
			return s.serviceSlotRepo.SoftDeleteRecurringSchedule(ctx, tx, businessID, *slot.RecurringScheduleID)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(skippedIDs) > 0 {
		dates, err := s.serviceSlotRepo.GetSlotDates(ctx, skippedIDs)
		if err != nil {
			return err
		}
		return errs.ValidationErrors{{
			Field: "deleteFutureRecurring",
			Message: fmt.Sprintf(
				"Deleted available slots. Kept — already booked: %s.",
				strings.Join(dates, ", "),
			),
		}}
	}
	return nil
}

func (s *serviceSlotService) GetServiceSlots(ctx context.Context, businessID int64, date string, staffID *int64, serviceID *int64, unassignedOnly bool) ([]param.ServiceSlotParam, error) {
	return s.serviceSlotRepo.GetServiceSlotsByBusinessAndDate(ctx, businessID, date, staffID, serviceID, unassignedOnly)
}

func (s *serviceSlotService) GetAvailableStaffForSlot(ctx context.Context, serviceSlotID int64, businessID int64) ([]param.StaffParam, error) {
	slot, err := s.serviceSlotRepo.GetServiceSlotByID(ctx, serviceSlotID, businessID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, slotNotFound()
		}
		return nil, err
	}
	weekday, err := weekdayOf(slot.Date)
	if err != nil {
		return nil, err
	}
	return s.serviceSlotRepo.GetAvailableStaff(ctx, businessID, slot.Date, weekday, slot.StartTime, slot.EndTime, slot.ServiceSlotID)
}
