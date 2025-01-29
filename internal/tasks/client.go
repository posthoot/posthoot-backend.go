package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kori/internal/utils/logger"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	limiter "kori/internal/tasks/rate"
)

// TaskClient handles task enqueuing with improved error handling and context support
type TaskClient struct {
	client       *asynq.Client
	logger       *logger.Logger
	redisOptions *redis.Options
	redisClient  *redis.Client
}

type RateLimiter struct {
	Rate   int
	Burst  int
	Period time.Duration
}

func (c *TaskClient) GetClient() *asynq.Client {
	return c.client
}

// NewTaskClient creates a new TaskClient with the given Redis configuration
func NewTaskClient(redisAddr, username, password string, db int) *TaskClient {
	redisOpt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Username: username,
		Password: password,
		DB:       db,
	}

	redisClient := redis.NewClient(
		&redis.Options{
			Addr:     redisAddr,
			Username: username,
			Password: password,
			DB:       db,
		},
	)

	return &TaskClient{
		client: asynq.NewClient(redisOpt),
		redisOptions: &redis.Options{
			Addr:     redisAddr,
			Username: username,
			Password: password,
			DB:       db,
		},
		redisClient: redisClient,
		logger:      logger.New("TASKS"),
	}
}

// Close closes the underlying asynq client
func (c *TaskClient) Close() error {
	return c.client.Close()
}

func GetEmailQueueName(smtpSettingsID string) string {
	return fmt.Sprintf("email:smtp:%s", smtpSettingsID)
}

// EnqueueEmailTask enqueues an email sending task
func (c *TaskClient) EnqueueEmailTask(ctx context.Context, task EmailTask) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal email task: %w", err)
	}

	redisClient := c.redisClient

	defer redisClient.Close()

	limiterKey := GetEmailQueueName(task.SMTPConfigID)

	// Use sliding window rate limiter with Redis
	rateLimiter := limiter.NewQueueRateLimiter(redisClient, limiter.QueueConfig{
		Name: limiterKey,
		RateLimit: limiter.RateLimit{
			Window:  time.Second,
			MaxJobs: task.MaxSendRate,
		},
	})

	// Use provider as identifier for rate limiting
	allowed, err := rateLimiter.Allow(ctx, limiterKey)
	if err != nil {
		return fmt.Errorf("rate limiter error: %w", err)
	}

	if !allowed {
		err := fmt.Errorf("rate limit exceeded for provider %s", limiterKey)
		// Return error to trigger asynq retry with configured backoff
		return c.logger.Error("❌ Rate limit exceeded %s", err)
	}

	info, err := c.client.EnqueueContext(ctx,
		asynq.NewTask(TaskTypeEmailSend, payload),
		asynq.Queue(QueueCritical),
		asynq.Timeout(TimeoutMedium),
		asynq.MaxRetry(RetryDefault),
	)
	if err != nil {
		return fmt.Errorf("failed to enqueue email task: %w", err)
	}

	c.logger.Info("Enqueued email task [%s] in queue %s for email %s",
		info.ID, info.Queue, task.EmailID)
	return nil
}

// EnqueueCampaignTask enqueues a campaign processing task
func (c *TaskClient) EnqueueCampaignTask(ctx context.Context, task CampaignTask, processIn time.Duration) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal campaign task: %w", err)
	}

	info, err := c.client.EnqueueContext(ctx,
		asynq.NewTask(TaskTypeCampaignProcess, payload),
		asynq.ProcessIn(processIn),
		asynq.Queue(QueueDefault),
		asynq.Timeout(TimeoutLong),
		asynq.MaxRetry(RetryMax),
	)
	if err != nil {
		return fmt.Errorf("failed to enqueue campaign task: %w", err)
	}

	c.logger.Info("Enqueued campaign task [%s] in queue %s for campaign %s scheduled at %s",
		info.ID, info.Queue, task.CampaignID, task.ScheduledAt)
	return nil
}

// EnqueueWebhookDeliveryTask enqueues a webhook delivery task
func (c *TaskClient) EnqueueWebhookDeliveryTask(ctx context.Context, task WebhookDeliveryTask) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook task: %w", err)
	}

	info, err := c.client.EnqueueContext(ctx,
		asynq.NewTask(TaskTypeWebhookDelivery, payload),
		asynq.Queue(QueueDefault),
		asynq.MaxRetry(RetryDefault),
	)
	if err != nil {
		return fmt.Errorf("failed to enqueue webhook task: %w", err)
	}

	c.logger.Info("Enqueued webhook task [%s] in queue %s for webhook %s and event %s",
		info.ID, info.Queue, task.WebhookID, task.Event)
	return nil
}

// EnqueueDomainVerificationTask enqueues a domain verification task
func (c *TaskClient) EnqueueDomainVerificationTask(ctx context.Context, task DomainVerificationTask) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal domain verification task: %w", err)
	}

	info, err := c.client.EnqueueContext(ctx,
		asynq.NewTask(TaskTypeDomainVerification, payload),
		asynq.Queue(QueueLow),
		asynq.MaxRetry(RetryMin),
	)
	if err != nil {
		return fmt.Errorf("failed to enqueue domain verification task: %w", err)
	}

	c.logger.Info("Enqueued domain verification task [%s] in queue %s for domain %s",
		info.ID, info.Queue, task.DomainID)
	return nil
}

// EnqueueContactImportTask enqueues a contact import task
func (c *TaskClient) EnqueueContactImportTask(ctx context.Context, task ContactImportTask) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal contact import task: %w", err)
	}

	info, err := c.client.EnqueueContext(ctx,
		asynq.NewTask(TaskTypeContactImport, payload),
		asynq.Queue(QueueDefault),
		asynq.MaxRetry(RetryDefault),
	)
	if err != nil {
		return fmt.Errorf("failed to enqueue contact import task: %w", err)
	}

	c.logger.Info("Enqueued contact import task [%s] in queue %s for import %s and team %s",
		info.ID, info.Queue, task.ImportID, task.TeamID)
	return nil
}

// EnqueueLLMEmailWriterTask enqueues an LLM email writer task
func (c *TaskClient) EnqueueLLMEmailWriterTask(ctx context.Context, task LLMEmailWriterTask) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal LLM email writer task: %w", err)
	}

	info, err := c.client.EnqueueContext(ctx,
		asynq.NewTask(TaskTypeLLMEmailWriter, payload),
		asynq.Queue(QueueCritical),
		asynq.Timeout(TimeoutMedium),
		asynq.MaxRetry(RetryMin),
	)
	if err != nil {
		return fmt.Errorf("failed to enqueue LLM email writer task: %w", err)
	}

	c.logger.Info("Enqueued LLM email writer task [%s] in queue %s for email %s, template %s and model %s",
		info.ID, info.Queue, task.EmailID, task.TemplateID, task.ModelID)
	return nil
}
