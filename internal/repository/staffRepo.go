package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"fyp/domain/param"
	"fyp/internal/interfaces"
)

// staffHasBookingSQL returns an EXISTS clause reporting whether staffAlias has
// a genuinely active booking (pending/accepted/rescheduled) — for use as a
// computed column, mirroring optionHasBookingSQL in serviceRepo.go.
func staffHasBookingSQL(staffAlias string) string {
	return fmt.Sprintf(`EXISTS (
		SELECT 1 FROM fyp_fuli_bookings b
		JOIN fyp_fuli_service_slot_options sso ON b.slot_option_id = sso.slot_option_id
		JOIN fyp_fuli_service_slots ss ON sso.service_slot_id = ss.service_slot_id
		WHERE ss.staff_id = %s.staff_id AND b.deleted_at IS NULL
			AND b.status IN ('pending', 'accepted', 'rescheduled')
	)`, staffAlias)
}

type staffRepo struct {
	DB *sql.DB
}

func NewStaffRepo(db *sql.DB) interfaces.IStaffRepo {
	return &staffRepo{DB: db}
}

func (s *staffRepo) InsertStaff(ctx context.Context, tx *sql.Tx, p param.StaffParam) (*int64, error) {
	// Restore a previously soft-deleted record if one exists for this user.
	var staffID int64
	err := tx.QueryRowContext(ctx, `
		UPDATE fyp_fuli_staff SET
			business_id          = $2,
			staff_name           = $3,
			staff_contact_number = $4,
			position             = $5,
			deleted_at           = NULL
		WHERE user_id = $1 AND deleted_at IS NOT NULL
		RETURNING staff_id
	`, p.UserID, p.BusinessID, p.StaffName, p.StaffContactNumber, p.Position).Scan(&staffID)
	if err == nil {
		return &staffID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	// No soft-deleted record — normal insert (unique violation = active staff).
	row := tx.QueryRowContext(ctx, `
		INSERT INTO fyp_fuli_staff (user_id, business_id, staff_name, staff_contact_number, position)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING staff_id
	`, p.UserID, p.BusinessID, p.StaffName, p.StaffContactNumber, p.Position)
	return scanStaffID(row)
}

func (s *staffRepo) DeleteStaffWorkingHours(ctx context.Context, tx *sql.Tx, staffID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_staff_working_hours SET deleted_at = NOW()
		WHERE staff_id = $1 AND deleted_at IS NULL
	`, staffID)
	return err
}

func (s *staffRepo) InsertStaffWorkingHours(ctx context.Context, tx *sql.Tx, param param.StaffParam) error {
	for _, wh := range param.WorkingHours {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO fyp_fuli_staff_working_hours (
				staff_id,
				day,
				start_time,
				end_time
			)
			VALUES ($1, $2, $3, $4)
		`,
			param.StaffID,
			wh.Day,
			wh.StartTime.Format("15:04:05"),
			wh.EndTime.Format("15:04:05"),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *staffRepo) GetStaffByUserID(ctx context.Context, userID int64) (*param.StaffParam, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT
			staff_id,
			user_id,
			business_id,
			staff_name,
			staff_contact_number,
			position
		FROM fyp_fuli_staff
		WHERE user_id = $1
			AND deleted_at IS NULL
	`, userID)

	var staff param.StaffParam
	var contactNumber sql.NullString
	var position sql.NullString
	if err := row.Scan(
		&staff.StaffID,
		&staff.UserID,
		&staff.BusinessID,
		&staff.StaffName,
		&contactNumber,
		&position,
	); err != nil {
		return nil, err
	}

	if contactNumber.Valid {
		staff.StaffContactNumber = contactNumber.String
	}
	if position.Valid {
		staff.Position = position.String
	}

	return &staff, nil
}

func (s *staffRepo) GetStaffWorkingHours(ctx context.Context, staffID int64) ([]param.WorkingHourParam, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT day, start_time, end_time
		FROM fyp_fuli_staff_working_hours
		WHERE staff_id = $1 AND deleted_at IS NULL
		ORDER BY start_time
	`, staffID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hours []param.WorkingHourParam
	for rows.Next() {
		var wh param.WorkingHourParam
		if err := rows.Scan(&wh.Day, &wh.StartTime, &wh.EndTime); err != nil {
			return nil, err
		}
		hours = append(hours, wh)
	}
	return hours, rows.Err()
}

func (s *staffRepo) GetStaffByBusinessID(ctx context.Context, businessID int64) ([]param.StaffParam, error) {
	rows, err := s.DB.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			st.staff_id,
			st.user_id,
			st.business_id,
			st.staff_name,
			u.email,
			u.must_reset_password,
			st.staff_contact_number,
			st.position,
			%s
		FROM fyp_fuli_staff st
		JOIN fyp_fuli_users u ON u.user_id = st.user_id
		WHERE st.business_id = $1
			AND st.deleted_at IS NULL
		ORDER BY st.staff_id
	`, staffHasBookingSQL("st")), businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var staffList []param.StaffParam
	for rows.Next() {
		staff, err := scanStaffRow(rows)
		if err != nil {
			return nil, err
		}
		staffList = append(staffList, *staff)
	}
	return staffList, rows.Err()
}

func (s *staffRepo) UpdateStaff(ctx context.Context, tx *sql.Tx, p param.StaffParam) (*param.StaffParam, error) {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_staff
		SET staff_name = $1, staff_contact_number = $2, position = $3
		WHERE staff_id = $4 AND business_id = $5 AND deleted_at IS NULL
	`, p.StaffName, p.StaffContactNumber, p.Position, p.StaffID, p.BusinessID)
	if err != nil {
		return nil, err
	}
	return s.GetStaffByIDTx(ctx, tx, p.StaffID, p.BusinessID)
}

