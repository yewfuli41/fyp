package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"fyp/config"
	"fyp/database"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces"
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

func (s *authService) SignUp(ctx context.Context, signupParam param.SignUpParam) (*param.AuthResult, error) {
	if err := signupParam.ValidateSignUp(); err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(signupParam.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	signupParam.Password = string(hashedPassword)

	user, err := s.authRepo.SignUp(ctx, signupParam)
	if err != nil {
		if database.IsUniqueViolation(err, "users_email_key") {
			return nil, errs.ValidationErrors{
				{Field: "email", Message: "This email is already registered"},
			}
		}
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

func (s *authService) LogIn(ctx context.Context, logInParam param.LogInParam) (*param.AuthResult, error) {
	if err := logInParam.ValidateLogIn(); err != nil {
		return nil, err
	}

	user, err := s.authRepo.GetUser(ctx, logInParam.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.ValidationErrors{
				{Field: "email", Message: "Invalid email"},
			}
		}
		return nil, err
	}
	if user.LockedUntil != nil {
		if user.LockedUntil.After(time.Now()) {
			return nil, errs.LockedError{LockedUntil: *user.LockedUntil}
		}
		user.FailedLoginAttempts = 0
		user.LockedUntil = nil
		err := s.authRepo.UpdateUserLogInStatus(ctx, *user)
		if err != nil {
			return nil, err
		}
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(logInParam.Password))

	if err != nil {
		user.FailedLoginAttempts += 1
		if user.FailedLoginAttempts >= s.authConfig.MaxFailedLoginAttempts {
			lockedUntil := time.Now().Add(time.Duration(s.authConfig.LockDurationMinutes) * time.Minute)
			user.LockedUntil = &lockedUntil
		}
		err := s.authRepo.UpdateUserLogInStatus(ctx, *user)
		if err != nil {
			return nil, err
		}
		return nil, errs.ValidationErrors{
			{Field: "password", Message: "Wrong password"},
		}
	}
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	err = s.authRepo.UpdateUserLogInStatus(ctx, *user)
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

func (a *authService) GetUserProfile(ctx context.Context, email string) (*param.AuthUserParam, error) {
	user, err := a.authRepo.GetUser(ctx, email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *authService) GenerateToken(user *param.AuthUserParam) (string, error) {
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
		[]byte(s.authConfig.JWTSecret),
	)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
