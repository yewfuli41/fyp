package interfaces

type IEmailService interface {
	SendStaffWelcomeEmail(toEmail string) error
	// SendBookingStatusEmail — withName is whoever the booking is "with" from
	// the recipient's side: the business name for a customer, or the
	// customer's name for the owner/staff (never their own business name).
	SendBookingStatusEmail(toEmail, recipientName, withName, statusLabel, whenText string) error
	// SendNewBookingRequestEmail notifies the business side (owner and, if
	// assigned, the staff member) that a customer just placed a new booking
	// request — sent once, right when CreateBooking succeeds, before anyone
	// has accepted/rejected it.
	SendNewBookingRequestEmail(toEmail, recipientName, customerName, whenText string) error
	// SendLeaveApplicationSubmittedEmail notifies the business owner that a
	// staff member has applied for leave.
	SendLeaveApplicationSubmittedEmail(toEmail, ownerName, staffName, startDate, endDate string) error
	// SendStaffReassignedEmail notifies a customer that their booking's
	// assigned staff has changed (due to a leave approval or a working-hours edit).
	SendStaffReassignedEmail(toEmail, recipientName, businessName, whenText string) error
}
