package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/internal/repository"
	"regexp"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AuthRepo", func() {
	var (
		db       *sql.DB
		mock     sqlmock.Sqlmock
		repo     interfaces.IAuthRepo
		ctx      context.Context
		userCols []string
	)

	BeforeEach(func() {
		var err error
		db, mock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		repo = repository.NewAuthRepo(db)
		ctx = context.Background()
		userCols = []string{"user_id", "username", "email", "contact_number", "password", "failed_login_attempts", "locked_until"}
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("SignUp", func() {
		It("inserts a new user and returns the created user", func() {
			signUpParam := param.SignUpParam{
				Username:      "testuser",
				Email:         "test@example.com",
				ContactNumber: "12345678",
				Password:      "hashedpassword",
			}

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO users")).
				WithArgs(signUpParam.Username, signUpParam.Email, signUpParam.ContactNumber, signUpParam.Password).
				WillReturnRows(sqlmock.NewRows(userCols).
					AddRow(1, signUpParam.Username, signUpParam.Email, signUpParam.ContactNumber, signUpParam.Password, 0, nil))

			user, err := repo.SignUp(ctx, signUpParam)

			Expect(err).NotTo(HaveOccurred())
			Expect(user.UserID).To(Equal(int64(1)))
			Expect(user.Username).To(Equal(signUpParam.Username))
			Expect(user.Email).To(Equal(signUpParam.Email))
			Expect(*user.ContactNumber).To(Equal(signUpParam.ContactNumber))
		})

		It("returns an error if the insertion fails", func() {
			signUpParam := param.SignUpParam{
				Username: "testuser",
				Email:    "test@example.com",
			}

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO users")).
				WithArgs(signUpParam.Username, signUpParam.Email, signUpParam.ContactNumber, signUpParam.Password).
				WillReturnError(errors.New("db error"))

			user, err := repo.SignUp(ctx, signUpParam)

			Expect(err).To(HaveOccurred())
			Expect(user).To(BeNil())
			Expect(err.Error()).To(Equal("db error"))
		})
	})

	Describe("GetUser", func() {
		It("returns the user if found by email", func() {
			logInParam := param.LogInParam{
				Email: "test@example.com",
			}

			mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM users WHERE email = $1")).
				WithArgs(logInParam.Email).
				WillReturnRows(sqlmock.NewRows(userCols).
					AddRow(1, "testuser", logInParam.Email, "12345678", "hashedpassword", 0, nil))

			user, err := repo.GetUser(ctx, logInParam.Email)

			Expect(err).NotTo(HaveOccurred())
			Expect(user.Email).To(Equal(logInParam.Email))
			Expect(*user.ContactNumber).To(Equal("12345678"))
			Expect(user.LockedUntil).To(BeNil())
		})

		It("returns the user with all fields populated", func() {
			logInParam := param.LogInParam{
				Email: "test@example.com",
			}
			lockedUntil := time.Now().Add(time.Hour).Truncate(time.Second).UTC()

			mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM users WHERE email = $1")).
				WithArgs(logInParam.Email).
				WillReturnRows(sqlmock.NewRows(userCols).
					AddRow(1, "testuser", logInParam.Email, "12345678", "hashedpassword", 3, lockedUntil))

			user, err := repo.GetUser(ctx, logInParam.Email)

			Expect(err).NotTo(HaveOccurred())
			Expect(user.Email).To(Equal(logInParam.Email))
			Expect(*user.ContactNumber).To(Equal("12345678"))
			Expect(user.FailedLoginAttempts).To(Equal(3))
			Expect(*user.LockedUntil).To(Equal(lockedUntil))
		})

		It("returns sql.ErrNoRows if user is not found", func() {
			logInParam := param.LogInParam{
				Email: "notfound@example.com",
			}

			mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM users WHERE email = $1")).
				WithArgs(logInParam.Email).
				WillReturnError(sql.ErrNoRows)

			user, err := repo.GetUser(ctx, logInParam.Email)

			Expect(err).To(Equal(sql.ErrNoRows))
			Expect(user).To(BeNil())
		})
	})

	Describe("UpdateUserLogInStatus", func() {
		It("updates failed login attempts and locked_until", func() {
			lockedUntil := time.Now().Add(time.Minute).Truncate(time.Second)
			userParam := param.AuthUserParam{
				UserID:              1,
				FailedLoginAttempts: 3,
				LockedUntil:         &lockedUntil,
			}

			mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET failed_login_attempts = $1, locked_until = $2 WHERE user_id = $3")).
				WithArgs(userParam.FailedLoginAttempts, userParam.LockedUntil, userParam.UserID).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err := repo.UpdateUserLogInStatus(ctx, userParam)

			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an error if update fails", func() {
			userParam := param.AuthUserParam{
				UserID: 1,
			}

			mock.ExpectExec(regexp.QuoteMeta("UPDATE users")).
				WillReturnError(errors.New("update error"))

			err := repo.UpdateUserLogInStatus(ctx, userParam)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("update error"))
		})
	})
})
