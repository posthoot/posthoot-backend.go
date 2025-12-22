package models

import (
	"kori/internal/events"

	"gorm.io/gorm"
)

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

func (t *TeamInvite) AfterCreate(tx *gorm.DB) error {
	log.Info("Team invite created %v", t)
	events.Emit("invite.created", t)
	return nil
}

func (c *ContactImport) AfterCreate(tx *gorm.DB) error {
	log.Info("Contact import created %v", c)
	events.Emit("contact_import.created", c)
	return nil
}

func (c *Campaign) AfterCreate(tx *gorm.DB) error {
	log.Info("Campaign created %v", c)
	events.Emit("campaign.created", c)
	return nil
}

func (c *Contact) AfterCreate(tx *gorm.DB) error {
	log.Info("Contact created %v", c)
	events.Emit("contact.created", c)
	if err := SyncSubscribersCount(tx, c.List); err != nil {
		return err
	}
	return nil
}

func (c *Contact) AfterDelete(tx *gorm.DB) error {
	log.Info("Contact deleted %v", c)
	events.Emit("contact.deleted", c)
	if err := SyncSubscribersCount(tx, c.List); err != nil {
		return err
	}
	return nil
}

func (c *Contact) AfterUpdate(tx *gorm.DB) error {
	log.Info("Contact updated %v", c)
	events.Emit("contact.updated", c)
	if err := SyncSubscribersCount(tx, c.List); err != nil {
		return err
	}
	return nil
}

func SyncSubscribersCount(tx *gorm.DB, mailingList *MailingList) error {
	count := int64(0)
	err := tx.Model(mailingList).Where("list_id = ?", mailingList.ID).Where("status = ?", SubscriberStatusActive).Find(&Contact{}).Count(&count)
	if err != nil {
		return err.Error
	}
	mailingList.SubscribersCount = count
	return tx.Model(mailingList).Update("subscribers_count", count).Error
}
