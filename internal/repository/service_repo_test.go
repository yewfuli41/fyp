package repository_test

import (
	"context"
	"database/sql"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/internal/repository"
	"regexp"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServiceRepo", func() {
	var (
		db          *sql.DB
		mock        sqlmock.Sqlmock
		repo        interfaces.IServiceRepo
		ctx         context.Context
		serviceCols []string
		pkgCols     []string
	)

	BeforeEach(func() {
		var err error
		db, mock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		repo = repository.NewServiceRepo(db)
		ctx = context.Background()
		serviceCols = []string{"service_id", "business_id", "service_name", "description"}
		pkgCols = []string{"service_option_id", "service_id", "service_option_name", "description"}
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("InsertService", func() {
		It("inserts a new service", func() {
			p := param.ServiceParam{
				BusinessID:  1,
				ServiceName: "Massage",
			}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO services")).
				WithArgs(p.BusinessID, p.ServiceName, p.Description).
				WillReturnRows(sqlmock.NewRows(serviceCols).
					AddRow(10, 1, "Massage", nil))

			result, err := repo.InsertService(ctx, tx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceID).To(Equal(int64(10)))
		})
	})

	Describe("InsertServiceOption", func() {
		It("inserts a new service package", func() {
			p := param.ServiceOptionParam{
				ServiceID:          10,
				ServiceOptionName: "Deep Tissue",
			}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO service_options")).
				WithArgs(p.ServiceID, p.ServiceOptionName, p.Description).
				WillReturnRows(sqlmock.NewRows(pkgCols).
					AddRow(20, 10, "Deep Tissue", nil))

			result, err := repo.InsertServiceOption(ctx, tx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceOptionID).To(Equal(int64(20)))
		})
	})

	Describe("InsertServiceOptionItem", func() {
		It("inserts a new package item", func() {
			p := param.ServiceOptionItemParam{
				ServiceOptionID: 20,
				ServiceOptionItemName:  "Oil",
			}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("INSERT INTO service_option_items")).
				WithArgs(p.ServiceOptionID, p.ServiceOptionItemName).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err := repo.InsertServiceOptionItem(ctx, tx, p)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("UpdateService", func() {
		It("updates an existing service", func() {
			p := param.ServiceParam{
				ServiceID:   10,
				BusinessID:  1,
				ServiceName: "Updated Massage",
			}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("UPDATE services")).
				WithArgs(p.ServiceName, p.Description, p.ServiceID, p.BusinessID).
				WillReturnRows(sqlmock.NewRows(serviceCols).
					AddRow(10, 1, "Updated Massage", nil))

			result, err := repo.UpdateService(ctx, tx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceName).To(Equal("Updated Massage"))
		})
	})

	Describe("SoftDeleteService", func() {
		It("soft deletes a service", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE services SET deleted_at = NOW()")).
				WithArgs(int64(10), int64(1)).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err := repo.SoftDeleteService(ctx, tx, 10, 1)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("DeleteServiceOptionsByServiceID", func() {
		It("soft deletes packages and items for a service", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE service_option_items SET deleted_at = NOW()")).
				WithArgs(int64(10)).
				WillReturnResult(sqlmock.NewResult(1, 1))

			mock.ExpectExec(regexp.QuoteMeta("UPDATE service_options SET deleted_at = NOW()")).
				WithArgs(int64(10)).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err := repo.DeleteServiceOptionsByServiceID(ctx, tx, 10)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetServicesByBusinessID", func() {
		It("returns services for a business", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT service_id, business_id, service_name, description FROM services")).
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows(serviceCols).
					AddRow(10, 1, "Massage", nil).
					AddRow(11, 1, "Facial", nil))

			results, err := repo.GetServicesByBusinessID(ctx, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].ServiceID).To(Equal(int64(10)))
			Expect(results[1].ServiceID).To(Equal(int64(11)))
		})
	})

	Describe("GetServiceOptionsByServiceID", func() {
		It("returns packages for a service", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT service_option_id, service_id, service_option_name, description FROM service_options")).
				WithArgs(int64(10)).
				WillReturnRows(sqlmock.NewRows(pkgCols).
					AddRow(20, 10, "Deep Tissue", nil).
					AddRow(21, 10, "Swedish", nil))

			results, err := repo.GetServiceOptionsByServiceID(ctx, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].ServiceOptionID).To(Equal(int64(20)))
		})

		It("returns empty slice when no packages exist", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT service_option_id, service_id, service_option_name, description FROM service_options")).
				WithArgs(int64(99)).
				WillReturnRows(sqlmock.NewRows(pkgCols))

			results, err := repo.GetServiceOptionsByServiceID(ctx, 99)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})
	})

	Describe("GetServiceOptionItemsByOptionID", func() {
		It("returns items for a package", func() {
			itemCols := []string{"service_option_item_id", "service_option_id", "service_option_item_name"}

			mock.ExpectQuery(regexp.QuoteMeta("SELECT service_option_item_id, service_option_id, service_option_item_name FROM service_option_items")).
				WithArgs(int64(20)).
				WillReturnRows(sqlmock.NewRows(itemCols).
					AddRow(1, 20, "Oil").
					AddRow(2, 20, "Lotion"))

			results, err := repo.GetServiceOptionItemsByOptionID(ctx, 20)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].ServiceOptionItemName).To(Equal("Oil"))
		})

		It("returns empty slice when no items exist", func() {
			itemCols := []string{"service_option_item_id", "service_option_id", "service_option_item_name"}

			mock.ExpectQuery(regexp.QuoteMeta("SELECT service_option_item_id, service_option_id, service_option_item_name FROM service_option_items")).
				WithArgs(int64(99)).
				WillReturnRows(sqlmock.NewRows(itemCols))

			results, err := repo.GetServiceOptionItemsByOptionID(ctx, 99)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})
	})

	Describe("HasBookingForService", func() {
		It("returns true if bookings exist", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*)")).
				WithArgs(int64(10)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

			hasBooking, err := repo.HasBookingForService(ctx, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasBooking).To(BeTrue())
		})

		It("returns false if no bookings exist", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*)")).
				WithArgs(int64(10)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			hasBooking, err := repo.HasBookingForService(ctx, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasBooking).To(BeFalse())
		})
	})
})
