package param

import (
	"fmt"
	"fyp/domain/errs"
	"regexp"
	"unicode"
	"unicode/utf8"
)

const (
	minPasswordLength      = 8
	minContactNumberDigits = 10

	// Maximum string lengths, mirroring the VARCHAR limits in the database
	// schema so over-long values are rejected with a clean validation error
	// instead of surfacing a raw SQL error.
	maxUsernameLength      = 100
	maxEmailLength         = 255
	maxContactNumberLength = 30
	maxBusinessNameLength  = 255
	maxServiceNameLength   = 255
	maxPackageNameLength   = 255
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// maxLengthError returns a validation error when value exceeds max characters.
// The boolean reports whether the value was too long.
func maxLengthError(field, label, value string, max int) (errs.ValidationError, bool) {
	if utf8.RuneCountInString(value) > max {
		capitalizedLabel := capitalizeFirst(label)
		return errs.ValidationError{
			Field:   field,
			Message: fmt.Sprintf("%s must be at most %d characters", capitalizedLabel, max),
		}, true
	}
	return errs.ValidationError{}, false
}

// capitalizeFirst capitalizes the first character of a string.
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
