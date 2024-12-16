package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

// EmailConfig holds the configuration for the email server
type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
}

// Email represents an email message
type Email struct {
	From    string
	To      []string
	Subject string
	Body    string
}

// AppleMailHandler handles sending emails using Apple's custom mail via SMTP
type AppleMailHandler struct {
	Config EmailConfig
}

var MailHandler = NewAppleMailHandler(EmailConfig{
	Host: "smtp.gmail.com",
	Port: 587,
})

// NewAppleMailHandler creates a new AppleMailHandler
func NewAppleMailHandler(config EmailConfig) *AppleMailHandler {
	return &AppleMailHandler{
		Config: config,
	}
}

// SendEmail sends an email using the configured SMTP server
func (h *AppleMailHandler) SendEmail(email Email) error {

	// Set the SMTP server configuration
	h.Config.Username = os.Getenv("EMAIL_USERNAME")
	h.Config.Password = os.Getenv("EMAIL_PASSWORD")
	email.From = os.Getenv("EMAIL_FROM")

	// Authenticate with the SMTP server
	fmt.Println("Authenticating with SMTP server...")

	auth := sasl.NewPlainClient("", h.Config.Username, h.Config.Password)

	// Compose the email
	subject := fmt.Sprintf("Subject: %s\n", email.Subject)

	toFormatted := fmt.Sprintf("To: %s\nContent-Type: text/html; charset=UTF-8\n", strings.Join(email.To, ", "))

	msg := strings.NewReader(toFormatted + subject + email.Body)

	// Send the email
	addr := fmt.Sprintf("%s:%d", h.Config.Host, h.Config.Port)

	err := smtp.SendMail(addr, auth, email.From, email.To, msg)

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Println("Email sent successfully!")

	return nil
}
