package interfaces

import (
	"context"
	"database/sql"
	"fyp/domain/param"
	"time"
)

type IServiceSlotRepo interface {
	InsertServiceSlot(ctx context.Context, tx *sql.Tx, p param.ServiceSlotParam) (int64, error)
	InsertServiceSlotPackage(ctx context.Context, tx *sql.Tx, serviceSlotID int64, servicePackageID int64) error
	InsertRecurringSchedule(ctx context.Context, tx *sql.Tx, p param.ServiceSlotParam, servicePackageID int64, day string) error

	GetServiceSlotsByBusinessAndDate(ctx context.Context, businessID int64, date string, staffID *int64, serviceID *int64, unassignedOnly bool) ([]param.ServiceSlotParam, error)
	GetServiceSlotByID(ctx context.Context, serviceSlotID int64, businessID int64) (*param.ServiceSlotParam, error)

	ReassignStaff(ctx context.Context, tx *sql.Tx, serviceSlotID int64, businessID int64, staffID *int64) error
	SoftDeleteServiceSlot(ctx context.Context, tx *sql.Tx, serviceSlotID int64, businessID int64) error
	GetFutureRecurringSlotIDs(ctx context.Context, businessID int64, staffID *int64, startTime, endTime time.Time, fromDate string) ([]int64, error)
	SoftDeleteServiceSlots(ctx context.Context, tx *sql.Tx, slotIDs []int64, businessID int64) error
	SoftDeleteRecurringSchedules(ctx context.Context, tx *sql.Tx, businessID int64, staffID int64, weekday string, startTime, endTime time.Time) error

	StaffBelongsToBusiness(ctx context.Context, staffID int64, businessID int64) (bool, error)
	PackagesBelongToBusiness(ctx context.Context, businessID int64, packageIDs []int64) (bool, error)
	StaffCoversTime(ctx context.Context, staffID int64, weekday string, startTime, endTime time.Time) (bool, error)
	StaffHasOverlappingSlot(ctx context.Context, staffID int64, date string, startTime, endTime time.Time, excludeSlotID int64) (bool, error)
	GetAvailableStaff(ctx context.Context, businessID int64, date string, weekday string, startTime, endTime time.Time, excludeSlotID int64) ([]param.StaffParam, error)
}

type IServiceSlotService interface {
	CreateServiceSlot(ctx context.Context, p param.ServiceSlotParam) (*param.ServiceSlotParam, error)
	ReassignServiceSlotStaff(ctx context.Context, serviceSlotID int64, businessID int64, staffID *int64) (*param.ServiceSlotParam, error)
	DeleteServiceSlot(ctx context.Context, serviceSlotID int64, businessID int64, deleteFutureRecurring bool) error
	GetServiceSlots(ctx context.Context, businessID int64, date string, staffID *int64, serviceID *int64, unassignedOnly bool) ([]param.ServiceSlotParam, error)
	GetAvailableStaffForSlot(ctx context.Context, serviceSlotID int64, businessID int64) ([]param.StaffParam, error)
}
