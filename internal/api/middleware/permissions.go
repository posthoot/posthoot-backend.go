package middleware

import (
	"context"
	"errors"
	"kori/internal/models"

	"gorm.io/gorm"
)

var (
	ErrInsufficientPermissions = errors.New("insufficient permissions")
	ErrInvalidAPIKey           = errors.New("invalid API key")
	ErrInvalidUser             = errors.New("invalid user")
)

// ValidateAPIKeyPermissions checks if an API key has the required permissions
func ValidateAPIKeyPermissions(ctx context.Context, db *gorm.DB, apiKeyID string, requiredPermissions []string) error {
	var apiKey models.APIKey
	if err := db.WithContext(ctx).
		Preload("Permissions.ResourcePermission.Resource").
		First(&apiKey, "id = ?", apiKeyID).Error; err != nil {
		return ErrInvalidAPIKey
	}

	// Create a map of the API key's permissions for faster lookup
	permissionMap := make(map[string]bool)
	for _, perm := range apiKey.Permissions {
		if perm.ResourcePermission != nil && perm.ResourcePermission.Resource != nil {
			scope := perm.ResourcePermission.Scope
			permissionMap[scope] = true
		}
	}

	// Check if the API key has all required permissions
	for _, required := range requiredPermissions {
		if !permissionMap[required] {
			return ErrInsufficientPermissions
		}
	}

	return nil
}

// ValidateUserPermissions checks if a user has the required permissions
func ValidateUserPermissions(ctx context.Context, db *gorm.DB, userID string, requiredPermissions []string) error {
	var user models.User
	if err := db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return ErrInvalidUser
	}

	// Super admins and admins have all permissions
	if user.Role == models.UserRoleSuperAdmin || user.Role == models.UserRoleAdmin {
		return nil
	}

	// For regular users, check their specific permissions
	var permissions []models.UserPermission
	if err := db.WithContext(ctx).
		Preload("ResourcePermission.Resource").
		Where("user_id = ?", userID).
		Find(&permissions).Error; err != nil {
		return err
	}

	// Create a map of the user's permissions for faster lookup
	permissionMap := make(map[string]bool)
	for _, perm := range permissions {
		if perm.ResourcePermission != nil {
			scope := perm.ResourcePermission.Scope
			permissionMap[scope] = true
		}
	}

	// Check if the user has all required permissions
	for _, required := range requiredPermissions {
		if !permissionMap[required] {
			return ErrInsufficientPermissions
		}
	}

	return nil
}

// ValidateTeamPermissions checks if a user has the required permissions within a team
func ValidateTeamPermissions(ctx context.Context, db *gorm.DB, userID string, teamID string, requiredPermissions []string) error {
	var user models.User
	if err := db.WithContext(ctx).First(&user, "id = ? AND team_id = ?", userID, teamID).Error; err != nil {
		return ErrInvalidUser
	}

	// Super admins and admins have all permissions
	if user.Role == models.UserRoleSuperAdmin || user.Role == models.UserRoleAdmin {
		return nil
	}

	// For regular users, check their team-specific permissions
	var permissions []models.UserPermission
	if err := db.WithContext(ctx).
		Preload("ResourcePermission.Resource").
		Where("user_id = ? AND team_id = ?", userID, teamID).
		Find(&permissions).Error; err != nil {
		return err
	}

	// Create a map of the user's team permissions for faster lookup
	permissionMap := make(map[string]bool)
	for _, perm := range permissions {
		if perm.ResourcePermission != nil {
			scope := perm.ResourcePermission.Scope
			permissionMap[scope] = true
		}
	}

	// Check if the user has all required permissions in the team context
	for _, required := range requiredPermissions {
		if !permissionMap[required] {
			return ErrInsufficientPermissions
		}
	}

	return nil
}

// Helper function to check if a user has a specific permission scope
func HasPermissionScope(permissions []models.ResourcePermission, scope string) bool {
	for _, p := range permissions {
		if p.Scope == scope {
			return true
		}
	}
	return false
}

// Helper function to check if a user has a specific role
func HasRole(user models.User, roles ...models.UserRole) bool {
	for _, role := range roles {
		if user.Role == role {
			return true
		}
	}
	return false
}

// Helper function to check if a user is an admin or super admin
func IsAdminUser(user models.User) bool {
	return user.Role == models.UserRoleAdmin || user.Role == models.UserRoleSuperAdmin
}

// Helper function to build a permission scope string
func BuildPermissionScope(resource, action string) string {
	return action + ":" + resource
}
