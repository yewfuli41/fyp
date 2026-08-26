package interfaces

import (
	"context"
	"database/sql"
	"fyp/domain/param"
)

type IBookingRepo interface {
	// unassignedOnly, when true, narrows to owner-managed (no staff assigned)
	// slots and takes precedence over staffID.
	GetAvailableSlots(ctx context.Context, businessID int64, serviceOptionID int64, date string, staffID *int64, unassignedOnly bool, unavailable *param.StaffUnavailability) ([]param.ServiceSlotParam, error)
	// GetAvailableDates returns, as "YYYY-MM-DD" strings, every date in
	// [from, until] that has at least one bookable slot, optionally narrowed
	// by service, service option, and/or staff (any may be nil for "any").
	// unassignedOnly, when true, narrows instead to owner-managed slots and
	// takes precedence over staffID.
	GetAvailableDates(ctx context.Context, businessID int64, serviceID *int64, serviceOptionID *int64, staffID *int64, unassignedOnly bool, from string, until string, unavailable *param.StaffUnavailability) ([]string, error)
	InsertBooking(ctx context.Context, userID int64, slotOptionID int64, description *string) (*param.BookingParam, error)
	// InsertWalkInBooking records an already-accepted walk-in booking (no
	// pending approval step) against userID — the business owner or staff
	// member who recorded it, since there's no separate walk-in customer
	// account.
	InsertWalkInBooking(ctx context.Context, userID int64, slotOptionID int64) (*param.BookingParam, error)
	GetRecentlyBookedBusinesses(ctx context.Context, userID int64) ([]param.BusinessProfileParam, error)
	GetBusinessBookings(ctx context.Context, businessID int64, staffID *int64) ([]param.BookingDetailParam, error)
	GetCustomerBookings(ctx context.Context, userID int64) ([]param.BookingDetailParam, error)
	// IsSlotOptionOwnedByUser reports whether the business that owns this slot
	// option is itself owned by userID — used to block an owner from booking
	// their own business as a customer.
	IsSlotOptionOwnedByUser(ctx context.Context, slotOptionID int64, userID int64) (bool, error)
	GetBookingDetail(ctx context.Context, bookingID int64) (*param.BookingDetailParam, error)
	GetBookingContext(ctx context.Context, bookingID int64) (*param.BookingContextParam, error)
	// GetBookingContextForSlot returns the active (pending/accepted/rescheduled)
	// booking's context for a service slot, or nil if it has none — used to
	// notify a customer when their slot's staff changes (leave approval,
	// working-hours edit).
	GetBookingContextForSlot(ctx context.Context, serviceSlotID int64) (*param.BookingContextParam, error)
	UpdateBookingStatus(ctx context.Context, tx *sql.Tx, bookingID int64, status string, decidedBy int64) error
	UpdateBookingDescription(ctx context.Context, tx *sql.Tx, bookingID int64, description *string) error
	UpdateBookingSlotOption(ctx context.Context, tx *sql.Tx, bookingID int64, newSlotOptionID int64, status string) error
	RejectOtherPendingBookingsForSlot(ctx context.Context, tx *sql.Tx, serviceSlotID int64, exceptBookingID int64) error
	SlotOptionIsAvailable(ctx context.Context, slotOptionID int64) (bool, error)
	// GetSlotOptionStaffUserID returns the login user id of the staff member a
	// slot option is assigned to, or nil if it's owner-managed (unassigned).
	GetSlotOptionStaffUserID(ctx context.Context, slotOptionID int64) (*int64, error)
	// GetSlotOptionAssignment returns the staff a slot option's slot is assigned
	// to (nil when owner-managed) and that slot's date, so a caller can tell
	// whether moving a booking onto it would land on a staff member's leave.
	GetSlotOptionAssignment(ctx context.Context, slotOptionID int64) (staffID *int64, date string, err error)
	// SweepPastBookings flips any pending/accepted/rescheduled booking whose
	// slot has already ended to "past" — called opportunistically before any
	// read or action so status is never stale.
	SweepPastBookings(ctx context.Context) error
}

type IBookingService interface {
	GetAvailableSlots(ctx context.Context, businessID int64, serviceOptionID int64, date string, staffID *int64, unassignedOnly bool, unavailable *param.StaffUnavailability) ([]param.ServiceSlotParam, error)
	GetAvailableDates(ctx context.Context, businessID int64, serviceID *int64, serviceOptionID *int64, staffID *int64, unassignedOnly bool, from string, until string, unavailable *param.StaffUnavailability) ([]string, error)
	CreateBooking(ctx context.Context, userID int64, slotOptionID int64, description *string) (*param.BookingParam, error)
	// RecordWalkIn marks the given (freshly created) slot option as an
	// already-accepted walk-in booking recorded by actorUserID.
	RecordWalkIn(ctx context.Context, actorUserID int64, slotOptionID int64) (*param.BookingDetailParam, error)
	GetRecentlyBookedBusinesses(ctx context.Context, userID int64) ([]param.BusinessProfileParam, error)
	GetBusinessBookings(ctx context.Context, businessID int64, staffID *int64) ([]param.BookingDetailParam, error)
	GetCustomerBookings(ctx context.Context, userID int64) ([]param.BookingDetailParam, error)
	AcceptBooking(ctx context.Context, userID int64, bookingID int64) (*param.BookingDetailParam, error)
	RejectBooking(ctx context.Context, userID int64, bookingID int64) (*param.BookingDetailParam, error)
	CancelBooking(ctx context.Context, userID int64, bookingID int64) (*param.BookingDetailParam, error)
	RescheduleBooking(ctx context.Context, userID int64, bookingID int64, newSlotOptionID int64) (*param.BookingDetailParam, error)
	AcceptReschedule(ctx context.Context, userID int64, bookingID int64) (*param.BookingDetailParam, error)
	// UpdateBookingDescription lets the customer edit just their note on an
	// otherwise-immutable booking (date/time/option can't be changed here).
	UpdateBookingDescription(ctx context.Context, userID int64, bookingID int64, description *string) (*param.BookingDetailParam, error)
}
