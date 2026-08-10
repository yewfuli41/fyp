package service

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/database"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"strings"

	"github.com/labstack/gommon/log"
)

type bookingService struct {
	bookingRepo  interfaces.IBookingRepo
	emailService interfaces.IEmailService
	serviceRepo  interfaces.IServiceRepo
	tx           *database.Transaction
}

func NewBookingService(db *sql.DB, bookingRepo interfaces.IBookingRepo, emailService interfaces.IEmailService, serviceRepo interfaces.IServiceRepo) interfaces.IBookingService {
	return &bookingService{
		bookingRepo:  bookingRepo,
		emailService: emailService,
		serviceRepo:  serviceRepo,
		tx:           database.NewTransaction(db),
	}
}

func (s *bookingService) GetAvailableSlots(ctx context.Context, businessID int64, serviceOptionID int64, date string, staffID *int64, unassignedOnly bool) ([]param.ServiceSlotParam, error) {
	if err := s.bookingRepo.SweepPastBookings(ctx); err != nil {
		log.Errorf("failed to sweep past bookings: %v", err)
	}
	// bookingRepo.GetAvailableSlots itself checks the option's own
	// [effective_from, effective_until] window against date, so a slot only
	// comes back if the option is actually offerable then.
	return s.bookingRepo.GetAvailableSlots(ctx, businessID, serviceOptionID, date, staffID, unassignedOnly)
}

func (s *bookingService) GetAvailableDates(ctx context.Context, businessID int64, serviceID *int64, serviceOptionID *int64, staffID *int64, unassignedOnly bool, from string, until string) ([]string, error) {
	if err := s.bookingRepo.SweepPastBookings(ctx); err != nil {
		log.Errorf("failed to sweep past bookings: %v", err)
	}
	return s.bookingRepo.GetAvailableDates(ctx, businessID, serviceID, serviceOptionID, staffID, unassignedOnly, from, until)
}

func (s *bookingService) GetRecentlyBookedBusinesses(ctx context.Context, userID int64) ([]param.BusinessProfileParam, error) {
	if err := s.bookingRepo.SweepPastBookings(ctx); err != nil {
		log.Errorf("failed to sweep past bookings: %v", err)
	}
	return s.bookingRepo.GetRecentlyBookedBusinesses(ctx, userID)
}

func (s *bookingService) CreateBooking(ctx context.Context, userID int64, slotOptionID int64, description *string) (*param.BookingParam, error) {
	if slotOptionID <= 0 {
		return nil, errs.ValidationErrors{{Field: "slotOptionId", Message: "Please select a time slot"}}
	}
	if description != nil {
		trimmed := strings.TrimSpace(*description)
		if trimmed == "" {
			description = nil
		} else {
			description = &trimmed
		}
	}
	// Check whether this slot belongs to a business owned by this user.
	isOwner, err := s.bookingRepo.IsSlotOptionOwnedByUser(ctx, slotOptionID, userID)
	if err != nil {
		return nil, err
	}
	if isOwner {
		return nil, errs.ValidationErrors{{
			Field:   "slotOptionId",
			Message: "You cannot book an appointment with your own business.",
		}}
	}
	available, err := s.bookingRepo.SlotOptionIsAvailable(ctx, slotOptionID)
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, errs.ValidationErrors{{Field: "slotOptionId", Message: "This time slot is no longer available."}}
	}
	booking, err := s.bookingRepo.InsertBooking(ctx, userID, slotOptionID, description)
	if err != nil {
		if database.IsUniqueViolation(err, "uq_active_booking_per_slot") {
			return nil, errs.ValidationErrors{{Field: "slotOptionId", Message: "This time slot has already been booked"}}
		}
		return nil, err
	}
	s.notifyNewBooking(ctx, booking.BookingID)
	return booking, nil
}

