package airley

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"kori/internal/models"
	"kori/internal/utils/base64"
	console "kori/internal/utils/logger"

	"kori/internal/handlers"
)

var log = console.New("SEEDER")

func LoadAirleyTemplates(db *gorm.DB) error {
	team, err := models.GetTeamByName("Team Airley", db)
	if err != nil {
		return log.Error("Failed to get team", err)
	}

	templates, err := models.LoadJSON[models.InitialTemplates]("internal/models/seeder/airley/templates.json")
	if err != nil {
		return err
	}

	// get api key
	apiKey := &models.APIKey{}
	if err := db.Where("team_id = ? AND name = ?", team.ID, "Airley API Key").First(apiKey).Error; err != nil {
		// create api key
		apiKey := &models.APIKey{
			Name:   "Airley API Key",
			TeamID: team.ID,
		}
		if err := db.Create(apiKey).Error; err != nil {
			return log.Error("Failed to create api key", err)
		}
	}

	log.Info("Created api key %s", apiKey.ID)

	// Create mailing lists
	for _, list := range templates.MailingLists {
		// check if list already exists
		var existingList models.MailingList
		if err := db.Where("name = ? AND team_id = ?", list.Name, team.ID).First(&existingList).Error; err == nil {
			return log.Error("Mailing list already exists", err)
		} else if err != gorm.ErrRecordNotFound {
			return log.Error("Failed to check if mailing list exists", err)
		}

		list.TeamID = team.ID
		if err := db.Create(&list).Error; err != nil {
			return log.Error("Failed to create mailing list", err)
		}
	}

	// upload

	// Create templates
	for _, tmpl := range templates.Templates {
		tmpl.TeamID = team.ID
		category := models.EmailCategory{}
		if err := db.Where("name = ? AND team_id = ?", tmpl.CategoryName, team.ID).First(&category).Error; err != nil {
			return log.Error("Failed to find category", err)
		}

		file, err := CreateHTMLFile(db, team.ID, tmpl.HtmlFileID)
		if err != nil {
			return log.Error("Failed to create html file", err)
		}

		log.Info("Created html file %s", file.ID)

		template := models.Template{
			Name:       tmpl.Name,
			Subject:    tmpl.Subject,
			HtmlFileID: file.ID,
			DesignJSON: base64.EncodeToBase64(string(tmpl.DesignJSON)),
			TeamID:     team.ID,
			CategoryID: category.ID,
			Variables:  tmpl.Variables,
		}
		if err := db.Create(&template).Error; err != nil {
			return log.Error("Failed to create template", err)
		}
	}

	port, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		return log.Error("Failed to convert smtp port to int", err)
	}

	// create smtp config
	smtpConfig := models.SMTPConfig{
		TeamID:       team.ID,
		Host:         os.Getenv("SMTP_HOST"),
		Port:         port,
		Username:     os.Getenv("SMTP_USERNAME"),
		Password:     os.Getenv("SMTP_PASSWORD"),
		FromEmail:    os.Getenv("SMTP_FROM_EMAIL"),
		Provider:     os.Getenv("SMTP_PROVIDER"),
		IsDefault:    true,
		IsActive:     true,
		RequiresAuth: true,
		SupportsTLS:  true,
	}

	if err := db.Create(&smtpConfig).Error; err != nil {
		return log.Error("Failed to create smtp config", err)
	}

	log.Success("Successfully loaded all templates")

	return nil
}

// write a function to create a html file based on string first we need to saniztinze the string remove = and -
func sanitizeString(str string) string {
	re := regexp.MustCompile(`\\`)      // Match all backslashes
	return re.ReplaceAllString(str, "") // Remove them
}

// CreateHTMLFile creates a new HTML file from content string
func CreateHTMLFile(db *gorm.DB, teamId string, content string) (*models.File, error) {
	log.Info("Creating HTML file for team %s", teamId)
	// Sanitize content
	sanitizedContent := sanitizeString(content)

	// Create file in memory
	fileBytes := []byte(sanitizedContent)

	fileName := fmt.Sprintf("tmp/%s.html", uuid.New().String())

	// Get storage handler
	storage := handlers.GetStorageHandler()
	if storage == nil {
		return nil, log.Error("storage handler not configured", nil)
	}

	// Upload file
	url, err := storage.UploadFile(context.Background(), fileBytes, fileName, types.ObjectCannedACLAuthenticatedRead)
	if err != nil {
		return nil, log.Error("failed to upload file", err)
	}

	log.Info("Uploaded file to %s", url)

	// Create file record
	file := &models.File{
		TeamID: teamId,
		Path:   url[strings.LastIndex(url, "/")+1:],
		Name:   fileName,
		Size:   int64(len(fileBytes)),
		Type:   "text/html",
	}

	if err := db.Create(file).Error; err != nil {
		return nil, log.Error("failed to create file record", err)
	}

	log.Info("Created file record %s", file.ID)

	return file, nil
}
