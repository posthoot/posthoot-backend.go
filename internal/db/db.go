package db

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kori/internal/config"
	"kori/internal/models"
	console "kori/internal/utils/logger"
)

var DB *gorm.DB
var log = console.New("DB")
var discordWebhookURL string

// SetDiscordWebhook sets the Discord webhook URL for monitoring
func SetDiscordWebhook(webhookURL string) {
	discordWebhookURL = webhookURL
}

// sendToDiscord sends connection pool stats to Discord
func sendToDiscord(stats sql.DBStats) error {
	if discordWebhookURL == "" {
		return nil
	}

	message := fmt.Sprintf("```\n🗄️ DB Connection Pool Stats:\n"+
		"🔝 Max Open Connections: %d\n"+
		"📊 Open Connections: %d\n"+
		"🔄 In Use Connections: %d\n"+
		"💤 Idle Connections: %d\n"+
		"⏳ Waited for Connection: %d\n"+
		"⌛ Wait Duration: %v\n"+
		"🚫 Max Idle Closed: %d\n"+
		"⏰ Max Lifetime Closed: %d\n```",
		stats.MaxOpenConnections,
		stats.OpenConnections,
		stats.InUse,
		stats.Idle,
		stats.WaitCount,
		stats.WaitDuration,
		stats.MaxIdleClosed,
		stats.MaxLifetimeClosed)

	payload := map[string]interface{}{
		"content": message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord payload: %w", err)
	}

	resp, err := http.Post(discordWebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send to Discord: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Discord API returned status code: %d for message: %s", resp.StatusCode)
	}

	return nil
}

func Connect(cfg *config.Config) error {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	log.Info("Connecting to database...")
	maxRetries := 5
	var err error
	for i := 0; i < maxRetries; i++ {
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger:                                   logger.Default.LogMode(logger.Info),
			DisableForeignKeyConstraintWhenMigrating: true,
			PrepareStmt:                              true,
			AllowGlobalUpdate:                        false,
		})
		if err == nil {
			log.Info("DSN: %s", dsn)
			log.Success("Connected to database")

			// Configure connection pool
			sqlDB, err := DB.DB()
			if err != nil {
				return log.Error("Failed to get underlying *sql.DB instance", err)
			}

			// Set connection pool settings
			sqlDB.SetMaxOpenConns(100)                 // Maximum number of open connections to the database
			sqlDB.SetMaxIdleConns(10)                  // Maximum number of idle connections in the pool
			sqlDB.SetConnMaxLifetime(time.Hour)        // Maximum amount of time a connection may be reused
			sqlDB.SetConnMaxIdleTime(time.Minute * 30) // Maximum amount of time a connection may be idle

			// Run migrations
			if err := runMigrations(); err != nil {
				return log.Error("Failed to run migrations", err)
			}

			log.Success("Migrations completed")

			return nil
		}
		log.Warn("Failed to connect to database (attempt %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(time.Second * 5)
	}
	return log.Error("failed to connect to database after %d attempts", fmt.Errorf("failed to connect to database after %d attempts", maxRetries))
}

func runMigrations() error {
	log.Info("Running migrations...")
	// Begin transaction for migrations
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Defer rollback in case of error
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.AutoMigrate(
		// Base models without foreign keys
		&models.User{},
		&models.Team{},
		&models.BrandingSettings{},
		&models.Resource{},
		&models.Model{},

		// Models with single foreign key dependencies
		&models.PasswordReset{},
		&models.TeamSettings{},
		&models.Contact{},
		&models.MailingList{},
		&models.SMTPConfig{},
		&models.Domain{},
		&models.Webhook{},
		&models.Template{},
		&models.APIKey{},
		&models.TeamInvite{},
		&models.RateLimit{},
		&models.AuthTransaction{},
		&models.Campaign{},

		// Subscriber models
		&models.ContactImport{},

		// Email-related models
		&models.Email{},
		&models.EmailTracking{},
		&models.Delivery{},

		// Permission models
		&models.UserPermission{},
		&models.ResourcePermission{},
		&models.APIKeyPermission{},

		// Usage and monitoring
		&models.APIKeyUsage{},

		// Automation models
		&models.Automation{},
		&models.AutomationNode{},
		&models.AutomationNodeEdge{},
		&models.LLMEmailWriterJob{},

		// Subscription models
		&models.Subscription{},
		&models.Product{},
		&models.ProductFeatureConfig{},
	); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func GetDB() *gorm.DB {
	return DB
}

// MonitorConnectionPool logs database connection pool statistics periodically
func MonitorConnectionPool(interval time.Duration) {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Error("Failed to get underlying *sql.DB instance", err)
		return
	}

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			stats := sqlDB.Stats()

			// Log to console
			log.Info("DB Connection Pool Stats:")
			log.Info("  Max Open Connections: %d", stats.MaxOpenConnections)
			log.Info("  Open Connections: %d", stats.OpenConnections)
			log.Info("  In Use Connections: %d", stats.InUse)
			log.Info("  Idle Connections: %d", stats.Idle)
			log.Info("  Waited for Connection: %d", stats.WaitCount)
			log.Info("  Wait Duration: %v", stats.WaitDuration)
			log.Info("  Max Idle Closed: %d", stats.MaxIdleClosed)
			log.Info("  Max Lifetime Closed: %d", stats.MaxLifetimeClosed)

			// Send to Discord if configured
			if err := sendToDiscord(stats); err != nil {
				log.Error("Failed to send stats to Discord", err)
			}
		}
	}()
}

// GetConnectionStats returns the current database connection pool statistics
func GetConnectionStats() (*sql.DBStats, error) {
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying *sql.DB instance: %w", err)
	}
	stats := sqlDB.Stats()
	return &stats, nil
}
