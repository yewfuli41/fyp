package repository

import (
	"context"
	"database/sql"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/utils"
)

type serviceRepo struct {
	DB *sql.DB
}

func NewServiceRepo(db *sql.DB) interfaces.IServiceRepo {
	return &serviceRepo{DB: db}
}

func (r *serviceRepo) InsertService(ctx context.Context, tx *sql.Tx, p param.ServiceParam) (*param.ServiceParam, error) {
	row := tx.QueryRowContext(ctx, `
		INSERT INTO services (business_id, service_name, description)
		VALUES ($1, $2, $3)
		RETURNING service_id, business_id, service_name, description
	`, p.BusinessID, p.ServiceName, p.Description)

	return scanService(row)
}

func (r *serviceRepo) InsertServicePackage(ctx context.Context, tx *sql.Tx, p param.ServicePackageParam) (*param.ServicePackageParam, error) {
	row := tx.QueryRowContext(ctx, `
		INSERT INTO service_packages (service_id, service_package_name, description)
		VALUES ($1, $2, $3)
		RETURNING service_package_id, service_id, service_package_name, description
	`, p.ServiceID, p.ServicePackageName, p.Description)

	return scanServicePackage(row)
}

func (r *serviceRepo) InsertPackageItem(ctx context.Context, tx *sql.Tx, p param.PackageItemParam) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO package_items (service_package_id, package_item_name)
		VALUES ($1, $2)
	`, p.ServicePackageID, p.PackageItemName)
	return err
}

func (r *serviceRepo) UpdateService(ctx context.Context, tx *sql.Tx, p param.ServiceParam) (*param.ServiceParam, error) {
	row := tx.QueryRowContext(ctx, `
		UPDATE services
		SET service_name = $1, description = $2
		WHERE service_id = $3 AND business_id = $4 AND deleted_at IS NULL
		RETURNING service_id, business_id, service_name, description
	`, p.ServiceName, p.Description, p.ServiceID, p.BusinessID)

	return scanService(row)
}

func (r *serviceRepo) DeleteServicePackagesByServiceID(ctx context.Context, tx *sql.Tx, serviceID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE package_items SET deleted_at = NOW()
		WHERE service_package_id IN (
			SELECT service_package_id FROM service_packages WHERE service_id = $1
		) AND deleted_at IS NULL
	`, serviceID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE service_packages SET deleted_at = NOW()
		WHERE service_id = $1 AND deleted_at IS NULL
	`, serviceID)
	return err
}

func (r *serviceRepo) SoftDeleteService(ctx context.Context, tx *sql.Tx, serviceID int64, businessID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE services SET deleted_at = NOW()
		WHERE service_id = $1 AND business_id = $2 AND deleted_at IS NULL
	`, serviceID, businessID)
	return err
}

func (r *serviceRepo) GetServicesByBusinessID(ctx context.Context, businessID int64) ([]param.ServiceParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT service_id, business_id, service_name, description
		FROM services
		WHERE business_id = $1 AND deleted_at IS NULL
		ORDER BY service_id
	`, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []param.ServiceParam
	for rows.Next() {
		s, err := scanService(rows)
		if err != nil {
			return nil, err
		}
		services = append(services, *s)
	}
	return services, nil
}

func (r *serviceRepo) GetServicePackagesByServiceID(ctx context.Context, serviceID int64) ([]param.ServicePackageParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT service_package_id, service_id, service_package_name, description
		FROM service_packages
		WHERE service_id = $1 AND deleted_at IS NULL
		ORDER BY service_package_id
	`, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []param.ServicePackageParam
	for rows.Next() {
		p, err := scanServicePackage(rows)
		if err != nil {
			return nil, err
		}
		packages = append(packages, *p)
	}
	return packages, nil
}

func (r *serviceRepo) GetPackageItemsByPackageID(ctx context.Context, servicePackageID int64) ([]param.PackageItemParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT package_item_id, service_package_id, package_item_name
		FROM package_items
		WHERE service_package_id = $1 AND deleted_at IS NULL
		ORDER BY package_item_id
	`, servicePackageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []param.PackageItemParam
	for rows.Next() {
		var item param.PackageItemParam
		if err := rows.Scan(&item.PackageItemID, &item.ServicePackageID, &item.PackageItemName); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *serviceRepo) HasBookingForService(ctx context.Context, serviceID int64) (bool, error) {
	var count int
	err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM bookings b
		JOIN service_slot_packages ssp ON b.slot_package_id = ssp.slot_package_id
		JOIN service_packages sp ON ssp.service_package_id = sp.service_package_id
		WHERE sp.service_id = $1 AND b.deleted_at IS NULL
	`, serviceID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

type rowScannerService interface {
	Scan(dest ...any) error
}

func scanService(row rowScannerService) (*param.ServiceParam, error) {
	var s param.ServiceParam
	var description sql.NullString
	if err := row.Scan(&s.ServiceID, &s.BusinessID, &s.ServiceName, &description); err != nil {
		return nil, err
	}
	s.Description = utils.NullStringPtr(description)
	return &s, nil
}

func scanServicePackage(row rowScannerService) (*param.ServicePackageParam, error) {
	var p param.ServicePackageParam
	var description sql.NullString
	if err := row.Scan(&p.ServicePackageID, &p.ServiceID, &p.ServicePackageName, &description); err != nil {
		return nil, err
	}
	p.Description = utils.NullStringPtr(description)
	return &p, nil
}
