package param_test

import (
	"fyp/domain/errs"
	"fyp/domain/param"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AuthParam", func() {
	Describe("SignUpParam", func() {
		Describe("ValidateSignUp", func() {
			It("returns nil for valid parameters", func() {
				p := param.SignUpParam{
					Username:      "testuser",
					Email:         "test@example.com",
					ContactNumber: "0123456789",
					Password:      "password123",
				}
				Expect(p.ValidateSignUp()).To(Succeed())
			})

			It("returns errors for empty fields", func() {
				p := param.SignUpParam{}
				err := p.ValidateSignUp()
				Expect(err).To(HaveOccurred())

				var vErrs errs.ValidationErrors
				Expect(err).To(BeAssignableToTypeOf(vErrs))
				vErrs = err.(errs.ValidationErrors)
				Expect(vErrs).To(HaveLen(4))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "username", Message: "username is required"}))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "email", Message: "email is required"}))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "contactNumber", Message: "contact number is required"}))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "password", Message: "password is required"}))
			})

			It("returns error for invalid email format", func() {
				p := param.SignUpParam{
					Username:      "testuser",
					Email:         "invalid-email",
					ContactNumber: "0123456789",
					Password:      "password123",
				}
				err := p.ValidateSignUp()
				Expect(err).To(HaveOccurred())
				Expect(err).To(ContainElement(errs.ValidationError{Field: "email", Message: "email format is invalid"}))
			})

			It("returns error for short password", func() {
				p := param.SignUpParam{
					Username:      "testuser",
					Email:         "test@example.com",
					ContactNumber: "0123456789",
					Password:      "short",
				}
				err := p.ValidateSignUp()
				Expect(err).To(HaveOccurred())
				Expect(err).To(ContainElement(errs.ValidationError{Field: "password", Message: "password must be at least 8 characters"}))
			})

			It("returns error for short contact number", func() {
				p := param.SignUpParam{
					Username:      "testuser",
					Email:         "test@example.com",
					ContactNumber: "123",
					Password:      "password123",
				}
				err := p.ValidateSignUp()
				Expect(err).To(HaveOccurred())
				Expect(err).To(ContainElement(errs.ValidationError{Field: "contactNumber", Message: "contact number must have at least 10 digits"}))
			})
		})
	})

	Describe("LogInParam", func() {
		Describe("ValidateLogIn", func() {
			It("returns nil for valid parameters", func() {
				p := param.LogInParam{
					Email:    "test@example.com",
					Password: "password123",
				}
				Expect(p.ValidateLogIn()).To(Succeed())
			})

			It("returns errors for empty fields", func() {
				p := param.LogInParam{}
				err := p.ValidateLogIn()
				Expect(err).To(HaveOccurred())

				var vErrs errs.ValidationErrors
				Expect(err).To(BeAssignableToTypeOf(vErrs))
				vErrs = err.(errs.ValidationErrors)
				Expect(vErrs).To(HaveLen(2))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "email", Message: "email is required"}))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "password", Message: "password is required"}))
			})
		})
	})
})
