package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kori/internal/api"
	"kori/internal/config"
	"kori/internal/db"
	"kori/internal/workers"

	"github.com/joho/godotenv"
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
	defer db.Close()

	// Initialize worker server
	workerServer := workers.NewWorkerServer(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.Username, cfg.Redis.DB, cfg.Worker.Concurrency)
	go func() {
		if err := workerServer.Start(); err != nil {
			log.Printf("Worker server error: %v", err)
		}
	}()

	// Initialize API server
	apiServer := api.NewServer(cfg, db.GetDB())
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the servers
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown worker server
	if err := workerServer.Shutdown(ctx); err != nil {
		log.Printf("Failed to shutdown worker server: %v", err)
	}

	// Shutdown API server
	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("Failed to shutdown API server: %v", err)
	}

	log.Println("Servers shutdown gracefully")
}