// notifyNewBooking emails the business side (owner and, if assigned, the
// staff member) that a new booking request just came in — best-effort, same
// as notify(): a failed/slow email should never fail the booking itself.
func (s *bookingService) notifyNewBooking(ctx context.Context, bookingID int64) {
	c, err := s.bookingRepo.GetBookingContext(ctx, bookingID)
	if err != nil {
		log.Errorf("failed to load booking context for new-booking email (booking %d): %v", bookingID, err)
		return
	}
	send := func(email, name string) {
		if email == "" {
			return
		}
		if err := s.emailService.SendNewBookingRequestEmail(email, name, c.CustomerName, c.WhenText); err != nil {
			log.Errorf("failed to send new booking request email to %s: %v", email, err)
		}
	}
	send(c.OwnerEmail, c.OwnerName)
	if c.StaffEmail != nil {
		staffName := ""
		if c.StaffName != nil {
			staffName = *c.StaffName
		}
		send(*c.StaffEmail, staffName)
	}
}

// RecordWalkIn marks a just-created slot option as an already-accepted
// walk-in booking. The slot was created moments earlier for this same
// business by the same caller, so none of CreateBooking's
// availability/ownership checks apply here.
func (s *bookingService) RecordWalkIn(ctx context.Context, actorUserID int64, slotOptionID int64) (*param.BookingDetailParam, error) {
	booking, err := s.bookingRepo.InsertWalkInBooking(ctx, actorUserID, slotOptionID)
	if err != nil {
		return nil, err
	}
	return s.bookingRepo.GetBookingDetail(ctx, booking.BookingID)
}

func (s *bookingService) GetBusinessBookings(ctx context.Context, businessID int64, staffID *int64) ([]param.BookingDetailParam, error) {
	if err := s.bookingRepo.SweepPastBookings(ctx); err != nil {
		log.Errorf("failed to sweep past bookings: %v", err)
	}
	return s.bookingRepo.GetBusinessBookings(ctx, businessID, staffID)
}

func (s *bookingService) GetCustomerBookings(ctx context.Context, userID int64) ([]param.BookingDetailParam, error) {
	if err := s.bookingRepo.SweepPastBookings(ctx); err != nil {
		log.Errorf("failed to sweep past bookings: %v", err)
	}
	return s.bookingRepo.GetCustomerBookings(ctx, userID)
}

// ── authorization helpers ────────────────────────────────────────────────────

func isBusinessSide(userID int64, c *param.BookingContextParam) bool {
	if userID == c.BusinessOwnerID {
		return true
	}
	return c.SlotStaffUserID != nil && userID == *c.SlotStaffUserID
}

func statusIn(status string, allowed ...string) bool {
	for _, a := range allowed {
		if status == a {
			return true
		}
	}
	return false
}

// notify emails the party that did NOT perform the action.
func (s *bookingService) notify(c *param.BookingContextParam, actorIsBusiness bool, statusLabel string) {
	send := func(email, name string) {
		if email == "" {
			return
		}
		if err := s.emailService.SendBookingStatusEmail(email, name, c.BusinessName, statusLabel, c.WhenText); err != nil {
			log.Errorf("failed to send booking status email to %s: %v", email, err)
		}
	}
	if actorIsBusiness {
		send(c.CustomerEmail, c.CustomerName)
		return
	}
	// customer acted → notify the business (owner and, if assigned, the staff)
	send(c.OwnerEmail, c.OwnerName)
	if c.StaffEmail != nil {
		staffName := ""
		if c.StaffName != nil {
			staffName = *c.StaffName
		}
		send(*c.StaffEmail, staffName)
	}
}

// ── transitions ──────────────────────────────────────────────────────────────

func (s *bookingService) AcceptBooking(ctx context.Context, userID int64, bookingID int64) (*param.BookingDetailParam, error) {
	if err := s.bookingRepo.SweepPastBookings(ctx); err != nil {
		log.Errorf("failed to sweep past bookings: %v", err)
	}
	c, err := s.bookingRepo.GetBookingContext(ctx, bookingID)
	if err != nil {
		return nil, bookingNotFound(err)
	}
	if !isBusinessSide(userID, c) {
		return nil, fmt.Errorf("You are not allowed to manage this booking.")
	}
	if c.Status != "pending" {
		return nil, fmt.Errorf("Only pending bookings can be accepted.")
	}

	err = s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		if err := s.bookingRepo.UpdateBookingStatus(ctx, tx, bookingID, "accepted", userID); err != nil {
			return err
		}
		// Prevent other pending requests on the same slot from also being accepted.
		return s.bookingRepo.RejectOtherPendingBookingsForSlot(ctx, tx, c.ServiceSlotID, bookingID)
	})
	if err != nil {
		return nil, err
	}

	// TODO: schedule automated appointment reminder before the appointment time.
	s.notify(c, true, "accepted")
	return s.bookingRepo.GetBookingDetail(ctx, bookingID)
}

