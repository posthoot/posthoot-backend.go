package utils

import (
	"fmt"
	"kori/internal/db"
	"kori/internal/models"
	"kori/internal/utils/base64"
	"strings"
	"sync"
	"time"

	"kori/internal/utils/logger"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

// EmailConfig holds the configuration for the email server
type EmailConfig struct {
	Host         string
	Port         int
	Username     string
	Password     string
	MaxSendRate  int
	SupportsTLS  bool
	RequiresAuth bool
}

// BatchEmailResult represents the result of sending a batch of emails
type BatchEmailResult struct {
	Email *models.Email
	Error error
}

// EmailHandler handles sending emails via SMTP
type EmailHandler struct {
	rateLimiter chan struct{}
	logger      *logger.Logger
}

// NewEmailHandler creates a new EmailHandler with rate limiting
func NewEmailHandler(maxSendRate int) *EmailHandler {
	// Initialize rate limiter channel based on max send rate
	rateLimiter := make(chan struct{}, maxSendRate)

	// Fill the rate limiter initially
	for i := 0; i < maxSendRate; i++ {
		rateLimiter <- struct{}{}
	}

	return &EmailHandler{
		rateLimiter: rateLimiter,
		logger:      logger.New("email_handler"),
	}
}

// SendEmail sends a single email using the configured SMTP server
func (h *EmailHandler) SendEmail(email *models.Email) error {
	if email == nil {
		return fmt.Errorf("email is nil")
	}

	// Acquire rate limit token
	<-h.rateLimiter
	defer func() {
		// Release token after 1 second to maintain rate limit
		time.Sleep(time.Second)
		h.rateLimiter <- struct{}{}
	}()

	if email.Status != models.EmailStatusPending {
		return fmt.Errorf("email is not pending")
	}

	if email.SMTPConfig == nil {
		return fmt.Errorf("SMTP config is nil")
	}

	h.logger.Info("📧 Sending email to: %s using SMTP server: %s", email.To, email.SMTPConfig.Host)

	auth := sasl.NewPlainClient("", email.SMTPConfig.Username, email.SMTPConfig.Password)

	// Compose the email
	subject := fmt.Sprintf("Subject: %s\n", email.Subject)
	decodedBody, err := base64.DecodeFromBase64(email.Body)
	if err != nil {
		return fmt.Errorf("❌ failed to decode email body: %w", err)
	}
	toFormatted := fmt.Sprintf("To: %s\nContent-Type: text/html; charset=UTF-8\n", email.To)
	msg := strings.NewReader(toFormatted + subject + "\n" + decodedBody)

	// Send the email
	addr := fmt.Sprintf("%s:%d", email.SMTPConfig.Host, email.SMTPConfig.Port)
	err = smtp.SendMail(addr, auth, email.From, []string{email.To}, msg)

	if err != nil {
		email.Error = err.Error()
		email.Status = models.EmailStatusFailed
		if dbErr := h.UpdateEmail(email); dbErr != nil {
			return h.logger.Error("❌ Failed to update email status", dbErr)
		}
		return h.logger.Error("❌ failed to send email: %w", err)
	}

	email.SentAt = time.Now()
	email.Status = models.EmailStatusSent
	email.Error = ""

	if err := h.UpdateEmail(email); err != nil {
		return fmt.Errorf("❌ failed to update email: %w", err)
	}

	h.logger.Success("✅ Email sent successfully to: %s", email.To)
	return nil
}

// SendBatchEmails sends multiple emails in parallel with rate limiting
func (h *EmailHandler) SendBatchEmails(emails []*models.Email) []BatchEmailResult {
	results := make([]BatchEmailResult, len(emails))
	var wg sync.WaitGroup

	for i, email := range emails {
		wg.Add(1)
		go func(index int, e *models.Email) {
			defer wg.Done()
			err := h.SendEmail(e)
			results[index] = BatchEmailResult{
				Email: e,
				Error: err,
			}
		}(i, email)
	}

	wg.Wait()
	return results
}

// SendCampaignEmails sends campaign emails in batches
func (h *EmailHandler) SendCampaignEmails(campaign *models.Campaign, emails []*models.Email, batchSize int) []BatchEmailResult {
	totalEmails := len(emails)
	results := make([]BatchEmailResult, totalEmails)

	// Process in batches
	for i := 0; i < totalEmails; i += batchSize {
		end := i + batchSize
		if end > totalEmails {
			end = totalEmails
		}

		// Send batch
		batchResults := h.SendBatchEmails(emails[i:end])

		// Copy batch results to final results
		copy(results[i:end], batchResults)

		// Optional: Add delay between batches to prevent overwhelming the SMTP server
		if end < totalEmails {
			time.Sleep(time.Second * 2)
		}
	}

	return results
}

func (h *EmailHandler) UpdateEmail(email *models.Email) error {
	if email == nil {
		return fmt.Errorf("email is nil")
	}

	return db.GetDB().Updates(email).Error
}
