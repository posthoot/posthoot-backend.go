package routes

import (
	"os"

	"kori/internal/api/middleware"
	"kori/internal/handlers"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func SetupSubscriptionRoutes(e *echo.Echo, db *gorm.DB) {
	subscriptionHandler := handlers.NewSubscriptionHandler(db)
	authMiddleware := middleware.NewAuthMiddleware(os.Getenv("JWT_SECRET"))

	// Public subscription routes
	subscriptions := e.Group("/api/v1/subscriptions")
	subscriptions.POST("", subscriptionHandler.CreateSubscription)
	subscriptions.POST("/webhook", subscriptionHandler.HandleWebhook)

	// Protected subscription routes
	protected := subscriptions.Group("")
	protected.Use(authMiddleware.Middleware())
	protected.GET("/portal", subscriptionHandler.GetManagementPortal)
	protected.GET("/features", subscriptionHandler.GetFeatures)
}