func (s *bookingService) RejectBooking(ctx context.Context, userID int64, bookingID int64) (*param.BookingDetailParam, error) {
	if err := s.bookingRepo.SweepPastBookings(ctx); err != nil {
		log.Errorf("failed to sweep past bookings: %v", err)
	}
	c, err := s.bookingRepo.GetBookingContext(ctx, bookingID)
	if err != nil {
		return nil, bookingNotFound(err)
	}
	if !isBusinessSide(userID, c) {
		return nil, fmt.Errorf("You are not allowed to manage this booking.")
	}
	// "rescheduled" is included so the business can reject its own reschedule
	// proposal — it just can't accept it, that's the customer's call alone.
	if !statusIn(c.Status, "pending", "rescheduled") {
		return nil, fmt.Errorf("Only pending or rescheduled bookings can be rejected.")
	}

	if err := s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		if err := s.bookingRepo.UpdateBookingStatus(ctx, tx, bookingID, "rejected", userID); err != nil {
			return err
		}
		// This slot_option is no longer blocked by any active booking — drop
		// it if the option's own window has since shrunk to no longer cover
		// the slot's date while this booking was pending.
		return s.serviceRepo.DropSlotOptionIfExpired(ctx, tx, c.SlotOptionID)
	}); err != nil {
		return nil, err
	}

	s.notify(c, true, "rejected")
	return s.bookingRepo.GetBookingDetail(ctx, bookingID)
}

func (s *bookingService) CancelBooking(ctx context.Context, userID int64, bookingID int64) (*param.BookingDetailParam, error) {
	if err := s.bookingRepo.SweepPastBookings(ctx); err != nil {
		log.Errorf("failed to sweep past bookings: %v", err)
	}
	c, err := s.bookingRepo.GetBookingContext(ctx, bookingID)
	if err != nil {
		return nil, bookingNotFound(err)
	}
	business := isBusinessSide(userID, c)
	if !business && userID != c.CustomerUserID {
		return nil, fmt.Errorf("You are not allowed to manage this booking.")
	}
	if !statusIn(c.Status, "pending", "accepted", "rescheduled") {
		return nil, fmt.Errorf("This booking can no longer be cancelled.")
	}

	if err := s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		if err := s.bookingRepo.UpdateBookingStatus(ctx, tx, bookingID, "cancelled", userID); err != nil {
			return err
		}
		// Same as reject: this slot_option is now unblocked — drop it if it's
		// no longer within the option's own validity window.
		return s.serviceRepo.DropSlotOptionIfExpired(ctx, tx, c.SlotOptionID)
	}); err != nil {
		return nil, err
	}

	s.notify(c, business, "cancelled")
	return s.bookingRepo.GetBookingDetail(ctx, bookingID)
}

