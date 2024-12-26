package handlers

import (
	"kori/internal/events"
	"kori/internal/models"
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

type SendEmailRequest struct {
	TemplateID         string         `json:"templateId" validate:"required"`
	To                 string         `json:"to" validate:"required,email"`
	Variables          datatypes.JSON `json:"variables" validate:"required"`
	SMTPConfigProvider string         `json:"smtpConfigProvider"`
	Subject            string         `json:"subject" validate:"required"`
}

func SendEmail(c echo.Context) error {
	var req SendEmailRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Get teamID from context (set by auth middleware)
	teamID := c.Get("teamID").(string)

	email := models.Email{
		TeamID:     teamID,
		TemplateID: req.TemplateID,
		To:         req.To,
		SMTPConfig: &models.SMTPConfig{
			Provider: req.SMTPConfigProvider,
		},
		Subject: req.Subject,
		Data:    req.Variables,
	}

	events.Emit("email.send", email)

	return c.JSON(http.StatusOK, map[string]string{
		"status": "Email queued successfully",
	})
}
