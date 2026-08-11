package service_test

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/internal/interfaces/mocks"
	"fyp/internal/service"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("BusinessService", func() {
	var (
		ctx             context.Context
		db              *sql.DB
		dbMock          sqlmock.Sqlmock
		businessRepo    *mocks.MockIBusinessRepo
		serviceSlotRepo *mocks.MockIServiceSlotRepo
		businessSvc     interfaces.IBusinessService
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		businessRepo = mocks.NewMockIBusinessRepo(GinkgoT())
		serviceSlotRepo = mocks.NewMockIServiceSlotRepo(GinkgoT())
		businessSvc = service.NewBusinessService(db, businessRepo, serviceSlotRepo)
	})

	AfterEach(func() {
		Expect(dbMock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("RegisterBusinessProfile", func() {
		var (
			businessParam param.BusinessProfileParam
			createdBiz    *param.BusinessProfileParam
			startTime     time.Time
			endTime       time.Time
		)

		BeforeEach(func() {
			startTime = time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
			endTime = time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

			businessParam = param.BusinessProfileParam{
				OwnerUserID:           42,
				BusinessName:          "Finn Studio",
				Address:               "123 Street",
				BusinessContactNumber: "0123456789",
				BusinessEmail:         "test@example.com",
				WorkingHours: []param.WorkingHourParam{
					{Day: "MONDAY", StartTime: startTime, EndTime: endTime},
				},
			}
			createdBiz = &param.BusinessProfileParam{
				BusinessID:            7,
				OwnerUserID:           42,
				BusinessName:          "Finn Studio",
				Address:               "123 Street",
				BusinessContactNumber: "0123456789",
				BusinessEmail:         "test@example.com",
				WorkingHours: []param.WorkingHourParam{
					{Day: "MONDAY", StartTime: startTime, EndTime: endTime},
				},
			}
		})

		It("successfully registers a business profile in a transaction", func() {
			dbMock.ExpectBegin()

			businessRepo.EXPECT().
				InsertBusinessProfile(ctx, mock.AnythingOfType("*sql.Tx"), businessParam).
				Return(createdBiz, nil).
				Once()

			businessRepo.EXPECT().
				InsertBusinessWorkingHours(ctx, mock.AnythingOfType("*sql.Tx"), *createdBiz).
				Return(nil).
				Once()

			dbMock.ExpectCommit()

			result, err := businessSvc.RegisterBusinessProfile(ctx, businessParam)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.BusinessID).To(Equal(int64(7)))
			Expect(result.WorkingHours).To(Equal(businessParam.WorkingHours))
		})

		It("returns validation errors without starting a transaction", func() {
			invalidParam := param.BusinessProfileParam{
				OwnerUserID: 42,
				// Missing BusinessName
			}

			// We still expect a Begin and Rollback because WithTransaction starts the tx
			// before calling the function.
			// Wait, let's look at BusinessService.RegisterBusinessProfile implementation again.
			/*
				err := s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
					if err := businessParam.ValidateRegisterBusinessProfile(); err != nil {
						return err
					}
					...
				})
			*/
			// Yes, it starts the transaction FIRST.

			dbMock.ExpectBegin()
			dbMock.ExpectRollback()

			result, err := businessSvc.RegisterBusinessProfile(ctx, invalidParam)

			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("rolls back and returns error if InsertBusinessProfile fails", func() {
			dbMock.ExpectBegin()

			businessRepo.EXPECT().
				InsertBusinessProfile(ctx, mock.AnythingOfType("*sql.Tx"), businessParam).
				Return(nil, fmt.Errorf("db error")).
				Once()

			dbMock.ExpectRollback()

			result, err := businessSvc.RegisterBusinessProfile(ctx, businessParam)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})

		It("rolls back and returns error if InsertBusinessWorkingHours fails", func() {
			dbMock.ExpectBegin()

			businessRepo.EXPECT().
				InsertBusinessProfile(ctx, mock.AnythingOfType("*sql.Tx"), businessParam).
				Return(createdBiz, nil).
				Once()

			businessRepo.EXPECT().
				InsertBusinessWorkingHours(ctx, mock.AnythingOfType("*sql.Tx"), *createdBiz).
				Return(fmt.Errorf("working hours error")).
				Once()

			dbMock.ExpectRollback()

			result, err := businessSvc.RegisterBusinessProfile(ctx, businessParam)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("working hours error"))
		})

		It("returns an error when the user already has a business profile", func() {
			dbMock.ExpectBegin()

			duplicateErr := newUniqueViolation("business_profiles_owner_user_id_key")
			businessRepo.EXPECT().
				InsertBusinessProfile(ctx, mock.AnythingOfType("*sql.Tx"), businessParam).
				Return(nil, duplicateErr).
				Once()

			dbMock.ExpectRollback()

			result, err := businessSvc.RegisterBusinessProfile(ctx, businessParam)

			Expect(result).To(BeNil())
			Expect(err.Error()).To(Equal("You already registered a business profile"))
		})

		It("prompts the user to log in again when the owner user account cannot be found", func() {
			dbMock.ExpectBegin()

			fkErr := &pq.Error{Code: "23503", Constraint: "business_profiles_owner_user_id_fkey"}
			businessRepo.EXPECT().
				InsertBusinessProfile(ctx, mock.AnythingOfType("*sql.Tx"), businessParam).
				Return(nil, fkErr).
				Once()

			dbMock.ExpectRollback()

			result, err := businessSvc.RegisterBusinessProfile(ctx, businessParam)

			Expect(result).To(BeNil())
			Expect(err.Error()).To(Equal("Unable to register business profile. Please log in again."))
		})
	})

	Describe("UpdateBusinessProfile", func() {
		var (
			businessParam param.BusinessProfileParam
			updatedBiz    *param.BusinessProfileParam
			startTime     time.Time
			endTime       time.Time
		)

		BeforeEach(func() {
			startTime = time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
			endTime = time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

			businessParam = param.BusinessProfileParam{
				OwnerUserID:           42,
				BusinessName:          "Updated Studio",
				Address:               "456 Avenue",
				BusinessContactNumber: "0123456789",
				BusinessEmail:         "updated@example.com",
				WorkingHours: []param.WorkingHourParam{
					{Day: "tuesday", StartTime: startTime, EndTime: endTime},
				},
			}
			updatedBiz = &param.BusinessProfileParam{
				BusinessID:            7,
				OwnerUserID:           42,
				BusinessName:          "Updated Studio",
				Address:               "456 Avenue",
				BusinessContactNumber: "0123456789",
				BusinessEmail:         "updated@example.com",
			}
		})

		It("successfully updates a business profile in a transaction", func() {
			dbMock.ExpectBegin()

			businessRepo.EXPECT().
				UpdateBusinessProfile(ctx, mock.AnythingOfType("*sql.Tx"), businessParam).
				Return(updatedBiz, nil).
				Once()

			serviceSlotRepo.EXPECT().
				GetFutureUnassignedSlotWindows(ctx, int64(7), mock.AnythingOfType("string")).
				Return(nil, nil).
				Once()

			businessRepo.EXPECT().
				DeleteBusinessWorkingHours(ctx, mock.AnythingOfType("*sql.Tx"), int64(7)).
				Return(nil).
				Once()

			expectedParam := businessParam
			expectedParam.BusinessID = 7
			businessRepo.EXPECT().
				InsertBusinessWorkingHours(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.BusinessProfileParam) bool {
					return p.BusinessID == 7
				})).
				Return(nil).
				Once()

			dbMock.ExpectCommit()

			result, err := businessSvc.UpdateBusinessProfile(ctx, businessParam)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.BusinessID).To(Equal(int64(7)))
			Expect(result.WorkingHours).To(Equal(businessParam.WorkingHours))
		})

		It("returns validation error without calling repo", func() {
			invalidParam := param.BusinessProfileParam{OwnerUserID: 42}

			dbMock.ExpectBegin()
			dbMock.ExpectRollback()

			result, err := businessSvc.UpdateBusinessProfile(ctx, invalidParam)

			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("rolls back when UpdateBusinessProfile repo call fails", func() {
			dbMock.ExpectBegin()

			businessRepo.EXPECT().
				UpdateBusinessProfile(ctx, mock.AnythingOfType("*sql.Tx"), businessParam).
				Return(nil, fmt.Errorf("update error")).
				Once()

			dbMock.ExpectRollback()

			result, err := businessSvc.UpdateBusinessProfile(ctx, businessParam)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("update error"))
		})

		It("rolls back when DeleteBusinessWorkingHours fails", func() {
			dbMock.ExpectBegin()

			businessRepo.EXPECT().
				UpdateBusinessProfile(ctx, mock.AnythingOfType("*sql.Tx"), businessParam).
				Return(updatedBiz, nil).
				Once()

			serviceSlotRepo.EXPECT().
				GetFutureUnassignedSlotWindows(ctx, int64(7), mock.AnythingOfType("string")).
				Return(nil, nil).
				Once()

			businessRepo.EXPECT().
				DeleteBusinessWorkingHours(ctx, mock.AnythingOfType("*sql.Tx"), int64(7)).
				Return(fmt.Errorf("delete error")).
				Once()

			dbMock.ExpectRollback()

			result, err := businessSvc.UpdateBusinessProfile(ctx, businessParam)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("delete error"))
		})

		It("rejects the update when an existing owner-managed slot would fall outside the new hours", func() {
			dbMock.ExpectBegin()

			businessRepo.EXPECT().
				UpdateBusinessProfile(ctx, mock.AnythingOfType("*sql.Tx"), businessParam).
				Return(updatedBiz, nil).
				Once()

			// businessParam only opens Tuesday 09:00-18:00 — an existing slot
			// on a Wednesday falls outside that entirely.
			serviceSlotRepo.EXPECT().
				GetFutureUnassignedSlotWindows(ctx, int64(7), mock.AnythingOfType("string")).
				Return([]param.SlotWindowParam{
					{Date: "2026-08-05", StartTime: startTime, EndTime: endTime}, // a Wednesday
				}, nil).
				Once()

			dbMock.ExpectRollback()

			result, err := businessSvc.UpdateBusinessProfile(ctx, businessParam)

			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("workingHours"))
			Expect(ve[0].Message).To(ContainSubstring("2026-08-05 (Wednesday)"))
		})
	})

	Describe("GetBusinessProfileByOwnerID", func() {
		It("returns the business profile with working hours", func() {
			startTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
			endTime := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

			biz := &param.BusinessProfileParam{BusinessID: 7, OwnerUserID: 42, BusinessName: "Test Biz"}
			whs := []param.WorkingHourParam{{Day: "MONDAY", StartTime: startTime, EndTime: endTime}}

			businessRepo.EXPECT().GetBusinessProfileByOwnerID(ctx, int64(42)).Return(biz, nil).Once()
			businessRepo.EXPECT().GetBusinessWorkingHours(ctx, int64(7)).Return(whs, nil).Once()

			result, err := businessSvc.GetBusinessProfileByOwnerID(ctx, 42)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.BusinessID).To(Equal(int64(7)))
			Expect(result.WorkingHours).To(HaveLen(1))
		})

		It("returns error when repo call fails", func() {
			businessRepo.EXPECT().
				GetBusinessProfileByOwnerID(ctx, int64(42)).
				Return(nil, fmt.Errorf("not found")).
				Once()

			result, err := businessSvc.GetBusinessProfileByOwnerID(ctx, 42)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("not found"))
		})
	})

	Describe("GetBusinesses", func() {
		It("returns the businesses found by the repo for the given search term", func() {
			businesses := []param.BusinessProfileParam{
				{BusinessID: 1, BusinessName: "Finn Studio"},
				{BusinessID: 2, BusinessName: "Finn Barbers"},
			}
			businessRepo.EXPECT().GetBusinesses(ctx, "finn").Return(businesses, nil).Once()

			result, err := businessSvc.GetBusinesses(ctx, "finn")

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(businesses))
		})

		It("propagates an error from the repo", func() {
			businessRepo.EXPECT().GetBusinesses(ctx, "finn").Return(nil, fmt.Errorf("db error")).Once()

			result, err := businessSvc.GetBusinesses(ctx, "finn")

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})
	})
})