func (s *staffRepo) GetStaffByIDTx(ctx context.Context, tx *sql.Tx, staffID int64, businessID int64) (*param.StaffParam, error) {
	row := tx.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT
			st.staff_id,
			st.user_id,
			st.business_id,
			st.staff_name,
			u.email,
			u.must_reset_password,
			st.staff_contact_number,
			st.position,
			%s
		FROM fyp_fuli_staff st
		JOIN fyp_fuli_users u ON u.user_id = st.user_id
		WHERE st.staff_id = $1
			AND st.business_id = $2
			AND st.deleted_at IS NULL
	`, staffHasBookingSQL("st")), staffID, businessID)
	return scanStaffRow(row)
}

func (s *staffRepo) SoftDeleteStaff(ctx context.Context, tx *sql.Tx, staffID int64, businessID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_staff SET deleted_at = NOW()
		WHERE staff_id = $1 AND business_id = $2 AND deleted_at IS NULL
	`, staffID, businessID)
	return err
}

// HasBookingForStaff reports whether staffID has a genuinely active booking —
// past, cancelled, and rejected bookings don't block deleting the staff member.
func (s *staffRepo) HasBookingForStaff(ctx context.Context, staffID int64) (bool, error) {
	var count int
	err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM fyp_fuli_bookings b
		JOIN fyp_fuli_service_slot_options sso ON b.slot_option_id = sso.slot_option_id
		JOIN fyp_fuli_service_slots ss ON sso.service_slot_id = ss.service_slot_id
		WHERE ss.staff_id = $1 AND b.deleted_at IS NULL
			AND b.status IN ('pending', 'accepted', 'rescheduled')
	`, staffID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func scanStaffRow(row rowScannerService) (*param.StaffParam, error) {
	var staff param.StaffParam
	var contactNumber sql.NullString
	var position sql.NullString
	if err := row.Scan(
		&staff.StaffID,
		&staff.UserID,
		&staff.BusinessID,
		&staff.StaffName,
		&staff.StaffEmail,
		&staff.MustResetPassword,
		&contactNumber,
		&position,
		&staff.HasBooking,
	); err != nil {
		return nil, err
	}
	if contactNumber.Valid {
		staff.StaffContactNumber = contactNumber.String
	}
	if position.Valid {
		staff.Position = position.String
	}
	return &staff, nil
}

func scanStaffID(row rowScannerService) (*int64, error) {
	var staffID int64
	if err := row.Scan(&staffID); err != nil {
		return nil, err
	}
	return &staffID, nil
}
