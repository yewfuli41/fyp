package repository

import (
	"context"
	"database/sql"
	"fyp/domain/param"
	"fyp/internal/interfaces"
)

type profileRepo struct {
	DB *sql.DB
}

func NewProfileRepo(db *sql.DB) interfaces.IProfileRepo {
	return &profileRepo{
		DB: db,
	}
}

func (p *profileRepo) UpdateUser(ctx context.Context, param param.ProfileParam) error {
	_, err := p.DB.ExecContext(ctx, `UPDATE users SET username = $1, email = $2, contact_number = $3 WHERE user_id = $4`, param.Username, param.Email, param.ContactNumber, param.UserId)
	return err
}

func (p *profileRepo) GetPassword(ctx context.Context)
func (p *profileRepo) ChangePassword(ctx context.Context)
