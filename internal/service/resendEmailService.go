package service

import (
	"fmt"
	"fyp/config"
	"fyp/internal/interfaces"

	"github.com/resend/resend-go/v2"
)

type resendEmailService struct {
	emailCfg config.EmailConfig
	client   *resend.Client
}

func NewResendEmailService(emailCfg config.EmailConfig) interfaces.IEmailService {
	return &resendEmailService{
		emailCfg: emailCfg,
		client:   resend.NewClient(emailCfg.ResendAPIKey),
	}
}

func (e *resendEmailService) from() string {
	return fmt.Sprintf("%s <%s>", e.emailCfg.FromName, e.emailCfg.FromEmail)
}

func (e *resendEmailService) send(toEmail, subject, plainText, htmlContent string) error {
	_, err := e.client.Emails.Send(&resend.SendEmailRequest{
		From:    e.from(),
		To:      []string{toEmail},
		Subject: subject,
		Text:    plainText,
		Html:    htmlContent,
	})
	if err != nil {
		return fmt.Errorf("resend: %w", err)
	}
	return nil
}

func (e *resendEmailService) SendStaffWelcomeEmail(toEmail, tempPassword string) error {
	subject := "Welcome — your staff account is ready"
	plainText := fmt.Sprintf("You have been added as a staff member.\n\nEmail: %s\nTemporary password: %s\n\nLog in with these details — you'll be asked to set your own password straight away.", toEmail, tempPassword)
	htmlContent := fmt.Sprintf("<p>You have been added as a staff member.</p><p><strong>Email:</strong> %s<br><strong>Temporary password:</strong> %s</p><p>Log in with these details — you'll be asked to set your own password straight away.</p>", toEmail, tempPassword)
	return e.send(toEmail, subject, plainText, htmlContent)
}

func (e *resendEmailService) SendBookingStatusEmail(toEmail, recipientName, withName, statusLabel string) error {
	subject := fmt.Sprintf("Your booking with %s is now %s", withName, statusLabel)
	plainText := fmt.Sprintf("Hi %s,\n\nYour booking with %s is now %s.\n\nThank you.", recipientName, withName, statusLabel)
	htmlContent := fmt.Sprintf("<p>Hi %s,</p><p>Your booking with <strong>%s</strong> is now <strong>%s</strong>.</p><p>Thank you.</p>", recipientName, withName, statusLabel)
	return e.send(toEmail, subject, plainText, htmlContent)
}

func (e *resendEmailService) SendNewBookingRequestEmail(toEmail, recipientName, customerName, whenText string) error {
	subject := fmt.Sprintf("New booking request from %s", customerName)
	plainText := fmt.Sprintf("Hi %s,\n\n%s just requested a booking on %s.\n\nPlease accept or reject it from your calendar.", recipientName, customerName, whenText)
	htmlContent := fmt.Sprintf("<p>Hi %s,</p><p><strong>%s</strong> just requested a booking for <strong>%s</strong>.</p><p>Please accept or reject it from your calendar.</p>", recipientName, customerName, whenText)
	return e.send(toEmail, subject, plainText, htmlContent)
}

func (e *resendEmailService) SendLeaveApplicationSubmittedEmail(toEmail, ownerName, staffName, startDate, endDate string) error {
	subject := fmt.Sprintf("%s applied for leave", staffName)
	plainText := fmt.Sprintf("Hi %s,\n\n%s has applied for leave from %s to %s.\n\nPlease review it in Staff Availability.", ownerName, staffName, startDate, endDate)
	htmlContent := fmt.Sprintf("<p>Hi %s,</p><p><strong>%s</strong> has applied for leave from <strong>%s</strong> to <strong>%s</strong>.</p><p>Please review it in Staff Availability.</p>", ownerName, staffName, startDate, endDate)
	return e.send(toEmail, subject, plainText, htmlContent)
}

func (e *resendEmailService) SendStaffReassignedEmail(toEmail, recipientName, businessName, whenText string) error {
	subject := fmt.Sprintf("An update on your booking with %s", businessName)
	plainText := fmt.Sprintf("Hi %s,\n\nThe staff member assigned to your booking with %s (%s) has changed. Everything else about your booking stays the same.\n\nThank you.", recipientName, businessName, whenText)
	htmlContent := fmt.Sprintf("<p>Hi %s,</p><p>The staff member assigned to your booking with <strong>%s</strong> (%s) has changed. Everything else about your booking stays the same.</p><p>Thank you.</p>", recipientName, businessName, whenText)
	return e.send(toEmail, subject, plainText, htmlContent)
}
