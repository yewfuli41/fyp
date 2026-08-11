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

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("BookingService", func() {
	var (
		ctx          context.Context
		db           *sql.DB
		dbMock       sqlmock.Sqlmock
		bookingRepo  *mocks.MockIBookingRepo
		emailService *mocks.MockIEmailService
		serviceRepo  *mocks.MockIServiceRepo
		bookingSvc   interfaces.IBookingService

		customerUserID int64
		ownerUserID    int64
		bookingID      int64
		newSlotOption  int64
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		bookingRepo = mocks.NewMockIBookingRepo(GinkgoT())
		emailService = mocks.NewMockIEmailService(GinkgoT())
		serviceRepo = mocks.NewMockIServiceRepo(GinkgoT())
		bookingSvc = service.NewBookingService(db, bookingRepo, emailService, serviceRepo)

		customerUserID = 10
		ownerUserID = 1
		bookingID = 100
		newSlotOption = 200
	})

	AfterEach(func() {
		Expect(dbMock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("GetAvailableSlots", func() {
		It("sweeps past bookings then delegates straight to the repo (which itself checks the option's own validity window)", func() {
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().
				GetAvailableSlots(ctx, int64(1), int64(31), "2026-08-01", (*int64)(nil), false).
				Return([]param.ServiceSlotParam{{ServiceSlotID: 900}}, nil).
				Once()

			slots, err := bookingSvc.GetAvailableSlots(ctx, 1, 31, "2026-08-01", nil, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(slots).To(HaveLen(1))
		})
	})

	Describe("RescheduleBooking initiated by the customer", func() {
		It("sets status to pending (not rescheduled), so the customer cannot self-accept it", func() {
			ctxParam := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "accepted",
				CustomerUserID:  customerUserID,
				CustomerEmail:   "customer@example.com",
				CustomerName:    "Cust",
				BusinessOwnerID: ownerUserID,
				OwnerEmail:      "owner@example.com",
				OwnerName:       "Owner",
				SlotStaffUserID: nil, // owner-managed slot, no staff assigned
				BusinessName:    "Biz",
				WhenText:        "today",
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()
			bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, newSlotOption).Return(true, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingSlotOption(ctx, mock.Anything, bookingID, newSlotOption, "pending").
				Return(nil).Once()
			serviceRepo.EXPECT().
				DropSlotOptionIfExpired(ctx, mock.Anything, int64(0)).
				Return(nil).Once()
			dbMock.ExpectCommit()

			// Re-fetch after update — simulate the now-pending state.
			postUpdateCtx := *ctxParam
			postUpdateCtx.Status = "pending"
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(&postUpdateCtx, nil).Once()

			emailService.EXPECT().
				SendBookingStatusEmail("owner@example.com", "Owner", "Cust", "a reschedule request").
				Return(nil).Once()

			detail := &param.BookingDetailParam{BookingID: bookingID, Status: "pending"}
			bookingRepo.EXPECT().GetBookingDetail(ctx, bookingID).Return(detail, nil).Once()

			result, err := bookingSvc.RescheduleBooking(ctx, customerUserID, bookingID, newSlotOption)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("pending"))
		})
	})

	Describe("RescheduleBooking initiated by the business", func() {
		It("sets status to rescheduled, awaiting the customer's acceptance", func() {
			ctxParam := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "accepted",
				CustomerUserID:  customerUserID,
				CustomerEmail:   "customer@example.com",
				CustomerName:    "Cust",
				BusinessOwnerID: ownerUserID,
				OwnerEmail:      "owner@example.com",
				OwnerName:       "Owner",
				SlotStaffUserID: nil,
				BusinessName:    "Biz",
				WhenText:        "today",
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()
			bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, newSlotOption).Return(true, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingSlotOption(ctx, mock.Anything, bookingID, newSlotOption, "rescheduled").
				Return(nil).Once()
			serviceRepo.EXPECT().
				DropSlotOptionIfExpired(ctx, mock.Anything, int64(0)).
				Return(nil).Once()
			dbMock.ExpectCommit()

			postUpdateCtx := *ctxParam
			postUpdateCtx.Status = "rescheduled"
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(&postUpdateCtx, nil).Once()

			emailService.EXPECT().
				SendBookingStatusEmail("customer@example.com", "Cust", "Biz", "rescheduled").
				Return(nil).Once()

			detail := &param.BookingDetailParam{BookingID: bookingID, Status: "rescheduled"}
			bookingRepo.EXPECT().GetBookingDetail(ctx, bookingID).Return(detail, nil).Once()

			result, err := bookingSvc.RescheduleBooking(ctx, ownerUserID, bookingID, newSlotOption)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("rescheduled"))
		})

		It("moves a still-pending booking to rescheduled too, so the business can't later self-accept it", func() {
			ctxParam := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "pending",
				CustomerUserID:  customerUserID,
				CustomerEmail:   "customer@example.com",
				CustomerName:    "Cust",
				BusinessOwnerID: ownerUserID,
				OwnerEmail:      "owner@example.com",
				OwnerName:       "Owner",
				BusinessName:    "Biz",
				WhenText:        "today",
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()
			bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, newSlotOption).Return(true, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingSlotOption(ctx, mock.Anything, bookingID, newSlotOption, "rescheduled").
				Return(nil).Once()
			serviceRepo.EXPECT().
				DropSlotOptionIfExpired(ctx, mock.Anything, int64(0)).
				Return(nil).Once()
			dbMock.ExpectCommit()

			postUpdateCtx := *ctxParam
			postUpdateCtx.Status = "rescheduled"
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(&postUpdateCtx, nil).Once()

			emailService.EXPECT().
				SendBookingStatusEmail("customer@example.com", "Cust", "Biz", "rescheduled").
				Return(nil).Once()

			detail := &param.BookingDetailParam{BookingID: bookingID, Status: "rescheduled"}
			bookingRepo.EXPECT().GetBookingDetail(ctx, bookingID).Return(detail, nil).Once()

			result, err := bookingSvc.RescheduleBooking(ctx, ownerUserID, bookingID, newSlotOption)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("rescheduled"))

			// Now confirm the business genuinely cannot self-accept: AcceptBooking
			// only fires on "pending", and this booking is "rescheduled".
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(&postUpdateCtx, nil).Once()
			_, err = bookingSvc.AcceptBooking(ctx, ownerUserID, bookingID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Only pending bookings can be accepted."))
		})

		It("notifies both the old and new staff member when the owner reassigns the booking to a different staff's slot", func() {
			oldStaffUserID := int64(50)
			oldStaffEmail := "staffold@example.com"
			oldStaffName := "StaffOld"
			newStaffEmail := "staffnew@example.com"
			newStaffName := "StaffNew"

			ctxParam := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "accepted",
				CustomerUserID:  customerUserID,
				CustomerEmail:   "customer@example.com",
				CustomerName:    "Cust",
				BusinessOwnerID: ownerUserID,
				OwnerEmail:      "owner@example.com",
				OwnerName:       "Owner",
				SlotStaffUserID: &oldStaffUserID,
				StaffEmail:      &oldStaffEmail,
				StaffName:       &oldStaffName,
				BusinessName:    "Biz",
				WhenText:        "today",
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()
			bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, newSlotOption).Return(true, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingSlotOption(ctx, mock.Anything, bookingID, newSlotOption, "rescheduled").
				Return(nil).Once()
			serviceRepo.EXPECT().
				DropSlotOptionIfExpired(ctx, mock.Anything, int64(0)).
				Return(nil).Once()
			dbMock.ExpectCommit()

			// Re-fetch reflects the new slot's staff — a different person than before.
			postUpdateCtx := *ctxParam
			postUpdateCtx.Status = "rescheduled"
			postUpdateCtx.StaffEmail = &newStaffEmail
			postUpdateCtx.StaffName = &newStaffName
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(&postUpdateCtx, nil).Once()

			emailService.EXPECT().
				SendBookingStatusEmail("customer@example.com", "Cust", "Biz", "rescheduled").
				Return(nil).Once()
			emailService.EXPECT().
				SendBookingStatusEmail(oldStaffEmail, oldStaffName, "Cust", "rescheduled").
				Return(nil).Once()
			emailService.EXPECT().
				SendBookingStatusEmail(newStaffEmail, newStaffName, "Cust", "rescheduled").
				Return(nil).Once()

			detail := &param.BookingDetailParam{BookingID: bookingID, Status: "rescheduled"}
			bookingRepo.EXPECT().GetBookingDetail(ctx, bookingID).Return(detail, nil).Once()

			result, err := bookingSvc.RescheduleBooking(ctx, ownerUserID, bookingID, newSlotOption)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("rescheduled"))
		})

		It("notifies the staff member once, not twice, when the owner reschedules within the same staff's own slots", func() {
			staffUserID := int64(50)
			staffEmail := "staff@example.com"
			staffName := "Staff"

			ctxParam := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "accepted",
				CustomerUserID:  customerUserID,
				CustomerEmail:   "customer@example.com",
				CustomerName:    "Cust",
				BusinessOwnerID: ownerUserID,
				OwnerEmail:      "owner@example.com",
				OwnerName:       "Owner",
				SlotStaffUserID: &staffUserID,
				StaffEmail:      &staffEmail,
				StaffName:       &staffName,
				BusinessName:    "Biz",
				WhenText:        "today",
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()
			bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, newSlotOption).Return(true, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingSlotOption(ctx, mock.Anything, bookingID, newSlotOption, "rescheduled").
				Return(nil).Once()
			serviceRepo.EXPECT().
				DropSlotOptionIfExpired(ctx, mock.Anything, int64(0)).
				Return(nil).Once()
			dbMock.ExpectCommit()

			// Re-fetch shows the same staff still assigned after the move.
			postUpdateCtx := *ctxParam
			postUpdateCtx.Status = "rescheduled"
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(&postUpdateCtx, nil).Once()

			emailService.EXPECT().
				SendBookingStatusEmail("customer@example.com", "Cust", "Biz", "rescheduled").
				Return(nil).Once()
			// Exactly one call expected — mockery fails the test if the same
			// staff ends up emailed twice (once as "old", once as "new").
			emailService.EXPECT().
				SendBookingStatusEmail(staffEmail, staffName, "Cust", "rescheduled").
				Return(nil).Once()

			detail := &param.BookingDetailParam{BookingID: bookingID, Status: "rescheduled"}
			bookingRepo.EXPECT().GetBookingDetail(ctx, bookingID).Return(detail, nil).Once()

			result, err := bookingSvc.RescheduleBooking(ctx, ownerUserID, bookingID, newSlotOption)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("rescheduled"))
		})

		It("does not send an extra staff notification when a staff member reschedules their own booking (only the customer is notified)", func() {
			staffUserID := int64(50)
			staffEmail := "staff@example.com"
			staffName := "Staff"

			ctxParam := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "accepted",
				CustomerUserID:  customerUserID,
				CustomerEmail:   "customer@example.com",
				CustomerName:    "Cust",
				BusinessOwnerID: ownerUserID,
				OwnerEmail:      "owner@example.com",
				OwnerName:       "Owner",
				SlotStaffUserID: &staffUserID,
				StaffEmail:      &staffEmail,
				StaffName:       &staffName,
				BusinessName:    "Biz",
				WhenText:        "today",
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()
			bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, newSlotOption).Return(true, nil).Once()
			// A staff member (not the owner) may only reschedule onto their own slots.
			bookingRepo.EXPECT().GetSlotOptionStaffUserID(ctx, newSlotOption).Return(&staffUserID, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingSlotOption(ctx, mock.Anything, bookingID, newSlotOption, "rescheduled").
				Return(nil).Once()
			serviceRepo.EXPECT().
				DropSlotOptionIfExpired(ctx, mock.Anything, int64(0)).
				Return(nil).Once()
			dbMock.ExpectCommit()

			postUpdateCtx := *ctxParam
			postUpdateCtx.Status = "rescheduled"
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(&postUpdateCtx, nil).Once()

			emailService.EXPECT().
				SendBookingStatusEmail("customer@example.com", "Cust", "Biz", "rescheduled").
				Return(nil).Once()
			// No expectation is set for the staff's own email — the extra
			// staff-notification block only runs for an owner-initiated
			// reschedule, so mockery will fail this test if it fires here too.

			detail := &param.BookingDetailParam{BookingID: bookingID, Status: "rescheduled"}
			bookingRepo.EXPECT().GetBookingDetail(ctx, bookingID).Return(detail, nil).Once()

			result, err := bookingSvc.RescheduleBooking(ctx, staffUserID, bookingID, newSlotOption)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("rescheduled"))
		})
	})

	Describe("Full business-initiated reschedule workflow: accepted -> rescheduled -> customer accepts -> accepted", func() {
		It("lets the customer accept the business's reschedule proposal", func() {
			afterReschedule := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "rescheduled",
				CustomerUserID:  customerUserID,
				CustomerEmail:   "customer@example.com",
				CustomerName:    "Cust",
				BusinessOwnerID: ownerUserID,
				OwnerEmail:      "owner@example.com",
				OwnerName:       "Owner",
				BusinessName:    "Biz",
				WhenText:        "today",
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(afterReschedule, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingStatus(ctx, mock.Anything, bookingID, "accepted", customerUserID).
				Return(nil).Once()
			dbMock.ExpectCommit()

			emailService.EXPECT().
				SendBookingStatusEmail("owner@example.com", "Owner", "Cust", "accepted").
				Return(nil).Once()

			detail := &param.BookingDetailParam{BookingID: bookingID, Status: "accepted"}
			bookingRepo.EXPECT().GetBookingDetail(ctx, bookingID).Return(detail, nil).Once()

			result, err := bookingSvc.AcceptReschedule(ctx, customerUserID, bookingID)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("accepted"))
		})

		It("does not let the business self-accept its own reschedule proposal", func() {
			afterReschedule := &param.BookingContextParam{
				BookingID:      bookingID,
				Status:         "rescheduled",
				CustomerUserID: customerUserID,
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(afterReschedule, nil).Once()

			_, err := bookingSvc.AcceptReschedule(ctx, ownerUserID, bookingID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("You are not allowed to manage this booking."))
		})

		It("lets the business reject (withdraw) its own reschedule proposal instead", func() {
			afterReschedule := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "rescheduled",
				SlotOptionID:    777,
				CustomerUserID:  customerUserID,
				CustomerEmail:   "customer@example.com",
				CustomerName:    "Cust",
				BusinessOwnerID: ownerUserID,
				BusinessName:    "Biz",
				WhenText:        "today",
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(afterReschedule, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingStatus(ctx, mock.Anything, bookingID, "rejected", ownerUserID).
				Return(nil).Once()
			serviceRepo.EXPECT().
				DropSlotOptionIfExpired(ctx, mock.Anything, int64(777)).
				Return(nil).Once()
			dbMock.ExpectCommit()

			emailService.EXPECT().
				SendBookingStatusEmail("customer@example.com", "Cust", "Biz", "rejected").
				Return(nil).Once()

			detail := &param.BookingDetailParam{BookingID: bookingID, Status: "rejected"}
			bookingRepo.EXPECT().GetBookingDetail(ctx, bookingID).Return(detail, nil).Once()

			result, err := bookingSvc.RejectBooking(ctx, ownerUserID, bookingID)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("rejected"))
		})
	})

	Describe("Full customer-initiated reschedule workflow: accepted -> pending -> business accepts", func() {
		It("lets the business accept the customer's reschedule request", func() {
			pendingCtx := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "pending",
				ServiceSlotID:   555,
				CustomerUserID:  customerUserID,
				CustomerEmail:   "customer@example.com",
				CustomerName:    "Cust",
				BusinessOwnerID: ownerUserID,
				OwnerEmail:      "owner@example.com",
				OwnerName:       "Owner",
				BusinessName:    "Biz",
				WhenText:        "today",
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(pendingCtx, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingStatus(ctx, mock.Anything, bookingID, "accepted", ownerUserID).
				Return(nil).Once()
			bookingRepo.EXPECT().
				RejectOtherPendingBookingsForSlot(ctx, mock.Anything, pendingCtx.ServiceSlotID, bookingID).
				Return(nil).Once()
			dbMock.ExpectCommit()

			emailService.EXPECT().
				SendBookingStatusEmail("customer@example.com", "Cust", "Biz", "accepted").
				Return(nil).Once()

			detail := &param.BookingDetailParam{BookingID: bookingID, Status: "accepted"}
			bookingRepo.EXPECT().GetBookingDetail(ctx, bookingID).Return(detail, nil).Once()

			result, err := bookingSvc.AcceptBooking(ctx, ownerUserID, bookingID)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("accepted"))
		})

		It("lets the business reject the customer's reschedule request instead", func() {
			pendingCtx := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "pending",
				SlotOptionID:    778,
				CustomerUserID:  customerUserID,
				CustomerEmail:   "customer@example.com",
				CustomerName:    "Cust",
				BusinessOwnerID: ownerUserID,
				OwnerEmail:      "owner@example.com",
				OwnerName:       "Owner",
				BusinessName:    "Biz",
				WhenText:        "today",
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(pendingCtx, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingStatus(ctx, mock.Anything, bookingID, "rejected", ownerUserID).
				Return(nil).Once()
			serviceRepo.EXPECT().
				DropSlotOptionIfExpired(ctx, mock.Anything, int64(778)).
				Return(nil).Once()
			dbMock.ExpectCommit()

			emailService.EXPECT().
				SendBookingStatusEmail("customer@example.com", "Cust", "Biz", "rejected").
				Return(nil).Once()

			detail := &param.BookingDetailParam{BookingID: bookingID, Status: "rejected"}
			bookingRepo.EXPECT().GetBookingDetail(ctx, bookingID).Return(detail, nil).Once()

			result, err := bookingSvc.RejectBooking(ctx, ownerUserID, bookingID)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("rejected"))
		})
	})

	Describe("CreateBooking", func() {
		It("blocks an owner from booking their own business", func() {
			bookingRepo.EXPECT().IsSlotOptionOwnedByUser(ctx, newSlotOption, ownerUserID).Return(true, nil).Once()

			_, err := bookingSvc.CreateBooking(ctx, ownerUserID, newSlotOption, nil)
			Expect(err).To(HaveOccurred())
			ve, ok := err.(errs.ValidationErrors)
			Expect(ok).To(BeTrue())
			Expect(ve).To(ContainElement(errs.ValidationError{
				Field: "slotOptionId", Message: "You cannot book an appointment with your own business.",
			}))
		})

		It("emails the owner and assigned staff once the booking is created", func() {
			staffUserID := int64(20)
			staffEmail := "staff@example.com"
			staffName := "Staffer"
			created := &param.BookingParam{BookingID: bookingID, UserID: customerUserID, SlotOptionID: newSlotOption, Status: "pending"}
			ctxParam := &param.BookingContextParam{
				BookingID:       bookingID,
				CustomerUserID:  customerUserID,
				CustomerName:    "Cust",
				BusinessOwnerID: ownerUserID,
				OwnerEmail:      "owner@example.com",
				OwnerName:       "Owner",
				SlotStaffUserID: &staffUserID,
				StaffEmail:      &staffEmail,
				StaffName:       &staffName,
				BusinessName:    "Biz",
				WhenText:        "today",
			}

			bookingRepo.EXPECT().IsSlotOptionOwnedByUser(ctx, newSlotOption, customerUserID).Return(false, nil).Once()
			bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, newSlotOption).Return(true, nil).Once()
			bookingRepo.EXPECT().InsertBooking(ctx, customerUserID, newSlotOption, (*string)(nil)).Return(created, nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()
			emailService.EXPECT().
				SendNewBookingRequestEmail("owner@example.com", "Owner", "Cust", "today").
				Return(nil).Once()
			emailService.EXPECT().
				SendNewBookingRequestEmail("staff@example.com", "Staffer", "Cust", "today").
				Return(nil).Once()

			result, err := bookingSvc.CreateBooking(ctx, customerUserID, newSlotOption, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.BookingID).To(Equal(bookingID))
		})

		It("still returns the booking even if loading context for the notification email fails", func() {
			created := &param.BookingParam{BookingID: bookingID, UserID: customerUserID, SlotOptionID: newSlotOption, Status: "pending"}

			bookingRepo.EXPECT().IsSlotOptionOwnedByUser(ctx, newSlotOption, customerUserID).Return(false, nil).Once()
			bookingRepo.EXPECT().SlotOptionIsAvailable(ctx, newSlotOption).Return(true, nil).Once()
			bookingRepo.EXPECT().InsertBooking(ctx, customerUserID, newSlotOption, (*string)(nil)).Return(created, nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(nil, fmt.Errorf("db error")).Once()

			result, err := bookingSvc.CreateBooking(ctx, customerUserID, newSlotOption, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.BookingID).To(Equal(bookingID))
		})
	})

	Describe("Sweeping past bookings before every read/action", func() {
		It("flips past bookings to \"past\" before returning the business's list", func() {
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBusinessBookings(ctx, int64(1), (*int64)(nil)).
				Return([]param.BookingDetailParam{{BookingID: bookingID, Status: "past"}}, nil).Once()

			result, err := bookingSvc.GetBusinessBookings(ctx, 1, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Status).To(Equal("past"))
		})

		It("still returns the customer's list even if the sweep itself fails", func() {
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(fmt.Errorf("db unavailable")).Once()
			bookingRepo.EXPECT().GetCustomerBookings(ctx, customerUserID).
				Return([]param.BookingDetailParam{{BookingID: bookingID, Status: "accepted"}}, nil).Once()

			result, err := bookingSvc.GetCustomerBookings(ctx, customerUserID)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
		})
	})

	Describe("AcceptReschedule", func() {
		It("rejects the customer trying to accept their own now-pending reschedule request", func() {
			ctxParam := &param.BookingContextParam{
				BookingID:      bookingID,
				Status:         "pending", // set by the customer's own RescheduleBooking call above
				CustomerUserID: customerUserID,
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()

			_, err := bookingSvc.AcceptReschedule(ctx, customerUserID, bookingID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("This appointment has no reschedule to accept."))
		})
	})

	Describe("RecordWalkIn", func() {
		It("inserts the walk-in booking and returns its detail, with no availability/ownership checks", func() {
			slotOptionID := int64(321)
			created := &param.BookingParam{BookingID: bookingID, UserID: ownerUserID, SlotOptionID: slotOptionID, Status: "accepted", BookingType: "walk_in"}
			detail := &param.BookingDetailParam{BookingID: bookingID, Status: "accepted", BookingType: "walk_in"}

			bookingRepo.EXPECT().InsertWalkInBooking(ctx, ownerUserID, slotOptionID).Return(created, nil).Once()
			bookingRepo.EXPECT().GetBookingDetail(ctx, bookingID).Return(detail, nil).Once()

			result, err := bookingSvc.RecordWalkIn(ctx, ownerUserID, slotOptionID)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.BookingID).To(Equal(bookingID))
			Expect(result.BookingType).To(Equal("walk_in"))
		})

		It("propagates a DB error from InsertWalkInBooking without calling GetBookingDetail", func() {
			slotOptionID := int64(321)
			bookingRepo.EXPECT().InsertWalkInBooking(ctx, ownerUserID, slotOptionID).Return(nil, fmt.Errorf("db error")).Once()

			result, err := bookingSvc.RecordWalkIn(ctx, ownerUserID, slotOptionID)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("CancelBooking initiated by the customer", func() {
		It("cancels an accepted booking and notifies the business side (owner and staff)", func() {
			staffUserID := int64(20)
			staffEmail := "staff@example.com"
			staffName := "Staffer"
			ctxParam := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "accepted",
				SlotOptionID:    500,
				CustomerUserID:  customerUserID,
				CustomerName:    "Cust",
				BusinessOwnerID: ownerUserID,
				OwnerEmail:      "owner@example.com",
				OwnerName:       "Owner",
				SlotStaffUserID: &staffUserID,
				StaffEmail:      &staffEmail,
				StaffName:       &staffName,
				BusinessName:    "Biz",
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingStatus(ctx, mock.Anything, bookingID, "cancelled", customerUserID).
				Return(nil).Once()
			serviceRepo.EXPECT().
				DropSlotOptionIfExpired(ctx, mock.Anything, int64(500)).
				Return(nil).Once()
			dbMock.ExpectCommit()

			emailService.EXPECT().
				SendBookingStatusEmail("owner@example.com", "Owner", "Cust", "cancelled").
				Return(nil).Once()
			emailService.EXPECT().
				SendBookingStatusEmail("staff@example.com", "Staffer", "Cust", "cancelled").
				Return(nil).Once()

			detail := &param.BookingDetailParam{BookingID: bookingID, Status: "cancelled"}
			bookingRepo.EXPECT().GetBookingDetail(ctx, bookingID).Return(detail, nil).Once()

			result, err := bookingSvc.CancelBooking(ctx, customerUserID, bookingID)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("cancelled"))
		})
	})

	Describe("CancelBooking initiated by the business", func() {
		It("cancels a pending booking and notifies the customer instead", func() {
			ctxParam := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "pending",
				SlotOptionID:    501,
				CustomerUserID:  customerUserID,
				CustomerEmail:   "customer@example.com",
				CustomerName:    "Cust",
				BusinessOwnerID: ownerUserID,
				BusinessName:    "Biz",
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingStatus(ctx, mock.Anything, bookingID, "cancelled", ownerUserID).
				Return(nil).Once()
			serviceRepo.EXPECT().
				DropSlotOptionIfExpired(ctx, mock.Anything, int64(501)).
				Return(nil).Once()
			dbMock.ExpectCommit()

			emailService.EXPECT().
				SendBookingStatusEmail("customer@example.com", "Cust", "Biz", "cancelled").
				Return(nil).Once()

			detail := &param.BookingDetailParam{BookingID: bookingID, Status: "cancelled"}
			bookingRepo.EXPECT().GetBookingDetail(ctx, bookingID).Return(detail, nil).Once()

			result, err := bookingSvc.CancelBooking(ctx, ownerUserID, bookingID)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("cancelled"))
		})

		It("rejects a stranger (neither customer nor business side) trying to cancel", func() {
			ctxParam := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "accepted",
				CustomerUserID:  customerUserID,
				BusinessOwnerID: ownerUserID,
			}
			strangerUserID := int64(999)
			// Also exercises the best-effort sweep-failure branch (a failed sweep
			// should never block the actual cancel logic below it).
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(fmt.Errorf("sweep failed")).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()

			_, err := bookingSvc.CancelBooking(ctx, strangerUserID, bookingID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("You are not allowed to manage this booking."))
		})

		It("refuses to cancel a booking that is already rejected", func() {
			ctxParam := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "rejected",
				CustomerUserID:  customerUserID,
				BusinessOwnerID: ownerUserID,
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()

			_, err := bookingSvc.CancelBooking(ctx, customerUserID, bookingID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("This booking can no longer be cancelled."))
		})

		It("returns a not-found error when the booking context lookup fails with sql.ErrNoRows", func() {
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(nil, sql.ErrNoRows).Once()

			_, err := bookingSvc.CancelBooking(ctx, customerUserID, bookingID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Booking not found."))
		})

		It("propagates a generic DB error from the booking context lookup as-is (not the not-found message)", func() {
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(nil, fmt.Errorf("db unavailable")).Once()

			_, err := bookingSvc.CancelBooking(ctx, customerUserID, bookingID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("db unavailable"))
		})

		It("propagates a DB error from the transaction", func() {
			ctxParam := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "accepted",
				SlotOptionID:    502,
				CustomerUserID:  customerUserID,
				BusinessOwnerID: ownerUserID,
			}
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingStatus(ctx, mock.Anything, bookingID, "cancelled", customerUserID).
				Return(fmt.Errorf("update failed")).Once()
			dbMock.ExpectRollback()

			_, err := bookingSvc.CancelBooking(ctx, customerUserID, bookingID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("update failed"))
		})
	})

	Describe("UpdateBookingDescription", func() {
		It("lets the customer trim and update their own note", func() {
			ctxParam := &param.BookingContextParam{
				BookingID:      bookingID,
				Status:         "accepted",
				CustomerUserID: customerUserID,
			}
			newDescription := "  Please arrive early  "
			trimmed := "Please arrive early"
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingDescription(ctx, mock.Anything, bookingID, &trimmed).
				Return(nil).Once()
			dbMock.ExpectCommit()

			detail := &param.BookingDetailParam{BookingID: bookingID, Status: "accepted"}
			bookingRepo.EXPECT().GetBookingDetail(ctx, bookingID).Return(detail, nil).Once()

			result, err := bookingSvc.UpdateBookingDescription(ctx, customerUserID, bookingID, &newDescription)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.BookingID).To(Equal(bookingID))
		})

		It("normalizes a whitespace-only description to nil", func() {
			ctxParam := &param.BookingContextParam{
				BookingID:      bookingID,
				Status:         "pending",
				CustomerUserID: customerUserID,
			}
			blank := "   "
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingDescription(ctx, mock.Anything, bookingID, (*string)(nil)).
				Return(nil).Once()
			dbMock.ExpectCommit()

			detail := &param.BookingDetailParam{BookingID: bookingID, Status: "pending"}
			bookingRepo.EXPECT().GetBookingDetail(ctx, bookingID).Return(detail, nil).Once()

			result, err := bookingSvc.UpdateBookingDescription(ctx, customerUserID, bookingID, &blank)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.BookingID).To(Equal(bookingID))
		})

		It("rejects the business side trying to edit the customer's note", func() {
			ctxParam := &param.BookingContextParam{
				BookingID:       bookingID,
				Status:          "accepted",
				CustomerUserID:  customerUserID,
				BusinessOwnerID: ownerUserID,
			}
			desc := "New note"
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()

			_, err := bookingSvc.UpdateBookingDescription(ctx, ownerUserID, bookingID, &desc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("You are not allowed to manage this booking."))
		})

		It("refuses to edit the description of a rejected booking", func() {
			ctxParam := &param.BookingContextParam{
				BookingID:      bookingID,
				Status:         "rejected",
				CustomerUserID: customerUserID,
			}
			desc := "New note"
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()

			_, err := bookingSvc.UpdateBookingDescription(ctx, customerUserID, bookingID, &desc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("This booking can no longer be edited."))
		})

		It("returns a not-found error when the booking context lookup fails with sql.ErrNoRows", func() {
			desc := "New note"
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(nil, sql.ErrNoRows).Once()

			_, err := bookingSvc.UpdateBookingDescription(ctx, customerUserID, bookingID, &desc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Booking not found."))
		})

		It("propagates a DB error from the transaction", func() {
			ctxParam := &param.BookingContextParam{
				BookingID:      bookingID,
				Status:         "accepted",
				CustomerUserID: customerUserID,
			}
			desc := "New note"
			bookingRepo.EXPECT().GetBookingContext(ctx, bookingID).Return(ctxParam, nil).Once()

			dbMock.ExpectBegin()
			bookingRepo.EXPECT().
				UpdateBookingDescription(ctx, mock.Anything, bookingID, &desc).
				Return(fmt.Errorf("update failed")).Once()
			dbMock.ExpectRollback()

			_, err := bookingSvc.UpdateBookingDescription(ctx, customerUserID, bookingID, &desc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("update failed"))
		})
	})

	Describe("GetAvailableDates", func() {
		It("sweeps past bookings then delegates to the repo", func() {
			serviceID := int64(5)
			serviceOptionID := int64(31)
			staffID := int64(20)
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().
				GetAvailableDates(ctx, int64(1), &serviceID, &serviceOptionID, &staffID, false, "2026-08-01", "2026-08-31").
				Return([]string{"2026-08-05", "2026-08-06"}, nil).Once()

			dates, err := bookingSvc.GetAvailableDates(ctx, 1, &serviceID, &serviceOptionID, &staffID, false, "2026-08-01", "2026-08-31")
			Expect(err).NotTo(HaveOccurred())
			Expect(dates).To(Equal([]string{"2026-08-05", "2026-08-06"}))
		})

		It("still calls through to the repo even if the sweep itself fails", func() {
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(fmt.Errorf("db unavailable")).Once()
			bookingRepo.EXPECT().
				GetAvailableDates(ctx, int64(1), (*int64)(nil), (*int64)(nil), (*int64)(nil), true, "2026-08-01", "2026-08-31").
				Return(nil, fmt.Errorf("repo error")).Once()

			dates, err := bookingSvc.GetAvailableDates(ctx, 1, nil, nil, nil, true, "2026-08-01", "2026-08-31")
			Expect(err).To(HaveOccurred())
			Expect(dates).To(BeNil())
		})
	})

	Describe("GetRecentlyBookedBusinesses", func() {
		It("sweeps past bookings then delegates to the repo", func() {
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(nil).Once()
			bookingRepo.EXPECT().
				GetRecentlyBookedBusinesses(ctx, customerUserID).
				Return([]param.BusinessProfileParam{{BusinessID: 7, BusinessName: "Biz"}}, nil).Once()

			result, err := bookingSvc.GetRecentlyBookedBusinesses(ctx, customerUserID)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].BusinessName).To(Equal("Biz"))
		})

		It("still calls through to the repo (and propagates its error) even if the sweep itself fails", func() {
			bookingRepo.EXPECT().SweepPastBookings(ctx).Return(fmt.Errorf("db unavailable")).Once()
			bookingRepo.EXPECT().
				GetRecentlyBookedBusinesses(ctx, customerUserID).
				Return(nil, fmt.Errorf("repo error")).Once()

			result, err := bookingSvc.GetRecentlyBookedBusinesses(ctx, customerUserID)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})
})
