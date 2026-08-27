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
	StaffEmail         string
	StaffContactNumber string
	Position           string
	// Password is the owner-chosen temporary password for a brand-new staff
	// account — only used when no existing user is found for StaffEmail (see
	// staffService.RegisterStaff); ignored otherwise.
	Password          string
	MustResetPassword bool
	WorkingHours      []WorkingHourParam //from businessParam.go
	// true once this staff member has an active (pending/accepted/rescheduled)
	// booking on any of their slots — loaded for display, mirrors
	// ServiceOptionParam.HasBooking.
	HasBooking bool
	// Filled by GetStaffByUserID only — the staff member's own screens show
	// which business they belong to.
	BusinessName string
}

func (p StaffParam) ValidateRegisterStaff() error {
	var validationErrs errs.ValidationErrors

	if strings.TrimSpace(p.StaffName) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "staffName", Message: "Staff name is required"})
	} else if e, ok := maxLengthError("staffName", "staff name", p.StaffName, maxBusinessNameLength); ok {
		validationErrs = append(validationErrs, e)
	}

	if strings.TrimSpace(p.StaffEmail) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "staffEmail", Message: "Staff email is required"})
	} else if !emailRegex.MatchString(strings.TrimSpace(p.StaffEmail)) {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "staffEmail", Message: "Staff email format is invalid"})
	} else if e, ok := maxLengthError("staffEmail", "staff email", p.StaffEmail, maxEmailLength); ok {
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

	if p.Password == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "password", Message: "Temporary password is required"})
	} else if len(p.Password) < minPasswordLength {
		validationErrs = append(validationErrs, errs.ValidationError{
			Field:   "password",
			Message: fmt.Sprintf("Temporary password must be at least %d characters", minPasswordLength),
		})
	}

	validationErrs = append(validationErrs, ValidateWorkingHoursShape(p.WorkingHours)...)

	if len(validationErrs) > 0 {
		return validationErrs
	}

	return nil
}

func (p StaffParam) ValidateUpdateStaff() error {
	var validationErrs errs.ValidationErrors

	if strings.TrimSpace(p.StaffName) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "staffName", Message: "Staff name is required"})
	} else if e, ok := maxLengthError("staffName", "staff name", p.StaffName, maxBusinessNameLength); ok {
		validationErrs = append(validationErrs, e)
	}

	if strings.TrimSpace(p.StaffEmail) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "staffEmail", Message: "Staff email is required"})
	} else if !emailRegex.MatchString(strings.TrimSpace(p.StaffEmail)) {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "staffEmail", Message: "Staff email format is invalid"})
	} else if e, ok := maxLengthError("staffEmail", "staff email", p.StaffEmail, maxEmailLength); ok {
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

	if len(validationErrs) > 0 {
		return validationErrs
	}

	return nil
}
