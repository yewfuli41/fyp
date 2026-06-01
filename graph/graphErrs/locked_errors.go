package graphErrs

import (
	"errors"
	"fyp/domain/errs"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

func init() {
	registerGraphQLErrorHandler(handleLockedError)
}

func handleLockedError(err error) *gqlerror.Error {
	var lockedErr errs.LockedError

	if !errors.As(err, &lockedErr) {
		return nil
	}

	return &gqlerror.Error{
		Message: "account locked",
		Extensions: map[string]any{
			"lockedUntil": lockedErr.LockedUntil,
		},
	}
}
