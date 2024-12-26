package main

import (
	"context"
	"kori/internal/handlers"
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

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Failed to load environment variables: %v", err)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
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

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			log.Fatalf("Failed to flush logger", zap.Error(err))
		}
	}(logger)

	// Initialize task handlers
	taskHandler := tasks.NewTaskHandler(db.GetDB(), logger)

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
			logger.Error("Task server error", zap.Error(err))
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
			logger.Error("Task scheduler error", zap.Error(err))
		}
	}()

	// Initialize API server
	apiServer := api.NewServer(cfg, db.GetDB())
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

		if err := apiServer.Start(); err != nil {
			logger.Error("API server error", zap.Error(err))
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
		logger.Error("Failed to shutdown API server", zap.Error(err))
	}

	logger.Info("Servers shutdown gracefully")
}
