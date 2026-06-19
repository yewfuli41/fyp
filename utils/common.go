package utils

import (
	"database/sql"
	"unicode"
)

func NullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

// capitalizeFirst capitalizes the first character of a string.
func CapitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
