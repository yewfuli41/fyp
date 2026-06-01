package service_test

import (
	"context"
	"database/sql"
	"errors"
	"fyp/config"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces/mocks"
	"fyp/internal/service"
	"os"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

var _ = Describe("AuthService", func() {
	var (
		ctx        context.Context
		authRepo   *mocks.MockIAuthRepo
		authConfig config.AuthConfig
		authSvc    interface {
			SignUp(context.Context, param.SignUpParam) (*param.AuthResult, error)
			LogIn(context.Context, param.LogInParam) (*param.AuthResult, error)
		}
	)

	BeforeEach(func() {
		ctx = context.Background()
		authRepo = mocks.NewMockIAuthRepo(GinkgoT())
		authConfig = config.AuthConfig{
			MaxFailedLoginAttempts: 3,
			LockDurationMinutes:    15,
			JWTExpirationHours:     2,
		}
		authSvc = service.NewAuthService(authRepo, authConfig)
		setenv("JWT_SECRET", "test-secret")
	})

	Describe("SignUp", func() {
		It("hashes the password, creates the user, and returns a signed token", func() {
			signUpParam := param.SignUpParam{
				Username:      "finn",
				Email:         "finn@example.com",
				ContactNumber: "0123456789",
				Password:      "password123",
			}
			createdUser := &param.AuthUserParam{
				UserID:        42,
				Username:      signUpParam.Username,
				Email:         signUpParam.Email,
				ContactNumber: &signUpParam.ContactNumber,
			}

			authRepo.EXPECT().
				SignUp(ctx, mock.MatchedBy(func(received param.SignUpParam) bool {
					Expect(received.Username).To(Equal(signUpParam.Username))
					Expect(received.Email).To(Equal(signUpParam.Email))
					Expect(received.ContactNumber).To(Equal(signUpParam.ContactNumber))
					Expect(received.Password).NotTo(Equal(signUpParam.Password))
					Expect(bcrypt.CompareHashAndPassword([]byte(received.Password), []byte(signUpParam.Password))).To(Succeed())
					return true
				})).
				Return(createdUser, nil).
				Once()

			result, err := authSvc.SignUp(ctx, signUpParam)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.User).To(Equal(createdUser))
			expectTokenClaims(result.Token, createdUser, authConfig.JWTExpirationHours)
		})

		It("returns validation errors without calling the repo", func() {
			result, err := authSvc.SignUp(ctx, param.SignUpParam{})

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("validation failed"))
			Expect(err).To(BeAssignableToTypeOf(errs.ValidationErrors{}))
		})

		It("maps duplicate email errors to validation errors", func() {
			signUpParam := param.SignUpParam{
				Username:      "finn",
				Email:         "finn@example.com",
				ContactNumber: "0123456789",
				Password:      "password123",
			}
			duplicateEmailErr := newUniqueViolation("users_email_key")

			authRepo.EXPECT().
				SignUp(ctx, mock.AnythingOfType("param.SignUpParam")).
				Return(nil, duplicateEmailErr).
				Once()

			result, err := authSvc.SignUp(ctx, signUpParam)

			Expect(result).To(BeNil())
			Expect(err).To(Equal(errs.ValidationErrors{
				{Field: "email", Message: "This email is already registered"},
			}))
		})

		It("returns an error when the repo SignUp fails generically", func() {
			signUpParam := param.SignUpParam{
				Username:      "finn",
				Email:         "finn@example.com",
				ContactNumber: "0123456789",
				Password:      "password123",
			}
			dbErr := errors.New("database connection failed")

			authRepo.EXPECT().
				SignUp(ctx, mock.AnythingOfType("param.SignUpParam")).
				Return(nil, dbErr).
				Once()

			result, err := authSvc.SignUp(ctx, signUpParam)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError(dbErr))
		})

		It("returns an error when JWT_SECRET is missing", func() {
			signUpParam := param.SignUpParam{
				Username:      "finn",
				Email:         "finn@example.com",
				ContactNumber: "0123456789",
				Password:      "password123",
			}
			createdUser := &param.AuthUserParam{
				UserID:   42,
				Username: signUpParam.Username,
				Email:    signUpParam.Email,
			}
			setenv("JWT_SECRET", "")

			authRepo.EXPECT().
				SignUp(ctx, mock.AnythingOfType("param.SignUpParam")).
				Return(createdUser, nil).
				Once()

			result, err := authSvc.SignUp(ctx, signUpParam)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("JWT_SECRET is not set"))
		})
	})

	Describe("LogIn", func() {
		var (
			logInParam param.LogInParam
			user       *param.AuthUserParam
		)

		BeforeEach(func() {
			logInParam = param.LogInParam{
				Email:    "finn@example.com",
				Password: "password123",
			}
			user = &param.AuthUserParam{
				UserID:              42,
				Username:            "finn",
				Email:               logInParam.Email,
				Password:            hashPassword(logInParam.Password),
				FailedLoginAttempts: 2,
			}
		})

		It("returns validation errors without calling the repo", func() {
			result, err := authSvc.LogIn(ctx, param.LogInParam{})

			Expect(result).To(BeNil())
			Expect(err).To(MatchError("validation failed"))
			Expect(err).To(BeAssignableToTypeOf(errs.ValidationErrors{}))
		})

		It("returns invalid email when the user does not exist", func() {
			authRepo.EXPECT().
				GetUser(ctx, logInParam).
				Return(nil, sql.ErrNoRows).
				Once()

			result, err := authSvc.LogIn(ctx, logInParam)

			Expect(result).To(BeNil())
			Expect(err).To(Equal(errs.ValidationErrors{
				{Field: "email", Message: "Invalid email"},
			}))
		})

		It("returns an error when the repo GetUser fails generically", func() {
			dbErr := errors.New("database connection failed")
			authRepo.EXPECT().
				GetUser(ctx, logInParam).
				Return(nil, dbErr).
				Once()

			result, err := authSvc.LogIn(ctx, logInParam)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError(dbErr))
		})

		It("resets failed login state and returns a signed token for a valid password", func() {
			authRepo.EXPECT().
				GetUser(ctx, logInParam).
				Return(user, nil).
				Once()
			authRepo.EXPECT().
				UpdateUserLogInStatus(ctx, mock.MatchedBy(func(received param.AuthUserParam) bool {
					Expect(received.FailedLoginAttempts).To(Equal(0))
					Expect(received.LockedUntil).To(BeNil())
					return received.UserID == user.UserID
				})).
				Return(nil).
				Once()

			result, err := authSvc.LogIn(ctx, logInParam)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.User).To(Equal(user))
			Expect(user.FailedLoginAttempts).To(Equal(0))
			Expect(user.LockedUntil).To(BeNil())
			expectTokenClaims(result.Token, user, authConfig.JWTExpirationHours)
		})

		It("increments failed attempts for an invalid password", func() {
			logInParam.Password = "wrong-password"

			authRepo.EXPECT().
				GetUser(ctx, logInParam).
				Return(user, nil).
				Once()
			authRepo.EXPECT().
				UpdateUserLogInStatus(ctx, mock.MatchedBy(func(received param.AuthUserParam) bool {
					Expect(received.FailedLoginAttempts).To(Equal(3))
					Expect(received.LockedUntil).NotTo(BeNil())
					Expect(*received.LockedUntil).To(BeTemporally("~", time.Now().Add(15*time.Minute), 3*time.Second))
					return received.UserID == user.UserID
				})).
				Return(nil).
				Once()

			result, err := authSvc.LogIn(ctx, logInParam)

			Expect(result).To(BeNil())
			Expect(err).To(Equal(errs.ValidationErrors{
				{Field: "password", Message: "Wrong password"},
			}))
		})

		It("returns a locked error when the lockout has not expired", func() {
			lockedUntil := time.Now().Add(10 * time.Minute)
			user.LockedUntil = &lockedUntil

			authRepo.EXPECT().
				GetUser(ctx, logInParam).
				Return(user, nil).
				Once()

			result, err := authSvc.LogIn(ctx, logInParam)

			Expect(result).To(BeNil())
			Expect(err).To(Equal(errs.LockedError{LockedUntil: lockedUntil}))
		})

		It("clears an expired lockout before validating the password", func() {
			expiredLockUntil := time.Now().Add(-time.Minute)
			user.LockedUntil = &expiredLockUntil
			user.FailedLoginAttempts = 3

			authRepo.EXPECT().
				GetUser(ctx, logInParam).
				Return(user, nil).
				Once()
			authRepo.EXPECT().
				UpdateUserLogInStatus(ctx, mock.MatchedBy(func(received param.AuthUserParam) bool {
					Expect(received.FailedLoginAttempts).To(Equal(0))
					Expect(received.LockedUntil).To(BeNil())
					return received.UserID == user.UserID
				})).
				Return(nil).
				Once()
			authRepo.EXPECT().
				UpdateUserLogInStatus(ctx, mock.MatchedBy(func(received param.AuthUserParam) bool {
					Expect(received.FailedLoginAttempts).To(Equal(0))
					Expect(received.LockedUntil).To(BeNil())
					return received.UserID == user.UserID
				})).
				Return(nil).
				Once()

			result, err := authSvc.LogIn(ctx, logInParam)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.User).To(Equal(user))
			expectTokenClaims(result.Token, user, authConfig.JWTExpirationHours)
		})

		It("returns repo errors from login status updates", func() {
			updateErr := errors.New("update failed")

			authRepo.EXPECT().
				GetUser(ctx, logInParam).
				Return(user, nil).
				Once()
			authRepo.EXPECT().
				UpdateUserLogInStatus(ctx, mock.AnythingOfType("param.AuthUserParam")).
				Return(updateErr).
				Once()

			result, err := authSvc.LogIn(ctx, logInParam)

			Expect(result).To(BeNil())
			Expect(err).To(MatchError(updateErr))
		})
	})
})

