package models

import (
	"encoding/json"
	"fmt"
	"os"

	"gorm.io/gorm"
)

type JSONTemplate struct {
	*Template
	CategoryName string `json:"categoryName"`
}

type InitialTemplates struct {
	MailingLists []MailingList  `json:"mailing_lists"`
	Templates    []JSONTemplate `json:"templates"`
}

type InitialCategories struct {
	Categories []string `json:"categories"`
}

type InitialLists struct {
	Lists []MailingList `json:"lists"`
}

// LoadInitialData loads all initial data from JSON files
func LoadInitialData(db *gorm.DB, teamId string) error {
	log.Info("Loading initial data for team %s", teamId)

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
	categories, err := loadJSON[InitialCategories]("internal/models/seeder/initial-setup/initial-categories.json")
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
	lists, err := loadJSON[InitialLists]("internal/models/seeder/initial-setup/initial-lists.json")
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
	templates, err := loadJSON[InitialTemplates]("internal/models/seeder/initial-setup/initial-templates.json")
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

	templateIds := make(map[string]string)

	templateIds["WELCOME"] = ""
	templateIds["INVITE"] = ""

	// Create templates
	for _, tmpl := range templates.Templates {
		tmpl.TeamID = teamId
		category := EmailCategory{}
		if err := db.Where("name = ? AND team_id = ?", tmpl.CategoryName, teamId).First(&category).Error; err != nil {
			return log.Error("Failed to find category", err)
		}

		file := File{
			TeamID: teamId,
			Path:   tmpl.HtmlFileID,
			Name:   fmt.Sprintf("%s", tmpl.HtmlFileID),
		}

		if err := db.Create(&file).Error; err != nil {
			return log.Error("Failed to create file", err)
		}

		template := Template{
			Name:       tmpl.Name,
			Subject:    tmpl.Subject,
			HtmlFileID: file.ID,
			DesignJSON: tmpl.DesignJSON,
			TeamID:     teamId,
			CategoryID: category.ID,
			Variables:  tmpl.Variables,
		}
		if err := db.Create(&template).Error; err != nil {
			return log.Error("Failed to create template", err)
		}

		if tmpl.Name == "PLATFORM WELCOME" {
			templateIds["WELCOME"] = template.ID
		} else if tmpl.Name == "PLATFORM INVITE" {
			templateIds["INVITE"] = template.ID
		}
	}

	teamSettings := TeamSettings{
		TeamID:            teamId,
		WelcomeTemplateID: templateIds["WELCOME"],
		InviteTemplateID:  templateIds["INVITE"],
	}
	if err := db.Create(&teamSettings).Error; err != nil {
		return log.Error("Failed to create team settings", err)
	}

	return nil
}
