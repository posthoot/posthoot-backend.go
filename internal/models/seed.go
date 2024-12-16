package models

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log"
	"os"
	"strings"

	"gorm.io/gorm"
)

// Default resources and their actions
var defaultResources = []Resource{
	// Campaign resources
	{Name: "campaigns", Action: "create"},
	{Name: "campaigns", Action: "read"},
	{Name: "campaigns", Action: "update"},
	{Name: "campaigns", Action: "delete"},

	// Template resources
	{Name: "templates", Action: "create"},
	{Name: "templates", Action: "read"},
	{Name: "templates", Action: "update"},
	{Name: "templates", Action: "delete"},

	// Mailing list resources
	{Name: "lists", Action: "create"},
	{Name: "lists", Action: "read"},
	{Name: "lists", Action: "update"},
	{Name: "lists", Action: "delete"},

	// Contact resources
	{Name: "contacts", Action: "create"},
	{Name: "contacts", Action: "read"},
	{Name: "contacts", Action: "update"},
	{Name: "contacts", Action: "delete"},

	// Team resources
	{Name: "teams", Action: "create"},
	{Name: "teams", Action: "read"},
	{Name: "teams", Action: "update"},
	{Name: "teams", Action: "delete"},

	// User resources
	{Name: "users", Action: "create"},
	{Name: "users", Action: "read"},
	{Name: "users", Action: "update"},
	{Name: "users", Action: "delete"},

	// API key resources
	{Name: "api_keys", Action: "create"},
	{Name: "api_keys", Action: "read"},
	{Name: "api_keys", Action: "update"},
	{Name: "api_keys", Action: "delete"},

	// Automation resources
	{Name: "automations", Action: "create"},
	{Name: "automations", Action: "read"},
	{Name: "automations", Action: "update"},
	{Name: "automations", Action: "delete"},

	// SMTP config resources
	{Name: "smtp_configs", Action: "create"},
	{Name: "smtp_configs", Action: "read"},
	{Name: "smtp_configs", Action: "update"},
	{Name: "smtp_configs", Action: "delete"},

	// Domain resources
	{Name: "domains", Action: "create"},
	{Name: "domains", Action: "read"},
	{Name: "domains", Action: "update"},
	{Name: "domains", Action: "delete"},

	// Permission resources
	{Name: "permissions", Action: "create"},
	{Name: "permissions", Action: "read"},
	{Name: "permissions", Action: "update"},
	{Name: "permissions", Action: "delete"},

	// Role resources
	{Name: "roles", Action: "create"},
	{Name: "roles", Action: "read"},
	{Name: "roles", Action: "update"},
	{Name: "roles", Action: "delete"},

	// Webhook resources
	{Name: "webhooks", Action: "create"},
	{Name: "webhooks", Action: "read"},
	{Name: "webhooks", Action: "update"},
	{Name: "webhooks", Action: "delete"},

	// Model resources
	{Name: "models", Action: "create"},
	{Name: "models", Action: "read"},
	{Name: "models", Action: "update"},
	{Name: "models", Action: "delete"},
}

// Role-based permission mappings
var rolePermissions = map[UserRole][]string{
	UserRoleAdmin: {
		// Admin has all permissions
		"campaigns:*", "templates:*", "lists:*", "contacts:*",
		"teams:*", "users:*", "api_keys:*", "automations:*",
		"smtp_configs:*", "domains:*",
		"permissions:*",
		"roles:*",
		"webhooks:*",
		"models:*",
	},
	UserRoleMember: {
		// Member has limited permissions
		"campaigns:read", "campaigns:create",
		"templates:read",
		"lists:read",
		"contacts:read", "contacts:create",
		"teams:read",
		"users:read",
		"automations:read",
		"smtp_configs:read",
		"domains:read",
		"webhooks:read",
		"models:read",
	},
}

