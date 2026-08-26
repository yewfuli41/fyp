package param_test

import (
	"fyp/domain/errs"
	"fyp/domain/param"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServiceSlotParam", func() {
	Describe("Validate", func() {
		var (
			startTime time.Time
			endTime   time.Time
		)

		BeforeEach(func() {
			startTime = time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
			endTime = time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC)
		})

		It("returns nil for a valid slot on a single future date", func() {
			p := param.ServiceSlotParam{
				ServiceOptionIDs: []int64{1},
				Date:             time.Now().AddDate(0, 0, 7).Format("2006-01-02"),
				StartTime:        startTime,
				EndTime:          endTime,
			}
			Expect(p.Validate()).To(Succeed())
		})

		It("returns nil for a valid recurring slot with days of week", func() {
			p := param.ServiceSlotParam{
				ServiceOptionIDs: []int64{1},
				DaysOfWeek:       []string{"monday", "wednesday"},
				StartTime:        startTime,
				EndTime:          endTime,
			}
			Expect(p.Validate()).To(Succeed())
		})

		It("returns error when no packages are selected", func() {
			p := param.ServiceSlotParam{
				DaysOfWeek: []string{"monday"},
				StartTime:  startTime,
				EndTime:    endTime,
			}
			err := p.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "serviceOptionIds", Message: "Please select a service and its option"},
			))
		})

		// UT-049 (Input & Parameter Validation (Domain Rules)).
		It("returns error when start time is not before end time", func() {
			p := param.ServiceSlotParam{
				ServiceOptionIDs: []int64{1},
				DaysOfWeek:       []string{"monday"},
				StartTime:        endTime,
				EndTime:          startTime,
			}
			err := p.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "startTime", Message: "Start time must be before end time"},
			))
		})

		// UT-049 (Input & Parameter Validation (Domain Rules)).
		It("returns error when start time equals end time", func() {
			p := param.ServiceSlotParam{
				ServiceOptionIDs: []int64{1},
				DaysOfWeek:       []string{"monday"},
				StartTime:        startTime,
				EndTime:          startTime,
			}
			err := p.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "startTime", Message: "Start time must be before end time"},
			))
		})

		// UT-050 (Input & Parameter Validation (Domain Rules)).
		It("returns error when neither date nor days of week are given", func() {
			p := param.ServiceSlotParam{
				ServiceOptionIDs: []int64{1},
				StartTime:        startTime,
				EndTime:          endTime,
			}
			err := p.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "schedule", Message: "Please select days of the week or a date"},
			))
		})

		// UT-050 (Input & Parameter Validation (Domain Rules)).
		It("returns error when both date and days of week are given", func() {
			p := param.ServiceSlotParam{
				ServiceOptionIDs: []int64{1},
				Date:             time.Now().AddDate(0, 0, 7).Format("2006-01-02"),
				DaysOfWeek:       []string{"monday"},
				StartTime:        startTime,
				EndTime:          endTime,
			}
			err := p.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "schedule", Message: "Choose either weekdays or a single date, not both"},
			))
		})

		It("returns error for an invalid date format", func() {
			p := param.ServiceSlotParam{
				ServiceOptionIDs: []int64{1},
				Date:             "not-a-date",
				StartTime:        startTime,
				EndTime:          endTime,
			}
			err := p.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "date", Message: "Invalid date"},
			))
		})

		// UT-051 (Input & Parameter Validation (Domain Rules)).
		It("returns error when the date and time are in the past", func() {
			p := param.ServiceSlotParam{
				ServiceOptionIDs: []int64{1},
				Date:             time.Now().AddDate(0, 0, -7).Format("2006-01-02"),
				StartTime:        startTime,
				EndTime:          endTime,
			}
			err := p.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.(errs.ValidationErrors)).To(ContainElement(
				errs.ValidationError{Field: "date", Message: "Date and time cannot be in the past"},
			))
		})

		It("returns multiple errors when several fields are invalid", func() {
			p := param.ServiceSlotParam{
				StartTime: endTime,
				EndTime:   startTime,
			}
			err := p.Validate()
			Expect(err).To(HaveOccurred())
			validationErrs := err.(errs.ValidationErrors)
			Expect(validationErrs).To(ContainElement(
				errs.ValidationError{Field: "serviceOptionIds", Message: "Please select a service and its option"},
			))
			Expect(validationErrs).To(ContainElement(
				errs.ValidationError{Field: "startTime", Message: "Start time must be before end time"},
			))
			Expect(validationErrs).To(ContainElement(
				errs.ValidationError{Field: "schedule", Message: "Please select days of the week or a date"},
			))
		})
	})
})
