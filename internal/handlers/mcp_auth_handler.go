package handlers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	"kori/internal/models"
)

// MCPAuthorize verifies a workspace API key without requiring a resource grant.
// Tool endpoints still enforce their own resource permissions on every call.
// This endpoint returns no identity, permissions, or credential data.
func MCPAuthorize(db *gorm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", "no-store")
		key := c.Request().Header.Get("X-API-Key")
		if key == "" || len(key) > 4096 {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API key")
		}
		var count int64
		if err := db.WithContext(c.Request().Context()).Model(&models.APIKey{}).
			Where("key = ? AND is_deleted = false AND team_id IS NOT NULL", key).
			Where("expires_at IS NULL OR expires_at = ? OR expires_at > ?", time.Time{}, time.Now()).Count(&count).Error; err != nil {
			return echo.NewHTTPError(503, "Authentication unavailable")
		}
		if count != 1 {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API key")
		}
		return c.NoContent(http.StatusNoContent)
	}
}
