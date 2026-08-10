package repository

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/utils"
	"time"
)

type serviceRepo struct {
	DB *sql.DB
}

func NewServiceRepo(db *sql.DB) interfaces.IServiceRepo {
	return &serviceRepo{DB: db}
}

func (r *serviceRepo) InsertService(ctx context.Context, tx *sql.Tx, p param.ServiceParam) (*param.ServiceParam, error) {
	row := tx.QueryRowContext(ctx, `
		INSERT INTO fyp_fuli_services (business_id, service_name, description)
		VALUES ($1, $2, $3)
		RETURNING service_id, business_id, service_name, description
	`, p.BusinessID, p.ServiceName, p.Description)

	return scanService(row)
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// InsertServiceOption creates a new, standalone option. EffectiveFrom
// defaults to today when empty.
func (r *serviceRepo) InsertServiceOption(ctx context.Context, tx *sql.Tx, p param.ServiceOptionParam) (*param.ServiceOptionParam, error) {
	row := tx.QueryRowContext(ctx, `
		INSERT INTO fyp_fuli_service_options (service_id, service_option_name, description, effective_from, effective_until, is_default)
		VALUES ($1, $2, $3, COALESCE($4::date, CURRENT_DATE), $5::date, $6)
		RETURNING service_option_id, service_id, service_option_name, description,
			to_char(effective_from, 'YYYY-MM-DD'), to_char(effective_until, 'YYYY-MM-DD'), is_default
	`, p.ServiceID, p.ServiceOptionName, p.Description, nilIfEmpty(p.EffectiveFrom), p.EffectiveUntil, p.IsDefault)

	return scanServiceOption(row)
}

func (r *serviceRepo) InsertServiceOptionItem(ctx context.Context, tx *sql.Tx, p param.ServiceOptionItemParam) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO fyp_fuli_service_option_items (service_option_id, service_option_item_name, created_at)
		VALUES ($1, $2, NOW())
	`, p.ServiceOptionID, p.ServiceOptionItemName)
	return err
}

func (r *serviceRepo) UpdateService(ctx context.Context, tx *sql.Tx, p param.ServiceParam) (*param.ServiceParam, error) {
	row := tx.QueryRowContext(ctx, `
		UPDATE fyp_fuli_services
		SET service_name = $1, description = $2
		WHERE service_id = $3 AND business_id = $4 AND deleted_at IS NULL
		RETURNING service_id, business_id, service_name, description
	`, p.ServiceName, p.Description, p.ServiceID, p.BusinessID)

	return scanService(row)
}

func (r *serviceRepo) SoftDeleteServiceOption(ctx context.Context, tx *sql.Tx, serviceOptionID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_service_option_items SET deleted_at = NOW()
		WHERE service_option_id = $1 AND deleted_at IS NULL
	`, serviceOptionID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE fyp_fuli_service_options SET deleted_at = NOW()
		WHERE service_option_id = $1 AND deleted_at IS NULL
	`, serviceOptionID)
	return err
}

// IsOptionEffectiveOn reports whether serviceOptionID's own validity window
// covers date.
func (r *serviceRepo) IsOptionEffectiveOn(ctx context.Context, serviceOptionID int64, date string) (bool, error) {
	var ok bool
	err := r.DB.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM fyp_fuli_service_options
			WHERE service_option_id = $1 AND deleted_at IS NULL
				AND effective_from <= $2::date
				AND (effective_until IS NULL OR effective_until >= $2::date)
		)
	`, serviceOptionID, date).Scan(&ok)
	return ok, err
}

// SetOptionWindow resizes an option's own validity window.
func (r *serviceRepo) SetOptionWindow(ctx context.Context, tx *sql.Tx, serviceOptionID int64, from string, until *string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_service_options
		SET effective_from = COALESCE($2::date, effective_from), effective_until = $3::date
		WHERE service_option_id = $1
	`, serviceOptionID, nilIfEmpty(from), until)
	return err
}

// SetServiceDefaultOption marks serviceOptionID as the service's default
// option and every other option under the service as not-default, atomically
// — so a service always has exactly one default.
func (r *serviceRepo) SetServiceDefaultOption(ctx context.Context, tx *sql.Tx, serviceID int64, serviceOptionID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_service_options
		SET is_default = (service_option_id = $2)
		WHERE service_id = $1
	`, serviceID, serviceOptionID)
	return err
}

func (r *serviceRepo) SoftDeleteService(ctx context.Context, tx *sql.Tx, serviceID int64, businessID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_services SET deleted_at = NOW()
		WHERE service_id = $1 AND business_id = $2 AND deleted_at IS NULL
	`, serviceID, businessID)
	return err
}

func (r *serviceRepo) GetServicesByBusinessID(ctx context.Context, businessID int64) ([]param.ServiceParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT service_id, business_id, service_name, description
		FROM fyp_fuli_services
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

// optionHasBookingSQL is HasBookingForOption's check inlined as a column
// expression, so a list of options can carry it without an N+1 query per row.
func optionHasBookingSQL(alias string) string {
	return fmt.Sprintf(`EXISTS (
		SELECT 1 FROM fyp_fuli_bookings b
		JOIN fyp_fuli_service_slot_options ssp ON b.slot_option_id = ssp.slot_option_id
		WHERE ssp.service_option_id = %s.service_option_id AND b.deleted_at IS NULL
			AND b.status IN ('pending', 'accepted', 'rescheduled')
	)`, alias)
}

// GetServiceOptionsByServiceID returns every live option for a service, the
// default first (see is_default), then by start date.
func (r *serviceRepo) GetServiceOptionsByServiceID(ctx context.Context, serviceID int64) ([]param.ServiceOptionParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT service_option_id, service_id, service_option_name, description,
			to_char(effective_from, 'YYYY-MM-DD'), to_char(effective_until, 'YYYY-MM-DD'), is_default,
			`+optionHasBookingSQL("fyp_fuli_service_options")+`
		FROM fyp_fuli_service_options
		WHERE service_id = $1 AND deleted_at IS NULL
		ORDER BY is_default DESC, effective_from ASC, service_option_id ASC
	`, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	today := time.Now().Format("2006-01-02")
	var packages []param.ServiceOptionParam
	for rows.Next() {
		p, err := scanServiceOptionWithBooking(rows)
		if err != nil {
			return nil, err
		}
		// Removed = as of today, this option's own window doesn't cover it.
		p.IsRemoved = !(p.EffectiveFrom <= today && (p.EffectiveUntil == nil || *p.EffectiveUntil >= today))
		packages = append(packages, *p)
	}
	return packages, nil
}

