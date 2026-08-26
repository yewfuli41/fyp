package repository

import (
	"context"
	"database/sql"
	"errors"
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

// excludeUnavailableStaff appends the clause that hides a staff member's slots
// across the range they can't work. Kept as one helper so GetAvailableSlots and
// GetAvailableDates can never drift apart on it — a date the calendar paints
// green must be a date the slot list can actually fill.
func excludeUnavailableStaff(query string, args []any, u *param.StaffUnavailability) (string, []any) {
	if u == nil {
		return query, args
	}
	args = append(args, u.StaffID, u.From, u.Until)
	return query + fmt.Sprintf(
		" AND NOT (ss.staff_id = $%d AND ss.date BETWEEN $%d::date AND $%d::date)",
		len(args)-2, len(args)-1, len(args),
	), args
}

func (r *bookingRepo) GetAvailableSlots(ctx context.Context, businessID int64, serviceOptionID int64, date string, staffID *int64, unassignedOnly bool, unavailable *param.StaffUnavailability) ([]param.ServiceSlotParam, error) {
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
		FROM fyp_fuli_service_slots ss
		LEFT JOIN fyp_fuli_staff st ON st.staff_id = ss.staff_id
		JOIN fyp_fuli_service_slot_options ssp ON ssp.service_slot_id = ss.service_slot_id
		JOIN fyp_fuli_service_options so ON so.service_option_id = ssp.service_option_id
		JOIN fyp_fuli_services s ON s.service_id = so.service_id
		WHERE ss.date = $2
			AND ss.deleted_at IS NULL
			AND ssp.deleted_at IS NULL
			AND ssp.service_option_id = $3
			AND s.business_id = $1
			AND so.effective_from <= ss.date
			AND (so.effective_until IS NULL OR so.effective_until >= ss.date)
			AND NOT EXISTS (
				SELECT 1 FROM fyp_fuli_bookings b
				WHERE b.slot_option_id = ssp.slot_option_id
					AND b.deleted_at IS NULL
					AND b.status NOT IN ('cancelled', 'rejected')
			)`
	args := []any{businessID, date, serviceOptionID}
	if unassignedOnly {
		query += " AND ss.staff_id IS NULL"
	} else if staffID != nil {
		args = append(args, *staffID)
		query += fmt.Sprintf(" AND ss.staff_id = $%d", len(args))
	}
	query, args = excludeUnavailableStaff(query, args, unavailable)
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

// GetAvailableDates returns the distinct dates in [from, until] that have at
// least one bookable slot, optionally narrowed to one service and/or one
// specific option of that service (either nil = any) and/or one staff member
// (nil = any staff — used by the owner; a staff caller passes their own id to
// see only their own slots). unassignedOnly, when true, narrows instead to
// owner-managed (no staff assigned) slots and wins over staffID. Each
// service_slot's date is checked against the effective window of the
// specific option version it was booked under — a version-agnostic version
// of the check GetAvailableSlots does for one already-resolved option.
func (r *bookingRepo) GetAvailableDates(ctx context.Context, businessID int64, serviceID *int64, serviceOptionID *int64, staffID *int64, unassignedOnly bool, from string, until string, unavailable *param.StaffUnavailability) ([]string, error) {
	query := `
		SELECT DISTINCT to_char(ss.date, 'YYYY-MM-DD')
		FROM fyp_fuli_service_slots ss
		JOIN fyp_fuli_service_slot_options ssp ON ssp.service_slot_id = ss.service_slot_id
		JOIN fyp_fuli_service_options so ON so.service_option_id = ssp.service_option_id
		JOIN fyp_fuli_services s ON s.service_id = so.service_id
		WHERE s.business_id = $1
			AND ss.date BETWEEN $2::date AND $3::date
			AND ss.date >= so.effective_from
			AND (so.effective_until IS NULL OR ss.date <= so.effective_until)
			AND ss.deleted_at IS NULL
			AND ssp.deleted_at IS NULL
			AND so.deleted_at IS NULL
			AND NOT EXISTS (
				SELECT 1 FROM fyp_fuli_bookings b
				WHERE b.slot_option_id = ssp.slot_option_id
					AND b.deleted_at IS NULL
					AND b.status NOT IN ('cancelled', 'rejected')
			)`
	args := []any{businessID, from, until}
	if serviceID != nil {
		args = append(args, *serviceID)
		query += fmt.Sprintf(" AND so.service_id = $%d", len(args))
	}
	if serviceOptionID != nil {
		args = append(args, *serviceOptionID)
		query += fmt.Sprintf(" AND so.service_option_id = $%d", len(args))
	}
	if unassignedOnly {
		query += " AND ss.staff_id IS NULL"
	} else if staffID != nil {
		args = append(args, *staffID)
		query += fmt.Sprintf(" AND ss.staff_id = $%d", len(args))
	}
	query, args = excludeUnavailableStaff(query, args, unavailable)
	query += " ORDER BY 1"

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		dates = append(dates, d)
	}
	return dates, rows.Err()
}

func (r *bookingRepo) getSlotOption(ctx context.Context, serviceSlotID int64, serviceOptionID int64) ([]param.SlotTierParam, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT ssp.slot_option_id, sp.service_option_id, sp.service_option_name, s.service_id, s.service_name
		FROM fyp_fuli_service_slot_options ssp
		JOIN fyp_fuli_service_options sp ON sp.service_option_id = ssp.service_option_id
		JOIN fyp_fuli_services s ON s.service_id = sp.service_id
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
		FROM fyp_fuli_bookings b
		JOIN fyp_fuli_service_slot_options sso ON sso.slot_option_id = b.slot_option_id
		JOIN fyp_fuli_service_options so ON so.service_option_id = sso.service_option_id
		JOIN fyp_fuli_services s ON s.service_id = so.service_id
		JOIN fyp_fuli_business_profiles bp ON bp.business_id = s.business_id
		WHERE b.user_id = $1 AND b.booking_type != 'walk_in' AND b.deleted_at IS NULL
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

// bookingDetailSelect is the shared projection/join for BookingDetailParam rows.
const bookingDetailSelect = `
	SELECT
		b.booking_id,
		b.status,
		b.booking_type,
		ss.service_slot_id,
		sso.slot_option_id,
		so.service_option_id,
		to_char(ss.date, 'YYYY-MM-DD'),
		ss.start_time,
		ss.end_time,
		s.service_name,
		so.service_option_name,
		st.staff_name,
		cu.username,
		cu.email,
		bp.business_id,
		bp.business_name,
		b.description,
		b.created_at
	FROM fyp_fuli_bookings b
	JOIN fyp_fuli_service_slot_options sso ON sso.slot_option_id = b.slot_option_id
	JOIN fyp_fuli_service_options so ON so.service_option_id = sso.service_option_id
	JOIN fyp_fuli_services s ON s.service_id = so.service_id
	JOIN fyp_fuli_business_profiles bp ON bp.business_id = s.business_id
	JOIN fyp_fuli_service_slots ss ON ss.service_slot_id = sso.service_slot_id
	JOIN fyp_fuli_users cu ON cu.user_id = b.user_id
	LEFT JOIN fyp_fuli_staff st ON st.staff_id = ss.staff_id
`

func scanBookingDetail(row rowScanner) (*param.BookingDetailParam, error) {
	var d param.BookingDetailParam
	var staffName, description sql.NullString
	if err := row.Scan(
		&d.BookingID, &d.Status, &d.BookingType,
		&d.ServiceSlotID, &d.SlotOptionID, &d.ServiceOptionID,
		&d.Date, &d.StartTime, &d.EndTime,
		&d.ServiceName, &d.OptionName, &staffName,
		&d.CustomerName, &d.CustomerEmail,
		&d.BusinessID, &d.BusinessName, &description, &d.CreatedAt,
	); err != nil {
		return nil, err
	}
	if staffName.Valid {
		d.StaffName = &staffName.String
	}
	if description.Valid {
		d.Description = &description.String
	}
	return &d, nil
}

func (r *bookingRepo) GetBusinessBookings(ctx context.Context, businessID int64, staffID *int64) ([]param.BookingDetailParam, error) {
	query := bookingDetailSelect + `
		WHERE bp.business_id = $1
			AND b.deleted_at IS NULL
			AND b.status IN ('pending', 'accepted', 'rescheduled')`
	args := []any{businessID}
	if staffID != nil {
		args = append(args, *staffID)
		query += fmt.Sprintf(" AND ss.staff_id = $%d", len(args))
	}
	query += " ORDER BY ss.date, ss.start_time, b.booking_id"

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []param.BookingDetailParam
	for rows.Next() {
		d, err := scanBookingDetail(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *d)
	}
	return results, rows.Err()
}

func (r *bookingRepo) GetCustomerBookings(ctx context.Context, userID int64) ([]param.BookingDetailParam, error) {
	// Excludes walk-ins: those are recorded under the business owner/staff's
	// own user_id (there's no separate walk-in customer account), and must
	// not show up as if that same user had booked their own business.
	rows, err := r.DB.QueryContext(ctx, bookingDetailSelect+`
		WHERE b.user_id = $1
			AND b.booking_type != 'walk_in'
			AND b.deleted_at IS NULL
		ORDER BY ss.date DESC, ss.start_time DESC, b.booking_id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []param.BookingDetailParam
	for rows.Next() {
		d, err := scanBookingDetail(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *d)
	}
	return results, rows.Err()
}

func (r *bookingRepo) GetBookingDetail(ctx context.Context, bookingID int64) (*param.BookingDetailParam, error) {
	row := r.DB.QueryRowContext(ctx, bookingDetailSelect+`
		WHERE b.booking_id = $1 AND b.deleted_at IS NULL
	`, bookingID)
	return scanBookingDetail(row)
}

func (r *bookingRepo) GetBookingContext(ctx context.Context, bookingID int64) (*param.BookingContextParam, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT
			b.booking_id,
			b.status,
			ss.service_slot_id,
			sso.slot_option_id,
			b.user_id,
			cu.username,
			cu.email,
			bp.owner_user_id,
			ou.username,
			ou.email,
			su.user_id,
			st.staff_name,
			su.email,
			bp.business_name,
			to_char(ss.date, 'YYYY-MM-DD') || ' ' || to_char(ss.start_time, 'HH24:MI')
		FROM fyp_fuli_bookings b
		JOIN fyp_fuli_service_slot_options sso ON sso.slot_option_id = b.slot_option_id
		JOIN fyp_fuli_service_options so ON so.service_option_id = sso.service_option_id
		JOIN fyp_fuli_services s ON s.service_id = so.service_id
		JOIN fyp_fuli_business_profiles bp ON bp.business_id = s.business_id
		JOIN fyp_fuli_service_slots ss ON ss.service_slot_id = sso.service_slot_id
		JOIN fyp_fuli_users cu ON cu.user_id = b.user_id
		JOIN fyp_fuli_users ou ON ou.user_id = bp.owner_user_id
		LEFT JOIN fyp_fuli_staff st ON st.staff_id = ss.staff_id
		LEFT JOIN fyp_fuli_users su ON su.user_id = st.user_id
		WHERE b.booking_id = $1 AND b.deleted_at IS NULL
	`, bookingID)

	var c param.BookingContextParam
	var staffID sql.NullInt64
	var staffName, staffEmail sql.NullString
	if err := row.Scan(
		&c.BookingID, &c.Status, &c.ServiceSlotID, &c.SlotOptionID,
		&c.CustomerUserID, &c.CustomerName, &c.CustomerEmail,
		&c.BusinessOwnerID, &c.OwnerName, &c.OwnerEmail,
		&staffID, &staffName, &staffEmail,
		&c.BusinessName, &c.WhenText,
	); err != nil {
		return nil, err
	}
	if staffID.Valid {
		c.SlotStaffUserID = &staffID.Int64
	}
	if staffName.Valid {
		c.StaffName = &staffName.String
	}
	if staffEmail.Valid {
		c.StaffEmail = &staffEmail.String
	}
	return &c, nil
}

// GetBookingContextForSlot is GetBookingContext keyed by service_slot_id
// instead of booking_id, scoped to the slot's currently active booking (if
// any) — used to notify a customer when their slot's staff changes.
func (r *bookingRepo) GetBookingContextForSlot(ctx context.Context, serviceSlotID int64) (*param.BookingContextParam, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT
			b.booking_id,
			b.status,
			ss.service_slot_id,
			sso.slot_option_id,
			b.user_id,
			cu.username,
			cu.email,
			bp.owner_user_id,
			ou.username,
			ou.email,
			su.user_id,
			st.staff_name,
			su.email,
			bp.business_name,
			to_char(ss.date, 'YYYY-MM-DD') || ' ' || to_char(ss.start_time, 'HH24:MI')
		FROM fyp_fuli_bookings b
		JOIN fyp_fuli_service_slot_options sso ON sso.slot_option_id = b.slot_option_id
		JOIN fyp_fuli_service_options so ON so.service_option_id = sso.service_option_id
		JOIN fyp_fuli_services s ON s.service_id = so.service_id
		JOIN fyp_fuli_business_profiles bp ON bp.business_id = s.business_id
		JOIN fyp_fuli_service_slots ss ON ss.service_slot_id = sso.service_slot_id
		JOIN fyp_fuli_users cu ON cu.user_id = b.user_id
		JOIN fyp_fuli_users ou ON ou.user_id = bp.owner_user_id
		LEFT JOIN fyp_fuli_staff st ON st.staff_id = ss.staff_id
		LEFT JOIN fyp_fuli_users su ON su.user_id = st.user_id
		WHERE ss.service_slot_id = $1
			AND b.deleted_at IS NULL
			AND b.status IN ('pending', 'accepted', 'rescheduled')
	`, serviceSlotID)

	var c param.BookingContextParam
	var staffID sql.NullInt64
	var staffName, staffEmail sql.NullString
	if err := row.Scan(
		&c.BookingID, &c.Status, &c.ServiceSlotID, &c.SlotOptionID,
		&c.CustomerUserID, &c.CustomerName, &c.CustomerEmail,
		&c.BusinessOwnerID, &c.OwnerName, &c.OwnerEmail,
		&staffID, &staffName, &staffEmail,
		&c.BusinessName, &c.WhenText,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if staffID.Valid {
		c.SlotStaffUserID = &staffID.Int64
	}
	if staffName.Valid {
		c.StaffName = &staffName.String
	}
	if staffEmail.Valid {
		c.StaffEmail = &staffEmail.String
	}
	return &c, nil
}

func (r *bookingRepo) UpdateBookingStatus(ctx context.Context, tx *sql.Tx, bookingID int64, status string, decidedBy int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_bookings
		SET status = $1, decided_at = NOW(), decided_by = $2
		WHERE booking_id = $3 AND deleted_at IS NULL
	`, status, decidedBy, bookingID)
	return err
}

func (r *bookingRepo) UpdateBookingDescription(ctx context.Context, tx *sql.Tx, bookingID int64, description *string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_bookings
		SET description = $1
		WHERE booking_id = $2 AND deleted_at IS NULL
	`, description, bookingID)
	return err
}