func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	Expect(err).NotTo(HaveOccurred())
	return string(hash)
}

func expectTokenClaims(tokenString string, user *param.AuthUserParam, expiresInHours int) {
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		Expect(token.Method).To(Equal(jwt.SigningMethodHS256))
		return []byte("test-secret"), nil
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(parsedToken.Valid).To(BeTrue())

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	Expect(ok).To(BeTrue())
	Expect(claims["sub"]).To(Equal("42"))
	Expect(claims["email"]).To(Equal(user.Email))
	Expect(claims["username"]).To(Equal(user.Username))
	Expect(claims["iat"]).To(BeNumerically("~", float64(time.Now().Unix()), 3))
	Expect(claims["exp"]).To(BeNumerically("~", float64(time.Now().Add(time.Duration(expiresInHours)*time.Hour).Unix()), 3))
}

func newUniqueViolation(constraint string) error {
	return &pq.Error{
		Code:       "23505",
		Constraint: constraint,
	}
}

func setenv(key, value string) {
	originalValue, hadOriginalValue := os.LookupEnv(key)
	Expect(os.Setenv(key, value)).To(Succeed())

	DeferCleanup(func() {
		if hadOriginalValue {
			Expect(os.Setenv(key, originalValue)).To(Succeed())
			return
		}
		Expect(os.Unsetenv(key)).To(Succeed())
	})
}
