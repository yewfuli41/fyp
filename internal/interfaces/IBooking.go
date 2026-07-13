package interfaces

import (
	"context"
	"fyp/domain/param"
)

type IBookingRepo interface {
	GetAvailableSlots(ctx context.Context, businessID int64, serviceOptionID int64, date string, staffID *int64) ([]param.ServiceSlotParam, error)
	InsertBooking(ctx context.Context, userID int64, slotOptionID int64) (*param.BookingParam, error)
	GetRecentlyBookedBusinesses(ctx context.Context, userID int64) ([]param.BusinessProfileParam, error)
}

type IBookingService interface {
	GetAvailableSlots(ctx context.Context, businessID int64, serviceOptionID int64, date string, staffID *int64) ([]param.ServiceSlotParam, error)
	CreateBooking(ctx context.Context, userID int64, slotOptionID int64) (*param.BookingParam, error)
	GetRecentlyBookedBusinesses(ctx context.Context, userID int64) ([]param.BusinessProfileParam, error)
}