func (r *bookingRepo) UpdateBookingSlotOption(ctx context.Context, tx *sql.Tx, bookingID int64, newSlotOptionID int64, status string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_bookings
		SET slot_option_id = $1, status = $2, decided_at = NOW()
		WHERE booking_id = $3 AND deleted_at IS NULL
	`, newSlotOptionID, status, bookingID)
	return err
}

func (r *bookingRepo) RejectOtherPendingBookingsForSlot(ctx context.Context, tx *sql.Tx, serviceSlotID int64, exceptBookingID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_bookings
		SET status = 'rejected', decided_at = NOW()
		WHERE booking_id <> $2
			AND status = 'pending'
			AND deleted_at IS NULL
			AND slot_option_id IN (
				SELECT slot_option_id FROM fyp_fuli_service_slot_options
				WHERE service_slot_id = $1 AND deleted_at IS NULL
			)
	`, serviceSlotID, exceptBookingID)
	return err
}

func (r *bookingRepo) IsSlotOptionOwnedByUser(ctx context.Context, slotOptionID int64, userID int64) (bool, error) {
	var owned bool
	err := r.DB.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM fyp_fuli_service_slot_options sso
			JOIN fyp_fuli_service_options so ON so.service_option_id = sso.service_option_id
			JOIN fyp_fuli_services s ON s.service_id = so.service_id
			JOIN fyp_fuli_business_profiles bp ON bp.business_id = s.business_id
			WHERE sso.slot_option_id = $1 AND bp.owner_user_id = $2
		)
	`, slotOptionID, userID).Scan(&owned)
	return owned, err
}

func (r *bookingRepo) SweepPastBookings(ctx context.Context) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE fyp_fuli_bookings b
		SET status = 'past'
		FROM fyp_fuli_service_slot_options sso
		JOIN fyp_fuli_service_slots ss ON ss.service_slot_id = sso.service_slot_id
		WHERE b.slot_option_id = sso.slot_option_id
			AND b.status IN ('pending', 'accepted', 'rescheduled')
			AND b.deleted_at IS NULL
			AND (ss.date + ss.end_time) < NOW()
	`)
	return err
}

