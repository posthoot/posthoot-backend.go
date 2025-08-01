package main

import (
	"context"
	"kori/docs/swagger"
	"kori/internal/handlers"
	"kori/internal/models/seeder/airley"
	"kori/internal/utils/crypto"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kori/internal/api"
	"kori/internal/config"
	"kori/internal/db"
	"kori/internal/models"
	"kori/internal/services"
	"kori/internal/tasks"
	"kori/internal/utils/logger"

	"github.com/joho/godotenv"
)

// 🚀 Main function
// @Summary Main function
// @Description Main function
// @title Posthoot API
// @version 1.0
// @description API documentation for Posthoot application
// @host backyard.posthoot.com
// @BasePath /
// @schemes https

// @securityDefinitions.basic BasicAuth
// @in header
// @name Authorization

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-KEY

func main() {

	logger := logger.New("kori")

	// check if .env file exists
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		logger.Info("No .env file found, skipping environment variable loading")
	} else {
		logger.Info("Loading environment variables from .env file")
		if err := godotenv.Load(); err != nil {
			log.Fatalf("Failed to load environment variables: %v", err)
		}
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize keys
	if err := crypto.InitializeKeys(
		cfg.Crypto.PrivateKey); err != nil {
		log.Fatalf("Failed to initialize keys: %v", err)
	}

	// Connect to database
	if err := db.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		err := db.Close()
		if err != nil {
			log.Fatalf("Failed to close database connection: %v", err)
		}
	}()

	// Set up Discord webhook for monitoring
	if cfg.Monitor.DiscordWebhookURL != "" {
		db.SetDiscordWebhook(cfg.Monitor.DiscordWebhookURL)
		logger.Info("Discord webhook monitoring enabled")
	}

	// Start monitoring database connection pool
	db.MonitorConnectionPool(10 * time.Hour)

	db_instance := db.GetDB()

	// Initialize task handlers
	taskHandler := tasks.NewTaskHandler(db_instance)

	// Initialize task server
	taskServer := tasks.NewServer(
		cfg.Redis.Addr,
		cfg.Redis.Password,
		cfg.Redis.Username,
		cfg.Redis.DB,
		taskHandler,
		logger,
	)

	// Create a context for task server
	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	// Start task server
	go func() {
		if err := taskServer.Start(serverCtx); err != nil {
			logger.Error("Task server error", err)
		}
	}()

	// Initialize task scheduler
	taskScheduler := tasks.NewScheduler(
		cfg.Redis.Addr,
		cfg.Redis.Password,
		cfg.Redis.Username,
		cfg.Redis.DB,
		logger,
	)

	// Start task scheduler
	go func() {
		if err := taskScheduler.Start(); err != nil {
			logger.Error("Task scheduler error", err)
		}
	}()

	// Initialize API server
	apiServer := api.NewServer(cfg, db_instance)
	go func() {

		// Initialize S3 service
		s3Service, err := services.NewS3Service(
			cfg.Storage.S3.BucketName,
			cfg.Storage.S3.Endpoint,
			cfg.Storage.S3.Region,
			cfg.Storage.S3.AccessKey,
			cfg.Storage.S3.SecretKey,
		)

		if err != nil {
			log.Fatalf("Failed to initialize S3 service: %v", err)
		}

		// Register the URL generator
		models.RegisterFileURLGenerator(s3Service)
		handlers.RegisterStorageHandler(s3Service)

		if cfg.Airley.Enabled {
			// Seed Airley templates
			if err := airley.LoadAirleyTemplates(db_instance); err != nil {
				logger.Error("Warning: Failed to seed Airley templates: %v", err)
			} else {
				logger.Success("Successfully seeded Airley templates")
			}
		}
		logger.Success("API server started")

		// Swagger documentation
		swagger.SwaggerInfo.Title = "Posthoot API Documentation"
		swagger.SwaggerInfo.Description = "API documentation for Posthoot application"
		swagger.SwaggerInfo.Version = "1.0"
		swagger.SwaggerInfo.Host = "backyard.posthoot.com"
		swagger.SwaggerInfo.Schemes = []string{"https"}

		if err := apiServer.Start(); err != nil {
			logger.Error("API server error", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the servers
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop task scheduler
	taskScheduler.Stop()

	// Stop task server
	serverCancel()

	// Shutdown API server
	if err := apiServer.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown API server", err)
	}

	logger.Info("Servers shutdown gracefully")
}
