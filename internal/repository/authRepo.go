package repository

import (
	"context"
	"database/sql"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"time"
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
			failed_login_attempts,
			locked_until
	`,
		param.Username, param.Email, param.ContactNumber, param.Password)

	user, err := scanUser(row)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (a *authRepo) GetAllUsers(ctx context.Context, param param.LogInParam) (*param.AuthUserParam, error) {
	rows, err := a.DB.ExecContext(ctx, `SELECT * FROM users`)
	var users []string

}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (*param.AuthUserParam, error) {
	var user param.AuthUserParam
	var contactNumber sql.NullString
	var lockedUntil sql.NullTime

	if err := row.Scan(
		&user.UserID,
		&user.Username,
		&user.Email,
		&contactNumber,
		&user.FailedLoginAttempts,
		&lockedUntil,
	); err != nil {
		return nil, err
	}

	if contactNumber.Valid {
		user.ContactNumber = &contactNumber.String
	}
	if lockedUntil.Valid {
		lockedUntilValue := lockedUntil.Time.UTC().Truncate(time.Second)
		user.LockedUntil = &lockedUntilValue
	}

	return &user, nil
}
