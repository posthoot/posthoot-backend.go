package handlers

import (
	"crypto/tls"
	"net/http"

	"kori/internal/utils/logger"

	"github.com/labstack/echo/v4"
	"gopkg.in/gomail.v2"
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
	RequireSSL bool   `json:"requireSSL" default:"false"`
	To         string `json:"to" validate:"omitempty"`
}

func NewSMTPHandler() *SMTPHandler {
	return &SMTPHandler{}
}

// TestSMTPConnection tests SMTP connection with provided credentials
func (h *SMTPHandler) TestSMTPConnection(c echo.Context) error {
	var req SMTPTestRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body: "+err.Error())
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Create a new message
	m := gomail.NewMessage()
	m.SetHeader("From", req.From)
	if req.To != "" {
		m.SetHeader("To", req.To)
	} else {
		m.SetHeader("To", req.From)
	}
	m.SetHeader("Subject", "Test Email from Posthoot")
	m.SetBody("text/html", "Hello, this is a test email from Posthoot!")

	// Create dialer with TLS config
	d := gomail.NewDialer(req.Host, req.Port, req.Username, req.Password)

	if req.RequireTLS {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if req.RequireSSL {
		d.SSL = true
	}

	// Try to send test email
	if err := d.DialAndSend(m); err != nil {
		log.Error("Failed to send test email", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to send test email: "+err.Error())
	}

	log.Success("SMTP connection test successful")
	return c.JSON(http.StatusOK, map[string]string{
		"message": "SMTP connection test successful",
	})
}
