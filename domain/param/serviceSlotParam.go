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
	ServiceID          int64
	ServiceName        string
}

type ServiceSlotParam struct {
	ServiceSlotID       int64
	BusinessID          int64
	StaffID             *int64 // nil => owner-managed (no assigned staff)
	StaffName           string
	RecurringScheduleID *int64   // nil => a one-off slot, not part of any weekday series
	Date                string   // "YYYY-MM-DD" — single specific date (mutually exclusive with DaysOfWeek)
	DaysOfWeek          []string // recurring weekdays, e.g. ["monday","wednesday"]
	StartTime           time.Time
	EndTime             time.Time
	CreatedBy           int64
	HasBooking          bool // loaded for display — true once a customer has booked this slot

	ServiceOptionIDs []int64            // input: packages to make bookable
	Packages          []SlotTierParam // loaded for display
}

func (p ServiceSlotParam) Validate() error {
	var validationErrs errs.ValidationErrors

	if len(p.ServiceOptionIDs) == 0 {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "serviceOptionIds", Message: "Please complete required fields"})
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
	} else if hasDate {
		if parsed, err := time.Parse("2006-01-02", p.Date); err != nil {
			validationErrs = append(validationErrs, errs.ValidationError{Field: "date", Message: "Invalid date"})
		} else {
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
