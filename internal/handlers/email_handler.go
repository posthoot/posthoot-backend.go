package handlers

import (
	"kori/internal/db"
	"kori/internal/events"
	"kori/internal/models"
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

type SendEmailRequest struct {
	TemplateID         string         `json:"templateId" validate:"required"`
	To                 string         `json:"to" validate:"required,email"`
	Variables          datatypes.JSON `json:"data" validate:"required"`
	SMTPConfigProvider string         `json:"provider" validate:"required,oneof=CUSTOM GMAIL OUTLOOK AMAZON"`
	Subject            string         `json:"subject" validate:"required"`
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
	}

	events.Emit("email.send", &email)

	return c.JSON(http.StatusOK, map[string]string{
		"status": "Email queued successfully",
	})
}
