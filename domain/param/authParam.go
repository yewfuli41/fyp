package param

import (
	"fmt"
	"fyp/domain/errs"
	"strings"
)

type SignUpParam struct {
	Username      string
	Email         string
	ContactNumber string
	Password      string
}

type LogInParam struct {
	Email    string
	Password string
}

func (p SignUpParam) ValidateSignUp() error {
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
			Message: fmt.Sprintf("Contact number must have at least %d digits", minContactNumberDigits),
		})
	} else if e, ok := maxLengthError("contactNumber", "contact number", p.ContactNumber, maxContactNumberLength); ok {
		validationErrs = append(validationErrs, e)
	}

	if p.Password == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "password", Message: "Password is required"})
	} else if len(p.Password) < minPasswordLength {
		validationErrs = append(validationErrs, errs.ValidationError{
			Field:   "password",
			Message: fmt.Sprintf("Password must be at least %d characters", minPasswordLength),
		})
	}

	if len(validationErrs) > 0 {
		return validationErrs
	}

	return nil
}

func (p LogInParam) ValidateLogIn() error {
	var validationErrs errs.ValidationErrors

	if strings.TrimSpace(p.Email) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "email", Message: "Email is required"})
	}

	if p.Password == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "password", Message: "Password is required"})
	}

	if len(validationErrs) > 0 {
		return validationErrs
	}

	return nil
}