func (r *serviceRepo) GetServiceOptionItemsByOptionID(ctx context.Context, serviceOptionID int64) ([]param.ServiceOptionItemParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT service_option_item_id, service_option_id, service_option_item_name
		FROM fyp_fuli_service_option_items
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
		FROM fyp_fuli_service_option_items
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
		SELECT COUNT(*) FROM fyp_fuli_bookings b
		JOIN fyp_fuli_service_slot_options ssp ON b.slot_option_id = ssp.slot_option_id
		JOIN fyp_fuli_service_options sp ON ssp.service_option_id = sp.service_option_id
		WHERE sp.service_id = $1 AND b.deleted_at IS NULL
			AND b.status IN ('pending', 'accepted', 'rescheduled')
	`, serviceID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasBookingForOption reports whether this option has an active booking, so
// deleting it can't orphan one.
func (r *serviceRepo) HasBookingForOption(ctx context.Context, serviceOptionID int64) (bool, error) {
	var count int
	err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM fyp_fuli_bookings b
		JOIN fyp_fuli_service_slot_options ssp ON b.slot_option_id = ssp.slot_option_id
		WHERE ssp.service_option_id = $1 AND b.deleted_at IS NULL
			AND b.status IN ('pending', 'accepted', 'rescheduled')
	`, serviceOptionID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *serviceRepo) CascadeDeleteServiceSlots(ctx context.Context, tx *sql.Tx, serviceID int64) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_service_slot_options sso
		SET deleted_at = NOW()
		FROM fyp_fuli_service_options so
		WHERE sso.service_option_id = so.service_option_id
			AND so.service_id = $1
			AND sso.deleted_at IS NULL
	`, serviceID); err != nil {
		return err
	}

	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_service_slots ss
		SET deleted_at = NOW()
		WHERE ss.deleted_at IS NULL
			AND EXISTS (
				SELECT 1 FROM fyp_fuli_service_slot_options sso
				JOIN fyp_fuli_service_options so ON so.service_option_id = sso.service_option_id
				WHERE sso.service_slot_id = ss.service_slot_id AND so.service_id = $1
			)
			AND NOT EXISTS (
				SELECT 1 FROM fyp_fuli_service_slot_options sso2
				WHERE sso2.service_slot_id = ss.service_slot_id AND sso2.deleted_at IS NULL
			)
	`, serviceID)
	return err
}

// DropSlotOptionIfExpired soft-deletes a slot's offering of an option
// (identified by slotOptionID) if that option's own validity window no
// longer covers the slot's date — called once a booking that had held this
// offering is released (reject/cancel/reschedule-away), since the owner may
// have shrunk the option's window while it was booked. No-op if the window
// still covers it, or if there's no such active offering to check.
func (r *serviceRepo) DropSlotOptionIfExpired(ctx context.Context, tx *sql.Tx, slotOptionID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_service_slot_options sso
		SET deleted_at = NOW()
		FROM fyp_fuli_service_options so, fyp_fuli_service_slots ss
		WHERE sso.slot_option_id = $1
			AND sso.deleted_at IS NULL
			AND so.service_option_id = sso.service_option_id
			AND ss.service_slot_id = sso.service_slot_id
			AND NOT (so.effective_from <= ss.date AND (so.effective_until IS NULL OR so.effective_until >= ss.date))
	`, slotOptionID)
	return err
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
	var description, effectiveUntil sql.NullString
	if err := row.Scan(
		&p.ServiceOptionID, &p.ServiceID, &p.ServiceOptionName, &description,
		&p.EffectiveFrom, &effectiveUntil, &p.IsDefault,
	); err != nil {
		return nil, err
	}
	p.Description = utils.NullStringPtr(description)
	if effectiveUntil.Valid {
		p.EffectiveUntil = &effectiveUntil.String
	}
	return &p, nil
}

// scanServiceOptionWithBooking is scanServiceOption plus the trailing
// has_booking column added by optionHasBookingSQL.
func scanServiceOptionWithBooking(row rowScannerService) (*param.ServiceOptionParam, error) {
	var p param.ServiceOptionParam
	var description, effectiveUntil sql.NullString
	if err := row.Scan(
		&p.ServiceOptionID, &p.ServiceID, &p.ServiceOptionName, &description,
		&p.EffectiveFrom, &effectiveUntil, &p.IsDefault, &p.HasBooking,
	); err != nil {
		return nil, err
	}
	p.Description = utils.NullStringPtr(description)
	if effectiveUntil.Valid {
		p.EffectiveUntil = &effectiveUntil.String
	}
	return &p, nil
}
