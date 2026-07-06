package param_test

import (
	"fyp/domain/errs"
	"fyp/domain/param"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// These tests exercise maxLengthError indirectly through the exported validators,
// verifying boundary behaviour for every VARCHAR limit defined in validation.go.

var _ = Describe("maxLengthError boundaries", func() {
	Describe("username (max 100)", func() {
		It("accepts a username exactly at the limit", func() {
			p := param.SignUpParam{
				Username:      strings.Repeat("a", 100),
				Email:         "test@example.com",
				ContactNumber: "0123456789",
				Password:      "password123",
			}
			Expect(p.ValidateSignUp()).To(Succeed())
		})

		It("rejects a username one character over the limit", func() {
			p := param.SignUpParam{
				Username:      strings.Repeat("a", 101),
				Email:         "test@example.com",
				ContactNumber: "0123456789",
				Password:      "password123",
			}
			err := p.ValidateSignUp()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "username", Message: "Username must be at most 100 characters"},
			))
		})
	})

	Describe("email (max 255)", func() {
		// "@b.com" = 6 chars, so 249 + 6 = 255, 250 + 6 = 256
		It("accepts an email exactly at the limit", func() {
			p := param.SignUpParam{
				Username:      "user",
				Email:         strings.Repeat("a", 249) + "@b.com",
				ContactNumber: "0123456789",
				Password:      "password123",
			}
			Expect(p.ValidateSignUp()).To(Succeed())
		})

		It("rejects an email one character over the limit", func() {
			p := param.SignUpParam{
				Username:      "user",
				Email:         strings.Repeat("a", 250) + "@b.com",
				ContactNumber: "0123456789",
				Password:      "password123",
			}
			err := p.ValidateSignUp()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "email", Message: "Email must be at most 255 characters"},
			))
		})
	})

	Describe("contactNumber (max 30)", func() {
		It("accepts a contact number exactly at the limit", func() {
			p := param.SignUpParam{
				Username:      "user",
				Email:         "test@example.com",
				ContactNumber: strings.Repeat("1", 30),
				Password:      "password123",
			}
			Expect(p.ValidateSignUp()).To(Succeed())
		})

		It("rejects a contact number one character over the limit", func() {
			p := param.SignUpParam{
				Username:      "user",
				Email:         "test@example.com",
				ContactNumber: strings.Repeat("1", 31),
				Password:      "password123",
			}
			err := p.ValidateSignUp()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "contactNumber", Message: "Contact number must be at most 30 characters"},
			))
		})
	})

	Describe("businessName (max 255)", func() {
		validWH := func() []param.WorkingHourParam {
			return []param.WorkingHourParam{{
				Day:       "monday",
				StartTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC),
				EndTime:   time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC),
			}}
		}

		It("accepts a business name exactly at the limit", func() {
			p := param.BusinessProfileParam{
				BusinessName:          strings.Repeat("b", 255),
				Address:               "123 St",
				BusinessContactNumber: "0123456789",
				BusinessEmail:         "biz@example.com",
				WorkingHours:          validWH(),
			}
			Expect(p.ValidateRegisterBusinessProfile()).To(Succeed())
		})

		It("rejects a business name one character over the limit", func() {
			p := param.BusinessProfileParam{
				BusinessName:          strings.Repeat("b", 256),
				Address:               "123 St",
				BusinessContactNumber: "0123456789",
				BusinessEmail:         "biz@example.com",
				WorkingHours:          validWH(),
			}
			err := p.ValidateRegisterBusinessProfile()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "businessName", Message: "Business name must be at most 255 characters"},
			))
		})
	})

	Describe("serviceName (max 255)", func() {
		It("accepts a service name exactly at the limit", func() {
			p := param.ServiceParam{ServiceName: strings.Repeat("s", 255)}
			Expect(p.Validate()).To(Succeed())
		})

		It("rejects a service name one character over the limit", func() {
			p := param.ServiceParam{ServiceName: strings.Repeat("s", 256)}
			err := p.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "serviceName", Message: "Service name must be at most 255 characters"},
			))
		})
	})

	Describe("packageName (max 255)", func() {
		It("rejects a package name one character over the limit", func() {
			p := param.ServiceParam{
				ServiceName: "Massage",
				ServiceOptions: []param.ServiceOptionParam{
					{ServiceOptionName: strings.Repeat("p", 256)},
				},
			}
			err := p.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "serviceOptions[0].serviceOptionName", Message: "Service option name must be at most 255 characters"},
			))
		})
	})
})
