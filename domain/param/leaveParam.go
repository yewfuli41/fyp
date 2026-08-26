package param

import (
	"fyp/domain/errs"
	"strings"
	"time"
	"unicode/utf8"
)

// StaffUnavailability marks one staff member as unable to work across a date
// range, so their slots in it are not offered as reschedule targets. Needed
// while approving a leave: the application is still pending at that point, so
// ILeaveRepo.IsStaffOnLeave — which only counts approved leave — can't see it,
// and the owner would otherwise be offered the very slots the staff member is
// taking off.
type StaffUnavailability struct {
	StaffID int64
	From    string // "YYYY-MM-DD"
	Until   string // "YYYY-MM-DD"
}

// LeaveRescheduleParam is one "move this booking to that slot" decision made
// while approving a leave. The owner settles every affected booking in the UI
// first and the whole set is submitted with the approval, so no customer is
// moved for a leave that is still pending — see
// ILeaveService.ApproveLeaveApplication.
type LeaveRescheduleParam struct {
	BookingID       int64
	NewSlotOptionID int64
}

// LeaveApplicationParam is a staff member's leave request. Deliberately has
// no "type" (annual/personal/medical, ...) — just a date range and a
// free-text reason.
type LeaveApplicationParam struct {
	LeaveID       int64
	StaffID       int64
	BusinessID    int64 // the staff's business — used server-side for owner-scoping checks, not exposed over GraphQL
	StaffName     string
	Position      string
	StartDate     string // "YYYY-MM-DD"
	EndDate       string // "YYYY-MM-DD"
	Justification *string
	// FileURL is a base64 data URI (e.g. "data:application/pdf;base64,...")
	// of a supporting document, stored as-is — no separate file storage.
	FileURL *string
	Status  string // "pending" | "approved" | "rejected"
	Remark        *string
	DecidedAt     *time.Time
	CreatedAt     time.Time

	// AffectedBookings: output only, populated for the owner's review list —
	// the active bookings that would need a replacement staff if this leave
	// is approved.
	AffectedBookings []BookingDetailParam
}

func (p LeaveApplicationParam) ValidateApplyLeave() error {
	var validationErrs errs.ValidationErrors

	if strings.TrimSpace(p.StartDate) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "startDate", Message: "Start date is required"})
	}
	if strings.TrimSpace(p.EndDate) == "" {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "endDate", Message: "End date is required"})
	}
	if len(validationErrs) > 0 {
		return validationErrs
	}

	if _, err := time.Parse("2006-01-02", p.StartDate); err != nil {
		return errs.ValidationErrors{{Field: "startDate", Message: "Invalid start date"}}
	}
	if _, err := time.Parse("2006-01-02", p.EndDate); err != nil {
		return errs.ValidationErrors{{Field: "endDate", Message: "Invalid end date"}}
	}

	today := time.Now().Format("2006-01-02")
	if p.StartDate < today {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "startDate", Message: "Start date cannot be in the past"})
	}
	if p.EndDate < p.StartDate {
		validationErrs = append(validationErrs, errs.ValidationError{Field: "endDate", Message: "End date must be on or after the start date"})
	}
	if p.Justification != nil {
		if e, ok := maxLengthError("justification", "reason", *p.Justification, maxJustificationLength); ok {
			validationErrs = append(validationErrs, e)
		}
	}
	if e, ok := validateFileURL(p.FileURL); !ok {
		validationErrs = append(validationErrs, e)
	}

	if len(validationErrs) > 0 {
		return validationErrs
	}
	return nil
}

// validateFileURL checks an optional base64 data URI attachment. The bool
// reports whether the value is acceptable (true when fileURL is nil).
func validateFileURL(fileURL *string) (errs.ValidationError, bool) {
	if fileURL == nil {
		return errs.ValidationError{}, true
	}
	if !strings.HasPrefix(*fileURL, "data:") {
		return errs.ValidationError{Field: "fileUrl", Message: "Attachment must be an uploaded file."}, false
	}
	if utf8.RuneCountInString(*fileURL) > maxFileURLLength {
		return errs.ValidationError{Field: "fileUrl", Message: "Attachment is too large (max 5 MB)."}, false
	}
	return errs.ValidationError{}, true
}

// ValidateUpdateJustification checks just the reason text — used when
// editing a still-pending application's reason (the date range can't be
// touched once submitted; a new range would need a fresh ApplyLeave call so
// the overlap check runs again).
func (p LeaveApplicationParam) ValidateUpdateJustification() error {
	if p.Justification == nil {
		return nil
	}
	if e, ok := maxLengthError("justification", "reason", *p.Justification, maxJustificationLength); ok {
		return errs.ValidationErrors{e}
	}
	return nil
}

// ValidateReject requires a remark explaining why the leave was rejected —
// mirrors UC-10 S-1's "user fills in justification" step for the owner.
func (p LeaveApplicationParam) ValidateReject() error {
	if p.Remark == nil || strings.TrimSpace(*p.Remark) == "" {
		return errs.ValidationErrors{{Field: "remark", Message: "Please provide a reason for rejecting this leave application."}}
	}
	if e, ok := maxLengthError("remark", "remark", *p.Remark, maxRemarkLength); ok {
		return errs.ValidationErrors{e}
	}
	return nil
}
