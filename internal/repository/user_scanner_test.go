package repository_test

import (
	"database/sql"
	"fyp/internal/repository"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UserScanner", func() {
	var (
		db   *sql.DB
		mock sqlmock.Sqlmock
		cols []string
	)

	BeforeEach(func() {
		var err error
		db, mock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())
		cols = []string{"user_id", "username", "email", "contact_number", "password", "failed_login_attempts", "locked_until", "must_reset_password"}
	})

	Describe("ScanUser", func() {
		It("scans a fully populated user row", func() {
			lockedUntil := time.Now().Add(time.Hour).Truncate(time.Second).UTC()
			contactNumber := "1234567890"
			rows := sqlmock.NewRows(cols).
				AddRow(1, "testuser", "test@example.com", contactNumber, "password", 0, lockedUntil, true)

			mock.ExpectQuery("SELECT").WillReturnRows(rows)
			row := db.QueryRow("SELECT")

			user, err := repository.ScanUser(row)

			Expect(err).NotTo(HaveOccurred())
			Expect(user.UserID).To(Equal(int64(1)))
			Expect(user.Username).To(Equal("testuser"))
			Expect(user.Email).To(Equal("test@example.com"))
			Expect(*user.ContactNumber).To(Equal(contactNumber))
			Expect(user.Password).To(Equal("password"))
			Expect(user.FailedLoginAttempts).To(Equal(0))
			Expect(*user.LockedUntil).To(Equal(lockedUntil))
			Expect(user.MustResetPassword).To(BeTrue())
		})

		It("scans a user row with NULL optional fields", func() {
			rows := sqlmock.NewRows(cols).
				AddRow(1, "testuser", "test@example.com", nil, "password", 0, nil, false)

			mock.ExpectQuery("SELECT").WillReturnRows(rows)
			row := db.QueryRow("SELECT")

			user, err := repository.ScanUser(row)

			Expect(err).NotTo(HaveOccurred())
			Expect(user.ContactNumber).To(BeNil())
			Expect(user.LockedUntil).To(BeNil())
		})

		It("returns an error if scanning fails", func() {
			rows := sqlmock.NewRows(cols).
				AddRow(1, "testuser", "test@example.com", "1234567890", "password", 0, "invalid-time", false)

			mock.ExpectQuery("SELECT").WillReturnRows(rows)
			row := db.QueryRow("SELECT")

			user, err := repository.ScanUser(row)

			Expect(err).To(HaveOccurred())
			Expect(user).To(BeNil())
		})

		It("returns an error if no rows are found", func() {
			mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)
			row := db.QueryRow("SELECT")

			user, err := repository.ScanUser(row)

			Expect(err).To(Equal(sql.ErrNoRows))
			Expect(user).To(BeNil())
		})
	})
})
