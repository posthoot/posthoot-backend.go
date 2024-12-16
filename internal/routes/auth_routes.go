package routes

import (
	"kori/internal/handlers"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func SetupAuthRoutes(e *echo.Echo, db *gorm.DB) {
	authHandler := handlers.NewAuthHandler(db)

	// Auth routes group
	auth := e.Group("/api/v1/auth")

	// Register routes
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/password-reset", authHandler.RequestPasswordReset)
	auth.POST("/password-reset/verify", authHandler.VerifyResetCode)
}
