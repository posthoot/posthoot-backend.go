package services

import (
	"context"
	"fmt"
	"kori/internal/config"
	"kori/internal/db"
	"kori/internal/events"
	"kori/internal/models"
	"kori/internal/tasks"
	"kori/internal/utils"
	"kori/internal/utils/base64"
	"kori/internal/utils/logger"
	"os"

	"github.com/google/uuid"
)

var (
	log        = logger.New("EMAIL")
	cfg, _     = config.Load()
	taskClient *tasks.TaskClient
)

type sendEmailHandlerBody struct {
	teamId       string
	templateId   string
	to           string
	SMTPProvider string
	categoryId   string
	variables    map[string]string
	subject      string
	listId       string
	campaignId   string
	body         string
	cc           string
	bcc          string
	replyTo      string
}

func init() {
	// Initialize taskClient after config is loaded
	taskClient = tasks.NewTaskClient(
		cfg.Redis.Addr, // Use cfg instead of empty Config struct
		cfg.Redis.Username,
		cfg.Redis.Password,
		cfg.Redis.DB,
	)

	// Register event handlers
	events.On("invite.created", func(data interface{}) {
		invite := data.(*models.TeamInvite)
		log.Info("Sending invite email to %s", invite.Email)
		if err := sendTeamInviteEmail(invite); err != nil {
			err := log.Error("Failed to send invite email: %v", err)
			if err != nil {
				return
			}
		}
	})

	events.On("email.created", func(data interface{}) {
		email := data.(*models.Email)
		log.Info("Enqueueing email task for %s", email.ID)

		task := tasks.EmailTask{
			EmailID:      email.ID,
			AttemptNum:   1,
			SMTPConfigID: email.SMTPConfigID,
			MaxSendRate:  email.SMTPConfig.MaxSendRate,
		}

		log.Info("Enqueueing email task for %s", task.EmailID)

		if err := taskClient.EnqueueEmailTask(context.Background(), task); err != nil {
			err := log.Error("Failed to enqueue email task: %v", err)
			if err != nil {
				return
			}
		}
	})

	events.On("users.created", func(data interface{}) {
		user := data.(*models.User)
		log.Info("Sending welcome email to %s", user.Email)
		if err := sendWelcomeEmail(user); err != nil {
			err := log.Error("Failed to send welcome email: %v", err)
			if err != nil {
				return
			}
		}
	})

	events.On("email.send", func(data interface{}) {
		email := data.(*models.Email)
		log.Info("Sending email to %s", email.To)
		log.Info("Email data: %s", email.Data)
		var emailData map[string]string
		var err error
		if email.Data != nil {
			emailData, err = utils.JSONToMap(email.Data)
			if err != nil {
				log.Error("Failed to convert data to map: %v", err)
				return
			}
		}

		handler := &sendEmailHandlerBody{
			teamId:       email.TeamID,
			templateId:   email.TemplateID,
			to:           email.To,
			SMTPProvider: email.SMTPConfigID,
			categoryId:   email.CategoryID,
			variables:    emailData,
			subject:      email.Subject,
			listId:       "",
			campaignId:   email.CampaignID,
			body:         email.Body,
			cc:           email.CC,
			bcc:          email.BCC,
			replyTo:      email.ReplyTo,
		}

		if err := sendEmail(handler); err != nil {
			err := log.Error("Failed to send email: %v", err)
			if err != nil {
				return
			}
		}
	})
}

