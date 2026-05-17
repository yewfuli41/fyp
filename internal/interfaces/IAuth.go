package interfaces

import (
	"context"
	"fyp/domain/param"
)

type IAuthRepo interface {
	SignUp(ctx context.Context, param param.SignUpParam) error
}

type IAuthService interface {
	SignUp(ctx context.Context, param param.SignUpParam) error
}
