package service_test

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/config"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/internal/interfaces/mocks"
	"fyp/internal/service"
	"strings"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("ServiceSlotService", func() {
	var (
		ctx         context.Context
		db          *sql.DB
		dbMock      sqlmock.Sqlmock
		slotRepo    *mocks.MockIServiceSlotRepo
		serviceRepo *mocks.MockIServiceRepo
		leaveRepo   *mocks.MockILeaveRepo
		slotSvc     interfaces.IServiceSlotService
		startTime   time.Time
		endTime     time.Time
		futureDate  string
		staffID     int64
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		slotRepo = mocks.NewMockIServiceSlotRepo(GinkgoT())
		serviceRepo = mocks.NewMockIServiceRepo(GinkgoT())
		leaveRepo = mocks.NewMockILeaveRepo(GinkgoT())
		slotSvc = service.NewServiceSlotService(db, slotRepo, serviceRepo, leaveRepo, config.ServiceSlotConfig{RecurringHorizonWeeks: 1})

		startTime = time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
		endTime = time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC)
		futureDate = time.Now().AddDate(0, 0, 7).Format("2006-01-02")
		staffID = 5
	})

	AfterEach(func() {
		Expect(dbMock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("CreateServiceSlot", func() {
		It("returns a validation error without touching the repo", func() {
			result, err := slotSvc.CreateServiceSlot(ctx, param.ServiceSlotParam{})
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})

		// UT-038 (Authorization Testing).
		It("returns an error when the selected packages are invalid", func() {
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				Date:             futureDate,
				StartTime:        startTime,
				EndTime:          endTime,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(false, nil).Once()

			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{Field: "serviceOptionIds", Message: "One or more selected options are invalid."}))
		})

		// UT-013 (Slot Scheduling & Staff Assignment).
		It("returns an error when the assigned staff does not belong to the business", func() {
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				StaffID:          &staffID,
				Date:             futureDate,
				StartTime:        startTime,
				EndTime:          endTime,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()
			slotRepo.EXPECT().StaffBelongsToBusiness(ctx, staffID, int64(1)).Return(false, nil).Once()

			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})

		// UT-013 (Slot Scheduling & Staff Assignment).
		It("returns an error when the staff does not work during the requested time", func() {
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				StaffID:          &staffID,
				Date:             futureDate,
				StartTime:        startTime,
				EndTime:          endTime,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()
			slotRepo.EXPECT().StaffBelongsToBusiness(ctx, staffID, int64(1)).Return(true, nil).Once()
			slotRepo.EXPECT().StaffCoversTime(ctx, staffID, mock.Anything, startTime, endTime).Return(false, nil).Once()

			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})

		// UT-014 (Slot Scheduling & Staff Assignment).
		It("returns an error when the business has no working hours during the requested time", func() {
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				Date:             futureDate,
				StartTime:        startTime,
				EndTime:          endTime,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()
			slotRepo.EXPECT().BusinessCoversTime(ctx, int64(1), mock.Anything, startTime, endTime).Return(false, nil).Once()

			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("startTime"))
		})

		It("creates an owner-managed slot on a single date", func() {
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				Date:             futureDate,
				StartTime:        startTime,
				EndTime:          endTime,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()
			slotRepo.EXPECT().BusinessCoversTime(ctx, int64(1), mock.Anything, startTime, endTime).Return(true, nil).Once()
			serviceRepo.EXPECT().IsOptionEffectiveOn(ctx, int64(20), futureDate).Return(true, nil).Once()

			dbMock.ExpectBegin()
			slotRepo.EXPECT().
				InsertServiceSlot(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(sp param.ServiceSlotParam) bool {
					return sp.Date == futureDate && sp.RecurringScheduleID == nil && sp.StaffID == nil
				})).
				Return(int64(100), nil).Once()
			slotRepo.EXPECT().InsertServiceSlotOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(100), int64(20)).Return(nil).Once()
			dbMock.ExpectCommit()

			created := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(created, nil).Once()

			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceSlotID).To(Equal(int64(100)))
		})

		// UT-015 (Slot Scheduling & Staff Assignment).
		It("drops one option from the slot when its own window doesn't cover that date, keeping the other", func() {
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20, 21},
				Date:             futureDate,
				StartTime:        startTime,
				EndTime:          endTime,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20, 21}).Return(true, nil).Once()
			slotRepo.EXPECT().BusinessCoversTime(ctx, int64(1), mock.Anything, startTime, endTime).Return(true, nil).Once()
			serviceRepo.EXPECT().IsOptionEffectiveOn(ctx, int64(20), futureDate).Return(false, nil).Once()
			serviceRepo.EXPECT().IsOptionEffectiveOn(ctx, int64(21), futureDate).Return(true, nil).Once()

			dbMock.ExpectBegin()
			slotRepo.EXPECT().
				InsertServiceSlot(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(sp param.ServiceSlotParam) bool {
					return sp.Date == futureDate
				})).
				Return(int64(100), nil).Once()
			// Only the effective option (21) is attached — 20 is simply omitted.
			slotRepo.EXPECT().InsertServiceSlotOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(100), int64(21)).Return(nil).Once()
			dbMock.ExpectCommit()

			created := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(created, nil).Once()

			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceSlotID).To(Equal(int64(100)))
		})

		// UT-015 (Slot Scheduling & Staff Assignment).
		It("rejects a single (non-recurring, non-walk-in) slot when none of its options are effective on that date", func() {
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				Date:             futureDate,
				StartTime:        startTime,
				EndTime:          endTime,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()
			slotRepo.EXPECT().BusinessCoversTime(ctx, int64(1), mock.Anything, startTime, endTime).Return(true, nil).Once()
			serviceRepo.EXPECT().IsOptionEffectiveOn(ctx, int64(20), futureDate).Return(false, nil).Once()

			dbMock.ExpectBegin()
			dbMock.ExpectRollback()

			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("serviceOptionIds"))
			Expect(ve[0].Message).To(Equal("None of the selected options are offered on " + futureDate + "."))
		})

		It("creates a recurring, staff-assigned slot", func() {
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				DaysOfWeek:       []string{"monday"},
				StaffID:          &staffID,
				StartTime:        startTime,
				EndTime:          endTime,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()
			slotRepo.EXPECT().StaffBelongsToBusiness(ctx, staffID, int64(1)).Return(true, nil).Once()
			slotRepo.EXPECT().StaffCoversTime(ctx, staffID, "monday", startTime, endTime).Return(true, nil).Once()
			leaveRepo.EXPECT().IsStaffOnLeave(ctx, staffID, mock.AnythingOfType("string")).Return(false, nil).Once()
			serviceRepo.EXPECT().IsOptionEffectiveOn(ctx, int64(20), mock.AnythingOfType("string")).Return(true, nil).Once()

			dbMock.ExpectBegin()
			slotRepo.EXPECT().
				InsertRecurringSchedule(ctx, mock.AnythingOfType("*sql.Tx"), mock.Anything, "monday").
				Return(int64(9), nil).Once()
			slotRepo.EXPECT().
				InsertServiceSlot(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(sp param.ServiceSlotParam) bool {
					return sp.RecurringScheduleID != nil && *sp.RecurringScheduleID == 9
				})).
				Return(int64(200), nil).Once()
			slotRepo.EXPECT().InsertServiceSlotOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(200), int64(20)).Return(nil).Once()
			dbMock.ExpectCommit()

			created := &param.ServiceSlotParam{ServiceSlotID: 200, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(200), int64(1)).Return(created, nil).Once()

			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceSlotID).To(Equal(int64(200)))
		})

		// UT-013 (Slot Scheduling & Staff Assignment).
		It("returns an error when the assigned staff is on approved leave on the slot's date", func() {
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				StaffID:          &staffID,
				Date:             futureDate,
				StartTime:        startTime,
				EndTime:          endTime,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()
			slotRepo.EXPECT().StaffBelongsToBusiness(ctx, staffID, int64(1)).Return(true, nil).Once()
			slotRepo.EXPECT().StaffCoversTime(ctx, staffID, mock.Anything, startTime, endTime).Return(true, nil).Once()
			leaveRepo.EXPECT().IsStaffOnLeave(ctx, staffID, futureDate).Return(true, nil).Once()

			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(result).To(BeNil())
			// A plain (non-field) error, unlike the other validation errors here —
			// it's surfaced as a general form message on the frontend rather than
			// pinned to the time picker's field feedback.
			Expect(err).To(MatchError("This staff member is on approved leave on " + futureDate + " — pick a different date or staff member."))
		})

		// UT-004 (Booking Business Rules).
		It("records a backdated walk-in against the option effective on that past date", func() {
			// A walk-in is a record of what was actually sold that day, so the
			// option is resolved against the walk-in's own date — including an
			// option that has since ended, which is exactly the case a
			// backdated walk-in needs and "still offered today" would block.
			pastDate := time.Now().AddDate(0, 0, -11)
			pastISO := pastDate.Format("2006-01-02")
			weekday := strings.ToLower(pastDate.Weekday().String())
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				Date:             pastISO,
				StartTime:        startTime,
				EndTime:          endTime,
				AllowPast:        true,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()
			slotRepo.EXPECT().BusinessCoversTime(ctx, int64(1), weekday, startTime, endTime).Return(true, nil).Once()
			serviceRepo.EXPECT().IsOptionEffectiveOn(ctx, int64(20), pastISO).Return(true, nil).Once()

			dbMock.ExpectBegin()
			slotRepo.EXPECT().
				InsertServiceSlot(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(sp param.ServiceSlotParam) bool {
					return sp.Date == pastISO
				})).
				Return(int64(200), nil).Once()
			slotRepo.EXPECT().InsertServiceSlotOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(200), int64(20)).Return(nil).Once()
			dbMock.ExpectCommit()
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(200), int64(1)).
				Return(&param.ServiceSlotParam{ServiceSlotID: 200}, nil).Once()

			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceSlotID).To(Equal(int64(200)))
		})

		// UT-004 (Booking Business Rules).
		It("refuses a backdated walk-in for an option that wasn't offered on that date", func() {
			// The mirror case: an option that hadn't started yet must not be
			// recordable against an earlier date, and the slot must not be
			// left behind optionless.
			pastDate := time.Now().AddDate(0, 0, -11)
			pastISO := pastDate.Format("2006-01-02")
			weekday := strings.ToLower(pastDate.Weekday().String())
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				Date:             pastISO,
				StartTime:        startTime,
				EndTime:          endTime,
				AllowPast:        true,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()
			slotRepo.EXPECT().BusinessCoversTime(ctx, int64(1), weekday, startTime, endTime).Return(true, nil).Once()
			serviceRepo.EXPECT().IsOptionEffectiveOn(ctx, int64(20), pastISO).Return(false, nil).Once()

			dbMock.ExpectBegin()
			dbMock.ExpectRollback()

			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("serviceOptionIds"))
			Expect(ve[0].Message).To(ContainSubstring(pastISO))
		})

		It("creates a recurring, owner-managed slot with its own recurring_schedules row", func() {
			// Owner-managed recurring series used to be skipped entirely
			// (recurring_schedules.staff_id was NOT NULL) — meaning they had
			// no series record to renew or bulk-delete by. Now they get a
			// row too, keyed by business_id instead of staff.
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				DaysOfWeek:       []string{"monday"},
				StartTime:        startTime,
				EndTime:          endTime,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()
			slotRepo.EXPECT().BusinessCoversTime(ctx, int64(1), "monday", startTime, endTime).Return(true, nil).Once()
			serviceRepo.EXPECT().IsOptionEffectiveOn(ctx, int64(20), mock.AnythingOfType("string")).Return(true, nil).Once()

			dbMock.ExpectBegin()
			slotRepo.EXPECT().
				InsertRecurringSchedule(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(sp param.ServiceSlotParam) bool {
					return sp.StaffID == nil && sp.BusinessID == 1
				}), "monday").
				Return(int64(9), nil).Once()
			slotRepo.EXPECT().
				InsertServiceSlot(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(sp param.ServiceSlotParam) bool {
					return sp.RecurringScheduleID != nil && *sp.RecurringScheduleID == 9 && sp.StaffID == nil
				})).
				Return(int64(200), nil).Once()
			slotRepo.EXPECT().InsertServiceSlotOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(200), int64(20)).Return(nil).Once()
			dbMock.ExpectCommit()

			created := &param.ServiceSlotParam{ServiceSlotID: 200, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(200), int64(1)).Return(created, nil).Once()

			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceSlotID).To(Equal(int64(200)))
		})

		// UT-018 (Slot Scheduling & Staff Assignment).
		It("bounds a recurring series by its own end date rather than the configured default", func() {
			// The suite's default is 1 week; an end date three weeks out must
			// win, producing three occurrences rather than one.
			today := time.Now()
			start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
			offset := (int(time.Monday) - int(start.Weekday()) + 7) % 7
			firstMonday := start.AddDate(0, 0, offset)
			thirdMonday := firstMonday.AddDate(0, 0, 14)

			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				DaysOfWeek:       []string{"monday"},
				StaffID:          &staffID,
				StartTime:        startTime,
				EndTime:          endTime,
				RecurringEndDate: thirdMonday.Format("2006-01-02"),
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()
			slotRepo.EXPECT().StaffBelongsToBusiness(ctx, staffID, int64(1)).Return(true, nil).Once()
			slotRepo.EXPECT().StaffCoversTime(ctx, staffID, "monday", startTime, endTime).Return(true, nil).Once()
			leaveRepo.EXPECT().IsStaffOnLeave(ctx, staffID, mock.AnythingOfType("string")).Return(false, nil).Times(3)
			serviceRepo.EXPECT().IsOptionEffectiveOn(ctx, int64(20), mock.AnythingOfType("string")).Return(true, nil).Times(3)

			dbMock.ExpectBegin()
			slotRepo.EXPECT().
				InsertRecurringSchedule(ctx, mock.AnythingOfType("*sql.Tx"), mock.Anything, "monday").
				Return(int64(9), nil).Once()
			slotRepo.EXPECT().
				InsertServiceSlot(ctx, mock.AnythingOfType("*sql.Tx"), mock.Anything).
				Return(int64(200), nil).Times(3)
			slotRepo.EXPECT().
				InsertServiceSlotOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(200), int64(20)).
				Return(nil).Times(3)
			dbMock.ExpectCommit()
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(200), int64(1)).
				Return(&param.ServiceSlotParam{ServiceSlotID: 200}, nil).Once()

			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceSlotID).To(Equal(int64(200)))
		})

		// UT-018 (Slot Scheduling & Staff Assignment).
		It("rejects an end date too early to catch any of the chosen weekdays", func() {
			// Yesterday: no upcoming Monday can fall on or before it.
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				DaysOfWeek:       []string{"monday"},
				StartTime:        startTime,
				EndTime:          endTime,
				RecurringEndDate: time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
			}

			// Refused by Validate before any repo call at all.
			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("recurringEndDate"))
		})

		// UT-015 (Slot Scheduling & Staff Assignment).
		It("skips recurring occurrences that fall outside every selected option's effective window, instead of creating an empty slot", func() {
			// 3-week horizon, but the option is only effective on the first
			// occurrence (e.g. an option with an effectiveUntil date) — the
			// later two Monday occurrences must simply not be created at all.
			horizonSvc := service.NewServiceSlotService(db, slotRepo, serviceRepo, leaveRepo, config.ServiceSlotConfig{RecurringHorizonWeeks: 3})
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				DaysOfWeek:       []string{"monday"},
				StaffID:          &staffID,
				StartTime:        startTime,
				EndTime:          endTime,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()
			slotRepo.EXPECT().StaffBelongsToBusiness(ctx, staffID, int64(1)).Return(true, nil).Once()
			slotRepo.EXPECT().StaffCoversTime(ctx, staffID, "monday", startTime, endTime).Return(true, nil).Once()
			leaveRepo.EXPECT().IsStaffOnLeave(ctx, staffID, mock.AnythingOfType("string")).Return(false, nil).Times(3)

			today := time.Now()
			start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
			offset := (int(time.Monday) - int(start.Weekday()) + 7) % 7
			firstMonday := start.AddDate(0, 0, offset).Format("2006-01-02")

			serviceRepo.EXPECT().IsOptionEffectiveOn(ctx, int64(20), mock.AnythingOfType("string")).
				RunAndReturn(func(_ context.Context, _ int64, date string) (bool, error) {
					// Only the very first resolved Monday is within the option's window.
					return date == firstMonday, nil
				}).Times(3)

			dbMock.ExpectBegin()
			slotRepo.EXPECT().
				InsertRecurringSchedule(ctx, mock.AnythingOfType("*sql.Tx"), mock.Anything, "monday").
				Return(int64(9), nil).Once()
			slotRepo.EXPECT().
				InsertServiceSlot(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(sp param.ServiceSlotParam) bool {
					return sp.RecurringScheduleID != nil && *sp.RecurringScheduleID == 9
				})).
				Return(int64(200), nil).Once() // only ONE insert — the other two occurrences are skipped
			slotRepo.EXPECT().InsertServiceSlotOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(200), int64(20)).Return(nil).Once()
			dbMock.ExpectCommit()

			created := &param.ServiceSlotParam{ServiceSlotID: 200, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(200), int64(1)).Return(created, nil).Once()

			result, err := horizonSvc.CreateServiceSlot(ctx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceSlotID).To(Equal(int64(200)))
		})

		It("rolls back when InsertServiceSlot fails", func() {
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				Date:             futureDate,
				StartTime:        startTime,
				EndTime:          endTime,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()
			slotRepo.EXPECT().BusinessCoversTime(ctx, int64(1), mock.Anything, startTime, endTime).Return(true, nil).Once()
			serviceRepo.EXPECT().IsOptionEffectiveOn(ctx, int64(20), futureDate).Return(true, nil).Once()

			dbMock.ExpectBegin()
			slotRepo.EXPECT().
				InsertServiceSlot(ctx, mock.AnythingOfType("*sql.Tx"), mock.Anything).
				Return(int64(0), fmt.Errorf("insert error")).Once()
			dbMock.ExpectRollback()

			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError("insert error"))
		})

		It("returns an error when no valid dates are resolved for the schedule", func() {
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				DaysOfWeek:       []string{"someday"},
				StartTime:        startTime,
				EndTime:          endTime,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()
			slotRepo.EXPECT().BusinessCoversTime(ctx, int64(1), "someday", startTime, endTime).Return(true, nil).Once()

			dbMock.ExpectBegin()
			// "someday" isn't a recognized weekday, so it resolves to zero
			// occurrences below — but insertRecurringSchedules still records
			// the series itself for every requested weekday regardless.
			slotRepo.EXPECT().
				InsertRecurringSchedule(ctx, mock.AnythingOfType("*sql.Tx"), mock.Anything, "someday").
				Return(int64(99), nil).Once()
			dbMock.ExpectCommit()

			result, err := slotSvc.CreateServiceSlot(ctx, p)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{Field: "schedule", Message: "No valid dates were resolved for the selected schedule."}))
		})
	})

	Describe("UpdateServiceSlot", func() {
		It("returns not found when the slot does not exist", func() {
			p := param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(nil, sql.ErrNoRows).Once()

			result, err := slotSvc.UpdateServiceSlot(ctx, p, nil)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{Field: "serviceSlotId", Message: "Service slot not found."}))
		})

		It("returns not found when the slot does not belong to the scoped staff", func() {
			otherStaff := int64(6)
			p := param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1}
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, StaffID: &otherStaff}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()

			result, err := slotSvc.UpdateServiceSlot(ctx, p, &staffID)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{Field: "serviceSlotId", Message: "Service slot not found."}))
		})

		It("returns a validation error for an invalid update", func() {
			p := param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1}
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()

			result, err := slotSvc.UpdateServiceSlot(ctx, p, nil)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		// UT-016 (Slot Scheduling & Staff Assignment).
		It("returns an error when an affected occurrence already has a booking", func() {
			p := param.ServiceSlotParam{
				ServiceSlotID: 100, BusinessID: 1,
				ServiceOptionIDs: []int64{20}, Date: futureDate, StartTime: startTime, EndTime: endTime,
			}
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().HasBookingForServiceSlot(ctx, int64(100)).Return(true, nil).Once()

			result, err := slotSvc.UpdateServiceSlot(ctx, p, nil)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("serviceSlotId"))
		})

		It("successfully updates a non-recurring slot", func() {
			p := param.ServiceSlotParam{
				ServiceSlotID: 100, BusinessID: 1,
				ServiceOptionIDs: []int64{21}, Date: futureDate, StartTime: startTime, EndTime: endTime,
			}
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().HasBookingForServiceSlot(ctx, int64(100)).Return(false, nil).Once()
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{21}).Return(true, nil).Once()
			slotRepo.EXPECT().BusinessCoversTime(ctx, int64(1), mock.Anything, startTime, endTime).Return(true, nil).Once()
			serviceRepo.EXPECT().IsOptionEffectiveOn(ctx, int64(21), futureDate).Return(true, nil).Once()

			dbMock.ExpectBegin()
			slotRepo.EXPECT().SoftDeleteServiceSlots(ctx, mock.AnythingOfType("*sql.Tx"), []int64{100}, int64(1)).Return(nil).Once()
			slotRepo.EXPECT().
				InsertServiceSlot(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(sp param.ServiceSlotParam) bool {
					return sp.Date == futureDate && sp.RecurringScheduleID == nil
				})).
				Return(int64(200), nil).Once()
			slotRepo.EXPECT().InsertServiceSlotOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(200), int64(21)).Return(nil).Once()
			dbMock.ExpectCommit()

			updated := &param.ServiceSlotParam{ServiceSlotID: 200, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(200), int64(1)).Return(updated, nil).Once()

			result, err := slotSvc.UpdateServiceSlot(ctx, p, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceSlotID).To(Equal(int64(200)))
		})
	})

	Describe("ReassignServiceSlotStaff", func() {
		It("returns not found when the slot does not exist", func() {
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(nil, sql.ErrNoRows).Once()

			result, err := slotSvc.ReassignServiceSlotStaff(ctx, 100, 1, &staffID)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{Field: "serviceSlotId", Message: "Service slot not found."}))
		})

		It("unassigns staff from a slot", func() {
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, Date: futureDate}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()

			dbMock.ExpectBegin()
			slotRepo.EXPECT().ReassignStaff(ctx, mock.AnythingOfType("*sql.Tx"), int64(100), int64(1), (*int64)(nil)).Return(nil).Once()
			dbMock.ExpectCommit()

			updated := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(updated, nil).Once()

			result, err := slotSvc.ReassignServiceSlotStaff(ctx, 100, 1, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceSlotID).To(Equal(int64(100)))
		})

		// UT-013 (Slot Scheduling & Staff Assignment).
		It("returns an error when the new staff does not belong to the business", func() {
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, Date: futureDate}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().StaffBelongsToBusiness(ctx, staffID, int64(1)).Return(false, nil).Once()

			result, err := slotSvc.ReassignServiceSlotStaff(ctx, 100, 1, &staffID)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		// UT-013 (Slot Scheduling & Staff Assignment).
		It("returns an error when the new staff does not cover the slot's time", func() {
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, Date: futureDate, StartTime: startTime, EndTime: endTime}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().StaffBelongsToBusiness(ctx, staffID, int64(1)).Return(true, nil).Once()
			slotRepo.EXPECT().StaffCoversTime(ctx, staffID, mock.Anything, startTime, endTime).Return(false, nil).Once()

			result, err := slotSvc.ReassignServiceSlotStaff(ctx, 100, 1, &staffID)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		// UT-013 (Slot Scheduling & Staff Assignment).
		It("returns an error when the new staff is on approved leave on the slot's date", func() {
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, Date: futureDate, StartTime: startTime, EndTime: endTime}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().StaffBelongsToBusiness(ctx, staffID, int64(1)).Return(true, nil).Once()
			slotRepo.EXPECT().StaffCoversTime(ctx, staffID, mock.Anything, startTime, endTime).Return(true, nil).Once()
			leaveRepo.EXPECT().IsStaffOnLeave(ctx, staffID, futureDate).Return(true, nil).Once()

			result, err := slotSvc.ReassignServiceSlotStaff(ctx, 100, 1, &staffID)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{Field: "staffId", Message: "This staff member is on approved leave on " + futureDate + " — pick a different staff member."}))
		})

		It("reassigns a slot to a new staff member", func() {
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, Date: futureDate, StartTime: startTime, EndTime: endTime}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().StaffBelongsToBusiness(ctx, staffID, int64(1)).Return(true, nil).Once()
			slotRepo.EXPECT().StaffCoversTime(ctx, staffID, mock.Anything, startTime, endTime).Return(true, nil).Once()
			leaveRepo.EXPECT().IsStaffOnLeave(ctx, staffID, futureDate).Return(false, nil).Once()

			dbMock.ExpectBegin()
			slotRepo.EXPECT().ReassignStaff(ctx, mock.AnythingOfType("*sql.Tx"), int64(100), int64(1), &staffID).Return(nil).Once()
			dbMock.ExpectCommit()

			updated := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, StaffID: &staffID}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(updated, nil).Once()

			result, err := slotSvc.ReassignServiceSlotStaff(ctx, 100, 1, &staffID)
			Expect(err).NotTo(HaveOccurred())
			Expect(*result.StaffID).To(Equal(staffID))
		})
	})

	Describe("DeleteServiceSlot", func() {
		It("returns not found when the slot does not exist", func() {
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(nil, sql.ErrNoRows).Once()

			err := slotSvc.DeleteServiceSlot(ctx, 100, 1, false, nil)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{Field: "serviceSlotId", Message: "Service slot not found."}))
		})

		It("returns not found when the slot does not belong to the scoped staff", func() {
			otherStaff := int64(6)
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, StaffID: &otherStaff}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()

			err := slotSvc.DeleteServiceSlot(ctx, 100, 1, false, &staffID)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{Field: "serviceSlotId", Message: "Service slot not found."}))
		})

		// UT-016 (Slot Scheduling & Staff Assignment).
		It("returns an error when the slot already has a booking", func() {
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().HasBookingForServiceSlot(ctx, int64(100)).Return(true, nil).Once()

			err := slotSvc.DeleteServiceSlot(ctx, 100, 1, false, nil)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("serviceSlotId"))
		})

		It("deletes a single occurrence", func() {
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().HasBookingForServiceSlot(ctx, int64(100)).Return(false, nil).Once()

			dbMock.ExpectBegin()
			slotRepo.EXPECT().SoftDeleteServiceSlot(ctx, mock.AnythingOfType("*sql.Tx"), int64(100), int64(1)).Return(nil).Once()
			dbMock.ExpectCommit()

			err := slotSvc.DeleteServiceSlot(ctx, 100, 1, false, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		// UT-017 (Slot Scheduling & Staff Assignment).
		It("deletes every occurrence from the selected slot's own date onward — not from today", func() {
			recurringID := int64(9)
			// Deliberately far in the future relative to "now" — if the
			// service ever regresses to using today's date instead of the
			// selected occurrence's, this exact-match expectation fails.
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, Date: "2026-09-01", RecurringScheduleID: &recurringID}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().GetFutureRecurringSlotIDs(ctx, int64(1), recurringID, "2026-09-01").Return([]int64{100, 105}, nil).Once()
			slotRepo.EXPECT().HasBookingForServiceSlot(ctx, int64(100)).Return(false, nil).Once()
			slotRepo.EXPECT().HasBookingForServiceSlot(ctx, int64(105)).Return(false, nil).Once()

			dbMock.ExpectBegin()
			slotRepo.EXPECT().SoftDeleteServiceSlots(ctx, mock.AnythingOfType("*sql.Tx"), []int64{100, 105}, int64(1)).Return(nil).Once()
			slotRepo.EXPECT().SoftDeleteRecurringSchedule(ctx, mock.AnythingOfType("*sql.Tx"), int64(1), recurringID).Return(nil).Once()
			dbMock.ExpectCommit()

			err := slotSvc.DeleteServiceSlot(ctx, 100, 1, true, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		// UT-017 (Slot Scheduling & Staff Assignment).
		It("deletes available future recurring occurrences and reports booked ones that were skipped", func() {
			recurringID := int64(9)
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, Date: "2026-08-01", RecurringScheduleID: &recurringID}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().GetFutureRecurringSlotIDs(ctx, int64(1), recurringID, mock.Anything).Return([]int64{100, 105, 106}, nil).Once()
			slotRepo.EXPECT().HasBookingForServiceSlot(ctx, int64(100)).Return(false, nil).Once()
			slotRepo.EXPECT().HasBookingForServiceSlot(ctx, int64(105)).Return(true, nil).Once()
			slotRepo.EXPECT().HasBookingForServiceSlot(ctx, int64(106)).Return(false, nil).Once()

			dbMock.ExpectBegin()
			slotRepo.EXPECT().SoftDeleteServiceSlots(ctx, mock.AnythingOfType("*sql.Tx"), []int64{100, 106}, int64(1)).Return(nil).Once()
			slotRepo.EXPECT().SoftDeleteRecurringSchedule(ctx, mock.AnythingOfType("*sql.Tx"), int64(1), recurringID).Return(nil).Once()
			dbMock.ExpectCommit()

			slotRepo.EXPECT().GetSlotDates(ctx, []int64{105}).Return([]string{"2026-08-08"}, nil).Once()

			err := slotSvc.DeleteServiceSlot(ctx, 100, 1, true, nil)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field:   "deleteFutureRecurring",
				Message: "Deleted available slots. Kept — already booked: 2026-08-08.",
			}))
		})
	})

	Describe("GetServiceSlots", func() {
		It("delegates to the repository", func() {
			slots := []param.ServiceSlotParam{{ServiceSlotID: 100}}
			slotRepo.EXPECT().GetServiceSlotsByBusinessAndDate(ctx, int64(1), futureDate, (*int64)(nil), (*int64)(nil), false).Return(slots, nil).Once()

			result, err := slotSvc.GetServiceSlots(ctx, 1, futureDate, nil, nil, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
		})

		It("returns an error from the repository", func() {
			slotRepo.EXPECT().GetServiceSlotsByBusinessAndDate(ctx, int64(1), futureDate, (*int64)(nil), (*int64)(nil), false).Return(nil, fmt.Errorf("db error")).Once()

			result, err := slotSvc.GetServiceSlots(ctx, 1, futureDate, nil, nil, false)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})
	})

	Describe("GetAvailableStaffForSlot", func() {
		It("returns not found when the slot does not exist", func() {
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(nil, sql.ErrNoRows).Once()

			result, err := slotSvc.GetAvailableStaffForSlot(ctx, 100, 1)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{Field: "serviceSlotId", Message: "Service slot not found."}))
		})

		It("returns available staff for the slot's date and time", func() {
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, Date: futureDate, StartTime: startTime, EndTime: endTime}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()

			staffList := []param.StaffParam{{StaffID: staffID, StaffName: "Alice"}}
			slotRepo.EXPECT().
				GetAvailableStaff(ctx, int64(1), futureDate, mock.Anything, startTime, endTime, int64(100)).
				Return(staffList, nil).Once()

			result, err := slotSvc.GetAvailableStaffForSlot(ctx, 100, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].StaffName).To(Equal("Alice"))
		})
	})
})
