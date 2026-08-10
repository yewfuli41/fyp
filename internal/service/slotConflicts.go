package service

import (
	"context"
	"database/sql"
	"fmt"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/interfaces"
	"strings"
	"time"

	"github.com/labstack/gommon/log"
)

// coveredByHours reports whether [start,end) on weekday is fully contained
// in some entry of hours for that day.
func coveredByHours(weekday string, start, end time.Time, hours []param.WorkingHourParam) bool {
	for _, wh := range hours {
		if wh.Day != weekday {
			continue
		}
		whStart := timeOfDay(wh.StartTime)
		whEnd := timeOfDay(wh.EndTime)
		if !start.Before(whStart) && !end.After(whEnd) {
			return true
		}
	}
	return false
}

// slotsOutsideHours filters assigned slots down to the ones whose window
// isn't covered by newHours on their own weekday.
func slotsOutsideHours(slots []param.AssignedSlotParam, newHours []param.WorkingHourParam) []param.AssignedSlotParam {
	var outside []param.AssignedSlotParam
	for _, s := range slots {
		weekday, err := weekdayOf(s.Date)
		if err != nil {
			continue
		}
		if !coveredByHours(weekday, timeOfDay(s.StartTime), timeOfDay(s.EndTime), newHours) {
			outside = append(outside, s)
		}
	}
	return outside
}

// resolveSlotConflicts is used by staffService.UpdateStaffWorkingHours to
// take a staff member's slots that fall outside their new hours out of
// service, and can't just delete them: a booked slot needs an explicit
// replacement staff (validated the same way ReassignServiceSlotStaff already
// validates one), while a non-booked slot can simply be unassigned back to
// owner-managed. (leaveService.ApproveLeaveApplication handles its own,
// simpler case directly — approving never blocks on picking a same-time
// replacement; see its doc comment.) Must run inside the caller's
// transaction; returns the booking contexts to email once it commits (see
// notifyStaffReassigned).
func resolveSlotConflicts(
	ctx context.Context, tx *sql.Tx,
	serviceSlotRepo interfaces.IServiceSlotRepo, bookingRepo interfaces.IBookingRepo,
	businessID int64, affected []param.AssignedSlotParam, reassignments []param.SlotReassignmentParam,
) ([]*param.BookingContextParam, error) {
	byID := make(map[int64]*param.SlotReassignmentParam, len(reassignments))
	for i := range reassignments {
		byID[reassignments[i].ServiceSlotID] = &reassignments[i]
	}

	var missingDates []string
	var toNotify []*param.BookingContextParam
	for _, slot := range affected {
		if !slot.HasBooking {
			if err := serviceSlotRepo.ReassignStaff(ctx, tx, slot.ServiceSlotID, businessID, nil); err != nil {
				return nil, err
			}
			continue
		}

		r, ok := byID[slot.ServiceSlotID]
		if !ok {
			missingDates = append(missingDates, slot.Date)
			continue
		}
		if r.StaffID != nil {
			staffOK, err := serviceSlotRepo.StaffBelongsToBusiness(ctx, *r.StaffID, businessID)
			if err != nil {
				return nil, err
			}
			if !staffOK {
				return nil, errs.ValidationErrors{{Field: "reassignments", Message: "Selected replacement staff was not found."}}
			}
			weekday, err := weekdayOf(slot.Date)
			if err != nil {
				return nil, err
			}
			covers, err := serviceSlotRepo.StaffCoversTime(ctx, *r.StaffID, weekday, slot.StartTime, slot.EndTime)
			if err != nil {
				return nil, err
			}
			if !covers {
				return nil, errs.ValidationErrors{{Field: "reassignments", Message: fmt.Sprintf("Replacement staff is not working on %s during that time.", slot.Date)}}
			}
		}
		if err := serviceSlotRepo.ReassignStaff(ctx, tx, slot.ServiceSlotID, businessID, r.StaffID); err != nil {
			return nil, err
		}
		bc, err := bookingRepo.GetBookingContextForSlot(ctx, slot.ServiceSlotID)
		if err != nil {
			return nil, err
		}
		if bc != nil {
			toNotify = append(toNotify, bc)
		}
	}
	if len(missingDates) > 0 {
		return nil, errs.ValidationErrors{{
			Field:   "reassignments",
			Message: fmt.Sprintf("Please pick a replacement staff for the affected booking(s) on: %s.", strings.Join(missingDates, ", ")),
		}}
	}
	return toNotify, nil
}

// notifyStaffReassigned emails every affected customer that their booking's
// staff has changed. Best-effort — logs failures without erroring the
// caller, matching bookingService.notify's fire-and-forget style.
func notifyStaffReassigned(emailService interfaces.IEmailService, contexts []*param.BookingContextParam) {
	for _, c := range contexts {
		if c.CustomerEmail == "" {
			continue
		}
		if err := emailService.SendStaffReassignedEmail(c.CustomerEmail, c.CustomerName, c.BusinessName, c.WhenText); err != nil {
			log.Errorf("failed to send staff-reassigned email to %s: %v", c.CustomerEmail, err)
		}
	}
}
