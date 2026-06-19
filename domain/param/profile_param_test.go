package param_test

import (
	"fyp/domain/errs"
	"fyp/domain/param"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ProfileParam", func() {
	Describe("ValidateProfile", func() {
		It("returns nil for valid parameters", func() {
			p := &param.ProfileParam{
				Username:      "testuser",
				Email:         "test@example.com",
				ContactNumber: "0123456789",
			}
			Expect(p.ValidateProfile()).To(Succeed())
		})

		It("returns errors for all empty fields", func() {
			p := &param.ProfileParam{}
			err := p.ValidateProfile()
			Expect(err).To(HaveOccurred())
			vErrs := err.(errs.ValidationErrors)
			fields := make([]string, len(vErrs))
			for i, e := range vErrs {
				fields[i] = e.Field
			}
			Expect(fields).To(ContainElements("username", "email", "contactNumber"))
		})

		It("returns error for invalid email format", func() {
			p := &param.ProfileParam{
				Username:      "testuser",
				Email:         "invalid-email",
				ContactNumber: "0123456789",
			}
			err := p.ValidateProfile()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "email", Message: "Email format is invalid"},
			))
		})

		It("returns error for short contact number", func() {
			p := &param.ProfileParam{
				Username:      "testuser",
				Email:         "test@example.com",
				ContactNumber: "123",
			}
			err := p.ValidateProfile()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "contactNumber", Message: "contact number must have at least 10 digits"},
			))
		})
	})
})
