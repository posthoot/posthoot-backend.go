package routes

import (
	"kori/internal/api/middleware"
	"kori/internal/config"
	"kori/internal/handlers"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// 📊 RegisterTrackingRoutes registers all tracking related routes
func RegisterTrackingRoutes(e *echo.Echo, h *handlers.TrackingHandler, cfg *config.Config, db *gorm.DB) {
	// Public tracking endpoints (no auth required)
	trackGroup := e.Group("/track")
	trackGroup.GET("/click/*", h.HandleEmailClick) // The * captures the rest of the URL
	trackGroup.GET("/open", h.HandleEmailOpen)

	// Analytics endpoints (require auth)
	analyticsGroup := e.Group("/analytics")
	// Add authentication middleware
	auth := middleware.NewAuthMiddleware(cfg.JWT.Secret)
	analyticsGroup.Use(auth.Middleware())

	analyticsGroup.Use(middleware.RequirePermissions(db, "analytics:read"))

	// Basic analytics
	analyticsGroup.GET("/email", h.GetEmailAnalytics)
	analyticsGroup.GET("/campaign", h.GetCampaignAnalytics)

	// Advanced analytics
	analyticsGroup.GET("/team/overview", h.GetTeamOverview)       // Team-wide stats
	analyticsGroup.GET("/campaign/compare", h.CompareCampaigns)   // Compare multiple campaigns
	analyticsGroup.GET("/heatmap", h.GetClickHeatmap)             // Click heatmap data
	analyticsGroup.GET("/engagement-times", h.GetEngagementTimes) // Best engagement times
	analyticsGroup.GET("/audience", h.GetAudienceInsights)        // Audience insights
	analyticsGroup.GET("/trends", h.GetTrendAnalysis)             // Trend analysis

	// Export endpoints
	analyticsGroup.GET("/export/email", h.ExportEmailAnalytics)       // Export email analytics
	analyticsGroup.GET("/export/campaign", h.ExportCampaignAnalytics) // Export campaign analytics
}
