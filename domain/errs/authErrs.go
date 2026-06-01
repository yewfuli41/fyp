package errs

import (
	"errors"
	"time"
)

type LockedError struct {
	LockedUntil time.Time
}

func (e LockedError) Error() string {
	return "account locked"
}

var ErrUnauthenticated = errors.New("unauthenticated")
