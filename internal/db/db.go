package db

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kori/internal/config"
	"kori/internal/models"
	console "kori/internal/utils/logger"
)

var DB *gorm.DB

var log = console.New("DB")

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

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Info),
		DisableForeignKeyConstraintWhenMigrating: true,
		PrepareStmt:                              true,
		AllowGlobalUpdate:                        false,
	})
	if err != nil {
		return log.Error("Failed to connect to database", err)
	}

	log.Info(fmt.Sprintf("DSN: %s", dsn))
	log.Success("Connected to database")

	// Run migrations
	if err := runMigrations(); err != nil {
		return log.Error("Failed to run migrations", err)
	}

	log.Success("Migrations completed")

	return nil
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

		// Email-related models
		&models.Email{},
		&models.EmailTracking{},
		&models.EmailReply{},
		&models.EmailBounce{},
		&models.EmailComplaint{},
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
