package processors

import (
	"context"
	"encoding/json"
	"fmt"
	"kori/internal/automation"
	"kori/internal/config"
	"kori/internal/models"
	"kori/internal/tasks"
	"kori/internal/utils"
	"kori/internal/utils/base64"
	"maps"
	"time"

	"gorm.io/gorm"
)

// EmailProcessor handles EMAIL nodes
type EmailProcessor struct {
	db         *gorm.DB
	taskClient *tasks.TaskClient
	cfg        *config.Config
}

// EmailNodeData represents the data structure for EMAIL nodes
type EmailNodeData struct {
	TemplateID   string `json:"templateId"`
	SMTPConfigID string `json:"smtpConfigId"`
	Subject      string `json:"subject,omitempty"`      // Override template subject
	CustomBody   string `json:"customBody,omitempty"`   // Override template body
	Variables    map[string]string `json:"variables,omitempty"` // Additional variables
}

// NewEmailProcessor creates a new EMAIL node processor
func NewEmailProcessor(db *gorm.DB, taskClient *tasks.TaskClient, cfg *config.Config) *EmailProcessor {
	return &EmailProcessor{
		db:         db,
		taskClient: taskClient,
		cfg:        cfg,
	}
}

// Type returns the node type this processor handles
func (p *EmailProcessor) Type() models.NodeType {
	return models.NodeTypeEmail
}

// Validate checks if the EMAIL node data is valid
func (p *EmailProcessor) Validate(node *models.AutomationNode) error {
	if node.Type != models.NodeTypeEmail {
		return fmt.Errorf("invalid node type for EmailProcessor")
	}

	var data EmailNodeData
	if err := json.Unmarshal(node.Data, &data); err != nil {
		return fmt.Errorf("invalid email node data: %w", err)
	}

	if data.TemplateID == "" && data.CustomBody == "" {
		return fmt.Errorf("either templateId or customBody must be specified")
	}

	if data.SMTPConfigID == "" {
		return fmt.Errorf("smtpConfigId is required")
	}

	return nil
}

// Process executes the EMAIL node logic
func (p *EmailProcessor) Process(ctx *automation.ExecutionContext, node *models.AutomationNode) (*automation.ProcessResult, error) {
	var data EmailNodeData
	if err := json.Unmarshal(node.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to parse email node data: %w", err)
	}

	// Get SMTP config
	smtpConfig, err := models.GetSMTPConfig(ctx.TeamID, data.SMTPConfigID, "", p.db)
	if err != nil {
		return nil, fmt.Errorf("failed to get SMTP config: %w", err)
	}

	// Get template if specified
	var template *models.Template
	var htmlBody string
	var subject string

	if data.TemplateID != "" {
		if err := p.db.Preload("HtmlFile").Preload("Category").Where("id = ?", data.TemplateID).First(&template).Error; err != nil {
			return nil, fmt.Errorf("failed to load template: %w", err)
		}

		// Get HTML content from template
		htmlBody, err = utils.GetHTMLFromURL(template.HtmlFile.SignedURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get HTML from template: %w", err)
		}

		subject = template.Subject
	} else {
		htmlBody = data.CustomBody
		subject = data.Subject
	}

	// Build variables for email personalization
	variables := make(map[string]string)

	// Default contact variables
	if ctx.Contact != nil {
		variables["email"] = ctx.Contact.Email
		variables["first_name"] = ctx.Contact.FirstName
		variables["last_name"] = ctx.Contact.LastName
		variables["name"] = fmt.Sprintf("%s %s", ctx.Contact.FirstName, ctx.Contact.LastName)
		variables["full_name"] = fmt.Sprintf("%s %s", ctx.Contact.FirstName, ctx.Contact.LastName)
		variables["company"] = ctx.Contact.Company
		variables["country"] = ctx.Contact.Country
		variables["city"] = ctx.Contact.City
		variables["state"] = ctx.Contact.State
		variables["zip"] = ctx.Contact.Zip
		variables["address"] = ctx.Contact.Address
		variables["phone"] = ctx.Contact.Phone
	}

	// Add custom variables from node data
	if data.Variables != nil {
		maps.Copy(variables, data.Variables)
	}

	// Add variables from execution context
	for key, val := range ctx.Variables {
		if strVal, ok := val.(string); ok {
			variables[key] = strVal
		}
	}

	// Replace variables in body and subject
	isMarketing := template != nil && template.Category.Name == "Marketing"
	parsedBody := utils.ReplaceVariables(htmlBody, variables, ctx.AutomationID, p.cfg, true, isMarketing)
	parsedSubject := utils.ReplaceVariables(subject, variables, ctx.AutomationID, p.cfg, false, false)

	// Decode subject if base64 encoded
	parsedSubject, err = base64.DecodeFromBase64(parsedSubject)
	if err != nil {
		return nil, fmt.Errorf("failed to decode subject: %w", err)
	}

	// Serialize variables for email data
	jsonData, err := utils.MapToJSON(variables)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize variables: %w", err)
	}

	// Create email record
	email := &models.Email{
		From:         smtpConfig.FromEmail,
		To:           ctx.Contact.Email,
		Subject:      parsedSubject,
		Body:         parsedBody,
		Data:         jsonData,
		Status:       models.EmailStatusPending,
		TeamID:       ctx.TeamID,
		TemplateID:   data.TemplateID,
		ContactID:    ctx.ContactID,
		SMTPConfigID: smtpConfig.ID,
		CategoryID:   "",
	}

	if template != nil {
		email.CategoryID = template.CategoryID
	}

	// Save email to database
	if err := p.db.Create(email).Error; err != nil {
		return nil, fmt.Errorf("failed to create email: %w", err)
	}

	// Enqueue email sending task
	emailTask := tasks.EmailTask{
		EmailID:      email.ID,
		AttemptNum:   1,
		SMTPConfigID: smtpConfig.ID,
		MaxSendRate:  smtpConfig.MaxSendRate,
		SendAt:       time.Now(),
	}

	if err := p.taskClient.EnqueueEmailTask(context.Background(), emailTask); err != nil {
		return nil, fmt.Errorf("failed to enqueue email task: %w", err)
	}

	// Get next nodes
	var edges []models.AutomationNodeEdge
	if err := p.db.Where("automation_id = ? AND source_id = ?", ctx.AutomationID, node.ID).Find(&edges).Error; err != nil {
		return nil, fmt.Errorf("failed to load edges: %w", err)
	}

	nextNodeIDs := make([]string, len(edges))
	for i, edge := range edges {
		nextNodeIDs[i] = edge.TargetID
	}

	// Update variables with email info
	updateVars := map[string]interface{}{
		"last_email_id":      email.ID,
		"last_email_subject": email.Subject,
		"last_email_sent_at": time.Now().Format(time.RFC3339),
	}

	return &automation.ProcessResult{
		NextNodeIDs: nextNodeIDs,
		Message:     fmt.Sprintf("Email queued for sending to %s", ctx.Contact.Email),
		UpdateVars:  updateVars,
		Data: map[string]interface{}{
			"emailId": email.ID,
		},
	}, nil
}
