package service

import (
	"context"
	"fyp/database"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces"
)

type bookingService struct {
	bookingRepo interfaces.IBookingRepo
}

func NewBookingService(bookingRepo interfaces.IBookingRepo) interfaces.IBookingService {
	return &bookingService{bookingRepo: bookingRepo}
}

func (s *bookingService) GetAvailableSlots(ctx context.Context, businessID int64, serviceOptionID int64, date string, staffID *int64) ([]param.ServiceSlotParam, error) {
	return s.bookingRepo.GetAvailableSlots(ctx, businessID, serviceOptionID, date, staffID)
}

func (s *bookingService) GetRecentlyBookedBusinesses(ctx context.Context, userID int64) ([]param.BusinessProfileParam, error) {
	return s.bookingRepo.GetRecentlyBookedBusinesses(ctx, userID)
}

func (s *bookingService) CreateBooking(ctx context.Context, userID int64, slotOptionID int64) (*param.BookingParam, error) {
	if slotOptionID <= 0 {
		return nil, errs.ValidationErrors{{Field: "slotOptionId", Message: "Please select a time slot"}}
	}
	booking, err := s.bookingRepo.InsertBooking(ctx, userID, slotOptionID)
	if err != nil {
		if database.IsUniqueViolation(err, "uq_active_booking_per_slot") {
			return nil, errs.ValidationErrors{{Field: "slotOptionId", Message: "This time slot has already been booked"}}
		}
		return nil, err
	}
	return booking, nil
}
