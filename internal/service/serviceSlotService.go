package service

import (
	"context"
	"database/sql"
	"errors"
	"fyp/database"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"strings"
	"time"
)

const recurringHorizonWeeks = 12

type serviceSlotService struct {
	serviceSlotRepo interfaces.IServiceSlotRepo
	tx              *database.Transaction
}

func NewServiceSlotService(db *sql.DB, serviceSlotRepo interfaces.IServiceSlotRepo) interfaces.IServiceSlotService {
	return &serviceSlotService{
		serviceSlotRepo: serviceSlotRepo,
		tx:              database.NewTransaction(db),
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

	// Resolve the set of concrete dates to create and the weekdays they cover.
	var dates []string
	var weekdays []string
	if p.IsRecurring() {
		weekdays = p.DaysOfWeek
		for _, wd := range p.DaysOfWeek {
			dates = append(dates, weekdayOccurrences(wd, recurringHorizonWeeks)...)
		}
	} else {
		wd, err := weekdayOf(p.Date)
		if err != nil {
			return nil, errs.ValidationErrors{{Field: "date", Message: "Invalid date"}}
		}
		dates = []string{p.Date}
		weekdays = []string{wd}
	}

	// Staff is optional; when assigned, validate it belongs to the business and
	// works every selected weekday during the time. No staff => owner-managed.
	if p.StaffID != nil {
		staffOK, err := s.serviceSlotRepo.StaffBelongsToBusiness(ctx, *p.StaffID, p.BusinessID)
		if err != nil {
			return nil, err
		}
		if !staffOK {
			return nil, errs.ValidationErrors{{Field: "staffId", Message: "Selected staff was not found."}}
		}
		for _, wd := range weekdays {
			covers, err := s.serviceSlotRepo.StaffCoversTime(ctx, *p.StaffID, wd, p.StartTime, p.EndTime)
			if err != nil {
				return nil, err
			}
			if !covers {
				return nil, errs.ValidationErrors{{Field: "startTime", Message: "Staff is not working on " + wd + " during the selected time."}}
			}
		}
	}

	var createdID int64
	err = s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		for _, date := range dates {
			slot := p
			slot.Date = date

			// Skip dates where an assigned staff already has an overlapping slot.
			if slot.StaffID != nil {
				overlap, err := s.serviceSlotRepo.StaffHasOverlappingSlot(ctx, *slot.StaffID, date, slot.StartTime, slot.EndTime, 0)
				if err != nil {
					return err
				}
				if overlap {
					continue
				}
			}

			id, err := s.serviceSlotRepo.InsertServiceSlot(ctx, tx, slot)
			if err != nil {
				return err
			}
			if err := s.insertSlotPackages(ctx, tx, id, slot.ServicePackageIDs); err != nil {
				return err
			}
			if createdID == 0 {
				createdID = id
			}
		}

		// Record the recurrence for staff-assigned weekday series.
		if p.IsRecurring() && p.StaffID != nil {
			for _, wd := range weekdays {
				for _, pkgID := range p.ServicePackageIDs {
					if err := s.serviceSlotRepo.InsertRecurringSchedule(ctx, tx, p, pkgID, wd); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if createdID == 0 {
		return nil, errs.ValidationErrors{{Field: "startTime", Message: "The selected times overlap existing slots."}}
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

		overlap, err := s.serviceSlotRepo.StaffHasOverlappingSlot(ctx, *staffID, slot.Date, slot.StartTime, slot.EndTime, slot.ServiceSlotID)
		if err != nil {
			return nil, err
		}
		if overlap {
			return nil, errs.ValidationErrors{{Field: "staffId", Message: "Staff already has a slot during this time."}}
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

func (s *serviceSlotService) DeleteServiceSlot(ctx context.Context, serviceSlotID int64, businessID int64, deleteFutureRecurring bool) error {
	slot, err := s.serviceSlotRepo.GetServiceSlotByID(ctx, serviceSlotID, businessID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return slotNotFound()
		}
		return err
	}

	weekday, err := weekdayOf(slot.Date)
	if err != nil {
		return err
	}

	return s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		if !deleteFutureRecurring {
			return s.serviceSlotRepo.SoftDeleteServiceSlot(ctx, tx, serviceSlotID, businessID)
		}

		ids, err := s.serviceSlotRepo.GetFutureRecurringSlotIDs(ctx, businessID, slot.StaffID, slot.StartTime, slot.EndTime, slot.Date)
		if err != nil {
			return err
		}
		if err := s.serviceSlotRepo.SoftDeleteServiceSlots(ctx, tx, ids, businessID); err != nil {
			return err
		}
		// Only staff-assigned series have recurring_schedules rows.
		if slot.StaffID != nil {
			return s.serviceSlotRepo.SoftDeleteRecurringSchedules(ctx, tx, businessID, *slot.StaffID, weekday, slot.StartTime, slot.EndTime)
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
