package graphErrs

import (
	"errors"
	"fyp/domain/errs"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

func init() {
	registerGraphQLErrorHandler(handleValidationError)
}

func handleValidationError(err error) *gqlerror.Error {
	var validationErrs errs.ValidationErrors

	if !errors.As(err, &validationErrs) {
		return nil
	}

	items := make([]errs.ValidationError, len(validationErrs))
	copy(items, validationErrs)

	return &gqlerror.Error{
		Message: "validation failed",
		Extensions: map[string]any{
			"validationErrors": items,
		},
	}
}