// SeedPermissions creates default resources and permissions
func SeedPermissions(db *gorm.DB) error {
	// Create resources
	for _, resource := range defaultResources {
		if err := db.FirstOrCreate(&resource, Resource{
			Name:   resource.Name,
			Action: resource.Action,
		}).Error; err != nil {
			return fmt.Errorf("failed to create resource %s:%s: %v", resource.Name, resource.Action, err)
		}
	}

	// Create resource permissions for each role
	for role, permissions := range rolePermissions {
		log.Printf("Creating permissions for role: %s", role)

		for _, permScope := range permissions {
			// Handle wildcard permissions
			if strings.HasSuffix(permScope, ":*") {
				resourceName := strings.TrimSuffix(permScope, ":*") // Remove :*
				var resources []Resource
				if err := db.Where("name = ?", resourceName).Find(&resources).Error; err != nil {
					return fmt.Errorf("failed to find resources for %s: %v", resourceName, err)
				}

				// Create permissions for all actions of this resource
				for _, resource := range resources {
					if err := createResourcePermission(db, resource); err != nil {
						return err
					}
				}
			} else {
				// Handle specific permissions
				parts := strings.Split(permScope, ":")
				if len(parts) != 2 {
					return fmt.Errorf("invalid permission scope format: %s", permScope)
				}

				resourceName, action := parts[0], parts[1]
				var resource Resource
				if err := db.Where("name = ? AND action = ?", resourceName, action).First(&resource).Error; err != nil {
					return fmt.Errorf("failed to find resource %s:%s: %v", resourceName, action, err)
				}

				if err := createResourcePermission(db, resource); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func createResourcePermission(db *gorm.DB, resource Resource) error {
	scope := fmt.Sprintf("%s:%s", resource.Name, resource.Action)

	permission := ResourcePermission{
		ResourceID: resource.ID,
		Scope:      scope,
	}

	if err := db.FirstOrCreate(&permission, ResourcePermission{
		ResourceID: resource.ID,
		Scope:      scope,
	}).Error; err != nil {
		return fmt.Errorf("failed to create permission %s: %v", scope, err)
	}

	return nil
}

// AssignDefaultPermissions assigns default permissions to a user based on their role
func AssignDefaultPermissions(db *gorm.DB, user *User) error {
	var permissions []ResourcePermission

	if user.Role == UserRoleAdmin {
		// For admin, get all resource permissions
		if err := db.Find(&permissions).Error; err != nil {
			return fmt.Errorf("failed to fetch permissions: %v", err)
		}
	} else {
		// For other roles, get specific permissions based on rolePermissions mapping
		rolePerm := rolePermissions[user.Role]
		for _, permScope := range rolePerm {
			if strings.HasSuffix(permScope, ":*") {
				// Handle wildcard permissions
				resourceName := strings.TrimSuffix(permScope, ":*")
				var resources []Resource
				if err := db.Where("name = ?", resourceName).Find(&resources).Error; err != nil {
					return fmt.Errorf("failed to find resources for %s: %v", resourceName, err)
				}

				for _, resource := range resources {
					var perm ResourcePermission
					if err := db.Where("resource_id = ?", resource.ID).First(&perm).Error; err != nil {
						return fmt.Errorf("failed to find permission for resource %s: %v", resource.Name, err)
					}
					permissions = append(permissions, perm)
				}
			} else {
				// Handle specific permissions
				parts := strings.Split(permScope, ":")
				if len(parts) != 2 {
					return fmt.Errorf("invalid permission scope format: %s", permScope)
				}

				resourceName, action := parts[0], parts[1]
				var resource Resource
				if err := db.Where("name = ? AND action = ?", resourceName, action).First(&resource).Error; err != nil {
					return fmt.Errorf("failed to find resource %s:%s: %v", resourceName, action, err)
				}

				var perm ResourcePermission
				if err := db.Where("resource_id = ?", resource.ID).First(&perm).Error; err != nil {
					return fmt.Errorf("failed to find permission for resource %s: %v", resource.Name, err)
				}
				permissions = append(permissions, perm)
			}
		}
	}

	// Create UserPermission entries
	for _, perm := range permissions {
		userPerm := UserPermission{
			UserID:               user.ID,
			ResourcePermissionID: perm.ID,
		}

		if err := db.FirstOrCreate(&userPerm, UserPermission{
			UserID:               user.ID,
			ResourcePermissionID: perm.ID,
		}).Error; err != nil {
			return fmt.Errorf("failed to create user permission: %v", err)
		}
	}

	return nil
}

func CreateSuperAdminFromEnv(db *gorm.DB) error {
	// check if super admin already exists
	var count int64
	db.Model(&User{}).Where("role = ?", UserRoleSuperAdmin).Count(&count)
	fmt.Println(count)
	if count > 0 {
		return nil
	}

	email, ok := os.LookupEnv("SUPERADMIN_EMAIL")

	if !ok {
		return fmt.Errorf("SUPERADMIN_EMAIL not set")
	}

	password, ok := os.LookupEnv("SUPERADMIN_PASSWORD")

	if !ok {
		return fmt.Errorf("SUPERADMIN_PASSWORD not set")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	name, ok := os.LookupEnv("SUPERADMIN_NAME")

	if !ok {
		return fmt.Errorf("SUPERADMIN_NAME not set")
	}

	role := UserRoleSuperAdmin

	team := Team{
		Name: name + "'s Team",
	}

	if err := db.Create(&team).Error; err != nil {
		return fmt.Errorf("failed to create team: %v", err)
	}

	user := User{
		FirstName: name,
		LastName:  "",
		Email:     email,
		Role:      role,
		Password:  string(hashedPassword),
		TeamID:    team.ID,
	}

	if err := db.Create(&user).Error; err != nil {
		return fmt.Errorf("failed to create superadmin user: %v", err)
	}

	return nil
}
