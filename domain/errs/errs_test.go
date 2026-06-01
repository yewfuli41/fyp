package errs_test

import (
	"fyp/domain/errs"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Errs", func() {
	Describe("LockedError", func() {
		It("returns the correct error message", func() {
			err := errs.LockedError{LockedUntil: time.Now()}
			Expect(err.Error()).To(Equal("account locked"))
		})
	})

	Describe("ValidationErrors", func() {
		It("returns the correct error message", func() {
			errs := errs.ValidationErrors{}
			Expect(errs.Error()).To(Equal("validation failed"))
		})
	})
})
