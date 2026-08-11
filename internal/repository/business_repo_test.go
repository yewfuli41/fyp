package repository_test

import (
	"context"
	"database/sql"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/internal/repository"
	"regexp"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BusinessRepo", func() {
	var (
		db           *sql.DB
		mock         sqlmock.Sqlmock
		repo         interfaces.IBusinessRepo
		ctx          context.Context
		businessCols []string
	)

	BeforeEach(func() {
		var err error
		db, mock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		repo = repository.NewBusinessRepo(db)
		ctx = context.Background()
		businessCols = []string{
			"business_id", "owner_user_id", "business_name", "description",
			"address", "image_url", "business_contact_number", "business_email",
		}
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("InsertBusinessProfile", func() {
		It("inserts a new business profile using the provided transaction", func() {
			desc := "A great business"
			businessParam := param.BusinessProfileParam{
				OwnerUserID:  1,
				BusinessName: "Test Biz",
				Description:  &desc,
			}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO business_profiles")).
				WithArgs(
					businessParam.OwnerUserID,
					businessParam.BusinessName,
					businessParam.Description,
					businessParam.Address,
					businessParam.ImageURL,
					businessParam.BusinessContactNumber,
					businessParam.BusinessEmail,
				).
				WillReturnRows(sqlmock.NewRows(businessCols).
					AddRow(7, 1, "Test Biz", desc, nil, nil, nil, nil))

			result, err := repo.InsertBusinessProfile(ctx, tx, businessParam)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.BusinessID).To(Equal(int64(7)))
		})
	})

	Describe("InsertBusinessWorkingHours", func() {
		It("inserts working hours using the provided transaction", func() {
			startTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
			endTime := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

			businessParam := param.BusinessProfileParam{
				BusinessID: 7,
				WorkingHours: []param.WorkingHourParam{
					{Day: "MONDAY", StartTime: startTime, EndTime: endTime},
					{Day: "TUESDAY", StartTime: startTime, EndTime: endTime},
				},
			}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			for _, wh := range businessParam.WorkingHours {
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO business_working_hours")).
					WithArgs(
						int64(7),
						wh.Day,
						wh.StartTime.Format("15:04:05"),
						wh.EndTime.Format("15:04:05"),
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			err := repo.InsertBusinessWorkingHours(ctx, tx, businessParam)

			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("UpdateBusinessProfile", func() {
		It("updates and returns the business profile", func() {
			p := param.BusinessProfileParam{
				OwnerUserID:           1,
				BusinessName:          "Updated Biz",
				Address:               "456 Ave",
				BusinessContactNumber: "0123456789",
				BusinessEmail:         "updated@example.com",
			}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("UPDATE business_profiles")).
				WithArgs(
					p.BusinessName,
					p.Description,
					p.Address,
					p.ImageURL,
					p.BusinessContactNumber,
					p.BusinessEmail,
					p.OwnerUserID,
				).
				WillReturnRows(sqlmock.NewRows(businessCols).
					AddRow(7, 1, "Updated Biz", nil, "456 Ave", nil, "0123456789", "updated@example.com"))

			result, err := repo.UpdateBusinessProfile(ctx, tx, p)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.BusinessID).To(Equal(int64(7)))
			Expect(result.BusinessName).To(Equal("Updated Biz"))
		})
	})

	Describe("DeleteBusinessWorkingHours", func() {
		It("deletes working hours for a business", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("DELETE FROM business_working_hours")).
				WithArgs(int64(7)).
				WillReturnResult(sqlmock.NewResult(1, 2))

			err := repo.DeleteBusinessWorkingHours(ctx, tx, 7)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetBusinessProfileByOwnerID", func() {
		It("returns the business profile for the given owner", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows(businessCols).
					AddRow(7, 1, "My Biz", nil, "123 St", nil, "0123456789", "biz@example.com"))

			result, err := repo.GetBusinessProfileByOwnerID(ctx, 1)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.BusinessID).To(Equal(int64(7)))
			Expect(result.BusinessName).To(Equal("My Biz"))
		})
	})

	Describe("BusinessEmailExists", func() {
		It("returns true when the business email already exists", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM business_profiles WHERE business_email = $1)")).
				WithArgs("taken@example.com").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

			exists, err := repo.BusinessEmailExists(ctx, "taken@example.com")

			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("returns false when the business email does not exist", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM business_profiles WHERE business_email = $1)")).
				WithArgs("free@example.com").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

			exists, err := repo.BusinessEmailExists(ctx, "free@example.com")

			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("propagates a DB error instead of reporting the email as free", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM business_profiles WHERE business_email = $1)")).
				WithArgs("error@example.com").
				WillReturnError(sql.ErrConnDone)

			exists, err := repo.BusinessEmailExists(ctx, "error@example.com")

			Expect(err).To(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})

	Describe("GetBusinesses", func() {
		It("returns all matching business profiles", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT business_id, owner_user_id, business_name, description, address")).
				WillReturnRows(sqlmock.NewRows(businessCols).
					AddRow(1, 1, "Alpha Biz", nil, "1 St", nil, "0111111111", "alpha@example.com").
					AddRow(2, 2, "Beta Biz", nil, "2 St", nil, "0222222222", "beta@example.com"))

			results, err := repo.GetBusinesses(ctx, "")

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].BusinessName).To(Equal("Alpha Biz"))
			Expect(results[1].BusinessName).To(Equal("Beta Biz"))
		})

		It("returns an empty slice when no business matches the search", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT business_id, owner_user_id, business_name, description, address")).
				WithArgs("%nonexistent%").
				WillReturnRows(sqlmock.NewRows(businessCols))

			results, err := repo.GetBusinesses(ctx, "nonexistent")

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})
	})

	Describe("GetBusinessByID", func() {
		It("returns the business profile when found", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT business_id, owner_user_id, business_name, description, address")).
				WithArgs(int64(7)).
				WillReturnRows(sqlmock.NewRows(businessCols).
					AddRow(7, 1, "My Biz", nil, "123 St", nil, "0123456789", "biz@example.com"))

			result, err := repo.GetBusinessByID(ctx, 7)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.BusinessID).To(Equal(int64(7)))
			Expect(result.BusinessName).To(Equal("My Biz"))
		})

		It("returns sql.ErrNoRows when the business is not found", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT business_id, owner_user_id, business_name, description, address")).
				WithArgs(int64(999)).
				WillReturnError(sql.ErrNoRows)

			result, err := repo.GetBusinessByID(ctx, 999)

			Expect(err).To(Equal(sql.ErrNoRows))
			Expect(result).To(BeNil())
		})
	})

	Describe("GetBusinessWorkingHours", func() {
		It("returns working hours for a business", func() {
			whCols := []string{"day", "start_time", "end_time"}
			startTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
			endTime := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

			mock.ExpectQuery(regexp.QuoteMeta("SELECT")).
				WithArgs(int64(7)).
				WillReturnRows(sqlmock.NewRows(whCols).
					AddRow("MONDAY", startTime, endTime).
					AddRow("TUESDAY", startTime, endTime))

			results, err := repo.GetBusinessWorkingHours(ctx, 7)

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].Day).To(Equal("MONDAY"))
		})
	})
})
