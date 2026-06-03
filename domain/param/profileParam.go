package param

import (
	"fmt"
	"fyp/domain/errs"
	"strings"
)

type ProfileParam struct {
	UserId        int
	Username      string
	Email         string
	ContactNumber string
}

func (p *ProfileParam) ValidateProfile() error {
	var validationErrs errs.ValidationErrors

	if strings.TrimSpace(p.Username) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "username", Message: "username is required"})
	}

	email := strings.TrimSpace(p.Email)
	if email == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "email", Message: "email is required"})
	} else if !emailRegex.MatchString(email) {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "email", Message: "email format is invalid"})
	}

	if strings.TrimSpace(p.ContactNumber) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "contactNumber", Message: "contact number is required"})
	} else if len(p.ContactNumber) < minContactNumberDigits {
		validationErrs = append(validationErrs, errs.ValidationError{
			Field:   "contactNumber",
			Message: fmt.Sprintf("contact number must have at least %d digits", minContactNumberDigits),
		})
	}

	if len(validationErrs) > 0 {
		return validationErrs
	}

	return nil
}
