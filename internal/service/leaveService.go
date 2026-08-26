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
	serviceRepo     interfaces.IServiceRepo
	emailService    interfaces.IEmailService
	tx              *database.Transaction
}

func NewLeaveService(
	db *sql.DB, leaveRepo interfaces.ILeaveRepo, serviceSlotRepo interfaces.IServiceSlotRepo,
	bookingRepo interfaces.IBookingRepo, businessRepo interfaces.IBusinessRepo,
	serviceRepo interfaces.IServiceRepo, emailService interfaces.IEmailService,
) interfaces.ILeaveService {
	return &leaveService{
		leaveRepo:       leaveRepo,
		serviceSlotRepo: serviceSlotRepo,
		bookingRepo:     bookingRepo,
		businessRepo:    businessRepo,
		serviceRepo:     serviceRepo,
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
// for a pending one, that's the list the owner has to settle before the leave
// can be approved at all (see ApproveLeaveApplication); each booking drops off
// as it is rescheduled elsewhere (it's no longer this staff's booking in this
// date range), so an approved application normally shows none left.
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

// ApproveLeaveApplication only goes through once every booking the leave
// affects has been handed a replacement time: a slot with no booking is freed
// back to owner-managed, but a slot still holding a customer's booking blocks
// the approval outright, since granting the leave would otherwise leave that
// customer with nobody to serve them.
// reschedules, when supplied, moves each named booking onto a new slot as part
// of the same transaction as the approval itself. Nothing is written until
// every one of them has been validated, so an approval the owner abandons
// half-way through — or one that fails on the last booking — leaves the leave
// pending and every booking exactly where the customer left it. Without that,
// each booking was committed (and the customer emailed) the moment it was
// picked, stranding customers on new times for leave that was never granted.
func (s *leaveService) ApproveLeaveApplication(
	ctx context.Context, businessID int64, leaveID int64, reschedules []param.LeaveRescheduleParam,
) (*param.LeaveApplicationParam, error) {
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

	moves, err := s.planLeaveReschedules(ctx, businessID, leave, reschedules)
	if err != nil {
		return nil, err
	}

	affected, err := s.serviceSlotRepo.GetAssignedSlotsInRange(ctx, leave.StaffID, leave.StartDate, leave.EndDate)
	if err != nil {
		return nil, err
	}

	movedSlots := make(map[int64]bool, len(moves))
	for _, m := range moves {
		movedSlots[m.booking.ServiceSlotID] = true
	}

	// Approving is all-or-nothing: a customer already booked with this staff
	// member has to be handed a new time — on someone else's slot — before the
	// leave can go through, or approving would quietly leave that booking with
	// nobody to serve it. The owner settles every affected booking first.
	for _, slot := range affected {
		if slot.HasBooking && !movedSlots[slot.ServiceSlotID] {
			return nil, errs.ValidationErrors{{
				Field:   "reschedules",
				Message: "Every booking affected by this leave must be rescheduled to a replacement staff member's time slot before the leave can be approved.",
			}}
		}
	}

	err = s.tx.WithTransaction(ctx, func(tx *sql.Tx) error {
		for _, m := range moves {
			if err := s.bookingRepo.UpdateBookingSlotOption(ctx, tx, m.booking.BookingID, m.newSlotOptionID, "rescheduled"); err != nil {
				return err
			}
			// The slot being vacated may offer an option whose effective
			// window has since closed — same cleanup RescheduleBooking does.
			if err := s.serviceRepo.DropSlotOptionIfExpired(ctx, tx, m.booking.SlotOptionID); err != nil {
				return err
			}
		}
		// Every remaining slot is now free of bookings (the guard above
		// proved it), so it is safe to unassign the whole range.
		for _, slot := range affected {
			if err := s.serviceSlotRepo.ReassignStaff(ctx, tx, slot.ServiceSlotID, businessID, nil); err != nil {
				return err
			}
		}
		return s.leaveRepo.UpdateLeaveStatus(ctx, tx, leaveID, "approved", nil)
	})
	if err != nil {
		if database.IsUniqueViolation(err, database.ConstraintActiveBookingPerSlot) {
			return nil, errs.ValidationErrors{{Field: "reschedules", Message: "One of the selected time slots was just taken. Please pick another."}}
		}
		return nil, err
	}

	// Only once the whole approval is committed does anyone hear about it.
	for _, m := range moves {
		s.notifyRescheduled(ctx, m.booking.BookingID)
	}

	return s.leaveRepo.GetLeaveApplicationByID(ctx, leaveID)
}

type leaveMove struct {
	booking         param.BookingDetailParam
	newSlotOptionID int64
}

// planLeaveReschedules validates every requested move up front — before the
// transaction opens — so a bad pick is reported without touching anything.
func (s *leaveService) planLeaveReschedules(
	ctx context.Context, businessID int64, leave *param.LeaveApplicationParam, reschedules []param.LeaveRescheduleParam,
) ([]leaveMove, error) {
	if len(reschedules) == 0 {
		return nil, nil
	}

	bookings, err := s.bookingRepo.GetBusinessBookings(ctx, businessID, &leave.StaffID)
	if err != nil {
		return nil, err
	}
	affected := make(map[int64]param.BookingDetailParam, len(bookings))
	for _, b := range bookings {
		if b.Date >= leave.StartDate && b.Date <= leave.EndDate {
			affected[b.BookingID] = b
		}
	}

	moves := make([]leaveMove, 0, len(reschedules))
	seenBooking := make(map[int64]bool, len(reschedules))
	seenTarget := make(map[int64]bool, len(reschedules))
	for _, r := range reschedules {
		booking, ok := affected[r.BookingID]
		if !ok {
			return nil, errs.ValidationErrors{{Field: "reschedules", Message: "A booking being rescheduled is not affected by this leave."}}
		}
		if seenBooking[r.BookingID] {
			return nil, errs.ValidationErrors{{Field: "reschedules", Message: "The same booking was rescheduled twice."}}
		}
		// Two bookings sent to one slot would otherwise be caught only by the
		// database constraint, half-way through the transaction.
		if seenTarget[r.NewSlotOptionID] {
			return nil, errs.ValidationErrors{{Field: "reschedules", Message: "Two bookings were moved onto the same time slot."}}
		}
		seenBooking[r.BookingID] = true
		seenTarget[r.NewSlotOptionID] = true

		available, err := s.bookingRepo.SlotOptionIsAvailable(ctx, r.NewSlotOptionID)
		if err != nil {
			return nil, err
		}
		if !available {
			return nil, errs.ValidationErrors{{Field: "reschedules", Message: "One of the selected time slots is no longer available."}}
		}

		// Moving a booking onto one of this staff member's own slots inside
		// the very leave being approved is no better than leaving it where it
		// is — they're off on that date either way. The picker already hides
		// these, but that's a client-side rule; enforce it here too so a stale
		// page (or anything not going through the picker) can't slip one past.
		targetStaffID, targetDate, err := s.bookingRepo.GetSlotOptionAssignment(ctx, r.NewSlotOptionID)
		if err != nil {
			return nil, err
		}
		if targetStaffID != nil && *targetStaffID == leave.StaffID &&
			targetDate >= leave.StartDate && targetDate <= leave.EndDate {
			return nil, errs.ValidationErrors{{
				Field:   "reschedules",
				Message: "One of the selected time slots is this staff member's own, on a day covered by this leave. Pick a slot on another date, or one covered by someone else.",
			}}
		}
		moves = append(moves, leaveMove{booking: booking, newSlotOptionID: r.NewSlotOptionID})
	}
	return moves, nil
}

// notifyRescheduled emails the customer that the business moved their booking.
// Best-effort: the reschedule is already committed, so a failed send is logged
// rather than surfaced as an approval failure.
func (s *leaveService) notifyRescheduled(ctx context.Context, bookingID int64) {
	c, err := s.bookingRepo.GetBookingContext(ctx, bookingID)
	if err != nil {
		log.Errorf("failed to load booking context for leave-reschedule email (booking %d): %v", bookingID, err)
		return
	}
	if c.CustomerEmail == "" {
		return
	}
	if err := s.emailService.SendBookingStatusEmail(c.CustomerEmail, c.CustomerName, c.BusinessName, "rescheduled"); err != nil {
		log.Errorf("failed to send leave-reschedule email to %s: %v", c.CustomerEmail, err)
	}
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
