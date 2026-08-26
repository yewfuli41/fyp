package param

import (
	"fyp/domain/errs"
	"strings"
	"time"
)

// SlotTierParam is a service package made bookable within a service slot,
// carrying the parent service name for display.
type SlotTierParam struct {
	SlotOptionID      int64
	ServiceOptionID   int64
	ServiceOptionName string
	ServiceID         int64
	ServiceName       string
}

type ServiceSlotParam struct {
	ServiceSlotID       int64
	BusinessID          int64
	StaffID             *int64 // nil => owner-managed (no assigned staff)
	StaffName           string
	RecurringScheduleID *int64   // nil => a one-off slot, not part of any weekday series
	Date                string   // "YYYY-MM-DD" — single specific date (mutually exclusive with DaysOfWeek)
	DaysOfWeek          []string // recurring weekdays, e.g. ["monday","wednesday"]
	// RecurringEndDate ("YYYY-MM-DD") is the last date a weekday series
	// generates an occurrence for. Only meaningful alongside DaysOfWeek.
	// Occurrences are generated once, at creation, so this is the series'
	// full extent — it is never topped up later.
	RecurringEndDate string
	StartTime        time.Time
	EndTime          time.Time
	CreatedBy        int64
	HasBooking       bool // loaded for display — true once a customer has booked this slot

	ServiceOptionIDs []int64         // input: packages to make bookable
	Packages         []SlotTierParam // loaded for display

	// AllowPast skips the "date and time cannot be in the past" check below —
	// set by RecordWalkIn, which creates a slot for something that's already
	// happened (the walk-in is being logged after the fact), unlike an
	// ordinary slot which is always for a future booking.
	AllowPast bool
}

func (p ServiceSlotParam) Validate() error {
	var validationErrs errs.ValidationErrors

	if len(p.ServiceOptionIDs) == 0 {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "serviceOptionIds", Message: "Please select a service and its option"})
	}
	if !p.StartTime.Before(p.EndTime) {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "startTime", Message: "Start time must be before end time"})
	}

	hasDate := strings.TrimSpace(p.Date) != ""
	hasDays := len(p.DaysOfWeek) > 0
	if !hasDate && !hasDays {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "schedule", Message: "Please select days of the week or a date"})
	} else if hasDate && hasDays {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "schedule", Message: "Choose either weekdays or a single date, not both"})
	} else if hasDays {
		// Optional: an omitted end date falls back to the configured default
		// in resolveSchedule. A supplied one must be a real, future date —
		// a series ending today or earlier would generate nothing at all.
		if end := strings.TrimSpace(p.RecurringEndDate); end != "" {
			parsed, err := time.Parse("2006-01-02", end)
			if err != nil {
				validationErrs = append(validationErrs, errs.ValidationError{Field: "recurringEndDate", Message: "Invalid end date"})
			} else {
				now := time.Now()
				today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
				if parsed.Before(today) {
					validationErrs = append(validationErrs, errs.ValidationError{Field: "recurringEndDate", Message: "End date cannot be in the past"})
				}
			}
		}
	} else if hasDate {
		if parsed, err := time.Parse("2006-01-02", p.Date); err != nil {
			validationErrs = append(validationErrs, errs.ValidationError{Field: "date", Message: "Invalid date"})
		} else if !p.AllowPast {
			slotStart := time.Date(
				parsed.Year(), parsed.Month(), parsed.Day(),
				p.StartTime.Hour(), p.StartTime.Minute(), p.StartTime.Second(), 0,
				time.UTC,
			)
			if slotStart.Before(time.Now()) {
				validationErrs = append(validationErrs, errs.ValidationError{Field: "date", Message: "Date and time cannot be in the past"})
			}
		}
	}

	if len(validationErrs) > 0 {
		return validationErrs
	}
	return nil
}

// SlotWindowParam is a minimal (date, time window) projection of a slot —
// used to check a working-hours edit against existing occurrences without
// loading full slot details.
type SlotWindowParam struct {
	Date      string // "YYYY-MM-DD"
	StartTime time.Time
	EndTime   time.Time
}

// AssignedSlotParam is a staff-assigned slot's identity + window + booking
// status — used to detect and resolve conflicts when a staff member goes on
// leave or their working hours shrink (see leaveService and
// staffService.UpdateStaffWorkingHours).
type AssignedSlotParam struct {
	ServiceSlotID int64
	Date          string // "YYYY-MM-DD"
	StartTime     time.Time
	EndTime       time.Time
	HasBooking    bool
}

// SlotReassignmentParam picks a replacement staff (or unassigns, when
// StaffID is nil) for one affected service slot.
type SlotReassignmentParam struct {
	ServiceSlotID int64
	StaffID       *int64
}
