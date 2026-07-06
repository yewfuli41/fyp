package interfaces

import (
	"context"
	"database/sql"
	"fyp/domain/param"
)

type IServiceRepo interface {
	InsertService(ctx context.Context, tx *sql.Tx, p param.ServiceParam) (*param.ServiceParam, error)
	InsertServiceOption(ctx context.Context, tx *sql.Tx, p param.ServiceOptionParam) (*param.ServiceOptionParam, error)
	InsertServiceOptionItem(ctx context.Context, tx *sql.Tx, p param.ServiceOptionItemParam) error
	UpdateService(ctx context.Context, tx *sql.Tx, p param.ServiceParam) (*param.ServiceParam, error)
	DeleteServiceOptionsByServiceID(ctx context.Context, tx *sql.Tx, serviceID int64) error
	SoftDeleteService(ctx context.Context, tx *sql.Tx, serviceID int64, businessID int64) error
	GetServicesByBusinessID(ctx context.Context, businessID int64) ([]param.ServiceParam, error)
	GetServiceOptionsByServiceID(ctx context.Context, serviceID int64) ([]param.ServiceOptionParam, error)
	GetServiceOptionItemsByOptionID(ctx context.Context, serviceOptionID int64) ([]param.ServiceOptionItemParam, error)
	GetServiceOptionItemsByOptionIDIncludeDeleted(ctx context.Context, serviceOptionID int64) ([]param.ServiceOptionItemParam, error)
	HasBookingForService(ctx context.Context, serviceID int64) (bool, error)
}

type IServiceService interface {
	CreateService(ctx context.Context, p param.ServiceParam) (*param.ServiceParam, error)
	UpdateService(ctx context.Context, p param.ServiceParam) (*param.ServiceParam, error)
	DeleteService(ctx context.Context, serviceID int64, businessID int64) error
	GetServicesByBusinessID(ctx context.Context, businessID int64) ([]param.ServiceParam, error)
}
