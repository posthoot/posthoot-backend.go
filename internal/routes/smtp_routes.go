package routes

import (
	"kori/internal/api/middleware"
	"kori/internal/config"
	"kori/internal/handlers"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func SetupSMTPRoutes(e *echo.Echo, config *config.Config, db *gorm.DB) {
	smtpHandler := handlers.NewSMTPHandler()

	// Create SMTP routes group
	smtp := e.Group("/api/v1/smtp")

	// Add authentication middleware
	auth := middleware.NewAuthMiddleware(config.JWT.Secret)
	smtp.Use(auth.Middleware())

	// Add permission middleware for SMTP management
	smtp.Use(middleware.RequirePermissions(db, "smtp_configs:read"))

	// SMTP test route
	// @Summary Test SMTP connection
	// @Description Test SMTP connection
	// @Accept json
	// @Produce json
	// @Success 200 {object} map[string]string "SMTP connection test successful"
	// @Failure 400 {object} map[string]string "Validation error or SMTP configuration not found"
	// @Failure 500 {object} map[string]string "Internal server error"
	// @Router /api/v1/smtp/test [post]
	smtp.POST("/test", smtpHandler.TestSMTPConnection)
}
