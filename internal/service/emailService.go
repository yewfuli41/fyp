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
	plainText := fmt.Sprintf("You have been added as a staff member.\n\nEmail: %s\nTemporary password: %s\n\nLog in with these details — you'll be asked to set your own password straight away.", toEmail, tempPassword)
	htmlContent := fmt.Sprintf("<p>You have been added as a staff member.</p><p><strong>Email:</strong> %s<br><strong>Temporary password:</strong> %s</p><p>Log in with these details — you'll be asked to set your own password straight away.</p>", toEmail, tempPassword)

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

func (e *sendGridEmailService) SendBookingStatusEmail(toEmail, recipientName, withName, statusLabel string) error {
	from := mail.NewEmail(e.emailCfg.FromName, e.emailCfg.FromEmail)
	to := mail.NewEmail(recipientName, toEmail)
	subject := fmt.Sprintf("Your booking with %s is now %s", withName, statusLabel)
	plainText := fmt.Sprintf("Hi %s,\n\nYour booking with %s is now %s.\n\nThank you.", recipientName, withName, statusLabel)
	htmlContent := fmt.Sprintf("<p>Hi %s,</p><p>Your booking with <strong>%s</strong> is now <strong>%s</strong>.</p><p>Thank you.</p>", recipientName, withName, statusLabel)

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

func (e *sendGridEmailService) SendNewBookingRequestEmail(toEmail, recipientName, customerName, whenText string) error {
	from := mail.NewEmail(e.emailCfg.FromName, e.emailCfg.FromEmail)
	to := mail.NewEmail("", toEmail)
	subject := fmt.Sprintf("New booking request from %s", customerName)
	plainText := fmt.Sprintf("Hi %s,\n\n%s just requested a booking for %s.\n\nPlease accept or reject it from your calendar.", recipientName, customerName, whenText)
	htmlContent := fmt.Sprintf("<p>Hi %s,</p><p><strong>%s</strong> just requested a booking for <strong>%s</strong>.</p><p>Please accept or reject it from your calendar.</p>", recipientName, customerName, whenText)

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

func (e *sendGridEmailService) SendLeaveApplicationSubmittedEmail(toEmail, ownerName, staffName, startDate, endDate string) error {
	from := mail.NewEmail(e.emailCfg.FromName, e.emailCfg.FromEmail)
	to := mail.NewEmail(ownerName, toEmail)
	subject := fmt.Sprintf("%s applied for leave", staffName)
	plainText := fmt.Sprintf("Hi %s,\n\n%s has applied for leave from %s to %s.\n\nPlease review it in Staff Availability.", ownerName, staffName, startDate, endDate)
	htmlContent := fmt.Sprintf("<p>Hi %s,</p><p><strong>%s</strong> has applied for leave from <strong>%s</strong> to <strong>%s</strong>.</p><p>Please review it in Staff Availability.</p>", ownerName, staffName, startDate, endDate)

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

func (e *sendGridEmailService) SendStaffReassignedEmail(toEmail, recipientName, businessName, whenText string) error {
	from := mail.NewEmail(e.emailCfg.FromName, e.emailCfg.FromEmail)
	to := mail.NewEmail(recipientName, toEmail)
	subject := fmt.Sprintf("An update on your booking with %s", businessName)
	plainText := fmt.Sprintf("Hi %s,\n\nThe staff member assigned to your booking with %s (%s) has changed. Everything else about your booking stays the same.\n\nThank you.", recipientName, businessName, whenText)
	htmlContent := fmt.Sprintf("<p>Hi %s,</p><p>The staff member assigned to your booking with <strong>%s</strong> (%s) has changed. Everything else about your booking stays the same.</p><p>Thank you.</p>", recipientName, businessName, whenText)

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
