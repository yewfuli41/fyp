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
			locked_until,
			must_reset_password
		)
		VALUES ($1, $2, $3, $4, 0, NULL, $5)
		RETURNING
			user_id,
			username,
			email,
			contact_number,
			password,
			failed_login_attempts,
			locked_until,
			must_reset_password
	`,
		param.Username, param.Email, param.ContactNumber, param.Password, param.MustResetPassword)

	user, err := ScanUser(row)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (a *authRepo) SignUpTx(ctx context.Context, tx *sql.Tx, param param.SignUpParam) (*param.AuthUserParam, error) {
	row := tx.QueryRowContext(ctx, `
		INSERT INTO users (
			username,
			email,
			contact_number,
			password,
			failed_login_attempts,
			locked_until,
			must_reset_password
		)
		VALUES ($1, $2, $3, $4, 0, NULL, $5)
		RETURNING
			user_id,
			username,
			email,
			contact_number,
			password,
			failed_login_attempts,
			locked_until,
			must_reset_password
	`,
		param.Username, param.Email, param.ContactNumber, param.Password, param.MustResetPassword)

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

func (a *authRepo) UpdatePassword(ctx context.Context, userID int64, hashedPassword string) error {
	_, err := a.DB.ExecContext(ctx, `UPDATE users SET password = $1, must_reset_password = FALSE WHERE user_id = $2`, hashedPassword, userID)
	return err
}

func (a *authRepo) UpdateUserEmailTx(ctx context.Context, tx *sql.Tx, userID int64, email string) error {
	_, err := tx.ExecContext(ctx, `UPDATE users SET email = $1 WHERE user_id = $2`, email, userID)
	return err
}
