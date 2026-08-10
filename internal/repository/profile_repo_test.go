package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/internal/repository"
	"regexp"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ProfileRepo", func() {
	var (
		db   *sql.DB
		mock sqlmock.Sqlmock
		repo interfaces.IProfileRepo
		ctx  context.Context
	)

	BeforeEach(func() {
		var err error
		db, mock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		repo = repository.NewProfileRepo(db)
		ctx = context.Background()
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("UpdateUser", func() {
		It("updates user information successfully", func() {
			profileParam := param.ProfileParam{
				UserId:        1,
				Username:      "updateduser",
				Email:         "updated@example.com",
				ContactNumber: "0123456789",
			}

			mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET username = $1, email = $2, contact_number = $3 WHERE user_id = $4")).
				WithArgs(profileParam.Username, profileParam.Email, profileParam.ContactNumber, profileParam.UserId).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err := repo.UpdateUser(ctx, profileParam)

			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an error if the update fails", func() {
			profileParam := param.ProfileParam{
				UserId: 1,
			}

			mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET")).
				WillReturnError(errors.New("db error"))

			err := repo.UpdateUser(ctx, profileParam)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("db error"))
		})
	})
})
