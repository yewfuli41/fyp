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
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "username", Message: "Username is required"}))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "email", Message: "Email is required"}))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "contactNumber", Message: "Contact number is required"}))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "password", Message: "Password is required"}))
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
				Expect(err).To(ContainElement(errs.ValidationError{Field: "email", Message: "Email format is invalid"}))
			})

			// UT-046 (Input & Parameter Validation (Domain Rules)).
			It("returns error for short password", func() {
				p := param.SignUpParam{
					Username:      "testuser",
					Email:         "test@example.com",
					ContactNumber: "0123456789",
					Password:      "short",
				}
				err := p.ValidateSignUp()
				Expect(err).To(HaveOccurred())
				Expect(err).To(ContainElement(errs.ValidationError{Field: "password", Message: "Password must be at least 8 characters"}))
			})

			// UT-047 (Input & Parameter Validation (Domain Rules)).
			It("returns error for short contact number", func() {
				p := param.SignUpParam{
					Username:      "testuser",
					Email:         "test@example.com",
					ContactNumber: "123",
					Password:      "password123",
				}
				err := p.ValidateSignUp()
				Expect(err).To(HaveOccurred())
				Expect(err).To(ContainElement(errs.ValidationError{Field: "contactNumber", Message: "Contact number must have at least 10 digits"}))
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
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "email", Message: "Email is required"}))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "password", Message: "Password is required"}))
			})
		})
	})

	Describe("ResetPasswordParam", func() {
		validResetPasswordParam := func() param.ResetPasswordParam {
			return param.ResetPasswordParam{
				UserID:      1,
				Email:       "test@example.com",
				NewPassword: "newpassword123",
			}
		}

		Describe("ValidateResetPassword", func() {
			It("returns nil for valid parameters", func() {
				p := validResetPasswordParam()
				Expect(p.ValidateResetPassword()).To(Succeed())
			})

			It("returns error when new password is empty", func() {
				p := validResetPasswordParam()
				p.NewPassword = ""

				err := p.ValidateResetPassword()
				Expect(err).To(HaveOccurred())

				var vErrs errs.ValidationErrors
				Expect(err).To(BeAssignableToTypeOf(vErrs))
				vErrs = err.(errs.ValidationErrors)
				Expect(vErrs).To(HaveLen(1))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "password", Message: "Password is required"}))
			})

			// UT-046 (Input & Parameter Validation (Domain Rules)).
			It("returns error when new password is too short", func() {
				p := validResetPasswordParam()
				p.NewPassword = "short"

				err := p.ValidateResetPassword()
				Expect(err).To(HaveOccurred())

				var vErrs errs.ValidationErrors
				Expect(err).To(BeAssignableToTypeOf(vErrs))
				vErrs = err.(errs.ValidationErrors)
				Expect(vErrs).To(HaveLen(1))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "password", Message: "Password must be at least 8 characters"}))
			})
		})
	})

	Describe("ChangePasswordParam", func() {
		validChangePasswordParam := func() param.ChangePasswordParam {
			return param.ChangePasswordParam{
				UserID:          1,
				Email:           "test@example.com",
				CurrentPassword: "currentpassword",
				NewPassword:     "newpassword123",
			}
		}

		Describe("ValidateChangePassword", func() {
			It("returns nil for valid parameters", func() {
				p := validChangePasswordParam()
				Expect(p.ValidateChangePassword()).To(Succeed())
			})

			It("returns errors for empty fields", func() {
				p := param.ChangePasswordParam{}
				err := p.ValidateChangePassword()
				Expect(err).To(HaveOccurred())

				var vErrs errs.ValidationErrors
				Expect(err).To(BeAssignableToTypeOf(vErrs))
				vErrs = err.(errs.ValidationErrors)
				Expect(vErrs).To(HaveLen(2))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "currentPassword", Message: "Current password is required"}))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "newPassword", Message: "New password is required"}))
			})

			It("returns error when current password is empty but new password is valid", func() {
				p := validChangePasswordParam()
				p.CurrentPassword = ""

				err := p.ValidateChangePassword()
				Expect(err).To(HaveOccurred())

				var vErrs errs.ValidationErrors
				Expect(err).To(BeAssignableToTypeOf(vErrs))
				vErrs = err.(errs.ValidationErrors)
				Expect(vErrs).To(HaveLen(1))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "currentPassword", Message: "Current password is required"}))
			})

			// UT-046 (Input & Parameter Validation (Domain Rules)).
			It("returns error when new password is too short", func() {
				p := validChangePasswordParam()
				p.NewPassword = "short"

				err := p.ValidateChangePassword()
				Expect(err).To(HaveOccurred())

				var vErrs errs.ValidationErrors
				Expect(err).To(BeAssignableToTypeOf(vErrs))
				vErrs = err.(errs.ValidationErrors)
				Expect(vErrs).To(HaveLen(1))
				Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "newPassword", Message: "New password must be at least 8 characters"}))
			})
		})
	})
})
