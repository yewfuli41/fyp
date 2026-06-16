package param

import (
	"fmt"
	"fyp/domain/errs"
	"strings"
)

type StaffParam struct {
	StaffID            int64
	BusinessID         int64
	UserID             int64
	StaffName          string
	StaffContactNumber string
	Position           string
	WorkingHours       []WorkingHourParam //from businessParam.go
}

func (p StaffParam) ValidateRegisterStaff() error {
	var validationErrs errs.ValidationErrors

	if strings.TrimSpace(p.StaffName) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "staffName", Message: "Staff name is required"})
	} else if e, ok := maxLengthError("staffName", "staff name", p.StaffName, maxBusinessNameLength); ok {
		validationErrs = append(validationErrs, e)
	}

	if strings.TrimSpace(p.StaffContactNumber) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "staffContactNumber", Message: "Staff contact number is required"})
	} else if len(strings.TrimSpace(p.StaffContactNumber)) < minContactNumberDigits {
		validationErrs = append(validationErrs, errs.ValidationError{
			Field:   "staffContactNumber",
			Message: fmt.Sprintf("Staff contact number must have at least %d digits", minContactNumberDigits),
		})
	} else if e, ok := maxLengthError("staffContactNumber", "staff contact number", p.StaffContactNumber, maxContactNumberLength); ok {
		validationErrs = append(validationErrs, e)
	}

	if strings.TrimSpace(p.Position) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "position", Message: "Position is required"})
	}

	if len(p.WorkingHours) == 0 {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "workingHours", Message: "At least one working hour is required"})
	}

	for i, wh := range p.WorkingHours {
		if !wh.StartTime.Before(wh.EndTime) {
			validationErrs = append(validationErrs, errs.ValidationError{
				Field:   fmt.Sprintf("workingHours[%d]", i),
				Message: "Start time must be before end time",
			})
		}
	}

	byDay := make(map[string][]int)
	for i, wh := range p.WorkingHours {
		byDay[wh.Day] = append(byDay[wh.Day], i)
	}
	overlapping := make(map[int]bool)
	for _, indices := range byDay {
		for i := 0; i < len(indices); i++ {
			for j := i + 1; j < len(indices); j++ {
				a := p.WorkingHours[indices[i]]
				b := p.WorkingHours[indices[j]]
				if a.StartTime.Before(b.EndTime) && b.StartTime.Before(a.EndTime) {
					overlapping[indices[i]] = true
					overlapping[indices[j]] = true
				}
			}
		}
	}
	for i := range overlapping {
		validationErrs = append(validationErrs, errs.ValidationError{
			Field:   fmt.Sprintf("workingHours[%d]", i),
			Message: "Working hours overlap with another entry on the same day",
		})
	}

	if len(validationErrs) > 0 {
		return validationErrs
	}

	return nil
}
