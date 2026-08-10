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
				SendBookingStatusEmail("owner@example.com", "Owner", "Cust", "a reschedule request", "today").
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
				SendBookingStatusEmail("customer@example.com", "Cust", "Biz", "rescheduled", "today").
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
				SendBookingStatusEmail("customer@example.com", "Cust", "Biz", "rescheduled", "today").
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
				SendBookingStatusEmail("owner@example.com", "Owner", "Cust", "accepted", "today").
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
				SendBookingStatusEmail("customer@example.com", "Cust", "Biz", "rejected", "today").
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
				SendBookingStatusEmail("customer@example.com", "Cust", "Biz", "accepted", "today").
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
				SendBookingStatusEmail("customer@example.com", "Cust", "Biz", "rejected", "today").
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
})