func (r *bookingRepo) SlotOptionIsAvailable(ctx context.Context, slotOptionID int64) (bool, error) {
	var available bool
	err := r.DB.QueryRowContext(ctx, `
		SELECT EXISTS(
				SELECT 1 FROM fyp_fuli_service_slot_options ssp
				JOIN fyp_fuli_service_slots ss ON ss.service_slot_id = ssp.service_slot_id
				WHERE ssp.slot_option_id = $1
					AND ssp.deleted_at IS NULL
					AND ss.deleted_at IS NULL
					AND (ss.date + ss.start_time) > NOW()
			)
			AND NOT EXISTS(
				SELECT 1 FROM fyp_fuli_bookings
				WHERE slot_option_id = $1
					AND deleted_at IS NULL
					AND status NOT IN ('cancelled', 'rejected')
			)
	`, slotOptionID).Scan(&available)
	return available, err
}

// GetSlotOptionStaffUserID returns the login user id of the staff member a
// slot option is assigned to, or nil if it's owner-managed (unassigned) —
// used to confine a staff member's reschedule choice to their own slots.
func (r *bookingRepo) GetSlotOptionStaffUserID(ctx context.Context, slotOptionID int64) (*int64, error) {
	var staffUserID sql.NullInt64
	err := r.DB.QueryRowContext(ctx, `
		SELECT st.user_id
		FROM fyp_fuli_service_slot_options ssp
		JOIN fyp_fuli_service_slots ss ON ss.service_slot_id = ssp.service_slot_id
		LEFT JOIN fyp_fuli_staff st ON st.staff_id = ss.staff_id
		WHERE ssp.slot_option_id = $1
	`, slotOptionID).Scan(&staffUserID)
	if err != nil {
		return nil, err
	}
	if staffUserID.Valid {
		return &staffUserID.Int64, nil
	}
	return nil, nil
}

