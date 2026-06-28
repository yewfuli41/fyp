package interfaces

import (
	"context"
	"database/sql"
	"fyp/domain/param"
)

type IStaffRepo interface {
	InsertStaff(ctx context.Context, tx *sql.Tx, p param.StaffParam) (*int64, error)
	InsertStaffWorkingHours(ctx context.Context, tx *sql.Tx, param param.StaffParam) error
	GetStaffByUserID(ctx context.Context, userID int64) (*param.StaffParam, error)
	GetStaffByBusinessID(ctx context.Context, businessID int64) ([]param.StaffParam, error)
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
}
