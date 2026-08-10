package param

import "time"

type BookingParam struct {
	BookingID    int64
	UserID       int64
	SlotOptionID int64
	Status       string
	BookingType  string
	Description  *string
	CreatedAt    time.Time
}

// BookingDetailParam is a flattened booking row for listing/display.
type BookingDetailParam struct {
	BookingID       int64
	Status          string
	BookingType     string
	ServiceSlotID   int64
	SlotOptionID    int64
	ServiceOptionID int64
	Date            string
	StartTime       time.Time
	EndTime         time.Time
	ServiceName     string
	OptionName      string
	StaffName       *string
	CustomerName    string
	CustomerEmail   string
	BusinessID      int64
	BusinessName    string
	Description     *string
	CreatedAt       time.Time
}

// BookingContextParam carries the identities needed to authorize a booking action
// and to target the status-change email at the correct party.
type BookingContextParam struct {
	BookingID       int64
	Status          string
	ServiceSlotID   int64
	SlotOptionID    int64
	CustomerUserID  int64
	CustomerName    string
	CustomerEmail   string
	BusinessOwnerID int64
	OwnerName       string
	OwnerEmail      string
	SlotStaffUserID *int64
	StaffName       *string
	StaffEmail      *string
	BusinessName    string
	WhenText        string
}
