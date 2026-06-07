package interfaces

import (
	"context"
	"database/sql"
	"fyp/domain/param"
)

type IBusinessRepo interface {
	InsertBusinessProfile(ctx context.Context, tx *sql.Tx, param param.BusinessProfileParam) (*param.BusinessProfileParam, error)
	InsertBusinessWorkingHours(ctx context.Context, tx *sql.Tx, param param.BusinessProfileParam) error
}

type IBusinessService interface {
	RegisterBusinessProfile(ctx context.Context, param param.BusinessProfileParam) (*param.BusinessProfileParam, error)
}