func sendTeamInviteEmail(invite *models.TeamInvite) error {
	// Start transaction
	tx := db.DB.Begin()
	if tx.Error != nil {
		return log.Error("failed to begin transaction", tx.Error)
	}

	// Get team details
	team := &models.Team{}
	if err := tx.First(team, invite.TeamID).Error; err != nil {
		tx.Rollback()
		return log.Error("failed to get team details", err)
	}

	// Get inviter details
	inviter := &models.User{}
	if err := tx.First(inviter, invite.InviterID).Error; err != nil {
		tx.Rollback()
		return log.Error("failed to get inviter details", err)
	}

	var template *models.Template

	if team.Settings[0].InviteTemplateID == "" {
		// Get invite template
		template = &models.Template{}
		if err := tx.Where("name = ? AND team_id = ?", "PLATFORM INVITE", invite.TeamID).First(template).Error; err != nil {
			tx.Rollback()
			return log.Error("failed to get invite template", err)
		}
	} else {
		template = &models.Template{}
		if err := tx.Where("id = ? AND team_id = ?", team.Settings[0].InviteTemplateID, invite.TeamID).First(template).Error; err != nil {
			tx.Rollback()
			return log.Error("failed to get invite template", err)
		}
	}

	// Get invite mailing list
	mailingList := &models.MailingList{}
	if err := tx.Where("name = ? AND team_id = ?", "Invited Users", invite.TeamID).First(mailingList).Error; err != nil {
		tx.Rollback()
		return log.Error("failed to get invite mailing list", err)
	}

	// Get or create contact
	contact := &models.Contact{}
	if err := tx.Where("email = ? AND team_id = ? AND list_id = ?", invite.Email, invite.TeamID, mailingList.ID).First(contact).Error; err != nil {
		contact = &models.Contact{
			Email:     invite.Email,
			FirstName: invite.Name,
			TeamID:    invite.TeamID,
			ListID:    mailingList.ID,
		}
		if err := tx.Create(contact).Error; err != nil {
			tx.Rollback()
			return log.Error("failed to create contact", err)
		}
	}

	// Get default SMTP config
	smtpConfig := &models.SMTPConfig{}
	if err := tx.Where("team_id = ?", invite.TeamID).First(smtpConfig).Error; err != nil {
		tx.Rollback()
		return log.Error("failed to get default smtp config", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return log.Error("failed to commit transaction", err)
	}

	handler := &sendEmailHandlerBody{
		teamId:       invite.TeamID,
		templateId:   template.ID,
		to:           invite.Email,
		SMTPProvider: smtpConfig.Provider,
		categoryId:   template.CategoryID,
		variables:    map[string]string{"inviter": fmt.Sprintf("%s %s", inviter.FirstName, inviter.LastName), "name": invite.Name, "role": string(invite.Role), "url": fmt.Sprintf("%s/team/accept-invite/%s", os.Getenv("OFFICE_URL"), invite.Code)},
		subject:      "Hey {{ name }} 👋🏻! You've been invited to join a team on Posthoot",
		listId:       mailingList.ID,
		campaignId:   "",
		body:         "",
		cc:           "",
		bcc:          "",
		replyTo:      "",
	}

	// Send email outside transaction since it's an external operation
	return sendEmail(handler)
}

func sendWelcomeEmail(user *models.User) error {
	tx := db.DB.Begin()
	team := &models.Team{}
	if err := tx.Where("name =?", os.Getenv("SUPERADMIN_TEAM_NAME")).First(team).Preload("Settings").Preload("Settings.BrandingSettings").Error; err != nil {
		tx.Rollback()
		return log.Error("failed to get team details", err)
	}

	// Get default SMTP config
	smtpConfig := &models.SMTPConfig{}
	if err := tx.Where("team_id = ?", user.TeamID).First(smtpConfig).Error; err != nil {
		tx.Rollback()
		return log.Error("failed to get default smtp config", err)
	}

	// mailing list

	mailingList := &models.MailingList{}
	if err := tx.Where("team_id = ? AND name = ?", user.TeamID, "PLATFORM WELCOME").First(mailingList).Error; err != nil {
		tx.Rollback()
		return log.Error("failed to get mailing list", err)
	}

	handler := &sendEmailHandlerBody{
		teamId:       user.TeamID,
		templateId:   team.Settings[0].WelcomeTemplateID,
		to:           user.Email,
		SMTPProvider: smtpConfig.Provider,
		categoryId:   "",
		variables:    map[string]string{"name": user.FirstName},
		subject:      "Hey {{ name }} 👋🏻! We're glad to have you onboard 🎉",
		listId:       mailingList.ID,
		campaignId:   "",
		body:         "",
		cc:           "",
		bcc:          "",
		replyTo:      "",
	}

	err := sendEmail(handler)
	if err != nil {
		return err
	}
	return nil
}

func sendEmail(
	handler *sendEmailHandlerBody,
) error {
	// Start transaction
	tx := db.DB.Begin()
	if tx.Error != nil {
		return log.Error("failed to begin transaction", tx.Error)
	}

	definedID := uuid.New()

	// Get SMTP config
	smtpConfig, err := models.GetSMTPConfig(handler.teamId, handler.SMTPProvider, "", tx)
	if err != nil {
		tx.Rollback()
		return log.Error("failed to get team smtp config ❌", err)
	}

	// Get template
	template := &models.Template{}
	if handler.templateId != "" {
		if err := tx.Where("id = ? AND team_id = ?", handler.templateId, handler.teamId).Preload("HtmlFile").First(template).Error; err != nil {
			tx.Rollback()
			return log.Error("failed to get template ❌", err)
		}
	}

	// Get or use default mailing list
	if handler.listId == "" {
		mailingList := &models.MailingList{}
		if err := tx.Where("team_id = ? AND name = ?", handler.teamId, "All Users").First(mailingList).Error; err != nil {
			tx.Rollback()
			return log.Error("failed to get default mailing list ❌", err)
		}
		handler.listId = mailingList.ID
	}

	// Get or create contact
	contact := &models.Contact{}
	if err := tx.Where("email = ? AND team_id = ? AND list_id = ?", handler.to, handler.teamId, handler.listId).First(contact).Error; err != nil {
		contactImport := &models.ContactImport{}
		contactImport.TeamID = handler.teamId
		contactImport.ListID = handler.listId
		contactImport.Status = models.ContactImportStatusCompleted
		if err := tx.Create(contactImport).Error; err != nil {
			tx.Rollback()
			return log.Error("failed to create contact import ❌", err)
		}
		contact = &models.Contact{
			Email:     handler.to,
			TeamID:    handler.teamId,
			ListID:    handler.listId,
			ImportID:  contactImport.ID,
			FirstName: handler.variables["name"],
		}
		if err := tx.Create(contact).Error; err != nil {
			tx.Rollback()
			return log.Error("failed to create contact ❌", err)
		}
	}

	if handler.categoryId == "" {
		category := &models.EmailCategory{}
		if err := tx.Where("name = ? AND team_id = ?", "Transactional", handler.teamId).First(category).Error; err != nil {
			tx.Rollback()
			return log.Error("failed to get category ❌", err)
		}
		handler.categoryId = category.ID
	}
	// Get category
	category := &models.EmailCategory{}
	if err := tx.Where("id = ? AND team_id = ?", handler.categoryId, handler.teamId).First(category).Error; err != nil {
		tx.Rollback()
		return log.Error("failed to get category ❌", err)
	}

	htmlFromTemplate := handler.body

	if handler.body == "" {
		// Get HTML content outside transaction since it's an external operation
		htmlFromTemplate, err = utils.GetHTMLFromURL(template.HtmlFile.SignedURL)
		if err != nil {
			tx.Rollback()
			return log.Error("failed to get html from template ❌", err)
		}
	}

	parsedBody := utils.ReplaceVariables(htmlFromTemplate, handler.variables, definedID.String(), cfg, true)
	parsedSubject := handler.subject
	if handler.subject == "" {
		parsedSubject = utils.ReplaceVariables(template.Subject, handler.variables, definedID.String(), cfg, false)
		parsedSubject, err = base64.DecodeFromBase64(parsedSubject)
		if err != nil {
			tx.Rollback()
			return log.Error("failed to decode subject ❌", err)
		}
	}

	jsonData, err := utils.MapToJSON(handler.variables)

	if err != nil {
		tx.Rollback()
		return log.Error("failed to convert variables to json ❌", err)
	}

	// Create email
	email := &models.Email{
		From:         smtpConfig.FromEmail,
		TeamID:       handler.teamId,
		TemplateID:   handler.templateId,
		To:           handler.to,
		Subject:      parsedSubject,
		Body:         parsedBody,
		Status:       models.EmailStatusPending,
		Data:         jsonData,
		ContactID:    contact.ID,
		CategoryID:   category.ID,
		SMTPConfigID: smtpConfig.ID,
		CampaignID:   handler.campaignId,
		CC:           handler.cc,
		BCC:          handler.bcc,
		ReplyTo:      handler.replyTo,
	}

	email.ID = definedID.String()
	if err := tx.Create(email).Error; err != nil {
		tx.Rollback()
		return log.Error("failed to create email ❌", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return log.Error("failed to commit transaction", err)
	}

	return nil // ✅
}
