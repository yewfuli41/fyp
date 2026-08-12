package interfaces

import (
	"context"
	"database/sql"
	"fyp/domain/param"
)

type IStaffRepo interface {
	InsertStaff(ctx context.Context, tx *sql.Tx, p param.StaffParam) (*int64, error)
	InsertStaffWorkingHours(ctx context.Context, tx *sql.Tx, param param.StaffParam) error
	DeleteStaffWorkingHours(ctx context.Context, tx *sql.Tx, staffID int64) error
	GetStaffByUserID(ctx context.Context, userID int64) (*param.StaffParam, error)
	GetStaffByBusinessID(ctx context.Context, businessID int64) ([]param.StaffParam, error)
	GetStaffWorkingHours(ctx context.Context, staffID int64) ([]param.WorkingHourParam, error)
	GetStaffByIDTx(ctx context.Context, tx *sql.Tx, staffID int64, businessID int64) (*param.StaffParam, error)
	UpdateStaff(ctx context.Context, tx *sql.Tx, p param.StaffParam) (*param.StaffParam, error)
	SoftDeleteStaff(ctx context.Context, tx *sql.Tx, staffID int64, businessID int64) error
	HasBookingForStaff(ctx context.Context, staffID int64) (bool, error)
}

type IStaffService interface {
	RegisterStaff(ctx context.Context, ownerParam *param.BusinessProfileParam, staffParam param.StaffParam, userParam param.AuthUserParam) error
	GetStaffProfileByUserID(ctx context.Context, userID int64) (*param.StaffParam, error)
	GetStaffByBusinessID(ctx context.Context, businessID int64) ([]param.StaffParam, error)
	UpdateStaff(ctx context.Context, p param.StaffParam) (*param.StaffParam, error)
	DeleteStaff(ctx context.Context, staffID int64, businessID int64) error
	// GetStaffHoursConflicts is a precheck for UpdateStaffWorkingHours — every
	// one of the staff's future slots that would fall outside the proposed
	// hours, booked and unbooked alike (see each slot's HasBooking), so the
	// caller can show a replacement-staff picker for the booked ones and warn
	// about the unbooked ones being unassigned before committing the change.
	GetStaffHoursConflicts(ctx context.Context, businessID int64, staffID int64, workingHours []param.WorkingHourParam) ([]param.ServiceSlotParam, error)
	// UpdateStaffWorkingHours requires a reassignment for every one of the
	// staff's future BOOKED slots that would fall outside the new hours
	// (see GetStaffHoursConflicts) and silently unassigns any non-booked one.
	UpdateStaffWorkingHours(ctx context.Context, businessID int64, staffID int64, workingHours []param.WorkingHourParam, reassignments []param.SlotReassignmentParam) (*param.StaffParam, error)
}
