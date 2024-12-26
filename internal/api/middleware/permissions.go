package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// Permission scopes
const (
	ScopeAdmin = "ADMIN"
	ScopeRead  = "READ"
	ScopeWrite = "WRITE"
)

// ValidateAPIKeyPermissions validates if an API key has the required permissions
func ValidateAPIKeyPermissions(ctx context.Context, db *gorm.DB, apiKeyID string, requiredPermissions []string) error {
	var permissions []string
	err := db.Raw(`
		SELECT CONCAT(resource, ':', scope) as permission
		FROM api_key_permissions
		WHERE key_id = ?
	`, apiKeyID).Scan(&permissions).Error
	if err != nil {
		return fmt.Errorf("failed to get permissions: %w", err)
	}

	// Check if the API key has admin scope
	for _, perm := range permissions {
		if strings.HasSuffix(perm, ":"+ScopeAdmin) {
			return nil // Admin has all permissions
		}
	}

	// Check each required permission
	for _, required := range requiredPermissions {
		hasPermission := false
		requiredParts := strings.Split(required, ":")
		if len(requiredParts) != 2 {
			continue // Invalid permission format
		}

		requiredResource := requiredParts[0]
		requiredScope := requiredParts[1]

		for _, perm := range permissions {
			permParts := strings.Split(perm, ":")
			if len(permParts) != 2 {
				continue
			}

			permResource := permParts[0]
			permScope := permParts[1]

			// Check if the permission matches
			if permResource == requiredResource {
				switch permScope {
				case ScopeAdmin:
					hasPermission = true
				case ScopeWrite:
					hasPermission = requiredScope == ScopeWrite || requiredScope == ScopeRead
				case ScopeRead:
					hasPermission = requiredScope == ScopeRead
				}
				if hasPermission {
					break
				}
			}
		}

		if !hasPermission {
			return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("missing required permission: %s", required))
		}
	}

	return nil
}

// ValidateMethodPermission validates if a given scope allows a specific HTTP method
func ValidateMethodPermission(method string, scope string) bool {
	switch scope {
	case ScopeAdmin:
		return true
	case ScopeWrite:
		return method == http.MethodPost || method == http.MethodPut ||
			method == http.MethodDelete || method == http.MethodPatch
	case ScopeRead:
		return method == http.MethodGet
	default:
		return false
	}
}

// GetRequiredPermissionForMethod returns the required permission scope for a given HTTP method
func GetRequiredPermissionForMethod(method string) string {
	switch method {
	case http.MethodGet:
		return ScopeRead
	case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		return ScopeWrite
	default:
		return ""
	}
}

// RequirePermissions middleware checks if the user/API key has the required permissions
func RequirePermissions(db *gorm.DB, requiredPermissions ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if user has admin access first
			if hasAdmin, ok := c.Get("hasAdminAccess").(bool); ok && hasAdmin {
				return next(c)
			}

			isAPIKey := c.Get("isAPIKey").(bool)
			method := c.Request().Method

			if isAPIKey {
				apiKeyID := c.Get("apiKeyID").(string)
				if err := ValidateAPIKeyPermissions(c.Request().Context(), db, apiKeyID, requiredPermissions); err != nil {
					return err
				}
			} else {
				// For JWT auth, check role-based permissions
				role := c.Get("role").(string)
				scopes := c.Get("scopes").([]string)

				// Admin role has all permissions
				if role == "admin" {
					return next(c)
				}

				// Check if user has any of the required permissions
				hasPermission := false
				for _, scope := range scopes {
					if ValidateMethodPermission(method, scope) {
						hasPermission = true
						break
					}
				}

				if !hasPermission {
					return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
				}
			}

			return next(c)
		}
	}
}
