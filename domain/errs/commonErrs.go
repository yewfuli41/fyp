package errs

import "errors"

var ErrInternal = errors.New("internal server error")
var ErrBusinessProfileNotFound = errors.New("Business profile not found")
