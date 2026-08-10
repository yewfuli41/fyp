package repository

import (
	"context"
	"database/sql"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/utils"
)

type leaveRepo struct {
	DB *sql.DB
}

func NewLeaveRepo(db *sql.DB) interfaces.ILeaveRepo {
	return &leaveRepo{DB: db}
}

const leaveApplicationSelect = `
	SELECT la.leave_id, la.staff_id, st.business_id, st.staff_name, st.position,
		to_char(la.start_date, 'YYYY-MM-DD'), to_char(la.end_date, 'YYYY-MM-DD'),
		la.justification, la.status, la.remark, la.decided_at, la.created_at
	FROM fyp_fuli_leave_applications la
	JOIN fyp_fuli_staff st ON st.staff_id = la.staff_id
`

func scanLeaveApplication(row rowScannerService) (*param.LeaveApplicationParam, error) {
	var p param.LeaveApplicationParam
	var position sql.NullString
	var justification, remark sql.NullString
	var decidedAt sql.NullTime
	if err := row.Scan(
		&p.LeaveID, &p.StaffID, &p.BusinessID, &p.StaffName, &position,
		&p.StartDate, &p.EndDate,
		&justification, &p.Status, &remark, &decidedAt, &p.CreatedAt,
	); err != nil {
		return nil, err
	}
	if position.Valid {
		p.Position = position.String
	}
	p.Justification = utils.NullStringPtr(justification)
	p.Remark = utils.NullStringPtr(remark)
	if decidedAt.Valid {
		p.DecidedAt = &decidedAt.Time
	}
	return &p, nil
}

// InsertLeaveApplication returns just the new row's ID — the caller should
// fetch the full record (via GetLeaveApplicationByID) only after the
// transaction commits. Fetching it here, mid-transaction, would read through
// a different connection than tx and not see the still-uncommitted insert.
func (r *leaveRepo) InsertLeaveApplication(ctx context.Context, tx *sql.Tx, p param.LeaveApplicationParam) (int64, error) {
	var leaveID int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO fyp_fuli_leave_applications (staff_id, start_date, end_date, justification)
		VALUES ($1, $2::date, $3::date, $4)
		RETURNING leave_id
	`, p.StaffID, p.StartDate, p.EndDate, p.Justification).Scan(&leaveID)
	return leaveID, err
}

func (r *leaveRepo) GetLeaveApplicationsByStaffID(ctx context.Context, staffID int64) ([]param.LeaveApplicationParam, error) {
	rows, err := r.DB.QueryContext(ctx, leaveApplicationSelect+`
		WHERE la.staff_id = $1 AND la.deleted_at IS NULL
		ORDER BY la.start_date DESC, la.leave_id DESC
	`, staffID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []param.LeaveApplicationParam
	for rows.Next() {
		p, err := scanLeaveApplication(rows)
		if err != nil {
			return nil, err
		}
		apps = append(apps, *p)
	}
	return apps, rows.Err()
}

func (r *leaveRepo) GetLeaveApplicationsByBusinessID(ctx context.Context, businessID int64) ([]param.LeaveApplicationParam, error) {
	rows, err := r.DB.QueryContext(ctx, leaveApplicationSelect+`
		WHERE st.business_id = $1 AND la.deleted_at IS NULL
		ORDER BY (la.status = 'pending') DESC, la.start_date DESC, la.leave_id DESC
	`, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []param.LeaveApplicationParam
	for rows.Next() {
		p, err := scanLeaveApplication(rows)
		if err != nil {
			return nil, err
		}
		apps = append(apps, *p)
	}
	return apps, rows.Err()
}

func (r *leaveRepo) GetLeaveApplicationByID(ctx context.Context, leaveID int64) (*param.LeaveApplicationParam, error) {
	row := r.DB.QueryRowContext(ctx, leaveApplicationSelect+`
		WHERE la.leave_id = $1 AND la.deleted_at IS NULL
	`, leaveID)
	return scanLeaveApplication(row)
}

func (r *leaveRepo) UpdateLeaveStatus(ctx context.Context, tx *sql.Tx, leaveID int64, status string, remark *string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_leave_applications
		SET status = $1, remark = $2, decided_at = NOW()
		WHERE leave_id = $3 AND deleted_at IS NULL
	`, status, remark, leaveID)
	return err
}

func (r *leaveRepo) UpdateLeaveJustification(ctx context.Context, tx *sql.Tx, leaveID int64, justification *string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_leave_applications
		SET justification = $1
		WHERE leave_id = $2 AND deleted_at IS NULL
	`, justification, leaveID)
	return err
}

func (r *leaveRepo) SoftDeleteLeaveApplication(ctx context.Context, tx *sql.Tx, leaveID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE fyp_fuli_leave_applications SET deleted_at = NOW()
		WHERE leave_id = $1 AND deleted_at IS NULL
	`, leaveID)
	return err
}

func (r *leaveRepo) HasOverlappingLeave(ctx context.Context, staffID int64, startDate, endDate string) (bool, error) {
	var exists bool
	err := r.DB.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM fyp_fuli_leave_applications
			WHERE staff_id = $1 AND deleted_at IS NULL
				AND status IN ('pending', 'approved')
				AND start_date <= $3::date AND end_date >= $2::date
		)
	`, staffID, startDate, endDate).Scan(&exists)
	return exists, err
}

func (r *leaveRepo) IsStaffOnLeave(ctx context.Context, staffID int64, date string) (bool, error) {
	var exists bool
	err := r.DB.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM fyp_fuli_leave_applications
			WHERE staff_id = $1 AND deleted_at IS NULL
				AND status = 'approved'
				AND start_date <= $2::date AND end_date >= $2::date
		)
	`, staffID, date).Scan(&exists)
	return exists, err
}
