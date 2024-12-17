package models

import (
	"encoding/json"
	"fmt"
	"os"

	"gorm.io/gorm"
)

type InitialTemplates struct {
	MailingLists []MailingList `json:"mailing_lists"`
	Templates    []Template    `json:"templates"`
}

type InitialCategories struct {
	Categories []string `json:"categories"`
}

type InitialLists struct {
	Lists []MailingList `json:"lists"`
}

// LoadInitialData loads all initial data from JSON files
func LoadInitialData(db *gorm.DB, teamId string) error {
	log.Info(fmt.Sprintf("Loading initial data for team %s", teamId))

	// Load categories
	if err := loadCategories(db, teamId); err != nil {
		return log.Error("Failed to load categories", err)
	}

	// Load lists
	if err := loadLists(db, teamId); err != nil {
		return log.Error("Failed to load lists", err)
	}

	// Load templates and mailing lists
	if err := loadTemplates(db, teamId); err != nil {
		return log.Error("Failed to load templates", err)
	}

	log.Success("Successfully loaded all initial data")
	return nil
}

func loadJSON[T any](filename string) (*T, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", filename, err)
	}

	return &result, nil
}

func loadCategories(db *gorm.DB, teamId string) error {
	log.Info("Loading categories")
	categories, err := loadJSON[InitialCategories]("internal/models/initial-seup/initial-categories.json")
	if err != nil {
		return err
	}

	for _, category := range categories.Categories {
		if err := db.Create(&EmailCategory{
			Name:   category,
			TeamID: teamId,
		}).Error; err != nil {
			return log.Error("Failed to create category", err)
		}
	}

	return nil
}

func loadLists(db *gorm.DB, teamId string) error {
	log.Info("Loading lists")
	lists, err := loadJSON[InitialLists]("internal/models/initial-seup/initial-lists.json")
	if err != nil {
		return err
	}

	for _, list := range lists.Lists {
		list.TeamID = teamId
		if err := db.Create(&list).Error; err != nil {
			return log.Error("Failed to create list", err)
		}
	}

	return nil
}

func loadTemplates(db *gorm.DB, teamId string) error {
	templates, err := loadJSON[InitialTemplates]("internal/models/initial-seup/initial-templates.json")
	if err != nil {
		return err
	}

	// Create mailing lists
	for _, list := range templates.MailingLists {
		list.TeamID = teamId
		if err := db.Create(&list).Error; err != nil {
			return log.Error("Failed to create mailing list", err)
		}
	}

	// Create templates
	for _, tmpl := range templates.Templates {
		tmpl.TeamID = teamId
		category := EmailCategory{}
		if err := db.Where("name = ? AND team_id = ?", tmpl.Category, teamId).First(&category).Error; err != nil {
			return log.Error("Failed to find category", err)
		}
		tmpl.Category = nil
		tmpl.CategoryID = category.ID
		if err := db.Create(&tmpl).Error; err != nil {
			return log.Error("Failed to create template", err)
		}
	}

	return nil
}
