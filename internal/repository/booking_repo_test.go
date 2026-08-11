package repository_test

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/internal/interfaces"
	"fyp/internal/repository"
	"regexp"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BookingRepo", func() {
	var (
		db                 *sql.DB
		mock               sqlmock.Sqlmock
		repo               interfaces.IBookingRepo
		ctx                context.Context
		bookingDetailCols  []string
		bookingContextCols []string
		slotCols           []string
		pkgCols            []string
		startTime          time.Time
		endTime            time.Time
		createdAt          time.Time
	)

	BeforeEach(func() {
		var err error
		db, mock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		repo = repository.NewBookingRepo(db)
		ctx = context.Background()

		bookingDetailCols = []string{
			"booking_id", "status", "booking_type",
			"service_slot_id", "slot_option_id", "service_option_id",
			"date", "start_time", "end_time",
			"service_name", "service_option_name", "staff_name",
			"username", "email",
			"business_id", "business_name", "description", "created_at",
		}
		bookingContextCols = []string{
			"booking_id", "status", "service_slot_id", "slot_option_id",
			"user_id", "username", "email",
			"owner_user_id", "owner_username", "owner_email",
			"staff_user_id", "staff_name", "staff_email",
			"business_name", "when_text",
		}
		slotCols = []string{
			"service_slot_id", "staff_id", "staff_name", "recurring_schedule_id",
			"date", "start_time", "end_time", "created_by", "has_booking",
		}
		pkgCols = []string{"slot_option_id", "service_option_id", "service_option_name", "service_id", "service_name"}

		startTime = time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
		endTime = time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC)
		createdAt = time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("InsertBooking", func() {
		It("inserts a pending online booking with a description", func() {
			desc := "please be gentle"

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO fyp_fuli_bookings")).
				WithArgs(int64(1), int64(20), &desc).
				WillReturnRows(sqlmock.NewRows(
					[]string{"booking_id", "user_id", "slot_option_id", "status", "booking_type", "description", "created_at"},
				).AddRow(100, 1, 20, "pending", "online", desc, createdAt))

			result, err := repo.InsertBooking(ctx, 1, 20, &desc)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.BookingID).To(Equal(int64(100)))
			Expect(result.Status).To(Equal("pending"))
			Expect(result.BookingType).To(Equal("online"))
			Expect(*result.Description).To(Equal(desc))
		})

		It("inserts a booking with a nil description", func() {
			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO fyp_fuli_bookings")).
				WithArgs(int64(1), int64(20), nil).
				WillReturnRows(sqlmock.NewRows(
					[]string{"booking_id", "user_id", "slot_option_id", "status", "booking_type", "description", "created_at"},
				).AddRow(101, 1, 20, "pending", "online", nil, createdAt))

			result, err := repo.InsertBooking(ctx, 1, 20, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.BookingID).To(Equal(int64(101)))
			Expect(result.Description).To(BeNil())
		})

		It("propagates a unique-constraint violation from the database as-is", func() {
			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO fyp_fuli_bookings")).
				WithArgs(int64(1), int64(20), nil).
				WillReturnError(fmt.Errorf(`pq: duplicate key value violates unique constraint "bookings_slot_option_id_key"`))

			result, err := repo.InsertBooking(ctx, 1, 20, nil)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("InsertWalkInBooking", func() {
		It("inserts an already-accepted walk-in booking", func() {
			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO fyp_fuli_bookings")).
				WithArgs(int64(1), int64(20)).
				WillReturnRows(sqlmock.NewRows(
					[]string{"booking_id", "user_id", "slot_option_id", "status", "booking_type", "created_at"},
				).AddRow(102, 1, 20, "accepted", "walk_in", createdAt))

			result, err := repo.InsertWalkInBooking(ctx, 1, 20)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.BookingID).To(Equal(int64(102)))
			Expect(result.Status).To(Equal("accepted"))
			Expect(result.BookingType).To(Equal("walk_in"))
		})
	})

	Describe("GetBookingContext", func() {
		It("returns the full booking context including staff", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_bookings b")).
				WithArgs(int64(100)).
				WillReturnRows(sqlmock.NewRows(bookingContextCols).
					AddRow(100, "accepted", 200, 20,
						1, "customer", "customer@example.com",
						2, "owner", "owner@example.com",
						5, "Alice", "alice@example.com",
						"Test Biz", "2026-08-10 09:00"))

			result, err := repo.GetBookingContext(ctx, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.BookingID).To(Equal(int64(100)))
			Expect(result.CustomerName).To(Equal("customer"))
			Expect(result.OwnerName).To(Equal("owner"))
			Expect(*result.SlotStaffUserID).To(Equal(int64(5)))
			Expect(*result.StaffName).To(Equal("Alice"))
			Expect(*result.StaffEmail).To(Equal("alice@example.com"))
			Expect(result.BusinessName).To(Equal("Test Biz"))
			Expect(result.WhenText).To(Equal("2026-08-10 09:00"))
		})

		It("returns nil staff fields for an owner-managed slot", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_bookings b")).
				WithArgs(int64(101)).
				WillReturnRows(sqlmock.NewRows(bookingContextCols).
					AddRow(101, "pending", 201, 21,
						1, "customer", "customer@example.com",
						2, "owner", "owner@example.com",
						nil, nil, nil,
						"Test Biz", "2026-08-11 10:00"))

			result, err := repo.GetBookingContext(ctx, 101)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.SlotStaffUserID).To(BeNil())
			Expect(result.StaffName).To(BeNil())
			Expect(result.StaffEmail).To(BeNil())
		})

		It("returns an error when the booking does not exist", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_bookings b")).
				WithArgs(int64(999)).
				WillReturnError(sql.ErrNoRows)

			result, err := repo.GetBookingContext(ctx, 999)
			Expect(err).To(MatchError(sql.ErrNoRows))
			Expect(result).To(BeNil())
		})
	})

	Describe("GetBookingContextForSlot", func() {
		It("returns the active booking's context for the slot", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_bookings b")).
				WithArgs(int64(200)).
				WillReturnRows(sqlmock.NewRows(bookingContextCols).
					AddRow(100, "accepted", 200, 20,
						1, "customer", "customer@example.com",
						2, "owner", "owner@example.com",
						5, "Alice", "alice@example.com",
						"Test Biz", "2026-08-10 09:00"))

			result, err := repo.GetBookingContextForSlot(ctx, 200)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.BookingID).To(Equal(int64(100)))
			Expect(*result.SlotStaffUserID).To(Equal(int64(5)))
		})

		It("returns nil, nil when the slot has no active booking", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_bookings b")).
				WithArgs(int64(300)).
				WillReturnError(sql.ErrNoRows)

			result, err := repo.GetBookingContextForSlot(ctx, 300)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("GetBookingDetail", func() {
		It("returns the booking detail", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_bookings b")).
				WithArgs(int64(100)).
				WillReturnRows(sqlmock.NewRows(bookingDetailCols).
					AddRow(100, "accepted", "online",
						200, 20, 10,
						"2026-08-10", startTime, endTime,
						"Massage", "Deep Tissue", "Alice",
						"customer", "customer@example.com",
						7, "Test Biz", "note", createdAt))

			result, err := repo.GetBookingDetail(ctx, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.BookingID).To(Equal(int64(100)))
			Expect(*result.StaffName).To(Equal("Alice"))
			Expect(*result.Description).To(Equal("note"))
		})

		It("returns an error when the booking does not exist", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_bookings b")).
				WithArgs(int64(999)).
				WillReturnError(sql.ErrNoRows)

			result, err := repo.GetBookingDetail(ctx, 999)
			Expect(err).To(MatchError(sql.ErrNoRows))
			Expect(result).To(BeNil())
		})
	})

	Describe("UpdateBookingStatus", func() {
		It("updates the booking status and decision metadata", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_bookings")).
				WithArgs("accepted", int64(2), int64(100)).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err := repo.UpdateBookingStatus(ctx, tx, 100, "accepted", 2)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("UpdateBookingDescription", func() {
		It("updates the booking's description", func() {
			desc := "updated note"
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_bookings")).
				WithArgs(&desc, int64(100)).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err := repo.UpdateBookingDescription(ctx, tx, 100, &desc)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("UpdateBookingSlotOption", func() {
		It("reschedules the booking to a new slot option", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_bookings")).
				WithArgs(int64(30), "rescheduled", int64(100)).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err := repo.UpdateBookingSlotOption(ctx, tx, 100, 30, "rescheduled")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("RejectOtherPendingBookingsForSlot", func() {
		It("rejects every other pending booking for the slot", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_bookings")).
				WithArgs(int64(200), int64(100)).
				WillReturnResult(sqlmock.NewResult(0, 2))

			err := repo.RejectOtherPendingBookingsForSlot(ctx, tx, 200, 100)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("SlotOptionIsAvailable", func() {
		It("returns true when the slot option is bookable", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(")).
				WithArgs(int64(20)).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

			ok, err := repo.SlotOptionIsAvailable(ctx, 20)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("returns false when the slot option is already booked or in the past", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(")).
				WithArgs(int64(21)).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

			ok, err := repo.SlotOptionIsAvailable(ctx, 21)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})

	Describe("GetSlotOptionStaffUserID", func() {
		It("returns the staff login user id when the slot is staff-assigned", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slot_options ssp")).
				WithArgs(int64(20)).
				WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(5))

			staffUserID, err := repo.GetSlotOptionStaffUserID(ctx, 20)
			Expect(err).NotTo(HaveOccurred())
			Expect(staffUserID).NotTo(BeNil())
			Expect(*staffUserID).To(Equal(int64(5)))
		})

		It("returns nil when the slot is owner-managed", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slot_options ssp")).
				WithArgs(int64(21)).
				WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(nil))

			staffUserID, err := repo.GetSlotOptionStaffUserID(ctx, 21)
			Expect(err).NotTo(HaveOccurred())
			Expect(staffUserID).To(BeNil())
		})
	})

	Describe("IsSlotOptionOwnedByUser", func() {
		It("returns true when the user owns the business behind the slot option", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(")).
				WithArgs(int64(20), int64(2)).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

			owned, err := repo.IsSlotOptionOwnedByUser(ctx, 20, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(owned).To(BeTrue())
		})

		It("returns false when the user does not own the business", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(")).
				WithArgs(int64(20), int64(9)).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

			owned, err := repo.IsSlotOptionOwnedByUser(ctx, 20, 9)
			Expect(err).NotTo(HaveOccurred())
			Expect(owned).To(BeFalse())
		})
	})

	Describe("GetBusinessBookings", func() {
		It("returns pending/accepted/rescheduled bookings for the business", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_bookings b")).
				WithArgs(int64(7)).
				WillReturnRows(sqlmock.NewRows(bookingDetailCols).
					AddRow(100, "pending", "online", 200, 20, 10, "2026-08-10", startTime, endTime,
						"Massage", "Deep Tissue", nil, "customer1", "c1@example.com", 7, "Test Biz", nil, createdAt).
					AddRow(101, "accepted", "online", 201, 21, 10, "2026-08-11", startTime, endTime,
						"Massage", "Swedish", "Alice", "customer2", "c2@example.com", 7, "Test Biz", "note", createdAt))

			results, err := repo.GetBusinessBookings(ctx, 7, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].BookingID).To(Equal(int64(100)))
			Expect(results[0].StaffName).To(BeNil())
			Expect(*results[1].StaffName).To(Equal("Alice"))
			Expect(*results[1].Description).To(Equal("note"))
		})

		It("filters to a single staff member's bookings when staffID is given", func() {
			staffID := int64(5)
			mock.ExpectQuery(regexp.QuoteMeta("AND ss.staff_id = $")).
				WithArgs(int64(7), staffID).
				WillReturnRows(sqlmock.NewRows(bookingDetailCols).
					AddRow(102, "pending", "online", 202, 22, 10, "2026-08-12", startTime, endTime,
						"Massage", "Deep Tissue", "Alice", "customer3", "c3@example.com", 7, "Test Biz", nil, createdAt))

			results, err := repo.GetBusinessBookings(ctx, 7, &staffID)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].BookingID).To(Equal(int64(102)))
		})
	})

	Describe("GetCustomerBookings", func() {
		It("returns the customer's own bookings, most recent first", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_bookings b")).
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows(bookingDetailCols).
					AddRow(101, "accepted", "online", 201, 21, 10, "2026-08-11", startTime, endTime,
						"Massage", "Swedish", "Alice", "customer", "c@example.com", 7, "Test Biz", nil, createdAt).
					AddRow(100, "pending", "online", 200, 20, 10, "2026-08-10", startTime, endTime,
						"Massage", "Deep Tissue", nil, "customer", "c@example.com", 7, "Test Biz", nil, createdAt))

			results, err := repo.GetCustomerBookings(ctx, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].BookingID).To(Equal(int64(101)))
			Expect(results[1].BookingID).To(Equal(int64(100)))
		})
	})

	Describe("GetRecentlyBookedBusinesses", func() {
		It("returns the businesses the user has recently booked with", func() {
			businessCols := []string{
				"business_id", "owner_user_id", "business_name", "description",
				"address", "image_url", "business_contact_number", "business_email",
			}
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_bookings b")).
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows(businessCols).
					AddRow(7, 2, "Test Biz", nil, "123 St", nil, "0123456789", "biz@example.com"))

			results, err := repo.GetRecentlyBookedBusinesses(ctx, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].BusinessID).To(Equal(int64(7)))
		})
	})

	Describe("SweepPastBookings", func() {
		It("flips any pending/accepted/rescheduled booking whose slot has ended to past", func() {
			mock.ExpectExec(regexp.QuoteMeta("SET status = 'past'")).
				WillReturnResult(sqlmock.NewResult(0, 3))

			err := repo.SweepPastBookings(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetAvailableSlots", func() {
		It("returns available slots with their packages when no staff filter is given", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(1), "2026-08-10", int64(10)).
				WillReturnRows(sqlmock.NewRows(slotCols).
					AddRow(200, nil, nil, nil, "2026-08-10", startTime, endTime, 1, false))

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slot_options ssp")).
				WithArgs(int64(200), int64(10)).
				WillReturnRows(sqlmock.NewRows(pkgCols).
					AddRow(20, 10, "Deep Tissue", 5, "Massage"))

			results, err := repo.GetAvailableSlots(ctx, 1, 10, "2026-08-10", nil, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].ServiceSlotID).To(Equal(int64(200)))
			Expect(results[0].Packages).To(HaveLen(1))
			Expect(results[0].Packages[0].ServiceOptionName).To(Equal("Deep Tissue"))
		})

		It("adds a staff filter as the last placeholder when staffID is given", func() {
			staffID := int64(5)
			mock.ExpectQuery(regexp.QuoteMeta("AND ss.staff_id = $4")).
				WithArgs(int64(1), "2026-08-10", int64(10), staffID).
				WillReturnRows(sqlmock.NewRows(slotCols))

			results, err := repo.GetAvailableSlots(ctx, 1, 10, "2026-08-10", &staffID, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})

		It("filters to unassigned slots only, overriding a given staffID", func() {
			staffID := int64(5)
			mock.ExpectQuery(regexp.QuoteMeta("AND ss.staff_id IS NULL")).
				WithArgs(int64(1), "2026-08-10", int64(10)).
				WillReturnRows(sqlmock.NewRows(slotCols))

			results, err := repo.GetAvailableSlots(ctx, 1, 10, "2026-08-10", &staffID, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})
	})

	Describe("GetAvailableDates", func() {
		It("returns distinct bookable dates in the range with no extra filters", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31").
				WillReturnRows(sqlmock.NewRows([]string{"date"}).
					AddRow("2026-08-10").
					AddRow("2026-08-12"))

			dates, err := repo.GetAvailableDates(ctx, 1, nil, nil, nil, false, "2026-08-01", "2026-08-31")
			Expect(err).NotTo(HaveOccurred())
			Expect(dates).To(Equal([]string{"2026-08-10", "2026-08-12"}))
		})

		It("narrows by service, option, and staff when all are given", func() {
			serviceID := int64(3)
			serviceOptionID := int64(10)
			staffID := int64(5)

			mock.ExpectQuery(regexp.QuoteMeta("AND so.service_id = $4")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31", serviceID, serviceOptionID, staffID).
				WillReturnRows(sqlmock.NewRows([]string{"date"}))

			dates, err := repo.GetAvailableDates(ctx, 1, &serviceID, &serviceOptionID, &staffID, false, "2026-08-01", "2026-08-31")
			Expect(err).NotTo(HaveOccurred())
			Expect(dates).To(BeEmpty())
		})

		It("filters to unassigned-only slots, overriding staffID", func() {
			staffID := int64(5)
			mock.ExpectQuery(regexp.QuoteMeta("AND ss.staff_id IS NULL")).
				WithArgs(int64(1), "2026-08-01", "2026-08-31").
				WillReturnRows(sqlmock.NewRows([]string{"date"}))

			dates, err := repo.GetAvailableDates(ctx, 1, nil, nil, &staffID, true, "2026-08-01", "2026-08-31")
			Expect(err).NotTo(HaveOccurred())
			Expect(dates).To(BeEmpty())
		})
	})
})
