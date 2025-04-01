package tasks

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"kori/internal/config"
	"kori/internal/models"
	"kori/internal/utils"
	"kori/internal/utils/logger"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/hibiken/asynq"
)

var (
	cfg, _ = config.Load()
)

// TaskHandler handles task processing with improved error handling and logging
type TaskHandler struct {
	db             *gorm.DB
	logger         *logger.Logger
	mailHandler    *utils.EmailHandler
	taskClient     *TaskClient
	storageHandler *utils.StorageHandler
}

// NewTaskHandler creates a new TaskHandler
func NewTaskHandler(db *gorm.DB) *TaskHandler {
	return &TaskHandler{
		db:             db,
		logger:         logger.New("task_handler"),
		mailHandler:    utils.NewEmailHandler(5), // Rate limit of 5 emails per second
		taskClient:     NewTaskClient(cfg.Redis.Addr, cfg.Redis.Username, cfg.Redis.Password, cfg.Redis.DB),
		storageHandler: utils.NewStorageHandler(),
	}
}

// HandleEmailSend processes an email sending task
func (h *TaskHandler) HandleEmailSend(ctx context.Context, t *asynq.Task) error {
	var task EmailTask
	if err := json.Unmarshal(t.Payload(), &task); err != nil {
		return fmt.Errorf("failed to unmarshal email task: %w", asynq.SkipRetry)
	}

	// Get email from db
	email, err := models.GetEmailByID(task.EmailID, h.db)
	if err != nil {
		return h.logger.Error("❌ failed to get email: %w", err)
	}

	h.logger.Info("📧 Processing email task ID: %s (Attempt: %d)", task.EmailID, task.AttemptNum)

	// Send email using SMTP handler
	if err := h.mailHandler.SendEmail(email); err != nil {
		task.Error = err.Error()
		task.AttemptNum++
		return h.logger.Error("❌ failed to send email: %w", err)
	}

	h.logger.Success("✅ Email sent successfully")

	// Check for after function in payload
	var payload map[string]interface{}
	if err := json.Unmarshal(t.Payload(), &payload); err == nil {
		if fn, ok := payload["after_func"].(func(context.Context, *asynq.Task) error); ok {
			if err := fn(ctx, t); err != nil {
				return fmt.Errorf("after func failed: %w", err)
			}
		}
	}

	return nil
}

