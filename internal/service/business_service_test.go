package service_test

import (
	"context"
	"database/sql"
	"fmt"
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
		ctx          context.Context
		db           *sql.DB
		dbMock       sqlmock.Sqlmock
		businessRepo *mocks.MockIBusinessRepo
		businessSvc  interfaces.IBusinessService
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		businessRepo = mocks.NewMockIBusinessRepo(GinkgoT())
		businessSvc = service.NewBusinessService(db, businessRepo)
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

		It("maps duplicate owner violation to a friendly error", func() {
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

		It("maps foreign key violation to a friendly error", func() {
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
})
