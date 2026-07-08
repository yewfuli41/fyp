package interfaces

import (
	"context"
	"database/sql"
	"fyp/domain/param"
)

type IBusinessRepo interface {
	InsertBusinessProfile(ctx context.Context, tx *sql.Tx, param param.BusinessProfileParam) (*param.BusinessProfileParam, error)
	InsertBusinessWorkingHours(ctx context.Context, tx *sql.Tx, param param.BusinessProfileParam) error
	UpdateBusinessProfile(ctx context.Context, tx *sql.Tx, param param.BusinessProfileParam) (*param.BusinessProfileParam, error)
	DeleteBusinessWorkingHours(ctx context.Context, tx *sql.Tx, businessID int64) error
	GetBusinessProfileByOwnerID(ctx context.Context, ownerID int64) (*param.BusinessProfileParam, error)
	GetBusinessWorkingHours(ctx context.Context, businessID int64) ([]param.WorkingHourParam, error)
	BusinessEmailExists(ctx context.Context, businessEmail string) (bool, error)
	GetBusinesses(ctx context.Context, search string) ([]param.BusinessProfileParam, error)
	GetBusinessByID(ctx context.Context, businessID int64) (*param.BusinessProfileParam, error)
}

type IBusinessService interface {
	RegisterBusinessProfile(ctx context.Context, param param.BusinessProfileParam) (*param.BusinessProfileParam, error)
	UpdateBusinessProfile(ctx context.Context, param param.BusinessProfileParam) (*param.BusinessProfileParam, error)
	GetBusinessProfileByOwnerID(ctx context.Context, ownerID int64) (*param.BusinessProfileParam, error)
	GetBusinesses(ctx context.Context, search string) ([]param.BusinessProfileParam, error)
}
