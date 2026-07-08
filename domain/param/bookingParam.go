package param

import "time"

type BookingParam struct {
	BookingID    int64
	UserID       int64
	SlotOptionID int64
	Status       string
	BookingType  string
	CreatedAt    time.Time
}
