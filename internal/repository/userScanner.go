package repository

import (
	"database/sql"
	"fyp/domain/param"
	"time"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func ScanUser(row rowScanner) (*param.AuthUserParam, error) {
	var user param.AuthUserParam
	var contactNumber sql.NullString
	var lockedUntil sql.NullTime

	if err := row.Scan(
		&user.UserID,
		&user.Username,
		&user.Email,
		&contactNumber,
		&user.Password,
		&user.FailedLoginAttempts,
		&lockedUntil,
	); err != nil {
		return nil, err
	}

	if contactNumber.Valid {
		user.ContactNumber = &contactNumber.String
	}
	if lockedUntil.Valid {
		lockedUntilValue := lockedUntil.Time.UTC().Truncate(time.Second)
		user.LockedUntil = &lockedUntilValue
	}

	return &user, nil
}
