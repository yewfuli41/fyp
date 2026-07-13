package repository

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/domain/param"
	"fyp/internal/interfaces"
)

type bookingRepo struct {
	DB *sql.DB
}

func NewBookingRepo(db *sql.DB) interfaces.IBookingRepo {
	return &bookingRepo{DB: db}
}

func (r *bookingRepo) GetAvailableSlots(ctx context.Context, businessID int64, serviceOptionID int64, date string, staffID *int64) ([]param.ServiceSlotParam, error) {
	query := `
		SELECT
			ss.service_slot_id,
			ss.staff_id,
			st.staff_name,
			ss.recurring_schedule_id,
			to_char(ss.date, 'YYYY-MM-DD'),
			ss.start_time,
			ss.end_time,
			ss.created_by,
			false AS has_booking
		FROM service_slots ss
		LEFT JOIN staff st ON st.staff_id = ss.staff_id
		JOIN service_slot_options ssp ON ssp.service_slot_id = ss.service_slot_id
		JOIN service_options so ON so.service_option_id = ssp.service_option_id
		JOIN services s ON s.service_id = so.service_id
		WHERE ss.date = $2
			AND ss.deleted_at IS NULL
			AND ssp.deleted_at IS NULL
			AND ssp.service_option_id = $3
			AND s.business_id = $1
			AND NOT EXISTS (
				SELECT 1 FROM bookings b
				WHERE b.slot_option_id = ssp.slot_option_id
					AND b.deleted_at IS NULL
					AND b.status NOT IN ('cancelled', 'rejected')
			)`
	args := []any{businessID, date, serviceOptionID}
	if staffID != nil {
		args = append(args, *staffID)
		query += fmt.Sprintf(" AND ss.staff_id = $%d", len(args))
	}
	query += " ORDER BY ss.start_time, ss.service_slot_id"

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []param.ServiceSlotParam
	for rows.Next() {
		slot, err := scanServiceSlot(rows)
		if err != nil {
			return nil, err
		}
		slots = append(slots, *slot)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range slots {
		pkgs, err := r.getSlotOption(ctx, slots[i].ServiceSlotID, serviceOptionID)
		if err != nil {
			return nil, err
		}
		slots[i].Packages = pkgs
	}
	return slots, nil
}

func (r *bookingRepo) getSlotOption(ctx context.Context, serviceSlotID int64, serviceOptionID int64) ([]param.SlotTierParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT ssp.slot_option_id, sp.service_option_id, sp.service_option_name, s.service_id, s.service_name
		FROM service_slot_options ssp
		JOIN service_options sp ON sp.service_option_id = ssp.service_option_id
		JOIN services s ON s.service_id = sp.service_id
		WHERE ssp.service_slot_id = $1 AND ssp.service_option_id = $2 AND ssp.deleted_at IS NULL
	`, serviceSlotID, serviceOptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pkgs []param.SlotTierParam
	for rows.Next() {
		var p param.SlotTierParam
		if err := rows.Scan(&p.SlotOptionID, &p.ServiceOptionID, &p.ServiceOptionName, &p.ServiceID, &p.ServiceName); err != nil {
			return nil, err
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, rows.Err()
}

func (r *bookingRepo) GetRecentlyBookedBusinesses(ctx context.Context, userID int64) ([]param.BusinessProfileParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT bp.business_id, bp.owner_user_id, bp.business_name, bp.description, bp.address,
		       bp.image_url, bp.business_contact_number, bp.business_email
		FROM bookings b
		JOIN service_slot_options sso ON sso.slot_option_id = b.slot_option_id
		JOIN service_options so ON so.service_option_id = sso.service_option_id
		JOIN services s ON s.service_id = so.service_id
		JOIN business_profiles bp ON bp.business_id = s.business_id
		WHERE b.user_id = $1 AND b.deleted_at IS NULL
		GROUP BY bp.business_id
		ORDER BY MAX(b.created_at) DESC
		LIMIT 6
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []param.BusinessProfileParam
	for rows.Next() {
		bp, err := ScanBusinessProfile(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *bp)
	}
	return results, rows.Err()
}

func (r *bookingRepo) InsertBooking(ctx context.Context, userID int64, slotOptionID int64) (*param.BookingParam, error) {
	var b param.BookingParam
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO bookings (user_id, slot_option_id, booking_group_id, status, booking_type)
		VALUES ($1, $2, nextval('booking_group_seq'), 'pending', 'online')
		RETURNING booking_id, user_id, slot_option_id, status, booking_type, created_at
	`, userID, slotOptionID).Scan(&b.BookingID, &b.UserID, &b.SlotOptionID, &b.Status, &b.BookingType, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}
