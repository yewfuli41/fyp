package service

import (
	"context"
	"database/sql"
	"errors"
	"fyp/database"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"strings"

	"github.com/labstack/gommon/log"
)

type leaveService struct {
	leaveRepo       interfaces.ILeaveRepo
	serviceSlotRepo interfaces.IServiceSlotRepo
	bookingRepo     interfaces.IBookingRepo
	businessRepo    interfaces.IBusinessRepo
	emailService    interfaces.IEmailService
	tx              *database.Transaction
}

func NewLeaveService(
	db *sql.DB, leaveRepo interfaces.ILeaveRepo, serviceSlotRepo interfaces.IServiceSlotRepo,
	bookingRepo interfaces.IBookingRepo, businessRepo interfaces.IBusinessRepo, emailService interfaces.IEmailService,
) interfaces.ILeaveService {
	return &leaveService{
		leaveRepo:       leaveRepo,
		serviceSlotRepo: serviceSlotRepo,
		bookingRepo:     bookingRepo,
		businessRepo:    businessRepo,
		emailService:    emailService,
		tx:              database.NewTransaction(db),
	}
}

func leaveNotFound() error {
	return errs.ValidationErrors{{Field: "leaveId", Message: "Leave application not found."}}
}

func (s *leaveService) getOwnLeave(ctx context.Context, leaveID int64) (*param.LeaveApplicationParam, error) {
	leave, err := s.leaveRepo.GetLeaveApplicationByID(ctx, leaveID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, leaveNotFound()
		}
		return nil, err
	}
	return leave, nil
}

func (s *leaveService) ApplyLeave(ctx context.Context, staffID int64, p param.LeaveApplicationParam) (*param.LeaveApplicationParam, error) {
	p.StaffID = staffID
	if err := p.ValidateApplyLeave(); err != nil {
		return nil, err
	}

	overlaps, err := s.leaveRepo.HasOverlappingLeave(ctx, staffID, p.StartDate, p.EndDate)
	if err != nil {
		return nil, err
	}
	if overlaps {
		return nil, errs.ValidationErrors{{Field: "startDate", Message: "You already have a leave application covering part of this date range."}}
	}

	var leaveID int64
	err = s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		id, err := s.leaveRepo.InsertLeaveApplication(ctx, tx, p)
		if err != nil {
			return err
		}
		leaveID = id
		return nil
	})
	if err != nil {
		return nil, err
	}

	created, err := s.leaveRepo.GetLeaveApplicationByID(ctx, leaveID)
	if err != nil {
		return nil, err
	}

	if business, err := s.businessRepo.GetBusinessByID(ctx, created.BusinessID); err == nil && business.BusinessEmail != "" {
		if err := s.emailService.SendLeaveApplicationSubmittedEmail(business.BusinessEmail, business.BusinessName, created.StaffName, created.StartDate, created.EndDate); err != nil {
			log.Errorf("failed to send leave-application-submitted email: %v", err)
		}
	}
	return created, nil
}

// UpdateLeaveApplication lets the applying staff member edit just their
// reason while the application is still awaiting a decision — the date
// range can't be changed here since a new range needs the overlap check a
// fresh ApplyLeave call already runs.
func (s *leaveService) UpdateLeaveApplication(ctx context.Context, staffID int64, leaveID int64, justification *string) (*param.LeaveApplicationParam, error) {
	leave, err := s.getOwnLeave(ctx, leaveID)
	if err != nil {
		return nil, err
	}
	if leave.StaffID != staffID {
		return nil, leaveNotFound()
	}
	if leave.Status != "pending" {
		return nil, errs.ValidationErrors{{Field: "leaveId", Message: "Only pending leave applications can be edited."}}
	}

	update := param.LeaveApplicationParam{Justification: justification}
	if err := update.ValidateUpdateJustification(); err != nil {
		return nil, err
	}
	if justification != nil {
		trimmed := strings.TrimSpace(*justification)
		if trimmed == "" {
			justification = nil
		} else {
			justification = &trimmed
		}
	}

	err = s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		return s.leaveRepo.UpdateLeaveJustification(ctx, tx, leaveID, justification)
	})
	if err != nil {
		return nil, err
	}
	return s.leaveRepo.GetLeaveApplicationByID(ctx, leaveID)
}

func (s *leaveService) DeleteLeaveApplication(ctx context.Context, staffID int64, leaveID int64) error {
	leave, err := s.getOwnLeave(ctx, leaveID)
	if err != nil {
		return err
	}
	if leave.StaffID != staffID {
		return leaveNotFound()
	}
	return s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		return s.leaveRepo.SoftDeleteLeaveApplication(ctx, tx, leaveID)
	})
}

