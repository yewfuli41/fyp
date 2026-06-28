package interfaces

import (
	"context"
	"database/sql"
	"fyp/domain/param"
)

type IServiceRepo interface {
	InsertService(ctx context.Context, tx *sql.Tx, p param.ServiceParam) (*param.ServiceParam, error)
	InsertServicePackage(ctx context.Context, tx *sql.Tx, p param.ServicePackageParam) (*param.ServicePackageParam, error)
	InsertPackageItem(ctx context.Context, tx *sql.Tx, p param.PackageItemParam) error
	UpdateService(ctx context.Context, tx *sql.Tx, p param.ServiceParam) (*param.ServiceParam, error)
	DeleteServicePackagesByServiceID(ctx context.Context, tx *sql.Tx, serviceID int64) error
	SoftDeleteService(ctx context.Context, tx *sql.Tx, serviceID int64, businessID int64) error
	GetServicesByBusinessID(ctx context.Context, businessID int64) ([]param.ServiceParam, error)
	GetServicePackagesByServiceID(ctx context.Context, serviceID int64) ([]param.ServicePackageParam, error)
	GetPackageItemsByPackageID(ctx context.Context, servicePackageID int64) ([]param.PackageItemParam, error)
	GetPackageItemsByPackageIDIncludeDeleted(ctx context.Context, servicePackageID int64) ([]param.PackageItemParam, error)
	HasBookingForService(ctx context.Context, serviceID int64) (bool, error)
}

type IServiceService interface {
	CreateService(ctx context.Context, p param.ServiceParam) (*param.ServiceParam, error)
	UpdateService(ctx context.Context, p param.ServiceParam) (*param.ServiceParam, error)
	DeleteService(ctx context.Context, serviceID int64, businessID int64) error
	GetServicesByBusinessID(ctx context.Context, businessID int64) ([]param.ServiceParam, error)
}
