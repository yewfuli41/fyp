package service

import (
	"context"
	"fyp/config"
	"fyp/domain/param"
	"fyp/internal/interfaces"

	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	authRepo   interfaces.IAuthRepo
	authConfig config.AuthConfig
}

func NewAuthService(authRepo interfaces.IAuthRepo, authConfig config.AuthConfig) interfaces.IAuthService {
	return &authService{
		authRepo:   authRepo,
		authConfig: authConfig,
	}
}

func (s *authService) HashPassword(password string) (string, error) {
	hashedPassword, err :=
		bcrypt.GenerateFromPassword(
			[]byte(password),
			bcrypt.DefaultCost,
		)

	if err != nil {
		return "", err
	}

	return string(
		hashedPassword,
	), nil
}

func (s *authService) SignUp(ctx context.Context, param param.SignUpParam) error {
	if err := param.Validate(); err != nil {
		return err
	}

	hashedPassword, err := s.HashPassword(param.Password)
	if err != nil {
		return err
	}
	param.Password = hashedPassword

	return s.authRepo.SignUp(ctx, param)
}