// HandleCampaignProcess processes a campaign task
func (h *TaskHandler) HandleCampaignProcess(ctx context.Context, t *asynq.Task) error {
	var task CampaignTask
	if err := json.Unmarshal(t.Payload(), &task); err != nil {
		return fmt.Errorf("failed to unmarshal campaign task: %w", asynq.SkipRetry)
	}

	h.logger.Info("📧 Processing campaign task ID: %s with batch size %d and offset %d",
		task.CampaignID, task.BatchSize, task.Offset)

	// Get campaign details
	campaign, err := models.GetCampaignByID(task.CampaignID, h.db)
	if err != nil {
		return h.logger.Error("❌ failed to get campaign: %w", err)
	}

	if campaign.Status == models.CampaignStatusCompleted {
		h.logger.Info("✅ Campaign %s is already completed", task.CampaignID)
		return nil
	}

	// Get the campaign's SMTP config
	smtpConfig, err := models.GetSMTPConfig(campaign.TeamID, campaign.SMTPConfigID, "", h.db)
	if err != nil {
		return h.logger.Error("❌ failed to get smtp config: %w", err)
	}

	// Get the campaign's email list
	emailList, err := models.GetEmailListByID(campaign.ListID, h.db)
	if err != nil {
		return h.logger.Error("❌ failed to get email list: %w", err)
	}

	// already processed contacts
	alreadyProcessedEmails, err := models.GetEmailsByCampaignID(campaign.ID, h.db)
	if err != nil {
		return h.logger.Error("❌ failed to get already processed emails: %w", err)
	}

	alreadyProcessedContacts := make([]string, len(alreadyProcessedEmails))
	for i, email := range alreadyProcessedEmails {
		alreadyProcessedContacts[i] = email.ContactID
	}

	// Get contacts for this batch, ordered by last email sent date
	var contacts []models.Contact
	query := h.db.Table("contacts").
		Select("contacts.*").
		Joins("LEFT JOIN emails ON emails.contact_id = contacts.id AND emails.campaign_id = ?", campaign.ID).
		Where("contacts.list_id = ? AND contacts.status = ?", emailList.ID, models.SubscriberStatusActive).
		Where("contacts.id NOT IN (?)", alreadyProcessedContacts).
		Where("contacts.status != ?", models.SubscriberStatusUnsubscribed)

	if err := query.Group("contacts.id").
		Order("MAX(emails.sent_at) ASC NULLS FIRST"). // Contacts with no emails come first
		Offset(campaign.Processed).
		Limit(task.BatchSize).
		Find(&contacts).Error; err != nil {
		return h.logger.Error("❌ failed to get contacts: %w", err)
	}

	// If no contacts in this batch, we're done
	if len(contacts) == 0 {
		h.logger.Info("✅ No more contacts to process for campaign %s", campaign.ID)
		campaign.Status = models.CampaignStatusCompleted
		if err := h.db.Save(campaign).Error; err != nil {
			return h.logger.Error("❌ failed to update campaign status: %w", err)
		}
		return nil
	}

	// Create emails for each contact
	emails := make([]*models.Email, len(contacts))
	for i, contact := range contacts {
		email := &models.Email{
			From:         smtpConfig.FromEmail,
			To:           contact.Email,
			Subject:      campaign.Template.Subject,
			Body:         campaign.Template.DesignJSON,
			Status:       models.EmailStatusPending,
			TeamID:       campaign.TeamID,
			TemplateID:   campaign.TemplateID,
			ContactID:    contact.ID,
			SMTPConfigID: smtpConfig.ID,
			CategoryID:   campaign.Template.CategoryID,
			CampaignID:   campaign.ID,
		}
		emails[i] = email
	}

	// Save all emails in a transaction
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		for _, email := range emails {
			if err := tx.Create(email).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return h.logger.Error("❌ failed to create emails: %w", err)
	}

	h.mailHandler.SendBatchEmails(emails)

	// Update campaign status if needed
	if task.Offset+task.BatchSize >= len(emailList.Contacts) {
		campaign.Status = models.CampaignStatusSending
		if err := h.db.Save(campaign).Error; err != nil {
			return h.logger.Error("❌ failed to update campaign status: %w", err)
		}
	}

	campaign.Processed += len(contacts)
	if err := h.db.Save(campaign).Error; err != nil {
		return h.logger.Error("❌ failed to update campaign processed: %w", err)
	}

	h.mailHandler.SendCampaignEmails(campaign, emails, task.BatchSize, campaign.BatchDelay)

	h.logger.Success("✅ Successfully processed campaign batch")

	// Enqueue next batch if needed
	if len(contacts) == task.BatchSize && campaign.Schedule != models.CampaignScheduleRecurring { // Only schedule next batch for non-recurring campaigns
		nextTask := CampaignTask{
			CampaignID: campaign.ID,
			BatchSize:  task.BatchSize,
			Offset:     campaign.Processed,
		}

		if err := h.taskClient.EnqueueCampaignTask(ctx, nextTask, campaign.BatchDelay); err != nil {
			return h.logger.Error("❌ failed to enqueue next batch: %w", err)
		}
	}

	h.logger.Success("✅ Successfully processed campaign batch")

	// Check for after function in payload
	var payload map[string]interface{}
	if err := json.Unmarshal(t.Payload(), &payload); err == nil {
		if fn, ok := payload["after_func"].(func(context.Context, *asynq.Task) error); ok {
			if err := fn(ctx, t); err != nil {
				return fmt.Errorf("after func failed: %w", err)
			}
		}
	}

	return nil
}

// HandleWebhookDelivery processes a webhook delivery task
func (h *TaskHandler) HandleWebhookDelivery(ctx context.Context, t *asynq.Task) error {
	var task WebhookDeliveryTask
	if err := json.Unmarshal(t.Payload(), &task); err != nil {
		return fmt.Errorf("failed to unmarshal webhook task: %w", asynq.SkipRetry)
	}

	h.logger.Info("processing webhook task %s with event %s and attempt %d", task.WebhookID, task.Event, task.AttemptNum)

	// TODO: Implement webhook delivery logic
	return nil
}

// HandleDomainVerification processes a domain verification task
func (h *TaskHandler) HandleDomainVerification(ctx context.Context, t *asynq.Task) error {
	var task DomainVerificationTask
	if err := json.Unmarshal(t.Payload(), &task); err != nil {
		return fmt.Errorf("failed to unmarshal domain verification task: %w", asynq.SkipRetry)
	}

	h.logger.Info("processing domain verification task %s", task.DomainID)

	// TODO: Implement domain verification logic
	return nil
}

