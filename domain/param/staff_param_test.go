package param_test

import (
	"fyp/domain/errs"
	"fyp/domain/param"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("StaffParam", func() {
	validWorkingHours := func() []param.WorkingHourParam {
		start := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
		end := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)
		return []param.WorkingHourParam{{Day: "monday", StartTime: start, EndTime: end}}
	}

	validParam := func() param.StaffParam {
		return param.StaffParam{
			BusinessID:         1,
			StaffName:          "John Smith",
			StaffEmail:         "john.smith@example.com",
			StaffContactNumber: "0123456789",
			Position:           "Therapist",
			Password:           "password123",
			WorkingHours:       validWorkingHours(),
		}
	}

	Describe("ValidateRegisterStaff", func() {
		It("returns nil for valid parameters", func() {
			p := validParam()
			Expect(p.ValidateRegisterStaff()).To(Succeed())
		})

		It("returns errors for all empty required fields", func() {
			p := param.StaffParam{}
			err := p.ValidateRegisterStaff()
			Expect(err).To(HaveOccurred())
			vErrs := err.(errs.ValidationErrors)
			fields := make([]string, len(vErrs))
			for i, e := range vErrs {
				fields[i] = e.Field
			}
			Expect(fields).To(ContainElements(
				"staffName", "staffEmail", "staffContactNumber", "position", "password", "workingHours",
			))
		})

		It("returns error for empty staff name", func() {
			p := validParam()
			p.StaffName = ""
			err := p.ValidateRegisterStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffName", Message: "Staff name is required"},
			))
		})

		It("returns error for staff name exceeding max length", func() {
			p := validParam()
			p.StaffName = strings.Repeat("a", 256)
			err := p.ValidateRegisterStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffName", Message: "Staff name must be at most 255 characters"},
			))
		})

		It("returns error for empty staff email", func() {
			p := validParam()
			p.StaffEmail = ""
			err := p.ValidateRegisterStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffEmail", Message: "Staff email is required"},
			))
		})

		It("returns error for invalid staff email format", func() {
			p := validParam()
			p.StaffEmail = "not-an-email"
			err := p.ValidateRegisterStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffEmail", Message: "Staff email format is invalid"},
			))
		})

		It("returns error for staff email exceeding max length", func() {
			p := validParam()
			p.StaffEmail = strings.Repeat("a", 250) + "@example.com"
			err := p.ValidateRegisterStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffEmail", Message: "Staff email must be at most 255 characters"},
			))
		})

		It("returns error for empty staff contact number", func() {
			p := validParam()
			p.StaffContactNumber = ""
			err := p.ValidateRegisterStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffContactNumber", Message: "Staff contact number is required"},
			))
		})

		It("returns error for staff contact number with too few digits", func() {
			p := validParam()
			p.StaffContactNumber = "123456789"
			err := p.ValidateRegisterStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffContactNumber", Message: "Staff contact number must have at least 10 digits"},
			))
		})

		It("returns error for staff contact number exceeding max length", func() {
			p := validParam()
			p.StaffContactNumber = strings.Repeat("1", 31)
			err := p.ValidateRegisterStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffContactNumber", Message: "Staff contact number must be at most 30 characters"},
			))
		})

		It("returns error for empty position", func() {
			p := validParam()
			p.Position = ""
			err := p.ValidateRegisterStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "position", Message: "Position is required"},
			))
		})

		It("returns error for empty password", func() {
			p := validParam()
			p.Password = ""
			err := p.ValidateRegisterStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "password", Message: "Temporary password is required"},
			))
		})

		// UT-046 (Input & Parameter Validation (Domain Rules)).
		It("returns error for password shorter than the minimum length", func() {
			p := validParam()
			p.Password = "pass1"
			err := p.ValidateRegisterStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "password", Message: "Temporary password must be at least 8 characters"},
			))
		})

		// UT-046 (Input & Parameter Validation (Domain Rules)).
		It("returns nil for a password exactly at the minimum length", func() {
			p := validParam()
			p.Password = "12345678"
			Expect(p.ValidateRegisterStaff()).To(Succeed())
		})

		It("returns error when working hours are empty", func() {
			p := validParam()
			p.WorkingHours = nil
			err := p.ValidateRegisterStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "workingHours", Message: "At least one working hour is required"},
			))
		})

		It("returns error when a working hour's start time is not before its end time", func() {
			p := validParam()
			start := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)
			end := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
			p.WorkingHours = []param.WorkingHourParam{{Day: "monday", StartTime: start, EndTime: end}}
			err := p.ValidateRegisterStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "workingHours[0]", Message: "Start time must be before end time"},
			))
		})
	})

	Describe("ValidateUpdateStaff", func() {
		validUpdateParam := func() param.StaffParam {
			return param.StaffParam{
				BusinessID:         1,
				StaffName:          "John Smith",
				StaffEmail:         "john.smith@example.com",
				StaffContactNumber: "0123456789",
				Position:           "Therapist",
			}
		}

		It("returns nil for valid parameters", func() {
			p := validUpdateParam()
			Expect(p.ValidateUpdateStaff()).To(Succeed())
		})

		It("returns errors for all empty required fields", func() {
			p := param.StaffParam{}
			err := p.ValidateUpdateStaff()
			Expect(err).To(HaveOccurred())
			vErrs := err.(errs.ValidationErrors)
			fields := make([]string, len(vErrs))
			for i, e := range vErrs {
				fields[i] = e.Field
			}
			Expect(fields).To(ContainElements("staffName", "staffEmail", "staffContactNumber", "position"))
			// ValidateUpdateStaff does not touch password or working hours.
			Expect(fields).NotTo(ContainElement("password"))
			Expect(fields).NotTo(ContainElement("workingHours"))
		})

		It("returns error for empty staff name", func() {
			p := validUpdateParam()
			p.StaffName = ""
			err := p.ValidateUpdateStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffName", Message: "Staff name is required"},
			))
		})

		It("returns error for staff name exceeding max length", func() {
			p := validUpdateParam()
			p.StaffName = strings.Repeat("a", 256)
			err := p.ValidateUpdateStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffName", Message: "Staff name must be at most 255 characters"},
			))
		})

		It("returns error for empty staff email", func() {
			p := validUpdateParam()
			p.StaffEmail = ""
			err := p.ValidateUpdateStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffEmail", Message: "Staff email is required"},
			))
		})

		It("returns error for invalid staff email format", func() {
			p := validUpdateParam()
			p.StaffEmail = "not-an-email"
			err := p.ValidateUpdateStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffEmail", Message: "Staff email format is invalid"},
			))
		})

		It("returns error for staff email exceeding max length", func() {
			p := validUpdateParam()
			p.StaffEmail = strings.Repeat("a", 250) + "@example.com"
			err := p.ValidateUpdateStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffEmail", Message: "Staff email must be at most 255 characters"},
			))
		})

		It("returns error for empty staff contact number", func() {
			p := validUpdateParam()
			p.StaffContactNumber = ""
			err := p.ValidateUpdateStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffContactNumber", Message: "Staff contact number is required"},
			))
		})

		It("returns error for staff contact number with too few digits", func() {
			p := validUpdateParam()
			p.StaffContactNumber = "123456789"
			err := p.ValidateUpdateStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffContactNumber", Message: "Staff contact number must have at least 10 digits"},
			))
		})

		It("returns error for staff contact number exceeding max length", func() {
			p := validUpdateParam()
			p.StaffContactNumber = strings.Repeat("1", 31)
			err := p.ValidateUpdateStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "staffContactNumber", Message: "Staff contact number must be at most 30 characters"},
			))
		})

		It("returns error for empty position", func() {
			p := validUpdateParam()
			p.Position = ""
			err := p.ValidateUpdateStaff()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "position", Message: "Position is required"},
			))
		})
	})
})
