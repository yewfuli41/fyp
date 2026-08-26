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
		serviceRepo     *mocks.MockIServiceRepo
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
		serviceRepo = mocks.NewMockIServiceRepo(GinkgoT())
		emailService = mocks.NewMockIEmailService(GinkgoT())
		leaveSvc = service.NewLeaveService(db, leaveRepo, serviceSlotRepo, bookingRepo, businessRepo, serviceRepo, emailService)

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

		// UT-023 (Leave Application Rules).
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
		// UT-039 (Authorization Testing).
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
		// UT-024 (Leave Application Rules).
		It("rejects approving a non-pending application", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, BusinessID: businessID, Status: "approved"}, nil).Once()

			result, err := leaveSvc.ApproveLeaveApplication(ctx, businessID, 100, nil)
			Expect(result).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		// UT-025 (Leave Application Rules).
		It("blocks approving while a booked slot has no replacement — nothing is written", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, StaffID: staffID, BusinessID: businessID, Status: "pending", StartDate: futureStart, EndDate: futureEnd}, nil).Once()

			startTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
			endTime := time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC)
			booked := param.AssignedSlotParam{ServiceSlotID: 300, Date: futureStart, StartTime: startTime, EndTime: endTime, HasBooking: true}
			notBooked := param.AssignedSlotParam{ServiceSlotID: 301, Date: futureEnd, StartTime: startTime, EndTime: endTime, HasBooking: false}
			serviceSlotRepo.EXPECT().GetAssignedSlotsInRange(ctx, staffID, futureStart, futureEnd).
				Return([]param.AssignedSlotParam{booked, notBooked}, nil).Once()

			// No transaction is opened at all: the leave stays pending, slot
			// 301 keeps its staff member, and no customer is moved.
			result, err := leaveSvc.ApproveLeaveApplication(ctx, businessID, 100, nil)

			Expect(result).To(BeNil())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve[0].Field).To(Equal("reschedules"))
		})

		It("approves and frees the staff's slots when none of them is booked", func() {
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
				Return(&param.LeaveApplicationParam{LeaveID: 100, StaffID: staffID, BusinessID: businessID, Status: "pending", StartDate: futureStart, EndDate: futureEnd}, nil).Once()

			startTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
			endTime := time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC)
			notBooked := param.AssignedSlotParam{ServiceSlotID: 301, Date: futureEnd, StartTime: startTime, EndTime: endTime, HasBooking: false}
			serviceSlotRepo.EXPECT().GetAssignedSlotsInRange(ctx, staffID, futureStart, futureEnd).
				Return([]param.AssignedSlotParam{notBooked}, nil).Once()

			dbMock.ExpectBegin()
			serviceSlotRepo.EXPECT().ReassignStaff(ctx, mock.AnythingOfType("*sql.Tx"), int64(301), businessID, (*int64)(nil)).Return(nil).Once()
			leaveRepo.EXPECT().UpdateLeaveStatus(ctx, mock.AnythingOfType("*sql.Tx"), int64(100), "approved", (*string)(nil)).Return(nil).Once()
			dbMock.ExpectCommit()

			updated := &param.LeaveApplicationParam{LeaveID: 100, Status: "approved"}
			leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).Return(updated, nil).Once()

			result, err := leaveSvc.ApproveLeaveApplication(ctx, businessID, 100, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("approved"))
		})

		// Every test below covers the same guarantee from a different angle:
		// the reschedules an owner picks while settling a leave are held until
		// the approval itself commits, so a customer is never moved — nor
		// emailed — for a leave that stays pending.
		Describe("with reschedules", func() {
			var pendingLeave *param.LeaveApplicationParam
			var affectedBooking param.BookingDetailParam

			BeforeEach(func() {
				pendingLeave = &param.LeaveApplicationParam{
					LeaveID: 100, StaffID: staffID, BusinessID: businessID, Status: "pending",
					StartDate: futureStart, EndDate: futureEnd,
				}
				affectedBooking = param.BookingDetailParam{
					BookingID: 200, ServiceSlotID: 300, SlotOptionID: 400, Date: futureStart,
				}
			})

			It("moves the booking, frees the vacated slot and approves in one transaction, emailing the customer only after it commits", func() {
				leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).Return(pendingLeave, nil).Once()
				bookingRepo.EXPECT().GetBusinessBookings(ctx, businessID, &staffID).
					Return([]param.BookingDetailParam{affectedBooking}, nil).Once()
				bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, int64(500)).Return(true, nil).Once()
				// Owner-managed target, so it is not the leaving staff's own slot.
				bookingRepo.EXPECT().GetSlotOptionAssignment(ctx, int64(500)).Return((*int64)(nil), futureStart, nil).Once()

				startTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
				endTime := time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC)
				booked := param.AssignedSlotParam{ServiceSlotID: 300, Date: futureStart, StartTime: startTime, EndTime: endTime, HasBooking: true}
				serviceSlotRepo.EXPECT().GetAssignedSlotsInRange(ctx, staffID, futureStart, futureEnd).
					Return([]param.AssignedSlotParam{booked}, nil).Once()

				dbMock.ExpectBegin()
				bookingRepo.EXPECT().
					UpdateBookingSlotOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(200), int64(500), "rescheduled").
					Return(nil).Once()
				serviceRepo.EXPECT().
					DropSlotOptionIfExpired(ctx, mock.AnythingOfType("*sql.Tx"), int64(400)).Return(nil).Once()
				// Slot 300 held the booking we just moved off it, so it is now
				// empty and gets freed back to owner-managed like any other.
				serviceSlotRepo.EXPECT().
					ReassignStaff(ctx, mock.AnythingOfType("*sql.Tx"), int64(300), businessID, (*int64)(nil)).Return(nil).Once()
				leaveRepo.EXPECT().
					UpdateLeaveStatus(ctx, mock.AnythingOfType("*sql.Tx"), int64(100), "approved", (*string)(nil)).Return(nil).Once()
				dbMock.ExpectCommit()

				bookingRepo.EXPECT().GetBookingContext(ctx, int64(200)).Return(&param.BookingContextParam{
					BookingID: 200, CustomerName: "Dana", CustomerEmail: "dana@example.com", BusinessName: "Salon",
				}, nil).Once()
				emailService.EXPECT().
					SendBookingStatusEmail("dana@example.com", "Dana", "Salon", "rescheduled").Return(nil).Once()

				leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
					Return(&param.LeaveApplicationParam{LeaveID: 100, Status: "approved"}, nil).Once()

				result, err := leaveSvc.ApproveLeaveApplication(ctx, businessID, 100, []param.LeaveRescheduleParam{
					{BookingID: 200, NewSlotOptionID: 500},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Status).To(Equal("approved"))
			})

			It("aborts the whole approval before any write when a chosen slot has been taken in the meantime", func() {
				leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).Return(pendingLeave, nil).Once()
				bookingRepo.EXPECT().GetBusinessBookings(ctx, businessID, &staffID).
					Return([]param.BookingDetailParam{affectedBooking}, nil).Once()
				bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, int64(500)).Return(false, nil).Once()

				// No ExpectBegin: the transaction is never opened, so the leave
				// is still pending and the booking still sits on its old slot.
				result, err := leaveSvc.ApproveLeaveApplication(ctx, businessID, 100, []param.LeaveRescheduleParam{
					{BookingID: 200, NewSlotOptionID: 500},
				})
				Expect(result).To(BeNil())
				ve, ok := err.(errs.ValidationErrors)
				Expect(ok).To(BeTrue())
				Expect(ve[0].Field).To(Equal("reschedules"))
			})

			It("refuses a booking that this leave doesn't actually affect", func() {
				leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).Return(pendingLeave, nil).Once()
				bookingRepo.EXPECT().GetBusinessBookings(ctx, businessID, &staffID).
					Return([]param.BookingDetailParam{affectedBooking}, nil).Once()

				result, err := leaveSvc.ApproveLeaveApplication(ctx, businessID, 100, []param.LeaveRescheduleParam{
					{BookingID: 999, NewSlotOptionID: 500},
				})
				Expect(result).To(BeNil())
				ve, ok := err.(errs.ValidationErrors)
				Expect(ok).To(BeTrue())
				Expect(ve[0].Field).To(Equal("reschedules"))
			})

			It("refuses to move two bookings onto the same slot", func() {
				second := param.BookingDetailParam{BookingID: 201, ServiceSlotID: 301, SlotOptionID: 401, Date: futureEnd}
				leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).Return(pendingLeave, nil).Once()
				bookingRepo.EXPECT().GetBusinessBookings(ctx, businessID, &staffID).
					Return([]param.BookingDetailParam{affectedBooking, second}, nil).Once()
				bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, int64(500)).Return(true, nil).Once()
				bookingRepo.EXPECT().GetSlotOptionAssignment(ctx, int64(500)).Return((*int64)(nil), futureStart, nil).Once()

				result, err := leaveSvc.ApproveLeaveApplication(ctx, businessID, 100, []param.LeaveRescheduleParam{
					{BookingID: 200, NewSlotOptionID: 500},
					{BookingID: 201, NewSlotOptionID: 500},
				})
				Expect(result).To(BeNil())
				ve, ok := err.(errs.ValidationErrors)
				Expect(ok).To(BeTrue())
				Expect(ve[0].Field).To(Equal("reschedules"))
			})

			// UT-027 (Leave Application Rules). The other side of the same
			// rule: only the leave's own dates are off-limits. A slot the same
			// staff member covers on a day they are back at work is a
			// perfectly good replacement, and blocking it would leave the
			// owner with nowhere to move the customer.
			It("accepts one of the leaving staff's own slots on a day outside their leave", func() {
				outsideLeave := time.Now().AddDate(0, 0, 20).Format("2006-01-02")

				leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).Return(pendingLeave, nil).Once()
				bookingRepo.EXPECT().GetBusinessBookings(ctx, businessID, &staffID).
					Return([]param.BookingDetailParam{affectedBooking}, nil).Once()
				bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, int64(500)).Return(true, nil).Once()
				// Same staff member, but well after the leave ends.
				bookingRepo.EXPECT().GetSlotOptionAssignment(ctx, int64(500)).
					Return(&staffID, outsideLeave, nil).Once()

				startTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
				endTime := time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC)
				booked := param.AssignedSlotParam{ServiceSlotID: 300, Date: futureStart, StartTime: startTime, EndTime: endTime, HasBooking: true}
				serviceSlotRepo.EXPECT().GetAssignedSlotsInRange(ctx, staffID, futureStart, futureEnd).
					Return([]param.AssignedSlotParam{booked}, nil).Once()

				dbMock.ExpectBegin()
				bookingRepo.EXPECT().
					UpdateBookingSlotOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(200), int64(500), "rescheduled").
					Return(nil).Once()
				serviceRepo.EXPECT().
					DropSlotOptionIfExpired(ctx, mock.AnythingOfType("*sql.Tx"), int64(400)).Return(nil).Once()
				serviceSlotRepo.EXPECT().
					ReassignStaff(ctx, mock.AnythingOfType("*sql.Tx"), int64(300), businessID, (*int64)(nil)).Return(nil).Once()
				leaveRepo.EXPECT().
					UpdateLeaveStatus(ctx, mock.AnythingOfType("*sql.Tx"), int64(100), "approved", (*string)(nil)).Return(nil).Once()
				dbMock.ExpectCommit()

				bookingRepo.EXPECT().GetBookingContext(ctx, int64(200)).Return(&param.BookingContextParam{
					BookingID: 200, CustomerName: "Dana", CustomerEmail: "dana@example.com", BusinessName: "Salon",
				}, nil).Once()
				emailService.EXPECT().
					SendBookingStatusEmail("dana@example.com", "Dana", "Salon", "rescheduled").Return(nil).Once()

				leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
					Return(&param.LeaveApplicationParam{LeaveID: 100, Status: "approved"}, nil).Once()

				result, err := leaveSvc.ApproveLeaveApplication(ctx, businessID, 100, []param.LeaveRescheduleParam{
					{BookingID: 200, NewSlotOptionID: 500},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Status).To(Equal("approved"))
			})

			// UT-027 (Leave Application Rules). The picker already hides these,
			// but that's a client-side rule — the server must refuse them too.
			It("refuses a slot belonging to the leaving staff on a day inside their own leave", func() {
				leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).Return(pendingLeave, nil).Once()
				bookingRepo.EXPECT().GetBusinessBookings(ctx, businessID, &staffID).
					Return([]param.BookingDetailParam{affectedBooking}, nil).Once()
				bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, int64(500)).Return(true, nil).Once()
				// Free, but it's this same staff member's slot on a leave day —
				// moving the customer there leaves nobody to serve them.
				bookingRepo.EXPECT().GetSlotOptionAssignment(ctx, int64(500)).
					Return(&staffID, futureEnd, nil).Once()

				// No ExpectBegin: refused before the transaction is opened.
				result, err := leaveSvc.ApproveLeaveApplication(ctx, businessID, 100, []param.LeaveRescheduleParam{
					{BookingID: 200, NewSlotOptionID: 500},
				})
				Expect(result).To(BeNil())
				ve, ok := err.(errs.ValidationErrors)
				Expect(ok).To(BeTrue())
				Expect(ve[0].Field).To(Equal("reschedules"))
				Expect(ve[0].Message).To(ContainSubstring("covered by this leave"))
			})

			// A slot of theirs on a date the leave doesn't cover is fine —
			// they're only away for the range they asked for.
			It("allows the leaving staff's own slot on a date outside the leave", func() {
				outside := "2030-01-01"
				leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).Return(pendingLeave, nil).Once()
				bookingRepo.EXPECT().GetBusinessBookings(ctx, businessID, &staffID).
					Return([]param.BookingDetailParam{affectedBooking}, nil).Once()
				bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, int64(500)).Return(true, nil).Once()
				bookingRepo.EXPECT().GetSlotOptionAssignment(ctx, int64(500)).
					Return(&staffID, outside, nil).Once()
				serviceSlotRepo.EXPECT().GetAssignedSlotsInRange(ctx, staffID, futureStart, futureEnd).
					Return([]param.AssignedSlotParam{}, nil).Once()

				dbMock.ExpectBegin()
				bookingRepo.EXPECT().
					UpdateBookingSlotOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(200), int64(500), "rescheduled").
					Return(nil).Once()
				serviceRepo.EXPECT().
					DropSlotOptionIfExpired(ctx, mock.AnythingOfType("*sql.Tx"), int64(400)).Return(nil).Once()
				leaveRepo.EXPECT().
					UpdateLeaveStatus(ctx, mock.AnythingOfType("*sql.Tx"), int64(100), "approved", (*string)(nil)).Return(nil).Once()
				dbMock.ExpectCommit()

				bookingRepo.EXPECT().GetBookingContext(ctx, int64(200)).Return(&param.BookingContextParam{
					BookingID: 200, CustomerName: "Dana", CustomerEmail: "dana@example.com", BusinessName: "Salon",
				}, nil).Once()
				emailService.EXPECT().
					SendBookingStatusEmail("dana@example.com", "Dana", "Salon", "rescheduled").Return(nil).Once()
				leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).
					Return(&param.LeaveApplicationParam{LeaveID: 100, Status: "approved"}, nil).Once()

				result, err := leaveSvc.ApproveLeaveApplication(ctx, businessID, 100, []param.LeaveRescheduleParam{
					{BookingID: 200, NewSlotOptionID: 500},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Status).To(Equal("approved"))
			})

			It("rolls the whole approval back — leave included — when a later booking in the batch fails to move", func() {
				second := param.BookingDetailParam{BookingID: 201, ServiceSlotID: 301, SlotOptionID: 401, Date: futureEnd}
				leaveRepo.EXPECT().GetLeaveApplicationByID(ctx, int64(100)).Return(pendingLeave, nil).Once()
				bookingRepo.EXPECT().GetBusinessBookings(ctx, businessID, &staffID).
					Return([]param.BookingDetailParam{affectedBooking, second}, nil).Once()
				bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, int64(500)).Return(true, nil).Once()
				bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, int64(501)).Return(true, nil).Once()
				bookingRepo.EXPECT().GetSlotOptionAssignment(ctx, int64(500)).Return((*int64)(nil), futureStart, nil).Once()
				bookingRepo.EXPECT().GetSlotOptionAssignment(ctx, int64(501)).Return((*int64)(nil), futureEnd, nil).Once()
				serviceSlotRepo.EXPECT().GetAssignedSlotsInRange(ctx, staffID, futureStart, futureEnd).
					Return([]param.AssignedSlotParam{}, nil).Once()

				dbMock.ExpectBegin()
				bookingRepo.EXPECT().
					UpdateBookingSlotOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(200), int64(500), "rescheduled").
					Return(nil).Once()
				serviceRepo.EXPECT().
					DropSlotOptionIfExpired(ctx, mock.AnythingOfType("*sql.Tx"), int64(400)).Return(nil).Once()
				// The second move blows up, so the first one — and the approval
				// itself — must go back with it. No email is ever sent.
				bookingRepo.EXPECT().
					UpdateBookingSlotOption(ctx, mock.AnythingOfType("*sql.Tx"), int64(201), int64(501), "rescheduled").
					Return(fmt.Errorf("slot vanished")).Once()
				dbMock.ExpectRollback()

				result, err := leaveSvc.ApproveLeaveApplication(ctx, businessID, 100, []param.LeaveRescheduleParam{
					{BookingID: 200, NewSlotOptionID: 500},
					{BookingID: 201, NewSlotOptionID: 501},
				})
				Expect(result).To(BeNil())
				Expect(err).To(MatchError(ContainSubstring("slot vanished")))
			})
		})
	})

	Describe("RejectLeaveApplication", func() {
		// UT-026 (Leave Application Rules).
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

		// UT-026 (Leave Application Rules).
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

		// UT-039 (Authorization Testing).
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