// HandleContactImport processes a contact import task
func (h *TaskHandler) HandleContactImport(ctx context.Context, t *asynq.Task) error {
	h.logger.Info("🚀 Starting contact import task")

	var task ContactImportTask
	if err := json.Unmarshal(t.Payload(), &task); err != nil {
		return h.logger.Error("❌ failed to unmarshal contact import task: %w", asynq.SkipRetry)
	}

	h.logger.Info("📥 Processing contact import task %s", task.ImportID)

	// get the contact import from the database
	contact_import, err := models.GetContactImportByID(task.ImportID, h.db)
	if err != nil {
		return h.logger.Error("❌ failed to get contact import: %w", err)
	}
	h.logger.Info("📋 Found contact import record: %v", contact_import)

	// get the file from the database
	file, err := models.GetFileByID(contact_import.FileID, h.db)
	if err != nil {
		return h.logger.Error("❌ failed to get file: %w", err)
	}
	h.logger.Info("📁 Found file record: %v", file)

	// we need to download the file from the storage provider
	h.logger.Info("⬇️ Downloading file from: %s", file.SignedURL)
	fileContent, err := h.storageHandler.DownloadFile(file.SignedURL)
	if err != nil {
		contact_import.Status = models.ContactImportStatusFailed
		if err := h.db.Save(contact_import).Error; err != nil {
			return h.logger.Error("❌ failed to update contact import status: %w", err)
		}
		return h.logger.Error("❌ failed to download file: %w", err)
	}
	h.logger.Info("✅ File downloaded successfully")

	// now we need to get the file content
	fileData := bytes.NewReader(fileContent)

	// we need to parse the file content
	reader := csv.NewReader(fileData)
	reader.Comma = ','
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	// now we need to parse the file content
	records, err := reader.ReadAll()
	if err != nil {
		contact_import.Status = models.ContactImportStatusFailed
		if err := h.db.Save(contact_import).Error; err != nil {
			return h.logger.Error("❌ failed to update contact import status: %w", err)
		}
		return h.logger.Error("❌ failed to read file: %w", err)
	}

	// Skip header row
	if len(records) <= 1 {
		contact_import.Status = models.ContactImportStatusFailed
		if err := h.db.Save(contact_import).Error; err != nil {
			return h.logger.Error("❌ failed to update contact import status: %w", err)
		}
		return h.logger.Error("❌ file has no data rows", fmt.Errorf("no data rows in file"))
	}

	h.logger.Info("📋 Records: %v", records)

	fieldsMap, err := utils.JSONToMap(contact_import.FieldsMap)
	if err != nil {
		contact_import.Status = models.ContactImportStatusFailed
		if err := h.db.Save(contact_import).Error; err != nil {
			return h.logger.Error("❌ failed to update contact import status: %w", err)
		}
		return h.logger.Error("❌ failed to parse fields map from json: %w", err)
	}

	h.logger.Info("📋 Fields map: %v", fieldsMap)

	// contacts to create
	var contacts []models.Contact

	// Process each record (skipping header)
	for _, record := range records[1:] {
		// Create a map of field values from the record
		recordMap := make(map[string]string)
		headers := records[0]

		for j, value := range record {
			if j < len(headers) {
				recordMap[headers[j]] = value
			}
		}

		// Map fields according to the fieldsMap
		contact := models.Contact{
			TeamID:   contact_import.TeamID,
			ListID:   contact_import.ListID,
			ImportID: contact_import.ID,
		}

		// Create reverse mapping (database fields to CSV fields)
		var reverseMap = make(map[string]string)
		for k, v := range fieldsMap {
			reverseMap[v] = k
		}

		// Helper function to get the CSV field name from database field
		getFieldValue := func(dbField string) string {
			// Check if the database field exists in reverse mapping
			if csvField, exists := reverseMap[dbField]; exists {
				return recordMap[csvField]
			}
			return ""
		}

		// Map all fields using the helper function
		contact.Email = getFieldValue("email")
		contact.FirstName = getFieldValue("first_name")
		contact.LastName = getFieldValue("last_name")
		contact.LinkedIn = getFieldValue("linkedin")
		contact.Twitter = getFieldValue("twitter")
		contact.Facebook = getFieldValue("facebook")
		contact.Instagram = getFieldValue("instagram")
		contact.Country = getFieldValue("country")
		contact.Phone = getFieldValue("phone")
		contact.City = getFieldValue("city")
		contact.State = getFieldValue("state")
		contact.Zip = getFieldValue("zip")
		contact.Address = getFieldValue("address")
		contact.Company = getFieldValue("company")

		// Store all fields in metadata
		contact.Metadata = datatypes.JSON(fmt.Sprintf("%v", recordMap))

		contacts = append(contacts, contact)

		h.logger.Info("processing record %v", contact)
	}

	// save the contacts
	if err := h.db.CreateInBatches(&contacts, 100).Error; err != nil {
		contact_import.Status = models.ContactImportStatusFailed
		if err := h.db.Save(contact_import).Error; err != nil {
			return h.logger.Error("❌ failed to update contact import status: %w", err)
		}
		return h.logger.Error("❌ failed to create contacts: %w", err)
	}

	contact_import.Status = models.ContactImportStatusCompleted
	if err := h.db.Save(contact_import).Error; err != nil {
		return h.logger.Error("❌ failed to update contact import status: %w", err)
	}

	return nil
}

// HandleLLMEmailWriter processes an LLM email writer task
func (h *TaskHandler) HandleLLMEmailWriter(ctx context.Context, t *asynq.Task) error {
	var task LLMEmailWriterTask
	if err := json.Unmarshal(t.Payload(), &task); err != nil {
		return fmt.Errorf("failed to unmarshal LLM email writer task: %w", asynq.SkipRetry)
	}

	h.logger.Info("processing LLM email writer task %s with template %s and model %s and attempt %d", task.EmailID, task.TemplateID, task.ModelID, task.AttemptNum)

	// TODO: Implement LLM email generation logic
	return nil
}
