package models

import "gorm.io/gorm"

func (a *APIKey) AfterCreate(tx *gorm.DB) error {

	// 🔍 Get resources for read and create actions
	var resources []Resource
	if err := tx.Where("name = ? AND action IN (?)", "emails", []string{"read", "create"}).Find(&resources).Error; err != nil {
		return err
	}

	log.Info("Found resources %v", resources)

	// 🔑 Get permissions for found resources
	var permissions []ResourcePermission
	var resourceIDs []string
	for _, r := range resources {
		resourceIDs = append(resourceIDs, r.ID)
	}

	log.Info("Found resources %v", resourceIDs)

	if err := tx.Where("resource_id IN ? AND scope IN ?", resourceIDs, []string{"emails:read", "emails:create"}).Find(&permissions).Error; err != nil {
		return err
	}

	log.Info("Found permissions %v", permissions)

	for _, p := range permissions {
		// create api key permission
		apiKeyPermission := &APIKeyPermission{
			KeyID:                a.ID,
			ResourcePermissionID: p.ID,
		}
		if err := tx.Create(apiKeyPermission).Error; err != nil {
			return err
		}
	}

	return nil
}