func (s *bookingService) RescheduleBooking(ctx context.Context, userID int64, bookingID int64, newSlotOptionID int64) (*param.BookingDetailParam, error) {
	if err := s.bookingRepo.SweepPastBookings(ctx); err != nil {
		log.Errorf("failed to sweep past bookings: %v", err)
	}
	c, err := s.bookingRepo.GetBookingContext(ctx, bookingID)
	if err != nil {
		return nil, bookingNotFound(err)
	}

	business := isBusinessSide(userID, c)
	if !business && userID != c.CustomerUserID {
		return nil, fmt.Errorf("You are not allowed to manage this booking.")
	}
	if !statusIn(c.Status, "pending", "accepted", "rescheduled") {
		return nil, fmt.Errorf("This booking can no longer be rescheduled.")
	}

	available, err := s.bookingRepo.SlotOptionIsAvailable(ctx, newSlotOptionID)
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, errs.ValidationErrors{{Field: "newSlotOptionId", Message: "The selected time slot is no longer available."}}
	}

	// A staff member (as opposed to the owner) may only reschedule a booking
	// onto one of their own slots — the owner can move it to any staff's slot
	// (or leave it owner-managed).
	if business && userID != c.BusinessOwnerID {
		newStaffUserID, err := s.bookingRepo.GetSlotOptionStaffUserID(ctx, newSlotOptionID)
		if err != nil {
			return nil, err
		}
		if newStaffUserID == nil || *newStaffUserID != userID {
			return nil, errs.ValidationErrors{{Field: "newSlotOptionId", Message: "You can only reschedule to your own slots."}}
		}
	}

	// A business reschedule is always a proposal the customer must accept via
	// AcceptReschedule — the business can never accept its own reschedule
	// (AcceptBooking only fires on "pending", so this must never stay
	// "pending" regardless of what the prior status was). A customer
	// reschedule becomes a pending request for the business to accept.
	newStatus := "pending"
	label := "a reschedule request"
	if business {
		newStatus = "rescheduled"
		label = "rescheduled"
	}

	// The slot the booking is leaving (c.SlotOptionID) becomes free — drop it
	// if the option's own window no longer covers its date (it may have
	// shrunk while this booking held it).
	freedSlotOptionID := c.SlotOptionID

	if err := s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		if err := s.bookingRepo.UpdateBookingSlotOption(ctx, tx, bookingID, newSlotOptionID, newStatus); err != nil {
			return err
		}
		return s.serviceRepo.DropSlotOptionIfExpired(ctx, tx, freedSlotOptionID)
	}); err != nil {
		if database.IsUniqueViolation(err, "uq_active_booking_per_slot") {
			return nil, errs.ValidationErrors{{Field: "newSlotOptionId", Message: "The selected time slot is no longer available."}}
		}
		return nil, err
	}

	// Re-fetch context for the new slot's time in the notification.
	if updated, cerr := s.bookingRepo.GetBookingContext(ctx, bookingID); cerr == nil {
		c = updated
	}
	s.notify(c, business, label)
	return s.bookingRepo.GetBookingDetail(ctx, bookingID)
}

func (s *bookingService) AcceptReschedule(ctx context.Context, userID int64, bookingID int64) (*param.BookingDetailParam, error) {
	if err := s.bookingRepo.SweepPastBookings(ctx); err != nil {
		log.Errorf("failed to sweep past bookings: %v", err)
	}
	c, err := s.bookingRepo.GetBookingContext(ctx, bookingID)
	if err != nil {
		return nil, bookingNotFound(err)
	}
	if userID != c.CustomerUserID {
		return nil, fmt.Errorf("You are not allowed to manage this booking.")
	}
	if c.Status != "rescheduled" {
		return nil, fmt.Errorf("This appointment has no reschedule to accept.")
	}

	if err := s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		return s.bookingRepo.UpdateBookingStatus(ctx, tx, bookingID, "accepted", userID)
	}); err != nil {
		return nil, err
	}

	// TODO: schedule automated appointment reminder before the appointment time.
	s.notify(c, false, "accepted")
	return s.bookingRepo.GetBookingDetail(ctx, bookingID)
}

func (s *bookingService) UpdateBookingDescription(ctx context.Context, userID int64, bookingID int64, description *string) (*param.BookingDetailParam, error) {
	c, err := s.bookingRepo.GetBookingContext(ctx, bookingID)
	if err != nil {
		return nil, bookingNotFound(err)
	}
	// Only the customer edits their own note — the business side views it
	// (via GetBookingDetail) but doesn't get to rewrite what the customer said.
	if userID != c.CustomerUserID {
		return nil, fmt.Errorf("You are not allowed to manage this booking.")
	}
	if !statusIn(c.Status, "pending", "accepted", "rescheduled") {
		return nil, fmt.Errorf("This booking can no longer be edited.")
	}
	if description != nil {
		trimmed := strings.TrimSpace(*description)
		if trimmed == "" {
			description = nil
		} else {
			description = &trimmed
		}
	}

	if err := s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		return s.bookingRepo.UpdateBookingDescription(ctx, tx, bookingID, description)
	}); err != nil {
		return nil, err
	}
	return s.bookingRepo.GetBookingDetail(ctx, bookingID)
}

func bookingNotFound(err error) error {
	if err == sql.ErrNoRows {
		return fmt.Errorf("Booking not found.")
	}
	return err
}
