package repository

import (
	"context"
	"database/sql"
	"fyp/domain/param"
	"fyp/internal/interfaces"
)

type staffRepo struct {
	DB *sql.DB
}

func NewStaffRepo(db *sql.DB) interfaces.IStaffRepo {
	return &staffRepo{DB: db}
}

func (s *staffRepo) InsertStaff(ctx context.Context, tx *sql.Tx, p param.StaffParam) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO staff (user_id, business_id, staff_name, staff_contact_number, position)
		VALUES ($1, $2, $3, $4, $5)
	`, p.UserID, p.BusinessID, p.StaffName, p.StaffContactNumber, p.Position)
	return err
}
