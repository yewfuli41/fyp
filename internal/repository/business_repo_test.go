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
			startTime, _ := time.Parse("15:04", "09:00")
			endTime, _ := time.Parse("15:04", "18:00")

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
})
