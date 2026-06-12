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
		validationErrs = append(validationErrs, errs.ValidationError{Field: "username", Message: "Username is required"})
	} else if e, ok := maxLengthError("username", "username", p.Username, maxUsernameLength); ok {
		validationErrs = append(validationErrs, e)
	}

	email := strings.TrimSpace(p.Email)
	if email == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "email", Message: "Email is required"})
	} else if !emailRegex.MatchString(email) {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "email", Message: "Email format is invalid"})
	} else if e, ok := maxLengthError("email", "email", email, maxEmailLength); ok {
		validationErrs = append(validationErrs, e)
	}

	if strings.TrimSpace(p.ContactNumber) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "contactNumber", Message: "Contact number is required"})
	} else if len(p.ContactNumber) < minContactNumberDigits {
		validationErrs = append(validationErrs, errs.ValidationError{
			Field:   "contactNumber",
			Message: fmt.Sprintf("contact number must have at least %d digits", minContactNumberDigits),
		})
	} else if e, ok := maxLengthError("contactNumber", "contact number", p.ContactNumber, maxContactNumberLength); ok {
		validationErrs = append(validationErrs, e)
	}

	if len(validationErrs) > 0 {
		return validationErrs
	}

	return nil
}
