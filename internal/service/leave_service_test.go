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

var _ = Describe("LeaveService", func() {
	var (
		ctx             context.Context
		db              *sql.DB
		dbMock          sqlmock.Sqlmock
		leaveRepo       *mocks.MockILeaveRepo
		serviceSlotRepo *mocks.MockIServiceSlotRepo
		bookingRepo     *mocks.MockIBookingRepo
		businessRepo    *mocks.MockIBusinessRepo
		emailService    *mocks.MockIEmailService
		leaveSvc        interfaces.ILeaveService
		futureStart     string
		futureEnd       string
		staffID         int64
		businessID      int64
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		leaveRepo = mocks.NewMockILeaveRepo(GinkgoT())
		serviceSlotRepo = mocks.NewMockIServiceSlotRepo(GinkgoT())
		bookingRepo = mocks.NewMockIBookingRepo(GinkgoT())
		businessRepo = mocks.NewMockIBusinessRepo(GinkgoT())
		emailService = mocks.NewMockIEmailService(GinkgoT())
		leaveSvc = service.NewLeaveService(db, leaveRepo, serviceSlotRepo, bookingRepo, businessRepo, emailService)

		futureStart = time.Now().AddDate(0, 0, 7).Format("2006-01-02")
		futureEnd = time.Now().AddDate(0, 0, 9).Format("2006-01-02")
		staffID = 5
		businessID = 1
	})

	AfterEach(func() {
		Expect(dbMock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("ApplyLeave", func() {
		It("returns a validation error without touching the repo", func() {
			result, err := leaveSvc.ApplyLeave(ctx, staffID, param.LeaveApplicationParam{})
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("rejects an application overlapping an existing one", func() {
			p := param.LeaveApplicationParam{StartDate: futureStart, EndDate: futureEnd}
			leaveRepo.EXPECT().HasOverlappingLeave(ctx, staffID, futureStart, futureEnd).Return(true, nil).Once()

			result, err := leaveSvc.ApplyLeave(ctx, staffID, p)
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("startDate"))
		})

		It("creates the application and emails the owner", func() {
			p := param.LeaveApplicationParam{StartDate: futureStart, EndDate: futureEnd}
			leaveRepo.EXPECT().HasOverlappingLeave(ctx, staffID, futureStart, futureEnd).Return(false, nil).Once()

			dbMock.ExpectBegin()
			leaveRepo.EXPECT().
				InsertLeaveApplication(ctx, mock.AnythingOfType("*sql.Tx"), mock.MatchedBy(func(p param.LeaveApplicationParam) bool {
					return p.StaffID == staffID && p.StartDate == futureStart
				})).
				Return(int64(100), nil).Once()
			dbMock.ExpectCommit()

			created := &param.LeaveApplicationParam{
				LeaveID: 100, StaffID: staffID, BusinessID: businessID, StaffName: "Alice",
				StartDate: futureStart, EndDate: futureEnd, Status: "pending",
			}
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).Return(created, nil).Once()

			businessRepo.EXPECT().GetBusinessByID(ctx, businessID).
				Return(&param.BusinessProfileParam{BusinessID: businessID, BusinessName: "Salon", BusinessEmail: "owner@example.com"}, nil).Once()
			emailService.EXPECT().
				SendLeaveApplicationSubmittedEmail("owner@example.com", "Salon", "Alice", futureStart, futureEnd).
				Return(nil).Once()

			result, err := leaveSvc.ApplyLeave(ctx, staffID, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.LeaveID).To(Equal(int64(100)))
		})
	})

	Describe("DeleteLeaveApplication", func() {
		It("returns not found when the application doesn't belong to this staff", func() {
			otherStaff := int64(9)
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, StaffID: otherStaff}, nil).Once()

			err := leaveSvc.DeleteLeaveApplication(ctx, staffID, 100)
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("leaveId"))
		})

		It("deletes the staff's own application", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, StaffID: staffID}, nil).Once()

			dbMock.ExpectBegin()
			leaveRepo.EXPECT().SoftDeleteLeaveApplication(ctx, mock.AnythingOfType("*sql.Tx"), int64(100)).Return(nil).Once()
			dbMock.ExpectCommit()

			err := leaveSvc.DeleteLeaveApplication(ctx, staffID, 100)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetBusinessLeaveApplications", func() {
		It("attaches affected bookings to pending and approved applications, but not rejected ones", func() {
			pending := param.LeaveApplicationParam{LeaveID: 1, StaffID: staffID, Status: "pending", StartDate: futureStart, EndDate: futureEnd}
			approved := param.LeaveApplicationParam{LeaveID: 2, StaffID: staffID, Status: "approved", StartDate: futureStart, EndDate: futureEnd}
			rejected := param.LeaveApplicationParam{LeaveID: 3, StaffID: staffID, Status: "rejected", StartDate: futureStart, EndDate: futureEnd}
			leaveRepo.EXPECT().GetLeaveApplicationsByBusinessID(ctx, businessID).
				Return([]param.LeaveApplicationParam{pending, approved, rejected}, nil).Once()

			inRange := param.BookingDetailParam{BookingID: 200, Date: futureStart}
			outOfRange := param.BookingDetailParam{BookingID: 201, Date: "2020-01-01"}
			// Called once for "pending" and once for "approved" — "rejected" is skipped entirely.
			bookingRepo.EXPECT().GetBusinessBookings(ctx, businessID, &staffID).
				Return([]param.BookingDetailParam{inRange, outOfRange}, nil).Twice()

			result, err := leaveSvc.GetBusinessLeaveApplications(ctx, businessID)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(3))
			Expect(result[0].AffectedBookings).To(HaveLen(1))
			Expect(result[0].AffectedBookings[0].BookingID).To(Equal(int64(200)))
			Expect(result[1].AffectedBookings).To(HaveLen(1))
			Expect(result[2].AffectedBookings).To(BeEmpty())
		})
	})

	Describe("ApproveLeaveApplication", func() {
		It("rejects approving a non-pending application", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, BusinessID: businessID, Status: "approved"}, nil).Once()

			result, err := leaveSvc.ApproveLeaveApplication(ctx, businessID, 100)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("approves immediately, leaving a booked slot untouched and unassigning the non-booked one — no replacement picks required", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, StaffID: staffID, BusinessID: businessID, Status: "pending", StartDate: futureStart, EndDate: futureEnd}, nil).Once()

			startTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
			endTime := time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC)
			booked := param.AssignedSlotParam{ServiceSlotID: 300, Date: futureStart, StartTime: startTime, EndTime: endTime, HasBooking: true}
			notBooked := param.AssignedSlotParam{ServiceSlotID: 301, Date: futureEnd, StartTime: startTime, EndTime: endTime, HasBooking: false}
			serviceSlotRepo.EXPECT().GetAssignedSlotsInRange(ctx, staffID, futureStart, futureEnd).
				Return([]param.AssignedSlotParam{booked, notBooked}, nil).Once()

			dbMock.ExpectBegin()
			// Slot 300 (booked) gets no ReassignStaff call at all — approving
			// never touches it; only the non-booked slot 301 is freed.
			serviceSlotRepo.EXPECT().ReassignStaff(ctx, mock.AnythingOfType("*sql.Tx"), int64(301), businessID, (*int64)(nil)).Return(nil).Once()
			leaveRepo.EXPECT().UpdateLeaveStatus(ctx, mock.AnythingOfType("*sql.Tx"), int64(100), "approved", (*string)(nil)).Return(nil).Once()
			dbMock.ExpectCommit()

			updated := &param.LeaveApplicationParam{LeaveID: 100, Status: "approved"}
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).Return(updated, nil).Once()

			result, err := leaveSvc.ApproveLeaveApplication(ctx, businessID, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("approved"))
		})
	})

	Describe("RejectLeaveApplication", func() {
		It("requires a remark", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, BusinessID: businessID, Status: "pending"}, nil).Once()

			result, err := leaveSvc.RejectLeaveApplication(ctx, businessID, 100, "   ")
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("remark"))
		})

		It("rejects the application with a remark", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, BusinessID: businessID, Status: "pending"}, nil).Once()

			dbMock.ExpectBegin()
			leaveRepo.EXPECT().
				UpdateLeaveStatus(ctx, mock.AnythingOfType("*sql.Tx"), int64(100), "rejected", mock.MatchedBy(func(r *string) bool { return r != nil && *r == "Too busy" })).
				Return(nil).Once()
			dbMock.ExpectCommit()

			updated := &param.LeaveApplicationParam{LeaveID: 100, Status: "rejected"}
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).Return(updated, nil).Once()

			result, err := leaveSvc.RejectLeaveApplication(ctx, businessID, 100, "Too busy")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("rejected"))
		})

		It("also reverses an already-approved application (no separate cancel)", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, BusinessID: businessID, Status: "approved"}, nil).Once()

			dbMock.ExpectBegin()
			leaveRepo.EXPECT().
				UpdateLeaveStatus(ctx, mock.AnythingOfType("*sql.Tx"), int64(100), "rejected", mock.MatchedBy(func(r *string) bool { return r != nil && *r == "No longer needed" })).
				Return(nil).Once()
			dbMock.ExpectCommit()

			updated := &param.LeaveApplicationParam{LeaveID: 100, Status: "rejected"}
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).Return(updated, nil).Once()

			result, err := leaveSvc.RejectLeaveApplication(ctx, businessID, 100, "No longer needed")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("rejected"))
		})

		It("refuses to reject an already-rejected application", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, BusinessID: businessID, Status: "rejected"}, nil).Once()

			result, err := leaveSvc.RejectLeaveApplication(ctx, businessID, 100, "Too busy")
			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("leaveId"))
		})
	})
})
