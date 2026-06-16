package interfaces

import (
	"context"
	"database/sql"
	"fyp/domain/param"
)

type IStaffRepo interface {
	InsertStaff(ctx context.Context, tx *sql.Tx, p param.StaffParam) error
}

type IStaffService interface {
	RegisterStaff(ctx context.Context, param param.StaffParam, businessWorkingHours param.WorkingHourParam)
}
