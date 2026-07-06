package interfaces

import (
	"context"
	"database/sql"
	"fyp/domain/param"
	"time"
)

type IServiceSlotRepo interface {
	InsertServiceSlot(ctx context.Context, tx *sql.Tx, p param.ServiceSlotParam) (int64, error)
	InsertServiceSlotOption(ctx context.Context, tx *sql.Tx, serviceSlotID int64, serviceOptionID int64) error
	// InsertRecurringSchedule creates one row per (staff, day) — not per
	// package — and returns its ID so callers can attach it to every
	// service_slots row generated for that weekday.
	InsertRecurringSchedule(ctx context.Context, tx *sql.Tx, p param.ServiceSlotParam, day string) (int64, error)

	GetServiceSlotsByBusinessAndDate(ctx context.Context, businessID int64, date string, staffID *int64, serviceID *int64, unassignedOnly bool) ([]param.ServiceSlotParam, error)
	GetServiceSlotByID(ctx context.Context, serviceSlotID int64, businessID int64) (*param.ServiceSlotParam, error)

	ReassignStaff(ctx context.Context, tx *sql.Tx, serviceSlotID int64, businessID int64, staffID *int64) error
	SoftDeleteServiceSlot(ctx context.Context, tx *sql.Tx, serviceSlotID int64, businessID int64) error
	// GetFutureRecurringSlotIDs returns every non-deleted slot (from fromDate
	// onward) belonging to the exact recurring series, by its recurring_schedule_id.
	GetFutureRecurringSlotIDs(ctx context.Context, businessID int64, recurringScheduleID int64, fromDate string) ([]int64, error)
	SoftDeleteServiceSlots(ctx context.Context, tx *sql.Tx, slotIDs []int64, businessID int64) error
	SoftDeleteRecurringSchedule(ctx context.Context, tx *sql.Tx, businessID int64, recurringScheduleID int64) error

	StaffBelongsToBusiness(ctx context.Context, staffID int64, businessID int64) (bool, error)
	OptionsBelongToBusiness(ctx context.Context, businessID int64, packageIDs []int64) (bool, error)
	StaffCoversTime(ctx context.Context, staffID int64, weekday string, startTime, endTime time.Time) (bool, error)
	GetAvailableStaff(ctx context.Context, businessID int64, date string, weekday string, startTime, endTime time.Time, excludeSlotID int64) ([]param.StaffParam, error)
	HasBookingForServiceSlot(ctx context.Context, serviceSlotID int64) (bool, error)
}

type IServiceSlotService interface {
	CreateServiceSlot(ctx context.Context, p param.ServiceSlotParam) (*param.ServiceSlotParam, error)
	// staffScope, when non-nil, restricts the operation to a slot currently
	// assigned to that staff ID (used when a staff member, not the owner, is
	// acting) — anything else is treated as not found.
	UpdateServiceSlot(ctx context.Context, p param.ServiceSlotParam, applyToFutureRecurring bool, staffScope *int64) (*param.ServiceSlotParam, error)
	ReassignServiceSlotStaff(ctx context.Context, serviceSlotID int64, businessID int64, staffID *int64) (*param.ServiceSlotParam, error)
	DeleteServiceSlot(ctx context.Context, serviceSlotID int64, businessID int64, deleteFutureRecurring bool, staffScope *int64) error
	GetServiceSlots(ctx context.Context, businessID int64, date string, staffID *int64, serviceID *int64, unassignedOnly bool) ([]param.ServiceSlotParam, error)
	GetAvailableStaffForSlot(ctx context.Context, serviceSlotID int64, businessID int64) ([]param.StaffParam, error)
}
