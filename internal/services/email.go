package services

import (
	"encoding/json"
	"fmt"
	"kori/internal/db"
	"kori/internal/events"
	"kori/internal/models"
	"kori/internal/utils"

	console "kori/internal/utils/logger"
)

var logger = console.New("EMAIL")

func init() {
	// Register event handlers
	events.On("invite.created", func(data interface{}) {
		invite := data.(*models.TeamInvite)
		logger.Info(fmt.Sprintf("Sending invite email to %s", invite.Email))
		sendTeamInviteEmail(invite)
	})
	events.On("users.created", func(data interface{}) {
		user := data.(*models.User)
		logger.Info(fmt.Sprintf("Sending welcome email to %s", user.Email))
		// sendWelcomeEmail(user)
	})
}

func sendTeamInviteEmail(invite *models.TeamInvite) error {
	// Start transaction
	tx := db.DB.Begin()
	if tx.Error != nil {
		return logger.Error("failed to begin transaction", tx.Error)
	}

	// Get team details
	team := &models.Team{}
	if err := tx.First(team, invite.TeamID).Error; err != nil {
		tx.Rollback()
		return logger.Error("failed to get team details", err)
	}

	// Get inviter details
	inviter := &models.User{}
	if err := tx.First(inviter, invite.InviterID).Error; err != nil {
		tx.Rollback()
		return logger.Error("failed to get inviter details", err)
	}

	var template *models.Template

	if team.Settings.InviteTemplateID == "" {
		// Get invite template
		template = &models.Template{}
		if err := tx.Where("name = ? AND team_id = ?", "PLATFORM INVITE", invite.TeamID).First(template).Error; err != nil {
			tx.Rollback()
			return logger.Error("failed to get invite template", err)
		}
	} else {
		template = &models.Template{}
		if err := tx.Where("id = ? AND team_id = ?", team.Settings.InviteTemplateID, invite.TeamID).First(template).Error; err != nil {
			tx.Rollback()
			return logger.Error("failed to get invite template", err)
		}
	}

	// Get invite mailing list
	mailingList := &models.MailingList{}
	if err := tx.Where("name = ? AND team_id = ?", "Invited Users", invite.TeamID).First(mailingList).Error; err != nil {
		tx.Rollback()
		return logger.Error("failed to get invite mailing list", err)
	}

	// Prepare email data
	emailData := json.RawMessage(fmt.Sprintf(`{
        "inviterName": "%s %s",
        "teamName": "%s",
        "inviteLink": "%s",
        "name": "%s"
    }`, inviter.FirstName, inviter.LastName, team.Name, fmt.Sprintf("/accept-invite/%s", invite.ID), invite.Name))

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
			return logger.Error("failed to create contact", err)
		}
	}

	// Get default SMTP config
	smtpConfig := &models.SMTPConfig{}
	if err := tx.Where("team_id = ?", invite.TeamID).First(smtpConfig).Error; err != nil {
		tx.Rollback()
		return logger.Error("failed to get default smtp config", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return logger.Error("failed to commit transaction", err)
	}

	// Send email outside transaction since it's an external operation
	return sendEmail(
		invite.TeamID,
		template.ID,
		invite.Email,
		map[string]string{
			"inviterName": fmt.Sprintf("%s %s", inviter.FirstName, inviter.LastName),
			"name":        invite.Name,
			"teamName":    team.Name,
			"inviteLink":  fmt.Sprintf("/accept-invite/%s", invite.ID),
		},
		smtpConfig.Provider,
		template.CategoryID,
		emailData,
		"",
		mailingList.ID,
		"",
	)
}

func sendEmail(teamId string, templateId string, to string, variables map[string]string, SMTPProvider string, categoryId string, data json.RawMessage, subject string, listId string, campaignId string) error {
	// Start transaction
	tx := db.DB.Begin()
	if tx.Error != nil {
		return logger.Error("failed to begin transaction", tx.Error)
	}

	// Get SMTP config
	smtpConfig := &models.SMTPConfig{}
	if err := tx.Where("team_id = ? AND provider = ?", teamId, SMTPProvider).First(smtpConfig).Error; err != nil {
		tx.Rollback()
		return logger.Error("failed to get team smtp config ❌", err)
	}

	// Get template
	template := &models.Template{}
	if err := tx.Where("id = ? AND team_id = ?", templateId, teamId).First(template).Error; err != nil {
		tx.Rollback()
		return logger.Error("failed to get template ❌", err)
	}

	// Get or use default mailing list
	if listId == "" {
		mailingList := &models.MailingList{}
		if err := tx.Where("team_id = ? AND name = ?", teamId, "All Users").First(mailingList).Error; err != nil {
			tx.Rollback()
			return logger.Error("failed to get default mailing list ❌", err)
		}
		listId = mailingList.ID
	}

	// Get or create contact
	contact := &models.Contact{}
	if err := tx.Where("email = ? AND team_id = ? AND list_id = ?", to, teamId, listId).First(contact).Error; err != nil {
		contact = &models.Contact{
			Email:     to,
			TeamID:    teamId,
			ListID:    listId,
			FirstName: variables["name"],
		}
		if err := tx.Create(contact).Error; err != nil {
			tx.Rollback()
			return logger.Error("failed to create contact ❌", err)
		}
	}

	// Get category
	category := &models.EmailCategory{}
	if err := tx.Where("id = ? AND team_id = ?", categoryId, teamId).First(category).Error; err != nil {
		tx.Rollback()
		return logger.Error("failed to get category ❌", err)
	}

	// Get HTML content outside transaction since it's an external operation
	htmlFromTemplate, err := utils.GetHTMLFromURL(template.HtmlFile)
	if err != nil {
		tx.Rollback()
		return logger.Error("failed to get html from template ❌", err)
	}

	parsedBody := utils.ReplaceVariables(htmlFromTemplate, variables)
	parsedSubject := subject
	if subject == "" {
		parsedSubject = utils.ReplaceVariables(template.Subject, variables)
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
		Data:         data,
		ContactID:    contact.ID,
		CategoryID:   category.ID,
		SMTPConfigID: smtpConfig.ID,
		CampaignID:   campaignId,
	}

	if err := tx.Create(email).Error; err != nil {
		tx.Rollback()
		return logger.Error("failed to create email ❌", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return logger.Error("failed to commit transaction", err)
	}

	return nil // ✅
}
