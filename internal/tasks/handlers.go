package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"kori/internal/models"
	"kori/internal/utils"
	"kori/internal/utils/logger"

	"gorm.io/gorm"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// TaskHandler handles task processing with improved error handling and logging
type TaskHandler struct {
	db          *gorm.DB
	logger      *logger.Logger
	mailHandler *utils.EmailHandler
}

// NewTaskHandler creates a new TaskHandler
func NewTaskHandler(db *gorm.DB) *TaskHandler {
	return &TaskHandler{
		db:          db,
		logger:      logger.New("task_handler"),
		mailHandler: utils.NewEmailHandler(5), // Rate limit of 5 emails per second
	}
}

// HandleEmailSend processes an email sending task
func (h *TaskHandler) HandleEmailSend(ctx context.Context, t *asynq.Task) error {
	var task EmailTask
	if err := json.Unmarshal(t.Payload(), &task); err != nil {
		return fmt.Errorf("failed to unmarshal email task: %w", asynq.SkipRetry)
	}

	h.logger.Info("📧 Processing email task ID: %s (Attempt: %d)", task.EmailID, task.AttemptNum)

	// Get email from db
	email, err := models.GetEmailByID(task.EmailID, h.db)
	if err != nil {
		return h.logger.Error("❌ failed to get email: %w", err)
	}

	// Send email using SMTP handler
	if err := h.mailHandler.SendEmail(email); err != nil {
		task.Error = err.Error()
		task.AttemptNum++
		return h.logger.Error("❌ failed to send email: %w", err)
	}

	h.logger.Success("✅ Email sent successfully")
	return nil
}

// HandleCampaignProcess processes a campaign task
func (h *TaskHandler) HandleCampaignProcess(ctx context.Context, t *asynq.Task) error {
	var task CampaignTask
	if err := json.Unmarshal(t.Payload(), &task); err != nil {
		return fmt.Errorf("failed to unmarshal campaign task: %w", asynq.SkipRetry)
	}

	h.logger.Info("processing campaign task",
		zap.String("campaign_id", task.CampaignID),
		zap.Int("batch_size", task.BatchSize),
		zap.Int("offset", task.Offset),
	)

	// TODO: Implement campaign processing logic
	return nil
}

// HandleWebhookDelivery processes a webhook delivery task
func (h *TaskHandler) HandleWebhookDelivery(ctx context.Context, t *asynq.Task) error {
	var task WebhookDeliveryTask
	if err := json.Unmarshal(t.Payload(), &task); err != nil {
		return fmt.Errorf("failed to unmarshal webhook task: %w", asynq.SkipRetry)
	}

	h.logger.Info("processing webhook task",
		zap.String("webhook_id", task.WebhookID),
		zap.String("event", task.Event),
		zap.Int("attempt", task.AttemptNum),
	)

	// TODO: Implement webhook delivery logic
	return nil
}

// HandleDomainVerification processes a domain verification task
func (h *TaskHandler) HandleDomainVerification(ctx context.Context, t *asynq.Task) error {
	var task DomainVerificationTask
	if err := json.Unmarshal(t.Payload(), &task); err != nil {
		return fmt.Errorf("failed to unmarshal domain verification task: %w", asynq.SkipRetry)
	}

	h.logger.Info("processing domain verification task",
		zap.String("domain_id", task.DomainID),
	)

	// TODO: Implement domain verification logic
	return nil
}

// HandleContactImport processes a contact import task
func (h *TaskHandler) HandleContactImport(ctx context.Context, t *asynq.Task) error {
	var task ContactImportTask
	if err := json.Unmarshal(t.Payload(), &task); err != nil {
		return fmt.Errorf("failed to unmarshal contact import task: %w", asynq.SkipRetry)
	}

	h.logger.Info("processing contact import task",
		zap.String("import_id", task.ImportID),
		zap.String("team_id", task.TeamID),
		zap.Int("batch_size", task.BatchSize),
		zap.Int("offset", task.Offset),
	)

	// TODO: Implement contact import logic
	return nil
}

// HandleLLMEmailWriter processes an LLM email writer task
func (h *TaskHandler) HandleLLMEmailWriter(ctx context.Context, t *asynq.Task) error {
	var task LLMEmailWriterTask
	if err := json.Unmarshal(t.Payload(), &task); err != nil {
		return fmt.Errorf("failed to unmarshal LLM email writer task: %w", asynq.SkipRetry)
	}

	h.logger.Info("processing LLM email writer task",
		zap.String("email_id", task.EmailID),
		zap.String("template_id", task.TemplateID),
		zap.String("model_id", task.ModelID),
		zap.Int("attempt", task.AttemptNum),
	)

	// TODO: Implement LLM email generation logic
	return nil
}
