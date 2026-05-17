package repository

import (
	"context"
	"database/sql"
	"fyp/domain/param"
	"fyp/internal/interfaces"
)

type authRepo struct {
	DB *sql.DB
}

func NewAuthRepo(db *sql.DB) interfaces.IAuthRepo {
	return &authRepo{
		DB: db,
	}
}

func (a *authRepo) SignUp(ctx context.Context, param param.SignUpParam) error {
	_, err := a.DB.ExecContext(ctx, `
		INSERT INTO users (
			username,
			email,
			contact_number,
			password,
			failed_login_attempts,
			locked_until
		)
		VALUES ($1, $2, $3, $4, 0, NULL)
	`,
		param.Username, param.Email, param.ContactNumber, param.Password)
	return err
}
