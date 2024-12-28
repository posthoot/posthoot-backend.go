package routes

import (
	"kori/internal/api/middleware"
	"kori/internal/config"
	"kori/internal/handlers"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func SetupEMAILRoutes(e *echo.Echo, config *config.Config, db *gorm.DB) {
	// Create EMAIL routes group
	email := e.Group("/api/v1/email")

	// Add authentication middleware
	auth := middleware.NewAuthMiddleware(config.JWT.Secret)
	email.Use(auth.Middleware())

	email.Use(middleware.RequirePermissions(db, "email:write"))

	email.POST("", handlers.SendEmail)
}
