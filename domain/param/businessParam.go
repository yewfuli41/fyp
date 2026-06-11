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
		validationErrs = append(validationErrs, errs.ValidationError{Field: "businessName", Message: "business name is required"})
	} else if e, ok := maxLengthError("businessName", "business name", p.BusinessName, maxBusinessNameLength); ok {
		validationErrs = append(validationErrs, e)
	}

	if strings.TrimSpace(p.Address) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "address", Message: "address is required"})
	}

	if strings.TrimSpace(p.BusinessContactNumber) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "businessContactNumber", Message: "business contact number is required"})
	} else if len(strings.TrimSpace(p.BusinessContactNumber)) < minContactNumberDigits {
		validationErrs = append(validationErrs, errs.ValidationError{
			Field:   "businessContactNumber",
			Message: fmt.Sprintf("business contact number must have at least %d digits", minContactNumberDigits),
		})
	} else if e, ok := maxLengthError("businessContactNumber", "business contact number", p.BusinessContactNumber, maxContactNumberLength); ok {
		validationErrs = append(validationErrs, e)
	}

	if strings.TrimSpace(p.BusinessEmail) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "businessEmail", Message: "business email is required"})
	} else if !emailRegex.MatchString(strings.TrimSpace(p.BusinessEmail)) {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "businessEmail", Message: "business email format is invalid"})
	} else if e, ok := maxLengthError("businessEmail", "business email", p.BusinessEmail, maxEmailLength); ok {
		validationErrs = append(validationErrs, e)
	}

	if len(p.WorkingHours) == 0 {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "workingHours", Message: "at least one working hour is required"})
	}

	for i, wh := range p.WorkingHours {
		if !wh.StartTime.Before(wh.EndTime) {
			validationErrs = append(validationErrs, errs.ValidationError{
				Field:   fmt.Sprintf("workingHours[%d]", i),
				Message: "start time must be before end time",
			})
		}
	}

	if len(validationErrs) > 0 {
		return validationErrs
	}

	return nil
}
