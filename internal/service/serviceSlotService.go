package service

import (
	"context"
	"database/sql"
	"errors"
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
	tx                *database.Transaction
	serviceSlotConfig config.ServiceSlotConfig
}

func NewServiceSlotService(db *sql.DB, serviceSlotRepo interfaces.IServiceSlotRepo, serviceSlotConfig config.ServiceSlotConfig) interfaces.IServiceSlotService {
	return &serviceSlotService{
		serviceSlotRepo:   serviceSlotRepo,
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
// selected weekday over the next recurringHorizonWeeks.
func (s *serviceSlotService) resolveSchedule(p param.ServiceSlotParam) ([]scheduledDate, []string, error) {
	if len(p.DaysOfWeek) > 0 {
		var scheduled []scheduledDate
		for _, wd := range p.DaysOfWeek {
			for _, date := range weekdayOccurrences(wd, s.serviceSlotConfig.RecurringHorizonWeeks) {
				scheduled = append(scheduled, scheduledDate{Date: date, Weekday: wd})
			}
		}
		return scheduled, p.DaysOfWeek, nil
	}
	wd, err := weekdayOf(p.Date)
	if err != nil {
		return nil, nil, errs.ValidationErrors{{Field: "date", Message: "Invalid date"}}
	}
	return []scheduledDate{{Date: p.Date, Weekday: wd}}, []string{wd}, nil
}

// validateStaffCoverage checks that an (optional) staff belongs to the
// business and works every given weekday during the requested time. No staff
// means owner-managed, which always passes.
func (s *serviceSlotService) validateStaffCoverage(ctx context.Context, staffID *int64, businessID int64, weekdays []string, startTime, endTime time.Time) error {
	if staffID == nil {
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

// insertSlotsForSchedule inserts one service_slots row (plus its packages)
// per scheduled date and returns the first inserted ID. recurringScheduleIDs
// maps weekday -> recurring_schedule_id (nil map for a non-recurring slot);
// each inserted slot is attached to its weekday's series, if any.
func (s *serviceSlotService) insertSlotsForSchedule(
	ctx context.Context, tx *sql.Tx, p param.ServiceSlotParam, scheduled []scheduledDate, recurringScheduleIDs map[string]int64,
) (int64, error) {
	var createdID int64
	for _, sd := range scheduled {
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
		if err := s.insertSlotPackages(ctx, tx, id, slot.ServicePackageIDs); err != nil {
			return 0, err
		}
		if createdID == 0 {
			createdID = id
		}
	}
	return createdID, nil
}

// insertRecurringSchedules records one (staff, day) series per selected
// weekday (recurring_schedules.staff_id is NOT NULL, so owner-managed slots
// never get rows here) and returns a weekday -> recurring_schedule_id lookup
// for insertSlotsForSchedule to attach to each generated slot.
func (s *serviceSlotService) insertRecurringSchedules(ctx context.Context, tx *sql.Tx, p param.ServiceSlotParam, weekdays []string) (map[string]int64, error) {
	if p.StaffID == nil {
		return nil, nil
	}
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

	packagesOK, err := s.serviceSlotRepo.PackagesBelongToBusiness(ctx, p.BusinessID, p.ServicePackageIDs)
	if err != nil {
		return nil, err
	}
	if !packagesOK {
		return nil, errs.ValidationErrors{{Field: "servicePackageIds", Message: "One or more selected packages are invalid."}}
	}

	scheduled, weekdays, err := s.resolveSchedule(p)
	if err != nil {
		return nil, err
	}

	if err := s.validateStaffCoverage(ctx, p.StaffID, p.BusinessID, weekdays, p.StartTime, p.EndTime); err != nil {
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

		id, err := s.insertSlotsForSchedule(ctx, tx, p, scheduled, recurringIDs)
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
		if err := s.serviceSlotRepo.InsertServiceSlotPackage(ctx, tx, slotID, pkgID); err != nil {
			return err
		}
	}
	return nil
}

// UpdateServiceSlot edits a slot's date/days-of-week, time, staff and
// packages. Once any affected occurrence has a booking, nothing here can
// change anymore — only staff reassignment (ReassignServiceSlotStaff) remains
// available for it.
//
// applyToFutureRecurring mirrors DeleteServiceSlot's flag: false edits just
// the opened occurrence; true edits it plus every future occurrence sharing
// its staff/time/weekday (the same series DeleteServiceSlot's "delete future
// recurring" targets) — those occurrences are replaced with the new schedule
// (which may itself be recurring across different weekdays, a single date, a
// different staff, etc).
//
// staffScope, when set (a staff member rather than the owner is acting),
// requires the slot to currently belong to that staff ID and forces the
// edited slot to stay assigned to them — a staff member can edit their own
// slots but can't reassign them away or touch anyone else's.
func (s *serviceSlotService) UpdateServiceSlot(ctx context.Context, p param.ServiceSlotParam, applyToFutureRecurring bool, staffScope *int64) (*param.ServiceSlotParam, error) {
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

	affectedIDs := []int64{p.ServiceSlotID}
	if applyToFutureRecurring && existing.RecurringScheduleID != nil {
		ids, err := s.serviceSlotRepo.GetFutureRecurringSlotIDs(ctx, p.BusinessID, *existing.RecurringScheduleID, existing.Date)
		if err != nil {
			return nil, err
		}
		if len(ids) > 0 {
			affectedIDs = ids
		}
	}

	for _, id := range affectedIDs {
		hasBooking, err := s.serviceSlotRepo.HasBookingForServiceSlot(ctx, id)
		if err != nil {
			return nil, err
		}
		if hasBooking {
			return nil, errs.ValidationErrors{{Field: "serviceSlotId", Message: "One or more occurrences already have a booking — only staff reassignment is allowed for those."}}
		}
	}

	packagesOK, err := s.serviceSlotRepo.PackagesBelongToBusiness(ctx, p.BusinessID, p.ServicePackageIDs)
	if err != nil {
		return nil, err
	}
	if !packagesOK {
		return nil, errs.ValidationErrors{{Field: "servicePackageIds", Message: "One or more selected packages are invalid."}}
	}

	scheduled, weekdays, err := s.resolveSchedule(p)
	if err != nil {
		return nil, err
	}

	if err := s.validateStaffCoverage(ctx, p.StaffID, p.BusinessID, weekdays, p.StartTime, p.EndTime); err != nil {
		return nil, err
	}

	var newID int64
	err = s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		if err := s.serviceSlotRepo.SoftDeleteServiceSlots(ctx, tx, affectedIDs, p.BusinessID); err != nil {
			return err
		}
		if applyToFutureRecurring && existing.RecurringScheduleID != nil {
			if err := s.serviceSlotRepo.SoftDeleteRecurringSchedule(ctx, tx, p.BusinessID, *existing.RecurringScheduleID); err != nil {
				return err
			}
		}

		var recurringIDs map[string]int64
		if len(p.DaysOfWeek) > 0 {
			recurringIDs, err = s.insertRecurringSchedules(ctx, tx, p, weekdays)
			if err != nil {
				return err
			}
		}

		id, err := s.insertSlotsForSchedule(ctx, tx, p, scheduled, recurringIDs)
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
		ids, err := s.serviceSlotRepo.GetFutureRecurringSlotIDs(ctx, businessID, *slot.RecurringScheduleID, slot.Date)
		if err != nil {
			return err
		}
		if len(ids) > 0 {
			affectedIDs = ids
		}
	}
	for _, id := range affectedIDs {
		hasBooking, err := s.serviceSlotRepo.HasBookingForServiceSlot(ctx, id)
		if err != nil {
			return err
		}
		if hasBooking {
			return errs.ValidationErrors{{Field: "serviceSlotId", Message: "One or more occurrences already have a booking — only staff reassignment is allowed for those."}}
		}
	}

	return s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		if !deleteFutureRecurring {
			return s.serviceSlotRepo.SoftDeleteServiceSlot(ctx, tx, serviceSlotID, businessID)
		}

		if err := s.serviceSlotRepo.SoftDeleteServiceSlots(ctx, tx, affectedIDs, businessID); err != nil {
			return err
		}
		if slot.RecurringScheduleID != nil {
			return s.serviceSlotRepo.SoftDeleteRecurringSchedule(ctx, tx, businessID, *slot.RecurringScheduleID)
		}
		return nil
	})
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
