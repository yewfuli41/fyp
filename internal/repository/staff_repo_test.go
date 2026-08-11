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

var _ = Describe("StaffRepo", func() {
	var (
		db                   *sql.DB
		mock                 sqlmock.Sqlmock
		repo                 interfaces.IStaffRepo
		ctx                  context.Context
		staffCols            []string
		staffColsWithBooking []string
	)

	BeforeEach(func() {
		var err error
		db, mock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		repo = repository.NewStaffRepo(db)
		ctx = context.Background()
		staffCols = []string{
			"staff_id", "user_id", "business_id", "staff_name",
			"staff_contact_number", "position",
		}
		staffColsWithBooking = []string{
			"staff_id", "user_id", "business_id", "staff_name",
			"email", "must_reset_password", "staff_contact_number", "position",
			"has_booking",
		}
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("InsertStaff", func() {
		p := param.StaffParam{
			UserID:             1,
			BusinessID:         2,
			StaffName:          "Alice",
			StaffContactNumber: "0123456789",
			Position:           "Therapist",
		}

		It("restores a previously soft-deleted staff record if one exists", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("UPDATE fyp_fuli_staff SET")).
				WithArgs(p.UserID, p.BusinessID, p.StaffName, p.StaffContactNumber, p.Position).
				WillReturnRows(sqlmock.NewRows([]string{"staff_id"}).AddRow(9))

			staffID, err := repo.InsertStaff(ctx, tx, p)

			Expect(err).NotTo(HaveOccurred())
			Expect(*staffID).To(Equal(int64(9)))
		})

		It("inserts a brand new staff record when no soft-deleted one exists", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("UPDATE fyp_fuli_staff SET")).
				WithArgs(p.UserID, p.BusinessID, p.StaffName, p.StaffContactNumber, p.Position).
				WillReturnError(sql.ErrNoRows)

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO fyp_fuli_staff")).
				WithArgs(p.UserID, p.BusinessID, p.StaffName, p.StaffContactNumber, p.Position).
				WillReturnRows(sqlmock.NewRows([]string{"staff_id"}).AddRow(10))

			staffID, err := repo.InsertStaff(ctx, tx, p)

			Expect(err).NotTo(HaveOccurred())
			Expect(*staffID).To(Equal(int64(10)))
		})

		It("propagates a database error from the restore check", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("UPDATE fyp_fuli_staff SET")).
				WithArgs(p.UserID, p.BusinessID, p.StaffName, p.StaffContactNumber, p.Position).
				WillReturnError(sql.ErrConnDone)

			staffID, err := repo.InsertStaff(ctx, tx, p)

			Expect(err).To(HaveOccurred())
			Expect(staffID).To(BeNil())
		})

		It("propagates a database error from the insert", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("UPDATE fyp_fuli_staff SET")).
				WithArgs(p.UserID, p.BusinessID, p.StaffName, p.StaffContactNumber, p.Position).
				WillReturnError(sql.ErrNoRows)

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO fyp_fuli_staff")).
				WithArgs(p.UserID, p.BusinessID, p.StaffName, p.StaffContactNumber, p.Position).
				WillReturnError(sql.ErrConnDone)

			staffID, err := repo.InsertStaff(ctx, tx, p)

			Expect(err).To(HaveOccurred())
			Expect(staffID).To(BeNil())
		})
	})

	Describe("InsertStaffWorkingHours", func() {
		startTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
		endTime := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

		It("inserts working hours using the provided transaction", func() {
			p := param.StaffParam{
				StaffID: 9,
				WorkingHours: []param.WorkingHourParam{
					{Day: "MONDAY", StartTime: startTime, EndTime: endTime},
					{Day: "TUESDAY", StartTime: startTime, EndTime: endTime},
				},
			}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			for _, wh := range p.WorkingHours {
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO fyp_fuli_staff_working_hours")).
					WithArgs(
						p.StaffID,
						wh.Day,
						wh.StartTime.Format("15:04:05"),
						wh.EndTime.Format("15:04:05"),
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			err := repo.InsertStaffWorkingHours(ctx, tx, p)

			Expect(err).NotTo(HaveOccurred())
		})

		It("propagates a database error", func() {
			p := param.StaffParam{
				StaffID: 9,
				WorkingHours: []param.WorkingHourParam{
					{Day: "MONDAY", StartTime: startTime, EndTime: endTime},
				},
			}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("INSERT INTO fyp_fuli_staff_working_hours")).
				WithArgs(p.StaffID, "MONDAY", startTime.Format("15:04:05"), endTime.Format("15:04:05")).
				WillReturnError(sql.ErrConnDone)

			err := repo.InsertStaffWorkingHours(ctx, tx, p)

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("DeleteStaffWorkingHours", func() {
		It("soft deletes working hours for a staff member", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_staff_working_hours SET deleted_at = NOW()")).
				WithArgs(int64(9)).
				WillReturnResult(sqlmock.NewResult(0, 2))

			err := repo.DeleteStaffWorkingHours(ctx, tx, 9)

			Expect(err).NotTo(HaveOccurred())
		})

		It("propagates a database error", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_staff_working_hours SET deleted_at = NOW()")).
				WithArgs(int64(9)).
				WillReturnError(sql.ErrConnDone)

			err := repo.DeleteStaffWorkingHours(ctx, tx, 9)

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetStaffByUserID", func() {
		It("returns the staff profile for the given user", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff")).
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows(staffCols).
					AddRow(9, 1, 2, "Alice", "0123456789", "Therapist"))

			result, err := repo.GetStaffByUserID(ctx, 1)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.StaffID).To(Equal(int64(9)))
			Expect(result.StaffName).To(Equal("Alice"))
			Expect(result.Position).To(Equal("Therapist"))
		})

		It("returns sql.ErrNoRows when the staff member is not found", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff")).
				WithArgs(int64(999)).
				WillReturnError(sql.ErrNoRows)

			result, err := repo.GetStaffByUserID(ctx, 999)

			Expect(err).To(Equal(sql.ErrNoRows))
			Expect(result).To(BeNil())
		})

		It("propagates a database error", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff")).
				WithArgs(int64(1)).
				WillReturnError(sql.ErrConnDone)

			result, err := repo.GetStaffByUserID(ctx, 1)

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("GetStaffWorkingHours", func() {
		It("returns working hours for a staff member", func() {
			whCols := []string{"day", "start_time", "end_time"}
			startTime := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
			endTime := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff_working_hours")).
				WithArgs(int64(9)).
				WillReturnRows(sqlmock.NewRows(whCols).
					AddRow("MONDAY", startTime, endTime).
					AddRow("TUESDAY", startTime, endTime))

			results, err := repo.GetStaffWorkingHours(ctx, 9)

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].Day).To(Equal("MONDAY"))
		})

		It("returns an empty slice when no working hours exist", func() {
			whCols := []string{"day", "start_time", "end_time"}

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff_working_hours")).
				WithArgs(int64(99)).
				WillReturnRows(sqlmock.NewRows(whCols))

			results, err := repo.GetStaffWorkingHours(ctx, 99)

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})

		It("propagates a database error", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff_working_hours")).
				WithArgs(int64(9)).
				WillReturnError(sql.ErrConnDone)

			results, err := repo.GetStaffWorkingHours(ctx, 9)

			Expect(err).To(HaveOccurred())
			Expect(results).To(BeNil())
		})
	})

	Describe("GetStaffByBusinessID", func() {
		It("returns staff for a business", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff st")).
				WithArgs(int64(2)).
				WillReturnRows(sqlmock.NewRows(staffColsWithBooking).
					AddRow(9, 1, 2, "Alice", "alice@example.com", false, "0123456789", "Therapist", false).
					AddRow(10, 3, 2, "Bob", "bob@example.com", true, nil, nil, true))

			results, err := repo.GetStaffByBusinessID(ctx, 2)

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].StaffID).To(Equal(int64(9)))
			Expect(results[0].HasBooking).To(BeFalse())
			Expect(results[1].StaffID).To(Equal(int64(10)))
			Expect(results[1].HasBooking).To(BeTrue())
			Expect(results[1].Position).To(Equal(""))
		})

		It("returns an empty slice when no staff exist", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff st")).
				WithArgs(int64(999)).
				WillReturnRows(sqlmock.NewRows(staffColsWithBooking))

			results, err := repo.GetStaffByBusinessID(ctx, 999)

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})

		It("propagates a database error", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff st")).
				WithArgs(int64(2)).
				WillReturnError(sql.ErrConnDone)

			results, err := repo.GetStaffByBusinessID(ctx, 2)

			Expect(err).To(HaveOccurred())
			Expect(results).To(BeNil())
		})
	})

	Describe("UpdateStaff", func() {
		p := param.StaffParam{
			StaffID:            9,
			BusinessID:         2,
			StaffName:          "Alice Updated",
			StaffContactNumber: "0198765432",
			Position:           "Senior Therapist",
		}

		It("updates a staff member and returns the refreshed record", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_staff")).
				WithArgs(p.StaffName, p.StaffContactNumber, p.Position, p.StaffID, p.BusinessID).
				WillReturnResult(sqlmock.NewResult(0, 1))

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff st")).
				WithArgs(p.StaffID, p.BusinessID).
				WillReturnRows(sqlmock.NewRows(staffColsWithBooking).
					AddRow(9, 1, 2, "Alice Updated", "alice@example.com", false, "0198765432", "Senior Therapist", false))

			result, err := repo.UpdateStaff(ctx, tx, p)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.StaffName).To(Equal("Alice Updated"))
		})

		It("propagates a database error from the update", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_staff")).
				WithArgs(p.StaffName, p.StaffContactNumber, p.Position, p.StaffID, p.BusinessID).
				WillReturnError(sql.ErrConnDone)

			result, err := repo.UpdateStaff(ctx, tx, p)

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("propagates a database error from the refresh fetch", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_staff")).
				WithArgs(p.StaffName, p.StaffContactNumber, p.Position, p.StaffID, p.BusinessID).
				WillReturnResult(sqlmock.NewResult(0, 1))

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff st")).
				WithArgs(p.StaffID, p.BusinessID).
				WillReturnError(sql.ErrNoRows)

			result, err := repo.UpdateStaff(ctx, tx, p)

			Expect(err).To(Equal(sql.ErrNoRows))
			Expect(result).To(BeNil())
		})
	})

	Describe("GetStaffByIDTx", func() {
		It("returns the staff record scoped to business", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff st")).
				WithArgs(int64(9), int64(2)).
				WillReturnRows(sqlmock.NewRows(staffColsWithBooking).
					AddRow(9, 1, 2, "Alice", "alice@example.com", false, "0123456789", "Therapist", false))

			result, err := repo.GetStaffByIDTx(ctx, tx, 9, 2)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.StaffID).To(Equal(int64(9)))
		})

		It("returns sql.ErrNoRows when the staff member is not found", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_staff st")).
				WithArgs(int64(999), int64(2)).
				WillReturnError(sql.ErrNoRows)

			result, err := repo.GetStaffByIDTx(ctx, tx, 999, 2)

			Expect(err).To(Equal(sql.ErrNoRows))
			Expect(result).To(BeNil())
		})
	})

	Describe("SoftDeleteStaff", func() {
		It("soft deletes a staff member", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_staff SET deleted_at = NOW()")).
				WithArgs(int64(9), int64(2)).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err := repo.SoftDeleteStaff(ctx, tx, 9, 2)

			Expect(err).NotTo(HaveOccurred())
		})

		It("propagates a database error", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_staff SET deleted_at = NOW()")).
				WithArgs(int64(9), int64(2)).
				WillReturnError(sql.ErrConnDone)

			err := repo.SoftDeleteStaff(ctx, tx, 9, 2)

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("HasBookingForStaff", func() {
		It("returns true if an active booking exists", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*)")).
				WithArgs(int64(9)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

			hasBooking, err := repo.HasBookingForStaff(ctx, 9)

			Expect(err).NotTo(HaveOccurred())
			Expect(hasBooking).To(BeTrue())
		})

		It("returns false if no active bookings exist", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*)")).
				WithArgs(int64(9)).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			hasBooking, err := repo.HasBookingForStaff(ctx, 9)

			Expect(err).NotTo(HaveOccurred())
			Expect(hasBooking).To(BeFalse())
		})

		It("propagates a database error", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*)")).
				WithArgs(int64(9)).
				WillReturnError(sql.ErrConnDone)

			hasBooking, err := repo.HasBookingForStaff(ctx, 9)

			Expect(err).To(HaveOccurred())
			Expect(hasBooking).To(BeFalse())
		})
	})
})
