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
	"strings"
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

		// UT-029 (Leave Application Rules).
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
		// UT-030 (Leave Application Rules).
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
		// UT-031 (Leave Application Rules).
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
		// UT-032 (Leave Application Rules).
		It("rejects approving a non-pending application", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, BusinessID: businessID, Status: "approved"}, nil).Once()

			result, err := leaveSvc.ApproveLeaveApplication(ctx, businessID, 100)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		// UT-033 (Leave Application Rules).
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
		// UT-034 (Leave Application Rules).
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

		// UT-034 (Leave Application Rules).
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

	Describe("GetMyLeaveApplications", func() {
		It("returns the staff's own leave applications", func() {
			apps := []param.LeaveApplicationParam{
				{LeaveID: 1, StaffID: staffID, Status: "pending"},
				{LeaveID: 2, StaffID: staffID, Status: "approved"},
			}
			leaveRepo.EXPECT().GetLeaveApplicationsByStaffID(ctx, staffID).Return(apps, nil).Once()

			result, err := leaveSvc.GetMyLeaveApplications(ctx, staffID)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(apps))
		})

		It("propagates an error from the repo", func() {
			leaveRepo.EXPECT().GetLeaveApplicationsByStaffID(ctx, staffID).Return(nil, fmt.Errorf("db error")).Once()

			result, err := leaveSvc.GetMyLeaveApplications(ctx, staffID)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})
	})

	Describe("UpdateLeaveApplication", func() {
		It("propagates an error when the leave lookup itself fails", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(nil, fmt.Errorf("db error")).Once()

			justification := "New reason"
			result, err := leaveSvc.UpdateLeaveApplication(ctx, staffID, 100, &justification)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})

		// UT-030 (Leave Application Rules).
		It("returns not found when the application doesn't belong to this staff", func() {
			otherStaff := int64(9)
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, StaffID: otherStaff, Status: "pending"}, nil).Once()

			justification := "New reason"
			result, err := leaveSvc.UpdateLeaveApplication(ctx, staffID, 100, &justification)

			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("leaveId"))
		})

		It("refuses to edit an application that is no longer pending", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, StaffID: staffID, Status: "approved"}, nil).Once()

			justification := "New reason"
			result, err := leaveSvc.UpdateLeaveApplication(ctx, staffID, 100, &justification)

			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("leaveId"))
			Expect(ve[0].Message).To(ContainSubstring("Only pending"))
		})

		It("returns a validation error when the justification is too long, without starting a transaction", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, StaffID: staffID, Status: "pending"}, nil).Once()

			tooLong := strings.Repeat("a", 1001)
			result, err := leaveSvc.UpdateLeaveApplication(ctx, staffID, 100, &tooLong)

			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("justification"))
		})

		It("trims the justification and updates it for the staff's own pending application", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, StaffID: staffID, Status: "pending"}, nil).Once()

			justification := "  Updated reason  "
			dbMock.ExpectBegin()
			leaveRepo.EXPECT().
				UpdateLeaveJustification(ctx, mock.AnythingOfType("*sql.Tx"), int64(100), mock.MatchedBy(func(j *string) bool {
					return j != nil && *j == "Updated reason"
				})).
				Return(nil).Once()
			dbMock.ExpectCommit()

			updated := &param.LeaveApplicationParam{LeaveID: 100, StaffID: staffID, Status: "pending", Justification: &justification}
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).Return(updated, nil).Once()

			result, err := leaveSvc.UpdateLeaveApplication(ctx, staffID, 100, &justification)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.LeaveID).To(Equal(int64(100)))
		})

		It("treats a blank justification as clearing it", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, StaffID: staffID, Status: "pending"}, nil).Once()

			blank := "   "
			dbMock.ExpectBegin()
			leaveRepo.EXPECT().
				UpdateLeaveJustification(ctx, mock.AnythingOfType("*sql.Tx"), int64(100), (*string)(nil)).
				Return(nil).Once()
			dbMock.ExpectCommit()

			updated := &param.LeaveApplicationParam{LeaveID: 100, StaffID: staffID, Status: "pending"}
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).Return(updated, nil).Once()

			result, err := leaveSvc.UpdateLeaveApplication(ctx, staffID, 100, &blank)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.LeaveID).To(Equal(int64(100)))
		})

		It("rolls back and propagates an error from UpdateLeaveJustification", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, StaffID: staffID, Status: "pending"}, nil).Once()

			justification := "Updated reason"
			dbMock.ExpectBegin()
			leaveRepo.EXPECT().
				UpdateLeaveJustification(ctx, mock.AnythingOfType("*sql.Tx"), int64(100), mock.AnythingOfType("*string")).
				Return(fmt.Errorf("db error")).Once()
			dbMock.ExpectRollback()

			result, err := leaveSvc.UpdateLeaveApplication(ctx, staffID, 100, &justification)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("db error"))
		})
	})
})