func (s *leaveService) GetMyLeaveApplications(ctx context.Context, staffID int64) ([]param.LeaveApplicationParam, error) {
	return s.leaveRepo.GetLeaveApplicationsByStaffID(ctx, staffID)
}

// GetBusinessLeaveApplications attaches, to every PENDING or APPROVED
// application, the active bookings its staff still has in that date range —
// for a pending one, that's what the owner should weigh before deciding; for
// an approved one, approving never auto-resolves a booked slot (see
// ApproveLeaveApplication), so this keeps listing each one under the leave
// until the owner reschedules it elsewhere, at which point it naturally
// drops off (it's no longer this staff's booking in this date range).
// Rejected applications skip this (nothing left to act on).
func (s *leaveService) GetBusinessLeaveApplications(ctx context.Context, businessID int64) ([]param.LeaveApplicationParam, error) {
	apps, err := s.leaveRepo.GetLeaveApplicationsByBusinessID(ctx, businessID)
	if err != nil {
		return nil, err
	}
	for i := range apps {
		if apps[i].Status == "rejected" {
			continue
		}
		staffID := apps[i].StaffID
		bookings, err := s.bookingRepo.GetBusinessBookings(ctx, businessID, &staffID)
		if err != nil {
			return nil, err
		}
		for _, b := range bookings {
			if b.Date >= apps[i].StartDate && b.Date <= apps[i].EndDate {
				apps[i].AffectedBookings = append(apps[i].AffectedBookings, b)
			}
		}
	}
	return apps, nil
}

// ApproveLeaveApplication never blocks on picking replacements: a slot with
// no booking is freed back to owner-managed right away, while a booked slot
// is simply left alone (still assigned, still visible) — the customer's
// booking is unaffected until the owner explicitly reschedules it (via the
// ordinary RescheduleBooking flow, which already requires the customer to
// accept the new time). GetBusinessLeaveApplications keeps surfacing any
// such booking under this leave as AffectedBookings until that happens.
func (s *leaveService) ApproveLeaveApplication(ctx context.Context, businessID int64, leaveID int64) (*param.LeaveApplicationParam, error) {
	leave, err := s.getOwnLeave(ctx, leaveID)
	if err != nil {
		return nil, err
	}
	if leave.BusinessID != businessID {
		return nil, leaveNotFound()
	}
	if leave.Status != "pending" {
		return nil, errs.ValidationErrors{{Field: "leaveId", Message: "Only pending leave applications can be approved."}}
	}

	affected, err := s.serviceSlotRepo.GetAssignedSlotsInRange(ctx, leave.StaffID, leave.StartDate, leave.EndDate)
	if err != nil {
		return nil, err
	}

	err = s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		for _, slot := range affected {
			if slot.HasBooking {
				continue
			}
			if err := s.serviceSlotRepo.ReassignStaff(ctx, tx, slot.ServiceSlotID, businessID, nil); err != nil {
				return err
			}
		}
		return s.leaveRepo.UpdateLeaveStatus(ctx, tx, leaveID, "approved", nil)
	})
	if err != nil {
		return nil, err
	}

	return s.leaveRepo.GetLeaveApplicationByID(ctx, leaveID)
}

// RejectLeaveApplication also doubles as "reverse an approval": rejecting an
// already-approved leave moves it to rejected too (with a required remark
// either way) rather than just disappearing via a silent cancel — it does
// not undo any staff reassignments already made when it was approved.
func (s *leaveService) RejectLeaveApplication(ctx context.Context, businessID int64, leaveID int64, remark string) (*param.LeaveApplicationParam, error) {
	leave, err := s.getOwnLeave(ctx, leaveID)
	if err != nil {
		return nil, err
	}
	if leave.BusinessID != businessID {
		return nil, leaveNotFound()
	}
	if leave.Status != "pending" && leave.Status != "approved" {
		return nil, errs.ValidationErrors{{Field: "leaveId", Message: "Only pending or approved leave applications can be rejected."}}
	}

	rejection := param.LeaveApplicationParam{Remark: &remark}
	if err := rejection.ValidateReject(); err != nil {
		return nil, err
	}

	err = s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		return s.leaveRepo.UpdateLeaveStatus(ctx, tx, leaveID, "rejected", &remark)
	})
	if err != nil {
		return nil, err
	}
	return s.leaveRepo.GetLeaveApplicationByID(ctx, leaveID)
}
