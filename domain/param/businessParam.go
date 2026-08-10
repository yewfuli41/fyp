package param

import (
	"fmt"
	"fyp/domain/errs"
	"strings"
	"time"
)

type WorkingHourParam struct {
	Day       string
	StartTime time.Time
	EndTime   time.Time
}

// ValidateWorkingHoursShape checks a working-hours list is internally
// consistent — non-empty, each entry start < end, and no two entries on the
// same day overlap. Shared by staff registration and any later edit to a
// staff's (or business's) working hours.
func ValidateWorkingHoursShape(workingHours []WorkingHourParam) errs.ValidationErrors {
	var validationErrs errs.ValidationErrors

	if len(workingHours) == 0 {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "workingHours", Message: "At least one working hour is required"})
	}

	for i, wh := range workingHours {
		if !wh.StartTime.Before(wh.EndTime) {
			validationErrs = append(validationErrs, errs.ValidationError{
				Field:   fmt.Sprintf("workingHours[%d]", i),
				Message: "Start time must be before end time",
			})
		}
	}

	byDay := make(map[string][]int)
	for i, wh := range workingHours {
		byDay[wh.Day] = append(byDay[wh.Day], i)
	}
	overlapping := make(map[int]bool)
	for _, indices := range byDay {
		for i := 0; i < len(indices); i++ {
			for j := i + 1; j < len(indices); j++ {
				a := workingHours[indices[i]]
				b := workingHours[indices[j]]
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

	return validationErrs
}

type BusinessProfileParam struct {
	BusinessID            int64
	OwnerUserID           int64
	BusinessName          string
	Description           *string
	Address               string
	ImageURL              *string
	BusinessContactNumber string
	BusinessEmail         string
	WorkingHours          []WorkingHourParam
}

func (p BusinessProfileParam) ValidateRegisterBusinessProfile() error {
	var validationErrs errs.ValidationErrors

	if strings.TrimSpace(p.BusinessName) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "businessName", Message: "Business name is required"})
	} else if e, ok := maxLengthError("businessName", "business name", p.BusinessName, maxBusinessNameLength); ok {
		validationErrs = append(validationErrs, e)
	}

	if strings.TrimSpace(p.Address) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "address", Message: "Address is required"})
	}

	if strings.TrimSpace(p.BusinessContactNumber) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "businessContactNumber", Message: "Business contact number is required"})
	} else if len(strings.TrimSpace(p.BusinessContactNumber)) < minContactNumberDigits {
		validationErrs = append(validationErrs, errs.ValidationError{
			Field:   "businessContactNumber",
			Message: fmt.Sprintf("Business contact number must have at least %d digits", minContactNumberDigits),
		})
	} else if e, ok := maxLengthError("businessContactNumber", "business contact number", p.BusinessContactNumber, maxContactNumberLength); ok {
		validationErrs = append(validationErrs, e)
	}

	if strings.TrimSpace(p.BusinessEmail) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "businessEmail", Message: "Business email is required"})
	} else if !emailRegex.MatchString(strings.TrimSpace(p.BusinessEmail)) {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "businessEmail", Message: "Business email format is invalid"})
	} else if e, ok := maxLengthError("businessEmail", "business email", p.BusinessEmail, maxEmailLength); ok {
		validationErrs = append(validationErrs, e)
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
