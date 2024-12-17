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
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	log.Info("Connecting to database...")

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
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
	return DB.AutoMigrate(
		&models.User{},
		&models.PasswordReset{},
		&models.Team{},
		&models.Contact{},
		&models.MailingList{},
		&models.SMTPConfig{},
		&models.Domain{},
		&models.Webhook{},
		&models.Delivery{},
		&models.Template{},
		&models.Email{},
		&models.EmailTracking{},
		&models.EmailReply{},
		&models.EmailBounce{},
		&models.EmailComplaint{},
		&models.APIKey{},
		&models.TeamInvite{},
		&models.TeamSettings{},
		&models.RateLimit{},
		&models.APIKeyUsage{},
		&models.ResourcePermission{},
		&models.APIKeyPermission{},
		&models.Resource{},
		&models.UserPermission{},
		&models.Automation{},
		&models.AutomationNode{},
		&models.AutomationNodeEdge{},
		&models.LLMEmailWriterJob{},
		&models.Model{},
		&models.Campaign{},
	)
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
