package interfaces

import (
	"context"
	"database/sql"
	"fyp/domain/param"
)

type ILeaveRepo interface {
	// InsertLeaveApplication returns just the new row's ID — fetch the full
	// record via GetLeaveApplicationByID only after the transaction commits.
	InsertLeaveApplication(ctx context.Context, tx *sql.Tx, p param.LeaveApplicationParam) (int64, error)
	GetLeaveApplicationsByStaffID(ctx context.Context, staffID int64) ([]param.LeaveApplicationParam, error)
	// GetLeaveApplicationsByBusinessID returns every leave application (any
	// status) for the business, each carrying its staff's name/position and
	// businessID (for scoping) — but not AffectedBookings, which the service
	// layer fills in per-application.
	GetLeaveApplicationsByBusinessID(ctx context.Context, businessID int64) ([]param.LeaveApplicationParam, error)
	// GetLeaveApplicationByID returns a single application with its staff's
	// businessID/name/position joined in, so callers can check ownership
	// (StaffID for the applicant, BusinessID for the owner) without a second
	// lookup.
	GetLeaveApplicationByID(ctx context.Context, leaveID int64) (*param.LeaveApplicationParam, error)
	UpdateLeaveStatus(ctx context.Context, tx *sql.Tx, leaveID int64, status string, remark *string) error
	// UpdateLeaveJustification rewrites just the reason text — the service
	// layer is responsible for only allowing this while still pending.
	UpdateLeaveJustification(ctx context.Context, tx *sql.Tx, leaveID int64, justification *string) error
	SoftDeleteLeaveApplication(ctx context.Context, tx *sql.Tx, leaveID int64) error
	// HasOverlappingLeave reports whether the staff already has a pending or
	// approved application covering any day in [startDate, endDate].
	HasOverlappingLeave(ctx context.Context, staffID int64, startDate, endDate string) (bool, error)
	// IsStaffOnLeave reports whether the staff has an approved leave
	// application covering the given date — pending applications don't count,
	// since they haven't been decided yet and shouldn't block scheduling.
	IsStaffOnLeave(ctx context.Context, staffID int64, date string) (bool, error)
}

type ILeaveService interface {
	ApplyLeave(ctx context.Context, staffID int64, p param.LeaveApplicationParam) (*param.LeaveApplicationParam, error)
	// UpdateLeaveApplication lets the applying staff member edit just their
	// reason while the application is still pending.
	UpdateLeaveApplication(ctx context.Context, staffID int64, leaveID int64, justification *string) (*param.LeaveApplicationParam, error)
	DeleteLeaveApplication(ctx context.Context, staffID int64, leaveID int64) error
	GetMyLeaveApplications(ctx context.Context, staffID int64) ([]param.LeaveApplicationParam, error)
	GetBusinessLeaveApplications(ctx context.Context, businessID int64) ([]param.LeaveApplicationParam, error)
	// ApproveLeaveApplication never blocks on picking replacements — see its
	// doc comment for why.
	ApproveLeaveApplication(ctx context.Context, businessID int64, leaveID int64) (*param.LeaveApplicationParam, error)
	// RejectLeaveApplication also reverses an already-approved leave (moving
	// it to rejected, with a required remark) — there is no separate cancel.
	RejectLeaveApplication(ctx context.Context, businessID int64, leaveID int64, remark string) (*param.LeaveApplicationParam, error)
}
