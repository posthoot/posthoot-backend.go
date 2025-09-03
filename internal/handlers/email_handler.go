package handlers

import (
	"kori/internal/db"
	"kori/internal/events"
	"kori/internal/models"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

type SendEmailRequest struct {
	TemplateID         string         `json:"templateId"`
	To                 string         `json:"to" validate:"required,email"`
	Variables          datatypes.JSON `json:"data" validate:"required,json"`
	SMTPConfigProvider string         `json:"provider" validate:"omitempty,oneof=CUSTOM GMAIL OUTLOOK AMAZON"`
	Subject            string         `json:"subject"`
	Body               string         `json:"html"`
	CC                 string         `json:"cc"`
	BCC                string         `json:"bcc"`
	ReplyTo            string         `json:"replyTo"`
	Test               bool           `json:"test"`
	SendAt             time.Time      `json:"scheduleAt"`
}

// SendEmail sends an email using the provided template and variables
// @Summary Send an email
// @Description Send an email using the provided template and variables
// @Tags Email
// @Accept json
// @Produce json
// @Param request body SendEmailRequest true "Email request"
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Router /email [post]
func SendEmail(c echo.Context) error {
	var req SendEmailRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Get teamID from context (set by auth middleware)
	teamID := c.Get("teamID").(string)

	tx := db.GetDB().Begin()

	smtpConfig, err := models.GetSMTPConfig(teamID, "", req.SMTPConfigProvider, tx)

	if err != nil {
		tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get SMTP config")
	}

	email := models.Email{
		TeamID:       teamID,
		TemplateID:   req.TemplateID,
		To:           req.To,
		SMTPConfigID: smtpConfig.ID,
		Subject:      req.Subject,
		Data:         req.Variables,
		Body:         req.Body,
		CC:           req.CC,
		BCC:          req.BCC,
		ReplyTo:      req.ReplyTo,
		Test:         req.Test,
		SendAt:       req.SendAt,
	}

	events.Emit("email.send", &email)

	return c.JSON(http.StatusOK, map[string]string{
		"status": "Email queued successfully",
	})
}
