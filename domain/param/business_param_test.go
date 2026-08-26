package param_test

import (
	"fyp/domain/errs"
	"fyp/domain/param"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BusinessProfileParam", func() {
	var validStart, validEnd time.Time

	BeforeEach(func() {
		validStart = time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
		validEnd = time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)
	})

	validParam := func() param.BusinessProfileParam {
		return param.BusinessProfileParam{
			BusinessName:          "Test Biz",
			Address:               "123 Street",
			BusinessContactNumber: "0123456789",
			BusinessEmail:         "test@example.com",
		}
	}

	Describe("ValidateRegisterBusinessProfile", func() {
		It("returns nil for valid parameters with working hours", func() {
			p := validParam()
			p.WorkingHours = []param.WorkingHourParam{
				{Day: "monday", StartTime: validStart, EndTime: validEnd},
			}
			Expect(p.ValidateRegisterBusinessProfile()).To(Succeed())
		})

		It("returns errors for all empty required fields", func() {
			p := param.BusinessProfileParam{}
			err := p.ValidateRegisterBusinessProfile()
			Expect(err).To(HaveOccurred())
			vErrs := err.(errs.ValidationErrors)
			fields := make([]string, len(vErrs))
			for i, e := range vErrs {
				fields[i] = e.Field
			}
			Expect(fields).To(ContainElements("businessName", "address", "businessContactNumber", "businessEmail", "workingHours"))
		})

		It("returns error for invalid email format", func() {
			p := validParam()
			p.BusinessEmail = "not-an-email"
			p.WorkingHours = []param.WorkingHourParam{{Day: "monday", StartTime: validStart, EndTime: validEnd}}
			err := p.ValidateRegisterBusinessProfile()
			Expect(err).To(HaveOccurred())
			Expect(err).To(ContainElement(errs.ValidationError{Field: "businessEmail", Message: "Business email format is invalid"}))
		})

		It("returns error for short contact number", func() {
			p := validParam()
			p.BusinessContactNumber = "123"
			p.WorkingHours = []param.WorkingHourParam{{Day: "monday", StartTime: validStart, EndTime: validEnd}}
			err := p.ValidateRegisterBusinessProfile()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "businessContactNumber", Message: "Business contact number must have at least 10 digits"},
			))
		})

		// UT-049 (Input & Parameter Validation (Domain Rules)).
		It("returns error when start time is not before end time", func() {
			p := validParam()
			p.WorkingHours = []param.WorkingHourParam{
				{Day: "monday", StartTime: validEnd, EndTime: validStart},
			}
			err := p.ValidateRegisterBusinessProfile()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "workingHours[0]", Message: "Start time must be before end time"},
			))
		})

		// UT-048 (Input & Parameter Validation (Domain Rules)).
		It("returns error for overlapping working hours on the same day", func() {
			start1 := time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC)
			end1 := time.Date(0, 1, 1, 12, 0, 0, 0, time.UTC)
			start2 := time.Date(0, 1, 1, 11, 0, 0, 0, time.UTC)
			end2 := time.Date(0, 1, 1, 13, 0, 0, 0, time.UTC)

			p := validParam()
			p.WorkingHours = []param.WorkingHourParam{
				{Day: "monday", StartTime: start1, EndTime: end1},
				{Day: "monday", StartTime: start2, EndTime: end2},
			}
			err := p.ValidateRegisterBusinessProfile()
			Expect(err).To(HaveOccurred())
			vErrs := err.(errs.ValidationErrors)
			fields := make([]string, len(vErrs))
			for i, e := range vErrs {
				fields[i] = e.Field
			}
			Expect(fields).To(ContainElements("workingHours[0]", "workingHours[1]"))
		})

		// UT-048 (Input & Parameter Validation (Domain Rules)).
		It("allows touching working hours on the same day (end == next start)", func() {
			start1 := time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC)
			end1 := time.Date(0, 1, 1, 12, 0, 0, 0, time.UTC)
			start2 := time.Date(0, 1, 1, 12, 0, 0, 0, time.UTC)
			end2 := time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC)

			p := validParam()
			p.WorkingHours = []param.WorkingHourParam{
				{Day: "monday", StartTime: start1, EndTime: end1},
				{Day: "monday", StartTime: start2, EndTime: end2},
			}
			Expect(p.ValidateRegisterBusinessProfile()).To(Succeed())
		})

		// UT-048 (Input & Parameter Validation (Domain Rules)).
		It("allows overlapping hours on different days", func() {
			p := validParam()
			p.WorkingHours = []param.WorkingHourParam{
				{Day: "monday", StartTime: validStart, EndTime: validEnd},
				{Day: "tuesday", StartTime: validStart, EndTime: validEnd},
			}
			Expect(p.ValidateRegisterBusinessProfile()).To(Succeed())
		})
	})
})
