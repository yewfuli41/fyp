package param_test

import (
	"fyp/domain/errs"
	"fyp/domain/param"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LeaveApplicationParam", func() {
	var today, tomorrow, dayAfterTomorrow, yesterday, dayBeforeYesterday string

	BeforeEach(func() {
		today = time.Now().Format("2006-01-02")
		tomorrow = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
		dayAfterTomorrow = time.Now().AddDate(0, 0, 2).Format("2006-01-02")
		yesterday = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		dayBeforeYesterday = time.Now().AddDate(0, 0, -2).Format("2006-01-02")
	})

	validParam := func() param.LeaveApplicationParam {
		return param.LeaveApplicationParam{
			StaffID:    1,
			BusinessID: 1,
			StaffName:  "Jane Doe",
			Position:   "Therapist",
			StartDate:  tomorrow,
			EndDate:    dayAfterTomorrow,
		}
	}

	Describe("ValidateApplyLeave", func() {
		It("returns nil for valid parameters", func() {
			p := validParam()
			Expect(p.ValidateApplyLeave()).To(Succeed())
		})

		It("allows the start date to be today (not in the past)", func() {
			p := validParam()
			p.StartDate = today
			p.EndDate = today
			Expect(p.ValidateApplyLeave()).To(Succeed())
		})

		It("allows a justification within the max length", func() {
			p := validParam()
			reason := strings.Repeat("a", 1000)
			p.Justification = &reason
			Expect(p.ValidateApplyLeave()).To(Succeed())
		})

		It("returns errors for empty start date and end date", func() {
			p := param.LeaveApplicationParam{}
			err := p.ValidateApplyLeave()
			Expect(err).To(HaveOccurred())
			vErrs := err.(errs.ValidationErrors)
			Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "startDate", Message: "Start date is required"}))
			Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "endDate", Message: "End date is required"}))
			Expect(vErrs).To(HaveLen(2))
		})

		It("returns error for a start date only", func() {
			p := param.LeaveApplicationParam{}
			err := p.ValidateApplyLeave()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(errs.ValidationError{Field: "startDate", Message: "Start date is required"}))
		})

		It("returns error for an invalid start date format", func() {
			p := validParam()
			p.StartDate = "not-a-date"
			err := p.ValidateApplyLeave()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(Equal(errs.ValidationErrors{
				{Field: "startDate", Message: "Invalid start date"},
			}))
		})

		It("returns error for an invalid end date format", func() {
			p := validParam()
			p.EndDate = "not-a-date"
			err := p.ValidateApplyLeave()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(Equal(errs.ValidationErrors{
				{Field: "endDate", Message: "Invalid end date"},
			}))
		})

		// UT-049 (Input & Parameter Validation (Domain Rules)).
		It("returns error when the start date is in the past", func() {
			p := validParam()
			p.StartDate = yesterday
			p.EndDate = today
			err := p.ValidateApplyLeave()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "startDate", Message: "Start date cannot be in the past"},
			))
		})

		It("returns error when the end date is before the start date", func() {
			p := validParam()
			p.StartDate = tomorrow
			p.EndDate = today
			err := p.ValidateApplyLeave()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "endDate", Message: "End date must be on or after the start date"},
			))
		})

		It("allows the end date to equal the start date", func() {
			p := validParam()
			p.StartDate = tomorrow
			p.EndDate = tomorrow
			Expect(p.ValidateApplyLeave()).To(Succeed())
		})

		// UT-050 (Input & Parameter Validation (Domain Rules)).
		It("returns error when justification exceeds the max length", func() {
			p := validParam()
			reason := strings.Repeat("a", 1001)
			p.Justification = &reason
			err := p.ValidateApplyLeave()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "justification", Message: "Reason must be at most 1000 characters"},
			))
		})

		It("accumulates multiple errors at once", func() {
			p := validParam()
			p.StartDate = yesterday
			p.EndDate = dayBeforeYesterday
			err := p.ValidateApplyLeave()
			Expect(err).To(HaveOccurred())
			vErrs := err.(errs.ValidationErrors)
			Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "startDate", Message: "Start date cannot be in the past"}))
			Expect(vErrs).To(ContainElement(errs.ValidationError{Field: "endDate", Message: "End date must be on or after the start date"}))
			Expect(vErrs).To(HaveLen(2))
		})
	})

	Describe("ValidateUpdateJustification", func() {
		It("returns nil when justification is nil", func() {
			p := param.LeaveApplicationParam{}
			Expect(p.ValidateUpdateJustification()).To(Succeed())
		})

		It("returns nil when justification is within the max length", func() {
			reason := "Family emergency, need a few days off."
			p := param.LeaveApplicationParam{Justification: &reason}
			Expect(p.ValidateUpdateJustification()).To(Succeed())
		})

		It("returns nil when justification is exactly at the max length", func() {
			reason := strings.Repeat("a", 1000)
			p := param.LeaveApplicationParam{Justification: &reason}
			Expect(p.ValidateUpdateJustification()).To(Succeed())
		})

		// UT-050 (Input & Parameter Validation (Domain Rules)).
		It("returns error when justification exceeds the max length", func() {
			reason := strings.Repeat("a", 1001)
			p := param.LeaveApplicationParam{Justification: &reason}
			err := p.ValidateUpdateJustification()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(Equal(errs.ValidationErrors{
				{Field: "justification", Message: "Reason must be at most 1000 characters"},
			}))
		})
	})

	Describe("ValidateReject", func() {
		It("returns nil when remark is provided", func() {
			remark := "Insufficient staffing coverage during requested period."
			p := param.LeaveApplicationParam{Remark: &remark}
			Expect(p.ValidateReject()).To(Succeed())
		})

		It("returns error when remark is nil", func() {
			p := param.LeaveApplicationParam{}
			err := p.ValidateReject()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(Equal(errs.ValidationErrors{
				{Field: "remark", Message: "Please provide a reason for rejecting this leave application."},
			}))
		})

		// UT-050 (Input & Parameter Validation (Domain Rules)).
		It("returns error when remark is empty", func() {
			remark := ""
			p := param.LeaveApplicationParam{Remark: &remark}
			err := p.ValidateReject()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(Equal(errs.ValidationErrors{
				{Field: "remark", Message: "Please provide a reason for rejecting this leave application."},
			}))
		})

		// UT-050 (Input & Parameter Validation (Domain Rules)).
		It("returns error when remark is only whitespace", func() {
			remark := "   "
			p := param.LeaveApplicationParam{Remark: &remark}
			err := p.ValidateReject()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(Equal(errs.ValidationErrors{
				{Field: "remark", Message: "Please provide a reason for rejecting this leave application."},
			}))
		})

		// UT-050 (Input & Parameter Validation (Domain Rules)).
		It("returns error when remark exceeds the max length", func() {
			remark := strings.Repeat("a", 1001)
			p := param.LeaveApplicationParam{Remark: &remark}
			err := p.ValidateReject()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(Equal(errs.ValidationErrors{
				{Field: "remark", Message: "Remark must be at most 1000 characters"},
			}))
		})

		It("returns nil when remark is exactly at the max length", func() {
			remark := strings.Repeat("a", 1000)
			p := param.LeaveApplicationParam{Remark: &remark}
			Expect(p.ValidateReject()).To(Succeed())
		})
	})
})
