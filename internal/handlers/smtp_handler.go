package handlers

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/smtp"

	"kori/internal/api/middleware"
	"kori/internal/utils/logger"

	"github.com/labstack/echo/v4"
)

var log = logger.New("smtp_handler")

type SMTPHandler struct{}

type SMTPTestRequest struct {
	Host       string `json:"host" validate:"required"`
	Port       int    `json:"port" validate:"required"`
	Username   string `json:"username" validate:"required"`
	Password   string `json:"password" validate:"required"`
	From       string `json:"from" validate:"required"`
	RequireTLS bool   `json:"requireTls" default:"false"`
}

func NewSMTPHandler() *SMTPHandler {
	return &SMTPHandler{}
}

// TestSMTPConnection tests SMTP connection with provided credentials
func (h *SMTPHandler) TestSMTPConnection(c echo.Context) error {
	teamID := middleware.GetTeamID(c)
	if teamID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "Team ID not found")
	}

	var req SMTPTestRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body: "+err.Error())
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	auth := smtp.PlainAuth("", req.Username, req.Password, req.Host)

	// Try to connect and authenticate
	addr := fmt.Sprintf("%s:%d", req.Host, req.Port)
	client, err := smtp.Dial(addr)
	if err != nil {
		log.Error("Failed to connect to SMTP server", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to connect to SMTP server")
	}
	defer client.Close()
	if !req.RequireTLS {
		if err = client.Auth(auth); err != nil {
			log.Error("SMTP authentication failed", err)
			return echo.NewHTTPError(http.StatusBadRequest, "SMTP authentication failed: "+err.Error())
		}
	}

	if req.RequireTLS {
		if err = client.StartTLS(
			&tls.Config{
				ServerName: req.Host,
			},
		); err != nil {
			log.Error("Failed to start TLS", err)
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to start TLS: "+err.Error())
		}
	}

	// maybe try to send a dummy email to our own email

	client.Mail(req.From)
	client.Rcpt(req.From)

	writer, err := client.Data()
	if err != nil {
		log.Error("Failed to send test email", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to send test email")
	}

	writer.Write([]byte("Hello, world! This is a test email from Posthoot."))
	writer.Close()

	client.Quit()

	log.Success("SMTP connection test successful for team %s", teamID)
	return c.JSON(http.StatusOK, map[string]string{
		"message": "SMTP connection test successful",
	})
}
