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

func (a *authRepo) SignUp(ctx context.Context, param param.SignUpParam) (*param.AuthUserParam, error) {
	row := a.DB.QueryRowContext(ctx, `
		INSERT INTO users (
			username,
			email,
			contact_number,
			password,
			failed_login_attempts,
			locked_until
		)
		VALUES ($1, $2, $3, $4, 0, NULL)
		RETURNING
			user_id,
			username,
			email,
			contact_number,
			password,
			failed_login_attempts,
			locked_until
	`,
		param.Username, param.Email, param.ContactNumber, param.Password)

	user, err := ScanUser(row)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (a *authRepo) GetUser(ctx context.Context, email string) (*param.AuthUserParam, error) {
	row := a.DB.QueryRowContext(ctx, `	
		SELECT * FROM users WHERE email = $1
	`, email)
	user, err := ScanUser(row)
	if err != nil {
		return nil, err
	}

	return user, nil

}

func (a *authRepo) UpdateUserLogInStatus(ctx context.Context, param param.AuthUserParam) error {
	_, err := a.DB.ExecContext(ctx, `UPDATE users SET failed_login_attempts = $1, locked_until = $2 WHERE user_id = $3`, param.FailedLoginAttempts, param.LockedUntil, param.UserID)
	return err
}
