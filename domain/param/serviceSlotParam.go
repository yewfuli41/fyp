package param

import (
	"fyp/domain/errs"
	"strings"
	"time"
)

// SlotPackageParam is a service package made bookable within a service slot,
// carrying the parent service name for display.
type SlotPackageParam struct {
	SlotPackageID      int64
	ServicePackageID   int64
	ServicePackageName string
	ServiceID          int64
	ServiceName        string
}

type ServiceSlotParam struct {
	ServiceSlotID int64
	BusinessID    int64
	StaffID       *int64 // nil => owner-managed (no assigned staff)
	StaffName     string
	Date          string   // "YYYY-MM-DD" — single specific date (mutually exclusive with DaysOfWeek)
	DaysOfWeek    []string // recurring weekdays, e.g. ["monday","wednesday"]
	StartTime     time.Time
	EndTime       time.Time
	CreatedBy     int64

	ServicePackageIDs []int64            // input: packages to make bookable
	Packages          []SlotPackageParam // loaded for display
}

func (p ServiceSlotParam) IsRecurring() bool {
	return len(p.DaysOfWeek) > 0
}

func (p ServiceSlotParam) Validate() error {
	var validationErrs errs.ValidationErrors

	if len(p.ServicePackageIDs) == 0 {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "servicePackageIds", Message: "Please complete required fields"})
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
	}

	if len(validationErrs) > 0 {
		return validationErrs
	}
	return nil
}
