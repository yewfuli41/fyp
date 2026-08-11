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
		pkgCols = []string{"service_option_id", "service_id", "service_option_name", "description", "effective_from", "effective_until", "is_default"}
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

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO fyp_fuli_services")).
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
				ServiceID:         10,
				ServiceOptionName: "Deep Tissue",
			}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO fyp_fuli_service_options")).
				WithArgs(p.ServiceID, p.ServiceOptionName, p.Description, nil, nil, false).
				WillReturnRows(sqlmock.NewRows(pkgCols).
					AddRow(20, 10, "Deep Tissue", nil, "2026-01-01", nil, false))

			result, err := repo.InsertServiceOption(ctx, tx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceOptionID).To(Equal(int64(20)))
		})
	})

	Describe("InsertServiceOptionItem", func() {
		It("inserts a new package item", func() {
			p := param.ServiceOptionItemParam{
				ServiceOptionID:       20,
				ServiceOptionItemName: "Oil",
			}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("INSERT INTO fyp_fuli_service_option_items")).
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

			mock.ExpectQuery(regexp.QuoteMeta("UPDATE fyp_fuli_services")).
				WithArgs(p.ServiceName, p.Description, p.ServiceID, p.BusinessID).
				WillReturnRows(sqlmock.NewRows(serviceCols).
					AddRow(10, 1, "Updated Massage", nil))

			result, err := repo.UpdateService(ctx, tx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceName).To(Equal("Updated Massage"))
		})
	})

	Describe("SoftDeleteServiceOption", func() {
		It("soft deletes an option and its items", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_service_option_items SET deleted_at = NOW()")).
				WithArgs(int64(20)).
				WillReturnResult(sqlmock.NewResult(1, 1))

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_service_options SET deleted_at = NOW()")).
				WithArgs(int64(20)).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err := repo.SoftDeleteServiceOption(ctx, tx, 20)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("IsOptionEffectiveOn", func() {
		It("returns true when the option's own window covers the date", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(")).
				WithArgs(int64(41), "2026-07-15").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

			ok, err := repo.IsOptionEffectiveOn(ctx, 41, "2026-07-15")
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("returns false when the date falls outside the option's window", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(")).
				WithArgs(int64(41), "2030-01-01").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

			ok, err := repo.IsOptionEffectiveOn(ctx, 41, "2030-01-01")
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})

	Describe("SetOptionWindow", func() {
		It("resizes an option's validity window", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			until := "2026-06-30"
			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_service_options")).
				WithArgs(int64(20), "2026-01-01", &until).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err := repo.SetOptionWindow(ctx, tx, 20, "2026-01-01", &until)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("SetServiceDefaultOption", func() {
		It("marks the given option as default and every other option under the service as not-default", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE service_options")).
				WithArgs(int64(10), int64(20)).
				WillReturnResult(sqlmock.NewResult(0, 3))

			err := repo.SetServiceDefaultOption(ctx, tx, 10, 20)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("SoftDeleteService", func() {
		It("soft deletes a service", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_services SET deleted_at = NOW()")).
				WithArgs(int64(10), int64(1)).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err := repo.SoftDeleteService(ctx, tx, 10, 1)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetServicesByBusinessID", func() {
		It("returns services for a business", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT service_id, business_id, service_name, description FROM fyp_fuli_services")).
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
		pkgColsWithBooking := []string{
			"service_option_id", "service_id", "service_option_name", "description",
			"effective_from", "effective_until", "is_default", "has_booking",
		}

		It("returns packages for a service", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_options")).
				WithArgs(int64(10)).
				WillReturnRows(sqlmock.NewRows(pkgColsWithBooking).
					AddRow(20, 10, "Deep Tissue", nil, "2026-01-01", nil, true, false).
					AddRow(21, 10, "Swedish", nil, "2026-01-01", nil, false, true))

			results, err := repo.GetServiceOptionsByServiceID(ctx, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].ServiceOptionID).To(Equal(int64(20)))
			Expect(results[0].HasBooking).To(BeFalse())
			Expect(results[1].HasBooking).To(BeTrue())
		})

		It("returns future live options as removed for temporary removal display", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_options")).
				WithArgs(int64(10)).
				WillReturnRows(sqlmock.NewRows(pkgColsWithBooking).
					AddRow(20, 10, "Deep Tissue", nil, "2030-01-01", nil, true, false))

			results, err := repo.GetServiceOptionsByServiceID(ctx, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].IsRemoved).To(BeTrue())
		})

		It("returns empty slice when no packages exist", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_options")).
				WithArgs(int64(99)).
				WillReturnRows(sqlmock.NewRows(pkgColsWithBooking))

			results, err := repo.GetServiceOptionsByServiceID(ctx, 99)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})
	})

	Describe("GetServiceOptionItemsByOptionID", func() {
		It("returns items for a package", func() {
			itemCols := []string{"service_option_item_id", "service_option_id", "service_option_item_name"}

			mock.ExpectQuery(regexp.QuoteMeta("SELECT service_option_item_id, service_option_id, service_option_item_name FROM fyp_fuli_service_option_items")).
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

			mock.ExpectQuery(regexp.QuoteMeta("SELECT service_option_item_id, service_option_id, service_option_item_name FROM fyp_fuli_service_option_items")).
				WithArgs(int64(99)).
				WillReturnRows(sqlmock.NewRows(itemCols))

			results, err := repo.GetServiceOptionItemsByOptionID(ctx, 99)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})
	})

	Describe("GetServiceOptionItemsByOptionIDIncludeDeleted", func() {
		It("returns items for a package including soft-deleted ones", func() {
			itemCols := []string{"service_option_item_id", "service_option_id", "service_option_item_name"}

			mock.ExpectQuery(regexp.QuoteMeta("SELECT service_option_item_id, service_option_id, service_option_item_name FROM service_option_items")).
				WithArgs(int64(20)).
				WillReturnRows(sqlmock.NewRows(itemCols).
					AddRow(1, 20, "Oil").
					AddRow(2, 20, "Lotion").
					AddRow(3, 20, "Towel"))

			results, err := repo.GetServiceOptionItemsByOptionIDIncludeDeleted(ctx, 20)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(3))
			Expect(results[0].ServiceOptionItemName).To(Equal("Oil"))
			Expect(results[2].ServiceOptionItemName).To(Equal("Towel"))
		})

		It("returns empty slice when no items exist", func() {
			itemCols := []string{"service_option_item_id", "service_option_id", "service_option_item_name"}

			mock.ExpectQuery(regexp.QuoteMeta("SELECT service_option_item_id, service_option_id, service_option_item_name FROM service_option_items")).
				WithArgs(int64(99)).
				WillReturnRows(sqlmock.NewRows(itemCols))

			results, err := repo.GetServiceOptionItemsByOptionIDIncludeDeleted(ctx, 99)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})
	})

	Describe("HasBookingForService", func() {
		It("returns true if an active booking exists", func() {
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

		It("only counts pending/accepted/rescheduled bookings, not rejected/cancelled/past ones", func() {
			mock.ExpectQuery(regexp.QuoteMeta("status IN ('pending', 'accepted', 'rescheduled')")).
				WithArgs(int64(10)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			hasBooking, err := repo.HasBookingForService(ctx, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasBooking).To(BeFalse())
		})
	})

	Describe("HasBookingForOption", func() {
		It("returns true if an active booking exists for the option", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*)")).
				WithArgs(int64(20)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

			hasBooking, err := repo.HasBookingForOption(ctx, 20)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasBooking).To(BeTrue())
		})

		It("returns false if no bookings exist for the option", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*)")).
				WithArgs(int64(20)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			hasBooking, err := repo.HasBookingForOption(ctx, 20)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasBooking).To(BeFalse())
		})
	})

	Describe("DropSlotOptionIfExpired", func() {
		It("soft-deletes the slot option in a single statement, scoped to when the option's own window no longer covers the slot's date", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_service_slot_options sso")).
				WithArgs(int64(777)).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err := repo.DropSlotOptionIfExpired(ctx, tx, 777)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("CascadeDeleteServiceSlots", func() {
		It("soft-deletes the service's slot offerings then any slot left with no active options", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_service_slot_options sso")).
				WithArgs(int64(10)).
				WillReturnResult(sqlmock.NewResult(0, 2))

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_service_slots ss")).
				WithArgs(int64(10)).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err := repo.CascadeDeleteServiceSlots(ctx, tx, 10)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
