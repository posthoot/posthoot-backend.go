package tasks

import (
	"context"
	"fmt"
	"kori/internal/utils/logger"

	"github.com/hibiken/asynq"
)

// Server handles task processing
type Server struct {
	server  *asynq.Server
	handler *TaskHandler
	logger  *logger.Logger
}

// NewServer creates a new task processing server
func NewServer(redisAddr, username, password string, db int, handler *TaskHandler, logger *logger.Logger) *Server {
	server := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     redisAddr,
			Username: username,
			Password: password,
			DB:       db,
		},
		asynq.Config{
			// Specify how many concurrent workers to use
			Concurrency: 10,
			// Optionally specify multiple queues with different priorities
			Queues: map[string]int{
				QueueCritical: 6, // High priority
				QueueDefault:  3, // Medium priority
				QueueLow:      1, // Low priority
			},
			// Enable strict priority, meaning higher priority queues are processed first
			StrictPriority: true,
		},
	)

	return &Server{
		server:  server,
		handler: handler,
		logger:  logger,
	}
}

// Start starts the task processing server
func (s *Server) Start(ctx context.Context) error {
	mux := asynq.NewServeMux()

	// Register task handlers
	mux.HandleFunc(TaskTypeEmailSend, s.handler.HandleEmailSend)
	mux.HandleFunc(TaskTypeEmailRetry, s.handler.HandleEmailSend)
	mux.HandleFunc(TaskTypeCampaignProcess, s.handler.HandleCampaignProcess)
	mux.HandleFunc(TaskTypeCampaignSchedule, s.handler.HandleCampaignProcess)
	mux.HandleFunc(TaskTypeWebhookDelivery, s.handler.HandleWebhookDelivery)
	mux.HandleFunc(TaskTypeWebhookRetry, s.handler.HandleWebhookDelivery)
	mux.HandleFunc(TaskTypeDomainVerification, s.handler.HandleDomainVerification)
	mux.HandleFunc(TaskTypeDomainCheck, s.handler.HandleDomainVerification)
	mux.HandleFunc(TaskTypeContactImport, s.handler.HandleContactImport)
	mux.HandleFunc(TaskTypeContactSync, s.handler.HandleContactImport)
	mux.HandleFunc(TaskTypeLLMEmailWriter, s.handler.HandleLLMEmailWriter)

	s.logger.Info("starting task processing server concurrency %d queues %v", 10, map[string]int{
		QueueCritical: 6,
		QueueDefault:  3,
		QueueLow:      1,
	})

	if err := s.server.Start(mux); err != nil {
		return fmt.Errorf("failed to start task server: %w", err)
	}

	return nil
}

// Stop stops the task processing server
func (s *Server) Stop() {
	s.server.Stop()
	s.logger.Info("task processing server stopped")
}

// Shutdown gracefully shuts down the task processing server
func (s *Server) Shutdown() {
	s.logger.Info("shutting down task processing server")
	s.server.Shutdown()
}