func (r *bookingRepo) GetSlotOptionAssignment(ctx context.Context, slotOptionID int64) (*int64, string, error) {
	var staffID sql.NullInt64
	var date string
	err := r.DB.QueryRowContext(ctx, `
		SELECT ss.staff_id, to_char(ss.date, 'YYYY-MM-DD')
		FROM fyp_fuli_service_slot_options ssp
		JOIN fyp_fuli_service_slots ss ON ss.service_slot_id = ssp.service_slot_id
		WHERE ssp.slot_option_id = $1
	`, slotOptionID).Scan(&staffID, &date)
	if err != nil {
		return nil, "", err
	}
	if staffID.Valid {
		return &staffID.Int64, date, nil
	}
	return nil, date, nil
}

func (r *bookingRepo) InsertBooking(ctx context.Context, userID int64, slotOptionID int64, description *string) (*param.BookingParam, error) {
	var b param.BookingParam
	var desc sql.NullString
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO fyp_fuli_bookings (user_id, slot_option_id, booking_group_id, status, booking_type, description)
		VALUES ($1, $2, nextval('fyp_fuli_booking_group_seq'), 'pending', 'online', $3)
		RETURNING booking_id, user_id, slot_option_id, status, booking_type, description, created_at
	`, userID, slotOptionID, description).Scan(
		&b.BookingID, &b.UserID, &b.SlotOptionID, &b.Status, &b.BookingType, &desc, &b.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		b.Description = &desc.String
	}
	return &b, nil
}

func (r *bookingRepo) InsertWalkInBooking(ctx context.Context, userID int64, slotOptionID int64) (*param.BookingParam, error) {
	var b param.BookingParam
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO fyp_fuli_bookings (user_id, slot_option_id, booking_group_id, status, booking_type, decided_by, decided_at)
		VALUES ($1, $2, nextval('fyp_fuli_booking_group_seq'), 'accepted', 'walk_in', $1, NOW())
		RETURNING booking_id, user_id, slot_option_id, status, booking_type, created_at
	`, userID, slotOptionID).Scan(
		&b.BookingID, &b.UserID, &b.SlotOptionID, &b.Status, &b.BookingType, &b.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &b, nil
}
