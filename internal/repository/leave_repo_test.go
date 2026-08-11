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

var _ = Describe("LeaveRepo", func() {
	var (
		db        *sql.DB
		mock      sqlmock.Sqlmock
		repo      interfaces.ILeaveRepo
		ctx       context.Context
		leaveCols []string
	)

	BeforeEach(func() {
		var err error
		db, mock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())

		repo = repository.NewLeaveRepo(db)
		ctx = context.Background()
		leaveCols = []string{
			"leave_id", "staff_id", "business_id", "staff_name", "position",
			"start_date", "end_date",
			"justification", "status", "remark", "decided_at", "created_at",
		}
	})

	AfterEach(func() {
		Expect(mock.ExpectationsWereMet()).To(Succeed())
	})

	Describe("InsertLeaveApplication", func() {
		It("inserts a new leave application using the provided transaction", func() {
			justification := "Family trip"
			p := param.LeaveApplicationParam{
				StaffID:       5,
				StartDate:     "2026-08-10",
				EndDate:       "2026-08-12",
				Justification: &justification,
			}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO fyp_fuli_leave_applications")).
				WithArgs(p.StaffID, p.StartDate, p.EndDate, p.Justification).
				WillReturnRows(sqlmock.NewRows([]string{"leave_id"}).AddRow(42))

			leaveID, err := repo.InsertLeaveApplication(ctx, tx, p)

			Expect(err).NotTo(HaveOccurred())
			Expect(leaveID).To(Equal(int64(42)))
		})

		It("propagates a database error", func() {
			p := param.LeaveApplicationParam{
				StaffID:   5,
				StartDate: "2026-08-10",
				EndDate:   "2026-08-12",
			}

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO fyp_fuli_leave_applications")).
				WithArgs(p.StaffID, p.StartDate, p.EndDate, p.Justification).
				WillReturnError(sql.ErrConnDone)

			leaveID, err := repo.InsertLeaveApplication(ctx, tx, p)

			Expect(err).To(HaveOccurred())
			Expect(leaveID).To(Equal(int64(0)))
		})
	})

	Describe("GetLeaveApplicationsByStaffID", func() {
		It("returns leave applications for a staff member", func() {
			decidedAt := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
			createdAt := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_leave_applications la")).
				WithArgs(int64(5)).
				WillReturnRows(sqlmock.NewRows(leaveCols).
					AddRow(1, 5, 2, "Alice", "Therapist", "2026-08-10", "2026-08-12", "Family trip", "approved", nil, decidedAt, createdAt).
					AddRow(2, 5, 2, "Alice", "Therapist", "2026-09-01", "2026-09-02", nil, "pending", nil, nil, createdAt))

			results, err := repo.GetLeaveApplicationsByStaffID(ctx, 5)

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0].LeaveID).To(Equal(int64(1)))
			Expect(results[0].StaffName).To(Equal("Alice"))
			Expect(*results[0].Justification).To(Equal("Family trip"))
			Expect(results[0].DecidedAt).NotTo(BeNil())
			Expect(results[1].Justification).To(BeNil())
			Expect(results[1].DecidedAt).To(BeNil())
		})

		It("returns an empty slice when no applications exist", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_leave_applications la")).
				WithArgs(int64(99)).
				WillReturnRows(sqlmock.NewRows(leaveCols))

			results, err := repo.GetLeaveApplicationsByStaffID(ctx, 99)

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})

		It("propagates a database error", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_leave_applications la")).
				WithArgs(int64(5)).
				WillReturnError(sql.ErrConnDone)

			results, err := repo.GetLeaveApplicationsByStaffID(ctx, 5)

			Expect(err).To(HaveOccurred())
			Expect(results).To(BeNil())
		})
	})

	Describe("GetLeaveApplicationsByBusinessID", func() {
		It("returns leave applications for a business", func() {
			createdAt := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_leave_applications la")).
				WithArgs(int64(2)).
				WillReturnRows(sqlmock.NewRows(leaveCols).
					AddRow(1, 5, 2, "Alice", "Therapist", "2026-08-10", "2026-08-12", "Family trip", "pending", nil, nil, createdAt))

			results, err := repo.GetLeaveApplicationsByBusinessID(ctx, 2)

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].BusinessID).To(Equal(int64(2)))
			Expect(results[0].Status).To(Equal("pending"))
		})

		It("returns an empty slice when no applications exist", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_leave_applications la")).
				WithArgs(int64(999)).
				WillReturnRows(sqlmock.NewRows(leaveCols))

			results, err := repo.GetLeaveApplicationsByBusinessID(ctx, 999)

			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeEmpty())
		})

		It("propagates a database error", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_leave_applications la")).
				WithArgs(int64(2)).
				WillReturnError(sql.ErrConnDone)

			results, err := repo.GetLeaveApplicationsByBusinessID(ctx, 2)

			Expect(err).To(HaveOccurred())
			Expect(results).To(BeNil())
		})
	})

	Describe("GetLeaveApplicationByID", func() {
		It("returns the leave application for a given ID", func() {
			createdAt := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)

			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_leave_applications la")).
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows(leaveCols).
					AddRow(1, 5, 2, "Alice", "Therapist", "2026-08-10", "2026-08-12", "Family trip", "pending", nil, nil, createdAt))

			result, err := repo.GetLeaveApplicationByID(ctx, 1)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.LeaveID).To(Equal(int64(1)))
			Expect(result.StaffID).To(Equal(int64(5)))
		})

		It("returns sql.ErrNoRows when the application is not found", func() {
			mock.ExpectQuery(regexp.QuoteMeta("FROM fyp_fuli_leave_applications la")).
				WithArgs(int64(999)).
				WillReturnError(sql.ErrNoRows)

			result, err := repo.GetLeaveApplicationByID(ctx, 999)

			Expect(err).To(Equal(sql.ErrNoRows))
			Expect(result).To(BeNil())
		})
	})

	Describe("UpdateLeaveStatus", func() {
		It("updates the status and remark of a leave application", func() {
			remark := "Approved by manager"

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_leave_applications")).
				WithArgs("approved", &remark, int64(1)).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err := repo.UpdateLeaveStatus(ctx, tx, 1, "approved", &remark)

			Expect(err).NotTo(HaveOccurred())
		})

		It("propagates a database error", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_leave_applications")).
				WithArgs("rejected", (*string)(nil), int64(1)).
				WillReturnError(sql.ErrConnDone)

			err := repo.UpdateLeaveStatus(ctx, tx, 1, "rejected", nil)

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("UpdateLeaveJustification", func() {
		It("updates the justification of a leave application", func() {
			justification := "Updated reason"

			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_leave_applications")).
				WithArgs(&justification, int64(1)).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err := repo.UpdateLeaveJustification(ctx, tx, 1, &justification)

			Expect(err).NotTo(HaveOccurred())
		})

		It("propagates a database error", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_leave_applications")).
				WithArgs((*string)(nil), int64(1)).
				WillReturnError(sql.ErrConnDone)

			err := repo.UpdateLeaveJustification(ctx, tx, 1, nil)

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("SoftDeleteLeaveApplication", func() {
		It("soft deletes a leave application", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_leave_applications SET deleted_at = NOW()")).
				WithArgs(int64(1)).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err := repo.SoftDeleteLeaveApplication(ctx, tx, 1)

			Expect(err).NotTo(HaveOccurred())
		})

		It("propagates a database error", func() {
			mock.ExpectBegin()
			tx, _ := db.Begin()

			mock.ExpectExec(regexp.QuoteMeta("UPDATE fyp_fuli_leave_applications SET deleted_at = NOW()")).
				WithArgs(int64(1)).
				WillReturnError(sql.ErrConnDone)

			err := repo.SoftDeleteLeaveApplication(ctx, tx, 1)

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("HasOverlappingLeave", func() {
		It("returns true when an overlapping leave application exists", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(")).
				WithArgs(int64(5), "2026-08-10", "2026-08-12").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

			ok, err := repo.HasOverlappingLeave(ctx, 5, "2026-08-10", "2026-08-12")

			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("returns false when no overlapping leave application exists", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(")).
				WithArgs(int64(5), "2026-08-10", "2026-08-12").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

			ok, err := repo.HasOverlappingLeave(ctx, 5, "2026-08-10", "2026-08-12")

			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

		It("propagates a database error", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(")).
				WithArgs(int64(5), "2026-08-10", "2026-08-12").
				WillReturnError(sql.ErrConnDone)

			ok, err := repo.HasOverlappingLeave(ctx, 5, "2026-08-10", "2026-08-12")

			Expect(err).To(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})

	Describe("IsStaffOnLeave", func() {
		It("returns true when the staff has an approved leave on the given date", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(")).
				WithArgs(int64(5), "2026-08-11").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

			ok, err := repo.IsStaffOnLeave(ctx, 5, "2026-08-11")

			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("returns false when the staff is not on approved leave", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(")).
				WithArgs(int64(5), "2026-08-11").
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

			ok, err := repo.IsStaffOnLeave(ctx, 5, "2026-08-11")

			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

		It("propagates a database error", func() {
			mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(")).
				WithArgs(int64(5), "2026-08-11").
				WillReturnError(sql.ErrConnDone)

			ok, err := repo.IsStaffOnLeave(ctx, 5, "2026-08-11")

			Expect(err).To(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})
})
