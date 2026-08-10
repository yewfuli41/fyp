package graphErrs_test

import (
	"errors"
	"fyp/domain/errs"
	"fyp/graph/graphErrs"
	"time"

	"github.com/vektah/gqlparser/v2/gqlerror"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GraphErrs", func() {
	Describe("ToGraphQLError", func() {
		It("maps errs.LockedError to gqlerror.Error", func() {
			lockedUntil := time.Now().Add(time.Hour)
			err := errs.LockedError{LockedUntil: lockedUntil}

			gqlErr := graphErrs.ToGraphQLError(err)

			Expect(gqlErr).To(BeAssignableToTypeOf(&gqlerror.Error{}))
			e := gqlErr.(*gqlerror.Error)
			Expect(e.Message).To(Equal("account locked"))
			Expect(e.Extensions["lockedUntil"]).To(Equal(lockedUntil))
		})

		It("maps errs.ValidationErrors to gqlerror.Error", func() {
			vErrs := errs.ValidationErrors{
				{Field: "email", Message: "invalid email"},
			}

			gqlErr := graphErrs.ToGraphQLError(vErrs)

			Expect(gqlErr).To(BeAssignableToTypeOf(&gqlerror.Error{}))
			e := gqlErr.(*gqlerror.Error)
			Expect(e.Message).To(Equal("validation failed"))
			Expect(e.Extensions["validationErrors"]).To(Equal([]errs.ValidationError(vErrs)))
		})

		It("returns the original error if no handler matches", func() {
			err := errors.New("some other error")
			gqlErr := graphErrs.ToGraphQLError(err)

			Expect(gqlErr).To(Equal(err))
		})
	})
})
