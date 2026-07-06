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

func (r *serviceRepo) InsertServiceOption(ctx context.Context, tx *sql.Tx, p param.ServiceOptionParam) (*param.ServiceOptionParam, error) {
	row := tx.QueryRowContext(ctx, `
		INSERT INTO service_options (service_id, service_option_name, description)
		VALUES ($1, $2, $3)
		RETURNING service_option_id, service_id, service_option_name, description
	`, p.ServiceID, p.ServiceOptionName, p.Description)

	return scanServiceOption(row)
}

func (r *serviceRepo) InsertServiceOptionItem(ctx context.Context, tx *sql.Tx, p param.ServiceOptionItemParam) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO service_option_items (service_option_id, service_option_item_name, created_at)
		VALUES ($1, $2, NOW())
	`, p.ServiceOptionID, p.ServiceOptionItemName)
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

func (r *serviceRepo) DeleteServiceOptionsByServiceID(ctx context.Context, tx *sql.Tx, serviceID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE service_option_items SET deleted_at = NOW()
		WHERE service_option_id IN (
			SELECT service_option_id FROM service_options WHERE service_id = $1
		) AND deleted_at IS NULL
	`, serviceID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE service_options SET deleted_at = NOW()
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

func (r *serviceRepo) GetServiceOptionsByServiceID(ctx context.Context, serviceID int64) ([]param.ServiceOptionParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT service_option_id, service_id, service_option_name, description
		FROM service_options
		WHERE service_id = $1 AND deleted_at IS NULL
		ORDER BY service_option_id
	`, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []param.ServiceOptionParam
	for rows.Next() {
		p, err := scanServiceOption(rows)
		if err != nil {
			return nil, err
		}
		packages = append(packages, *p)
	}
	return packages, nil
}

func (r *serviceRepo) GetServiceOptionItemsByOptionID(ctx context.Context, serviceOptionID int64) ([]param.ServiceOptionItemParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT service_option_item_id, service_option_id, service_option_item_name
		FROM service_option_items
		WHERE service_option_id = $1 AND deleted_at IS NULL
		ORDER BY service_option_item_id
	`, serviceOptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []param.ServiceOptionItemParam
	for rows.Next() {
		var item param.ServiceOptionItemParam
		if err := rows.Scan(&item.ServiceOptionItemID, &item.ServiceOptionID, &item.ServiceOptionItemName); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *serviceRepo) GetServiceOptionItemsByOptionIDIncludeDeleted(ctx context.Context, serviceOptionID int64) ([]param.ServiceOptionItemParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT service_option_item_id, service_option_id, service_option_item_name
		FROM service_option_items
		WHERE service_option_id = $1
		ORDER BY service_option_item_id
	`, serviceOptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []param.ServiceOptionItemParam
	for rows.Next() {
		var item param.ServiceOptionItemParam
		if err := rows.Scan(&item.ServiceOptionItemID, &item.ServiceOptionID, &item.ServiceOptionItemName); err != nil {
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
		JOIN service_slot_options ssp ON b.slot_option_id = ssp.slot_option_id
		JOIN service_options sp ON ssp.service_option_id = sp.service_option_id
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

func scanServiceOption(row rowScannerService) (*param.ServiceOptionParam, error) {
	var p param.ServiceOptionParam
	var description sql.NullString
	if err := row.Scan(&p.ServiceOptionID, &p.ServiceID, &p.ServiceOptionName, &description); err != nil {
		return nil, err
	}
	p.Description = utils.NullStringPtr(description)
	return &p, nil
}
