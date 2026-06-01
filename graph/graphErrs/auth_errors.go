package graphErrs

import (
	"errors"
	"fyp/domain/errs"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

func init() {
	registerGraphQLErrorHandler(handleAuthError)
}

func handleAuthError(err error) *gqlerror.Error {
	if !errors.Is(err, errs.ErrUnauthenticated) {
		return nil
	}

	return &gqlerror.Error{
		Message: "unauthenticated",
		Extensions: map[string]any{
			"code": "UNAUTHENTICATED",
		},
	}
}
