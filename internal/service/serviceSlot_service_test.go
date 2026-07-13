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
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("ServiceSlotService", func() {
	var (
		ctx        context.Context
		db         *sql.DB
		dbMock     sqlmock.Sqlmock
		slotRepo   *mocks.MockIServiceSlotRepo
		slotSvc    interfaces.IServiceSlotService
		startTime  time.Time
		endTime    time.Time
		futureDate string
		staffID    int64
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		slotRepo = mocks.NewMockIServiceSlotRepo(GinkgoT())
		slotSvc = service.NewServiceSlotService(db, slotRepo, config.ServiceSlotConfig{RecurringHorizonWeeks: 1})

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
			Expect(ve).To(ContainElement(errs.ValidationError{Field: "serviceOptionIds", Message: "One or more selected packages are invalid."}))
		})

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

		It("creates an owner-managed slot on a single date", func() {
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				Date:             futureDate,
				StartTime:        startTime,
				EndTime:          endTime,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()

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

		It("rolls back when InsertServiceSlot fails", func() {
			p := param.ServiceSlotParam{
				BusinessID:       1,
				ServiceOptionIDs: []int64{20},
				Date:             futureDate,
				StartTime:        startTime,
				EndTime:          endTime,
			}
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{20}).Return(true, nil).Once()

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

			dbMock.ExpectBegin()
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

			result, err := slotSvc.UpdateServiceSlot(ctx, p, false, nil)
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

			result, err := slotSvc.UpdateServiceSlot(ctx, p, false, &staffID)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{Field: "serviceSlotId", Message: "Service slot not found."}))
		})

		It("returns a validation error for an invalid update", func() {
			p := param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1}
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()

			result, err := slotSvc.UpdateServiceSlot(ctx, p, false, nil)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("returns an error when an affected occurrence already has a booking", func() {
			p := param.ServiceSlotParam{
				ServiceSlotID: 100, BusinessID: 1,
				ServiceOptionIDs: []int64{20}, Date: futureDate, StartTime: startTime, EndTime: endTime,
			}
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().HasBookingForServiceSlot(ctx, int64(100)).Return(true, nil).Once()

			result, err := slotSvc.UpdateServiceSlot(ctx, p, false, nil)
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

			result, err := slotSvc.UpdateServiceSlot(ctx, p, false, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceSlotID).To(Equal(int64(200)))
		})

		It("applies the update to every future occurrence of a recurring series", func() {
			p := param.ServiceSlotParam{
				ServiceSlotID: 100, BusinessID: 1,
				ServiceOptionIDs: []int64{21}, Date: futureDate, StartTime: startTime, EndTime: endTime,
			}
			recurringID := int64(9)
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, Date: "2026-08-01", RecurringScheduleID: &recurringID}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().GetFutureRecurringSlotIDs(ctx, int64(1), recurringID, "2026-08-01").Return([]int64{100, 105}, nil).Once()
			slotRepo.EXPECT().HasBookingForServiceSlot(ctx, int64(100)).Return(false, nil).Once()
			slotRepo.EXPECT().HasBookingForServiceSlot(ctx, int64(105)).Return(false, nil).Once()
			slotRepo.EXPECT().OptionsBelongToBusiness(ctx, int64(1), []int64{21}).Return(true, nil).Once()

			dbMock.ExpectBegin()
			slotRepo.EXPECT().SoftDeleteServiceSlots(ctx, mock.AnythingOfType("*sql.Tx"), []int64{100, 105}, int64(1)).Return(nil).Once()
			slotRepo.EXPECT().SoftDeleteRecurringSchedule(ctx, mock.AnythingOfType("*sql.Tx"), int64(1), recurringID).Return(nil).Once()
			slotRepo.EXPECT().
				InsertServiceSlot(ctx, mock.AnythingOfType("*sql.Tx"), mock.Anything).
				Return(int64(300), nil).Once()
			slotRepo.EXPECT().InsertServiceSlotOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(300), int64(21)).Return(nil).Once()
			dbMock.ExpectCommit()

			updated := &param.ServiceSlotParam{ServiceSlotID: 300, BusinessID: 1}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(300), int64(1)).Return(updated, nil).Once()

			result, err := slotSvc.UpdateServiceSlot(ctx, p, true, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceSlotID).To(Equal(int64(300)))
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

		It("returns an error when the new staff does not belong to the business", func() {
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, Date: futureDate}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().StaffBelongsToBusiness(ctx, staffID, int64(1)).Return(false, nil).Once()

			result, err := slotSvc.ReassignServiceSlotStaff(ctx, 100, 1, &staffID)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("returns an error when the new staff does not cover the slot's time", func() {
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, Date: futureDate, StartTime: startTime, EndTime: endTime}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().StaffBelongsToBusiness(ctx, staffID, int64(1)).Return(true, nil).Once()
			slotRepo.EXPECT().StaffCoversTime(ctx, staffID, mock.Anything, startTime, endTime).Return(false, nil).Once()

			result, err := slotSvc.ReassignServiceSlotStaff(ctx, 100, 1, &staffID)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("reassigns a slot to a new staff member", func() {
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, Date: futureDate, StartTime: startTime, EndTime: endTime}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().StaffBelongsToBusiness(ctx, staffID, int64(1)).Return(true, nil).Once()
			slotRepo.EXPECT().StaffCoversTime(ctx, staffID, mock.Anything, startTime, endTime).Return(true, nil).Once()

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

		It("deletes every future occurrence of a recurring series", func() {
			recurringID := int64(9)
			existing := &param.ServiceSlotParam{ServiceSlotID: 100, BusinessID: 1, Date: "2026-08-01", RecurringScheduleID: &recurringID}
			slotRepo.EXPECT().GetServiceSlotByID(ctx, int64(100), int64(1)).Return(existing, nil).Once()
			slotRepo.EXPECT().GetFutureRecurringSlotIDs(ctx, int64(1), recurringID, "2026-08-01").Return([]int64{100, 105}, nil).Once()
			slotRepo.EXPECT().HasBookingForServiceSlot(ctx, int64(100)).Return(false, nil).Once()
			slotRepo.EXPECT().HasBookingForServiceSlot(ctx, int64(105)).Return(false, nil).Once()

			dbMock.ExpectBegin()
			slotRepo.EXPECT().SoftDeleteServiceSlots(ctx, mock.AnythingOfType("*sql.Tx"), []int64{100, 105}, int64(1)).Return(nil).Once()
			slotRepo.EXPECT().SoftDeleteRecurringSchedule(ctx, mock.AnythingOfType("*sql.Tx"), int64(1), recurringID).Return(nil).Once()
			dbMock.ExpectCommit()

			err := slotSvc.DeleteServiceSlot(ctx, 100, 1, true, nil)
			Expect(err).NotTo(HaveOccurred())
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
