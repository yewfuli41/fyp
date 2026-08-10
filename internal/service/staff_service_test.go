package service_test

import (
	"context"
	"database/sql"
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

var _ = Describe("StaffService working hours", func() {
	var (
		ctx             context.Context
		db              *sql.DB
		dbMock          sqlmock.Sqlmock
		staffRepo       *mocks.MockIStaffRepo
		businessRepo    *mocks.MockIBusinessRepo
		authRepo        *mocks.MockIAuthRepo
		serviceSlotRepo *mocks.MockIServiceSlotRepo
		bookingRepo     *mocks.MockIBookingRepo
		emailService    *mocks.MockIEmailService
		staffSvc        interfaces.IStaffService
		businessID      int64
		staffID         int64
		businessHours   []param.WorkingHourParam
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		staffRepo = mocks.NewMockIStaffRepo(GinkgoT())
		businessRepo = mocks.NewMockIBusinessRepo(GinkgoT())
		authRepo = mocks.NewMockIAuthRepo(GinkgoT())
		serviceSlotRepo = mocks.NewMockIServiceSlotRepo(GinkgoT())
		bookingRepo = mocks.NewMockIBookingRepo(GinkgoT())
		emailService = mocks.NewMockIEmailService(GinkgoT())
		staffSvc = service.NewStaffService(db, staffRepo, businessRepo, authRepo, serviceSlotRepo, bookingRepo, emailService)

		businessID = 1
		staffID = 5
		businessHours = []param.WorkingHourParam{
			{Day: "monday", StartTime: time.Date(0, 1, 1, 8, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 20, 0, 0, 0, time.UTC)},
		}
	})

	AfterEach(func() {
		Expect(dbMock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("GetStaffHoursConflicts", func() {
		It("returns only the booked slots that fall outside the proposed hours", func() {
			newHours := []param.WorkingHourParam{
				{Day: "monday", StartTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 12, 0, 0, 0, time.UTC)},
			}
			bookedOutside := param.AssignedSlotParam{
				ServiceSlotID: 10, Date: "2026-08-03" /* Monday */, HasBooking: true,
				StartTime: time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 15, 0, 0, 0, time.UTC),
			}
			notBookedOutside := param.AssignedSlotParam{
				ServiceSlotID: 11, Date: "2026-08-03", HasBooking: false,
				StartTime: time.Date(0, 1, 1, 16, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 17, 0, 0, 0, time.UTC),
			}
			insideHours := param.AssignedSlotParam{
				ServiceSlotID: 12, Date: "2026-08-03", HasBooking: true,
				StartTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC),
			}
			serviceSlotRepo.EXPECT().GetFutureAssignedSlotWindows(ctx, staffID, mock.AnythingOfType("string")).
				Return([]param.AssignedSlotParam{bookedOutside, notBookedOutside, insideHours}, nil).Once()

			full := &param.ServiceSlotParam{ServiceSlotID: 10, Date: "2026-08-03"}
			serviceSlotRepo.EXPECT().GetServiceSlotByID(ctx, int64(10), businessID).Return(full, nil).Once()

			result, err := staffSvc.GetStaffHoursConflicts(ctx, businessID, staffID, newHours)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].ServiceSlotID).To(Equal(int64(10)))
		})
	})

	Describe("UpdateStaffWorkingHours", func() {
		It("returns a validation error for an empty schedule without touching the repo", func() {
			result, err := staffSvc.UpdateStaffWorkingHours(ctx, businessID, staffID, nil, nil)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("rejects hours outside the business's own hours", func() {
			newHours := []param.WorkingHourParam{
				{Day: "monday", StartTime: time.Date(0, 1, 1, 6, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)},
			}
			businessRepo.EXPECT().GetBusinessByID(ctx, businessID).
				Return(&param.BusinessProfileParam{BusinessID: businessID, WorkingHours: businessHours}, nil).Once()

			result, err := staffSvc.UpdateStaffWorkingHours(ctx, businessID, staffID, newHours, nil)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("rolls back when a booked slot outside the new hours has no reassignment", func() {
			newHours := []param.WorkingHourParam{
				{Day: "monday", StartTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 12, 0, 0, 0, time.UTC)},
			}
			businessRepo.EXPECT().GetBusinessByID(ctx, businessID).
				Return(&param.BusinessProfileParam{BusinessID: businessID, WorkingHours: businessHours}, nil).Once()

			booked := param.AssignedSlotParam{
				ServiceSlotID: 10, Date: "2026-08-03", HasBooking: true,
				StartTime: time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 15, 0, 0, 0, time.UTC),
			}
			serviceSlotRepo.EXPECT().GetFutureAssignedSlotWindows(ctx, staffID, mock.AnythingOfType("string")).
				Return([]param.AssignedSlotParam{booked}, nil).Once()

			dbMock.ExpectBegin()
			staffRepo.EXPECT().GetStaffByIDTx(ctx, mock.AnythingOfType("*sql.Tx"), staffID, businessID).
				Return(&param.StaffParam{StaffID: staffID, BusinessID: businessID}, nil).Once()
			dbMock.ExpectRollback()

			result, err := staffSvc.UpdateStaffWorkingHours(ctx, businessID, staffID, newHours, nil)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("reassignments"))
		})

		It("reassigns the booked slot, unassigns the non-booked one, and saves the new hours", func() {
			newHours := []param.WorkingHourParam{
				{Day: "monday", StartTime: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 12, 0, 0, 0, time.UTC)},
			}
			businessRepo.EXPECT().GetBusinessByID(ctx, businessID).
				Return(&param.BusinessProfileParam{BusinessID: businessID, WorkingHours: businessHours}, nil).Once()

			replacement := int64(8)
			bookedStart := time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC)
			bookedEnd := time.Date(0, 1, 1, 15, 0, 0, 0, time.UTC)
			booked := param.AssignedSlotParam{ServiceSlotID: 10, Date: "2026-08-03", HasBooking: true, StartTime: bookedStart, EndTime: bookedEnd}
			notBooked := param.AssignedSlotParam{
				ServiceSlotID: 11, Date: "2026-08-03", HasBooking: false,
				StartTime: time.Date(0, 1, 1, 16, 0, 0, 0, time.UTC), EndTime: time.Date(0, 1, 1, 17, 0, 0, 0, time.UTC),
			}
			serviceSlotRepo.EXPECT().GetFutureAssignedSlotWindows(ctx, staffID, mock.AnythingOfType("string")).
				Return([]param.AssignedSlotParam{booked, notBooked}, nil).Once()

			dbMock.ExpectBegin()
			staffRepo.EXPECT().GetStaffByIDTx(ctx, mock.AnythingOfType("*sql.Tx"), staffID, businessID).
				Return(&param.StaffParam{StaffID: staffID, BusinessID: businessID}, nil).Once()

			serviceSlotRepo.EXPECT().StaffBelongsToBusiness(ctx, replacement, businessID).Return(true, nil).Once()
			serviceSlotRepo.EXPECT().StaffCoversTime(ctx, replacement, mock.Anything, bookedStart, bookedEnd).Return(true, nil).Once()
			serviceSlotRepo.EXPECT().ReassignStaff(ctx, mock.AnythingOfType("*sql.Tx"), int64(10), businessID, &replacement).Return(nil).Once()
			bc := &param.BookingContextParam{BookingID: 400, CustomerEmail: "cust@example.com", CustomerName: "Cust", BusinessName: "Salon", WhenText: "Aug 3 14:00"}
			bookingRepo.EXPECT().GetBookingContextForSlot(ctx, int64(10)).Return(bc, nil).Once()
			serviceSlotRepo.EXPECT().ReassignStaff(ctx, mock.AnythingOfType("*sql.Tx"), int64(11), businessID, (*int64)(nil)).Return(nil).Once()

			staffRepo.EXPECT().DeleteStaffWorkingHours(ctx, mock.AnythingOfType("*sql.Tx"), staffID).Return(nil).Once()
			staffRepo.EXPECT().
				InsertStaffWorkingHours(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.StaffParam) bool {
					return p.StaffID == staffID && len(p.WorkingHours) == 1
				})).
				Return(nil).Once()
			dbMock.ExpectCommit()

			emailService.EXPECT().SendStaffReassignedEmail("cust@example.com", "Cust", "Salon", "Aug 3 14:00").Return(nil).Once()

			result, err := staffSvc.UpdateStaffWorkingHours(ctx, businessID, staffID, newHours, []param.SlotReassignmentParam{
				{ServiceSlotID: 10, StaffID: &replacement},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.WorkingHours).To(Equal(newHours))
		})
	})
})
