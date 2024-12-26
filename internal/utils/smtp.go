package utils

import (
	"fmt"
	"kori/internal/db"
	"kori/internal/models"
	"strings"
	"sync"
	"time"

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
	config      EmailConfig
	rateLimiter chan struct{}
}

var MailHandler *EmailHandler

// NewEmailHandler creates a new EmailHandler
func NewEmailHandler(config EmailConfig) *EmailHandler {
	// Initialize rate limiter channel based on max send rate
	rateLimiter := make(chan struct{}, config.MaxSendRate)

	// Fill the rate limiter initially
	for i := 0; i < config.MaxSendRate; i++ {
		rateLimiter <- struct{}{}
	}

	return &EmailHandler{
		config:      config,
		rateLimiter: rateLimiter,
	}
}

// SendEmail sends a single email using the configured SMTP server
func (h *EmailHandler) SendEmail(email *models.Email) error {
	// Acquire rate limit token
	<-h.rateLimiter
	defer func() {
		// Release token after 1 second to maintain rate limit
		time.Sleep(time.Second)
		h.rateLimiter <- struct{}{}
	}()

	smtpConfig := email.SMTPConfig

	auth := sasl.NewPlainClient("", smtpConfig.Username, smtpConfig.Password)

	// Compose the email
	subject := fmt.Sprintf("Subject: %s\n", email.Subject)
	toFormatted := fmt.Sprintf("To: %s\nContent-Type: text/html; charset=UTF-8\n", email.To)
	msg := strings.NewReader(toFormatted + subject + email.Body)

	// Send the email
	addr := fmt.Sprintf("%s:%d", smtpConfig.Host, smtpConfig.Port)
	err := smtp.SendMail(addr, auth, email.From, []string{email.To}, msg)

	if err != nil {
		email.Error = err.Error()
		email.Status = models.EmailStatusFailed
		h.UpdateEmail(email)
		return fmt.Errorf("failed to send email: %w", err)
	}

	email.SentAt = time.Now()
	email.Status = models.EmailStatusSent
	email.Error = ""

	if err := h.UpdateEmail(email); err != nil {
		return fmt.Errorf("failed to update email: %w", err)
	}

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
	db := db.GetDB()
	return db.Save(email).Error
}
