package interfaces

import (
	"context"
	"fyp/domain/param"
)

type IAuthRepo interface {
	SignUp(ctx context.Context, param param.SignUpParam) (*param.AuthUserParam, error)
	GetUser(ctx context.Context, email string) (*param.AuthUserParam, error)
	UpdateUserLogInStatus(ctx context.Context, param param.AuthUserParam) error
}

type IAuthService interface {
	SignUp(ctx context.Context, param param.SignUpParam) (*param.AuthResult, error)
	LogIn(ctx context.Context, logInParam param.LogInParam) (*param.AuthResult, error)
	GetUserProfile(ctx context.Context, email string) (*param.AuthUserParam, error)
}
