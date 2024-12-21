package routes

import (
	"kori/internal/api/middleware"
	"kori/internal/config"
	"kori/internal/handlers"

	"github.com/labstack/echo/v4"
)

func SetupSMTPRoutes(e *echo.Echo, config *config.Config) {
	smtpHandler := handlers.NewSMTPHandler()

	smtp := e.Group("/api/v1/smtp")
	auth := middleware.NewAuthMiddleware(config.JWT.Secret)
	smtp.Use(auth.Middleware())

	smtp.POST("/test", smtpHandler.TestSMTPConnection)
}
