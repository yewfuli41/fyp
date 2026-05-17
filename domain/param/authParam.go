package param

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	minPasswordLength      = 8
	minContactNumberDigits = 10
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type SignUpParam struct {
	Username      string
	Email         string
	ContactNumber string
	Password      string
}

type ValidationErrors map[string]string

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ""
	}

	messages := make([]string, 0, len(v))
	for field, message := range v {
		messages = append(messages, fmt.Sprintf("%s: %s", field, message))
	}

	return strings.Join(messages, ", ")
}

func (p SignUpParam) Validate() error {
	errs := ValidationErrors{}

	if strings.TrimSpace(p.Username) == "" {
		errs["username"] = "username is required"
	}

	email := strings.TrimSpace(p.Email)
	if email == "" {
		errs["email"] = "email is required"
	} else if !emailRegex.MatchString(email) {
		errs["email"] = "email format is invalid"
	}

	if strings.TrimSpace(p.ContactNumber) == "" {
		errs["contactNumber"] = "contact number is required"
	} else if len(p.ContactNumber) < minContactNumberDigits {
		errs["contactNumber"] = fmt.Sprintf("contact number must have at least %d digits", minContactNumberDigits)
	}

	if p.Password == "" {
		errs["password"] = "password is required"
	} else if len(p.Password) < minPasswordLength {
		errs["password"] = fmt.Sprintf("password must be at least %d characters", minPasswordLength)
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}
