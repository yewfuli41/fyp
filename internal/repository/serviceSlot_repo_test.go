package repository_test

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"fyp/internal/repository"
	"regexp"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServiceSlotRepo", func() {
	var (
		db        *sql.DB
		mock      sqlmock.Sqlmock
		repo      interfaces.IServiceSlotRepo
		ctx       context.Context
		slotCols  []string
		pkgCols   []string
		staffCols []string
		startTime time.Time
		endTime   time.Time
	)

	BeforeEach(func() {
		var err error
		db, mock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		repo = repository.NewServiceSlotRepo(db)
		ctx = context.Background()
		slotCols = []string{
			"service_slot_id", "staff_id", "staff_name", "recurring_schedule_id",
			"date", "start_time", "end_time", "created_by", "has_booking",
		}
		pkgCols = []string{"slot_option_id", "service_option_id", "service_option_name", "service_id", "service_name"}
		staffCols = []string{"staff_id", "user_id", "business_id", "staff_name", "email", "must_reset_password", "staff_contact_number", "position", "has_booking"}
		startTime = time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
		endTime = time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC)
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("InsertServiceSlot", func() {
		It("inserts an owner-managed, non-recurring slot", func() {
			p := param.ServiceSlotParam{
				Date:      "2026-08-10",
				StartTime: startTime,
				EndTime:   endTime,
				CreatedBy: 1,
			}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO fyp_fuli_service_slots")).
				WithArgs(nil, nil, p.Date, "09:00:00", "10:00:00", p.CreatedBy).
				WillReturnRows(sqlmock.NewRows([]string{"service_slot_id"}).AddRow(100))

			id, err := repo.InsertServiceSlot(ctx, tx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal(int64(100)))
		})

		It("inserts a staff-assigned recurring slot", func() {
			staffID := int64(5)
			recurringID := int64(9)
			p := param.ServiceSlotParam{
				StaffID:             &staffID,
				RecurringScheduleID: &recurringID,
				Date:                "2026-08-10",
				StartTime:           startTime,
				EndTime:             endTime,
				CreatedBy:           1,
			}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO fyp_fuli_service_slots")).
				WithArgs(staffID, recurringID, p.Date, "09:00:00", "10:00:00", p.CreatedBy).
				WillReturnRows(sqlmock.NewRows([]string{"service_slot_id"}).AddRow(101))

			id, err := repo.InsertServiceSlot(ctx, tx, p)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal(int64(101)))
		})
	})

	Describe("InsertServiceSlotOption", func() {
		It("attaches a package to a slot", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("INSERT INTO fyp_fuli_service_slot_options")).
				WithArgs(int64(20), int64(100)).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err := repo.InsertServiceSlotOption(ctx, tx, 100, 20)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("InsertRecurringSchedule", func() {
		It("creates a recurring schedule row with a NULL staff_id for an owner-managed slot", func() {
			p := param.ServiceSlotParam{BusinessID: 1, StartTime: startTime, EndTime: endTime, CreatedBy: 1}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO fyp_fuli_recurring_schedules")).
				WithArgs(int64(1), nil, "monday", "09:00:00", "10:00:00", p.CreatedBy).
				WillReturnRows(sqlmock.NewRows([]string{"recurring_schedule_id"}).AddRow(9))

			id, err := repo.InsertRecurringSchedule(ctx, tx, p, "monday")
			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal(int64(9)))
		})

		It("creates a recurring schedule row for a staff-assigned slot", func() {
			staffID := int64(5)
			p := param.ServiceSlotParam{BusinessID: 1, StaffID: &staffID, StartTime: startTime, EndTime: endTime, CreatedBy: 1}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO fyp_fuli_recurring_schedules")).
				WithArgs(int64(1), staffID, "monday", "09:00:00", "10:00:00", p.CreatedBy).
				WillReturnRows(sqlmock.NewRows([]string{"recurring_schedule_id"}).AddRow(9))

			id, err := repo.InsertRecurringSchedule(ctx, tx, p, "monday")
			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal(int64(9)))
		})
	})

	Describe("GetServiceSlotsByBusinessAndDate", func() {
		It("returns slots with their packages for a business and date", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(1), "2026-08-10").
				WillReturnRows(sqlmock.NewRows(slotCols).
					AddRow(100, nil, nil, nil, "2026-08-10", startTime, endTime, 1, false))

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slot_options ssp")).
				WithArgs(int64(100)).
				WillReturnRows(sqlmock.NewRows(pkgCols).
					AddRow(1, 20, "Deep Tissue", 10, "Massage"))

			results, err := repo.GetServiceSlotsByBusinessAndDate(ctx, 1, "2026-08-10", nil, nil, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].ServiceSlotID).To(Equal(int64(100)))
			Expect(results[0].Packages).To(HaveLen(1))
			Expect(results[0].Packages[0].ServiceOptionName).To(Equal("Deep Tissue"))
		})

		It("filters by staff ID when unassignedOnly is false", func() {
			staffID := int64(5)
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(1), "2026-08-10", staffID).
				WillReturnRows(sqlmock.NewRows(slotCols))

			results, err := repo.GetServiceSlotsByBusinessAndDate(ctx, 1, "2026-08-10", &staffID, nil, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})

		It("filters unassigned slots only, ignoring a given staff ID", func() {
			staffID := int64(5)
			mock.ExpectQuery(regexp.QuoteMeta("AND ss.staff_id IS NULL")).
				WithArgs(int64(1), "2026-08-10").
				WillReturnRows(sqlmock.NewRows(slotCols))

			results, err := repo.GetServiceSlotsByBusinessAndDate(ctx, 1, "2026-08-10", &staffID, nil, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})

		It("filters by service ID", func() {
			serviceID := int64(10)
			mock.ExpectQuery(regexp.QuoteMeta("sp.service_id = $")).
				WithArgs(int64(1), "2026-08-10", serviceID).
				WillReturnRows(sqlmock.NewRows(slotCols))

			results, err := repo.GetServiceSlotsByBusinessAndDate(ctx, 1, "2026-08-10", nil, &serviceID, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})

		It("returns an error when the query fails", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(1), "2026-08-10").
				WillReturnError(fmt.Errorf("db error"))

			results, err := repo.GetServiceSlotsByBusinessAndDate(ctx, 1, "2026-08-10", nil, nil, false)
			Expect(err).To(MatchError("db error"))
			Expect(results).To(BeNil())
		})
	})

	Describe("GetServiceSlotByID", func() {
		It("returns the slot along with its packages", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(100), int64(1)).
				WillReturnRows(sqlmock.NewRows(slotCols).
					AddRow(100, int64(5), "Alice", int64(9), "2026-08-10", startTime, endTime, 1, true))

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slot_options ssp")).
				WithArgs(int64(100)).
				WillReturnRows(sqlmock.NewRows(pkgCols).
					AddRow(1, 20, "Deep Tissue", 10, "Massage"))

			result, err := repo.GetServiceSlotByID(ctx, 100, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.ServiceSlotID).To(Equal(int64(100)))
			Expect(*result.StaffID).To(Equal(int64(5)))
			Expect(result.StaffName).To(Equal("Alice"))
			Expect(*result.RecurringScheduleID).To(Equal(int64(9)))
			Expect(result.HasBooking).To(BeTrue())
			Expect(result.Packages).To(HaveLen(1))
		})

		It("returns an error when the slot is not found", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(999), int64(1)).
				WillReturnError(sql.ErrNoRows)

			result, err := repo.GetServiceSlotByID(ctx, 999, 1)
			Expect(err).To(MatchError(sql.ErrNoRows))
			Expect(result).To(BeNil())
		})
	})

	Describe("ReassignStaff", func() {
		It("assigns a slot to a staff member", func() {
			staffID := int64(5)
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_service_slots ss")).
				WithArgs(staffID, int64(100), int64(1)).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err := repo.ReassignStaff(ctx, tx, 100, 1, &staffID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("unassigns a slot when staffID is nil", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_service_slots ss")).
				WithArgs(nil, int64(100), int64(1)).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err := repo.ReassignStaff(ctx, tx, 100, 1, nil)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("HasBookingForServiceSlot", func() {
		It("returns true if a booking exists", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*)")).
				WithArgs(int64(100)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

			hasBooking, err := repo.HasBookingForServiceSlot(ctx, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasBooking).To(BeTrue())
		})

		It("returns false if no booking exists", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*)")).
				WithArgs(int64(100)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			hasBooking, err := repo.HasBookingForServiceSlot(ctx, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasBooking).To(BeFalse())
		})
	})

	Describe("SoftDeleteServiceSlot", func() {
		It("soft deletes a single slot", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_service_slots ss")).
				WithArgs(int64(100), int64(1)).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err := repo.SoftDeleteServiceSlot(ctx, tx, 100, 1)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("SoftDeleteServiceSlots", func() {
		It("soft deletes multiple slots", func() {
			ids := []int64{100, 101}
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_service_slots ss")).
				WithArgs(pq.Array(ids), int64(1)).
				WillReturnResult(sqlmock.NewResult(1, 2))

			err := repo.SoftDeleteServiceSlots(ctx, tx, ids, 1)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetFutureRecurringSlotIDs", func() {
		It("returns every future slot in the recurring series", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(1), int64(9), "2026-08-10").
				WillReturnRows(sqlmock.NewRows([]string{"service_slot_id"}).
					AddRow(100).AddRow(107))

			ids, err := repo.GetFutureRecurringSlotIDs(ctx, 1, 9, "2026-08-10")
			Expect(err).NotTo(HaveOccurred())
			Expect(ids).To(Equal([]int64{100, 107}))
		})
	})

	Describe("SoftDeleteRecurringSchedule", func() {
		It("soft deletes a recurring schedule owned by the business", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_recurring_schedules rs")).
				WithArgs(int64(1), int64(9)).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err := repo.SoftDeleteRecurringSchedule(ctx, tx, 1, 9)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("StaffBelongsToBusiness", func() {
		It("returns true when the staff belongs to the business", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM fyp_fuli_staff")).
				WithArgs(int64(5), int64(1)).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

			ok, err := repo.StaffBelongsToBusiness(ctx, 5, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("returns false when the staff does not belong to the business", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM fyp_fuli_staff")).
				WithArgs(int64(6), int64(1)).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

			ok, err := repo.StaffBelongsToBusiness(ctx, 6, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})

	Describe("OptionsBelongToBusiness", func() {
		It("returns false without querying when no package IDs are given", func() {
			ok, err := repo.OptionsBelongToBusiness(ctx, 1, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

		It("returns true when every package belongs to the business", func() {
			ids := []int64{20, 21}
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM fyp_fuli_service_options")).
				WithArgs(int64(1), pq.Array(ids)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

			ok, err := repo.OptionsBelongToBusiness(ctx, 1, ids)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("returns false when some packages do not belong to the business", func() {
			ids := []int64{20, 21}
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM fyp_fuli_service_options")).
				WithArgs(int64(1), pq.Array(ids)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

			ok, err := repo.OptionsBelongToBusiness(ctx, 1, ids)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})

	Describe("StaffCoversTime", func() {
		It("returns true when the staff works during the requested time", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff_working_hours")).
				WithArgs(int64(5), "monday", "09:00:00", "10:00:00").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

			ok, err := repo.StaffCoversTime(ctx, 5, "monday", startTime, endTime)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("returns false when the staff does not work during the requested time", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff_working_hours")).
				WithArgs(int64(5), "monday", "09:00:00", "10:00:00").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

			ok, err := repo.StaffCoversTime(ctx, 5, "monday", startTime, endTime)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})

	Describe("GetAvailableStaff", func() {
		It("returns staff available for the given time slot", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff st")).
				WithArgs(int64(1), "monday", "09:00:00", "10:00:00", "2026-08-10", int64(100)).
				WillReturnRows(sqlmock.NewRows(staffCols).
					AddRow(5, 50, 1, "Alice", "alice@example.com", false, "0123456789", "Therapist", false))

			results, err := repo.GetAvailableStaff(ctx, 1, "2026-08-10", "monday", startTime, endTime, 100)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].StaffName).To(Equal("Alice"))
		})

		It("returns an error when the query fails", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff st")).
				WithArgs(int64(1), "monday", "09:00:00", "10:00:00", "2026-08-10", int64(100)).
				WillReturnError(fmt.Errorf("db error"))

			results, err := repo.GetAvailableStaff(ctx, 1, "2026-08-10", "monday", startTime, endTime, 100)
			Expect(err).To(MatchError("db error"))
			Expect(results).To(BeNil())
		})
	})

	Describe("GetRecurringSchedulesNeedingRenewal", func() {
		It("returns schedules whose horizon has fallen short", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_recurring_schedules rs")).
				WithArgs(int64(1), "2026-09-01").
				WillReturnRows(sqlmock.NewRows(
					[]string{"recurring_schedule_id", "staff_id", "day", "start_time", "end_time", "last_date"},
				).
					AddRow(42, 5, "monday", startTime, endTime, "2026-08-10").
					AddRow(43, 6, "friday", startTime, endTime, nil))

			results, err := repo.GetRecurringSchedulesNeedingRenewal(ctx, 1, "2026-09-01")
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].RecurringScheduleID).To(Equal(int64(42)))
			Expect(results[0].LastDate).To(Equal("2026-08-10"))
			Expect(results[1].LastDate).To(Equal("")) // no active occurrences left at all
		})

		It("returns an error when the query fails", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_recurring_schedules rs")).
				WithArgs(int64(1), "2026-09-01").
				WillReturnError(fmt.Errorf("db error"))

			results, err := repo.GetRecurringSchedulesNeedingRenewal(ctx, 1, "2026-09-01")
			Expect(err).To(MatchError("db error"))
			Expect(results).To(BeNil())
		})
	})

	Describe("GetOptionIDsForRecurringSchedule", func() {
		It("returns every option ever attached to the series", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slot_options sso")).
				WithArgs(int64(42)).
				WillReturnRows(sqlmock.NewRows([]string{"service_option_id"}).AddRow(9).AddRow(11))

			ids, err := repo.GetOptionIDsForRecurringSchedule(ctx, 42)
			Expect(err).NotTo(HaveOccurred())
			Expect(ids).To(Equal([]int64{9, 11}))
		})

		It("returns an error when the query fails", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slot_options sso")).
				WithArgs(int64(42)).
				WillReturnError(fmt.Errorf("db error"))

			ids, err := repo.GetOptionIDsForRecurringSchedule(ctx, 42)
			Expect(err).To(MatchError("db error"))
			Expect(ids).To(BeNil())
		})
	})

	Describe("GetSlotDates", func() {
		It("returns the dates of the given slots, sorted", func() {
			ids := []int64{100, 101}
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots")).
				WithArgs(pq.Array(ids)).
				WillReturnRows(sqlmock.NewRows([]string{"date"}).
					AddRow("2026-08-10").AddRow("2026-08-11"))

			dates, err := repo.GetSlotDates(ctx, ids)
			Expect(err).NotTo(HaveOccurred())
			Expect(dates).To(Equal([]string{"2026-08-10", "2026-08-11"}))
		})

		It("returns nil without querying when no slot IDs are given", func() {
			dates, err := repo.GetSlotDates(ctx, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(dates).To(BeNil())
		})
	})

	Describe("GetFutureUnassignedSlotWindows", func() {
		It("returns future owner-managed slot windows for a business", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(1), "2026-08-10").
				WillReturnRows(sqlmock.NewRows([]string{"date", "start_time", "end_time"}).
					AddRow("2026-08-10", startTime, endTime))

			windows, err := repo.GetFutureUnassignedSlotWindows(ctx, 1, "2026-08-10")
			Expect(err).NotTo(HaveOccurred())
			Expect(windows).To(HaveLen(1))
			Expect(windows[0].Date).To(Equal("2026-08-10"))
			Expect(windows[0].StartTime).To(Equal(startTime))
			Expect(windows[0].EndTime).To(Equal(endTime))
		})
	})

	Describe("GetAssignedSlotsInRange", func() {
		It("returns a staff's assigned slots within a date range", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(5), "2026-08-10", "2026-08-20").
				WillReturnRows(sqlmock.NewRows([]string{"service_slot_id", "date", "start_time", "end_time", "has_booking"}).
					AddRow(100, "2026-08-10", startTime, endTime, true))

			slots, err := repo.GetAssignedSlotsInRange(ctx, 5, "2026-08-10", "2026-08-20")
			Expect(err).NotTo(HaveOccurred())
			Expect(slots).To(HaveLen(1))
			Expect(slots[0].ServiceSlotID).To(Equal(int64(100)))
			Expect(slots[0].Date).To(Equal("2026-08-10"))
			Expect(slots[0].HasBooking).To(BeTrue())
		})
	})

	Describe("GetFutureAssignedSlotWindows", func() {
		It("returns a staff's future assigned slots", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_service_slots ss")).
				WithArgs(int64(5), "2026-08-10").
				WillReturnRows(sqlmock.NewRows([]string{"service_slot_id", "date", "start_time", "end_time", "has_booking"}).
					AddRow(101, "2026-08-15", startTime, endTime, false))

			slots, err := repo.GetFutureAssignedSlotWindows(ctx, 5, "2026-08-10")
			Expect(err).NotTo(HaveOccurred())
			Expect(slots).To(HaveLen(1))
			Expect(slots[0].ServiceSlotID).To(Equal(int64(101)))
			Expect(slots[0].Date).To(Equal("2026-08-15"))
			Expect(slots[0].HasBooking).To(BeFalse())
		})
	})

	Describe("BusinessCoversTime", func() {
		It("returns true when business working hours cover the requested time", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_business_working_hours")).
				WithArgs(int64(1), "monday", "09:00:00", "10:00:00").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

			ok, err := repo.BusinessCoversTime(ctx, 1, "monday", startTime, endTime)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("returns false when business working hours do not cover the requested time", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_business_working_hours")).
				WithArgs(int64(1), "monday", "09:00:00", "10:00:00").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

			ok, err := repo.BusinessCoversTime(ctx, 1, "monday", startTime, endTime)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})
})
