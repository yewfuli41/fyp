package interfaces

import (
	"context"
	"database/sql"
	"fyp/domain/param"
)

type IAuthRepo interface {
	SignUp(ctx context.Context, param param.SignUpParam) (*param.AuthUserParam, error)
	SignUpTx(ctx context.Context, tx *sql.Tx, param param.SignUpParam) (*param.AuthUserParam, error)
	GetUser(ctx context.Context, email string) (*param.AuthUserParam, error)
	UpdateUserLogInStatus(ctx context.Context, param param.AuthUserParam) error
	UpdatePassword(ctx context.Context, userID int64, hashedPassword string) error
	UpdateUserEmailTx(ctx context.Context, tx *sql.Tx, userID int64, email string) error
}

type IAuthService interface {
	SignUp(ctx context.Context, param param.SignUpParam) (*param.AuthResult, error)
	LogIn(ctx context.Context, logInParam param.LogInParam) (*param.AuthResult, error)
	GetUserProfile(ctx context.Context, email string) (*param.AuthUserParam, error)
	ResetPassword(ctx context.Context, resetPasswordParam param.ResetPasswordParam) error
}
