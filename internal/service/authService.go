package service

import (
	"context"
	"fmt"
	"fyp/config"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"os"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
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

func (s *authService) SignUp(ctx context.Context, signupParam param.SignUpParam) (*param.AuthResult, error) {
	if err := signupParam.ValidateSignUp(); err != nil {
		return nil, err
	}

	hashedPassword, err := s.HashPassword(signupParam.Password)
	if err != nil {
		return nil, err
	}
	signupParam.Password = hashedPassword

	user, err := s.authRepo.SignUp(ctx, signupParam)
	if err != nil {
		return nil, err
	}

	token, err := s.GenerateToken(user)
	if err != nil {
		return nil, err
	}

	return &param.AuthResult{
		Token: token,
		User:  user,
	}, nil
}

func (s *authService) GenerateToken(user *param.AuthUserParam) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", fmt.Errorf("JWT_SECRET is not set")
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(s.authConfig.JWTExpirationHours) * time.Hour)

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub": fmt.Sprintf(
				"%d",
				user.UserID,
			),
			"email":    user.Email,
			"username": user.Username,
			"iat":      now.Unix(),
			"exp":      expiresAt.Unix(),
		},
	)

	tokenString, err := token.SignedString(
		[]byte(secret),
	)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
