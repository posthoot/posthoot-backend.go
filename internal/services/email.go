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
	"kori/internal/utils/logger"
)

var log = logger.New("EMAIL")

func init() {
	taskClient := tasks.NewTaskClient(
		config.Config{}.Redis.Addr,
		config.Config{}.Redis.Password,
		config.Config{}.Redis.Username,
		config.Config{}.Redis.DB,
	)

	// Register event handlers
	events.On("invite.created", func(data interface{}) {
		invite := data.(*models.TeamInvite)
		log.Info("Sending invite email to %s", invite.Email)
		if err := sendTeamInviteEmail(invite); err != nil {
			log.Error("Failed to send invite email: %v", err)
		}
	})

	events.On("email.created", func(data interface{}) {
		email := data.(*models.Email)
		log.Info("Enqueueing email task for %s", email.ID)

		task := tasks.EmailTask{
			EmailID:    email.ID,
			AttemptNum: 1,
		}

		log.Info("Enqueueing email task for %s", task.EmailID)

		if err := taskClient.EnqueueEmailTask(context.Background(), task); err != nil {
			log.Error("Failed to enqueue email task: %v", err)
		}
	})

	events.On("users.created", func(data interface{}) {
		user := data.(*models.User)
		log.Info("Sending welcome email to %s", user.Email)
		if err := sendWelcomeEmail(user); err != nil {
			log.Error("Failed to send welcome email: %v", err)
		}
	})

	events.On("email.send", func(data interface{}) {
		email := data.(*models.Email)
		log.Info("Sending email to %s", email.To)
		log.Info("Email data: %s", email.Data)
		data, err := utils.JSONToMap(email.Data)
		if err != nil {
			err := log.Error("Failed to convert data to map: %v", err)
			if err != nil {
				return
			}
			return
		}
		if err := sendEmail(
			email.TeamID,
			email.TemplateID,
			email.To,
			email.SMTPConfigID,
			email.CategoryID,
			data.(map[string]string),
			email.Subject,
			"",
			email.CampaignID,
		); err != nil {
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

	// Send email outside transaction since it's an external operation
	return sendEmail(
		invite.TeamID,
		template.ID,
		invite.Email,
		smtpConfig.ID,
		template.CategoryID,
		map[string]string{
			"inviterName": fmt.Sprintf("%s %s", inviter.FirstName, inviter.LastName),
			"name":        invite.Name,
			"teamName":    team.Name,
			"inviteLink":  fmt.Sprintf("/accept-invite/%s", invite.ID),
		},
		"Hey there! You've been invited to join Kori",
		mailingList.ID,
		"",
	)
}

func sendWelcomeEmail(user *models.User) error {
	tx := db.DB.Begin()
	team := &models.Team{}
	if err := db.DB.First(team, user.TeamID).Preload("Settings").Preload("Settings.BrandingSettings").Error; err != nil {
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

	err := sendEmail(user.TeamID, team.Settings[0].WelcomeTemplateID, user.Email, smtpConfig.ID, "", map[string]string{"name": user.FirstName, "team": team.Name, "logo": team.Settings[0].BrandingSettings.LogoURL}, "Welcome to Kori", mailingList.ID, "")
	if err != nil {
		return err
	}
	return nil
}

func sendEmail(teamId string, templateId string, to string, SMTPProvider string, categoryId string, variables map[string]string, subject string, listId string, campaignId string) error {
	// Start transaction
	tx := db.DB.Begin()
	if tx.Error != nil {
		return log.Error("failed to begin transaction", tx.Error)
	}

	// Get SMTP config
	smtpConfig, err := models.GetSMTPConfig(teamId, SMTPProvider, "", tx)
	if err != nil {
		tx.Rollback()
		return log.Error("failed to get team smtp config ❌", err)
	}

	// Get template
	template := &models.Template{}
	if err := tx.Where("id = ? AND team_id = ?", templateId, teamId).Preload("HtmlFile").First(template).Error; err != nil {
		tx.Rollback()
		return log.Error("failed to get template ❌", err)
	}

	// Get or use default mailing list
	if listId == "" {
		mailingList := &models.MailingList{}
		if err := tx.Where("team_id = ? AND name = ?", teamId, "All Users").First(mailingList).Error; err != nil {
			tx.Rollback()
			return log.Error("failed to get default mailing list ❌", err)
		}
		listId = mailingList.ID
	}

	// Get or create contact
	contact := &models.Contact{}
	if err := tx.Where("email = ? AND team_id = ? AND list_id = ?", to, teamId, listId).First(contact).Error; err != nil {
		contactImport := &models.ContactImport{}
		contactImport.TeamID = teamId
		contactImport.ListID = listId
		contactImport.Status = models.ContactImportStatusCompleted
		if err := tx.Create(contactImport).Error; err != nil {
			tx.Rollback()
			return log.Error("failed to create contact import ❌", err)
		}
		contact = &models.Contact{
			Email:     to,
			TeamID:    teamId,
			ListID:    listId,
			ImportID:  contactImport.ID,
			FirstName: variables["name"],
		}
		if err := tx.Create(contact).Error; err != nil {
			tx.Rollback()
			return log.Error("failed to create contact ❌", err)
		}
	}

	if categoryId == "" {
		category := &models.EmailCategory{}
		if err := tx.Where("name = ? AND team_id = ?", "Transactional", teamId).First(category).Error; err != nil {
			tx.Rollback()
			return log.Error("failed to get category ❌", err)
		}
		categoryId = category.ID
	}
	// Get category
	category := &models.EmailCategory{}
	if err := tx.Where("id = ? AND team_id = ?", categoryId, teamId).First(category).Error; err != nil {
		tx.Rollback()
		return log.Error("failed to get category ❌", err)
	}

	// Get HTML content outside transaction since it's an external operation
	htmlFromTemplate, err := utils.GetHTMLFromURL(template.HtmlFile.SignedURL)
	if err != nil {
		tx.Rollback()
		return log.Error("failed to get html from template ❌", err)
	}

	parsedBody := utils.ReplaceVariables(htmlFromTemplate, variables)
	parsedSubject := subject
	if subject == "" {
		parsedSubject = utils.ReplaceVariables(template.Subject, variables)
	}

	jsonData, err := utils.MapToJSON(variables)

	if err != nil {
		tx.Rollback()
		return log.Error("failed to convert variables to json ❌", err)
	}

	// Create email
	email := &models.Email{
		From:         smtpConfig.Username,
		TeamID:       teamId,
		TemplateID:   template.ID,
		To:           to,
		Subject:      parsedSubject,
		Body:         parsedBody,
		Status:       models.EmailStatusPending,
		Data:         jsonData,
		ContactID:    contact.ID,
		CategoryID:   category.ID,
		SMTPConfigID: smtpConfig.ID,
		CampaignID:   campaignId,
	}

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
