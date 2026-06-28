package interfaces

type IEmailService interface {
	SendStaffWelcomeEmail(toEmail, tempPassword string) error
}
