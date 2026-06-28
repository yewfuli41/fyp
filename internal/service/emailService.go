package service

import (
	"fmt"
	"fyp/config"
	"fyp/internal/interfaces"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type sendGridEmailService struct {
	emailCfg config.EmailConfig
}

func NewSendGridEmailService(emailCfg config.EmailConfig) interfaces.IEmailService {
	return &sendGridEmailService{
		emailCfg: emailCfg,
	}
}

func (e *sendGridEmailService) SendStaffWelcomeEmail(toEmail, tempPassword string) error {
	from := mail.NewEmail(e.emailCfg.FromName, e.emailCfg.FromEmail)
	to := mail.NewEmail("", toEmail)
	subject := "Welcome — your staff account is ready"
	plainText := fmt.Sprintf("You have been added as a staff member.\n\nEmail: %s\nTemporary password: %s\n\nPlease log in and change your password.", toEmail, tempPassword)
	htmlContent := fmt.Sprintf("<p>You have been added as a staff member.</p><p><strong>Email:</strong> %s<br><strong>Temporary password:</strong> %s</p><p>Please log in and change your password.</p>", toEmail, tempPassword)

	message := mail.NewSingleEmail(from, subject, to, plainText, htmlContent)
	client := sendgrid.NewSendClient(e.emailCfg.SendGridAPIKey)
	resp, err := client.Send(message)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid returned status %d: %s", resp.StatusCode, resp.Body)
	}
	return nil
}
