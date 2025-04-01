package tasks

import (
	"context"
	"encoding/json"
	"errors"
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

// EnqueueCampaignTask enqueues a campaign task with support for cron scheduling
func (c *TaskClient) EnqueueCampaignTask(ctx context.Context, task CampaignTask, processIn time.Duration) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal campaign task: %w", err)
	}

	// Base task with type and payload
	t := asynq.NewTask(TaskTypeCampaignProcess, payload)

	// Configure task options based on scheduling type
	var opts []asynq.Option
	opts = append(opts,
		asynq.Queue(QueueDefault),
		asynq.Timeout(TimeoutLong),
		asynq.MaxRetry(RetryMax),
	)

	switch {
	case task.CronExpression != "":
		// For recurring campaigns using cron
		opts = append(opts,
			CronSchedule(task.CronExpression),
			// Remove Unique from base options to avoid duplicate
			asynq.Unique(24*time.Hour), // Prevent duplicate schedules
		)

		// Schedule the next run after this one completes
		opts = append(opts, AfterFunc(func(ctx context.Context, t *asynq.Task) error {
			// Decode the task payload
			var taskData CampaignTask
			if err := json.Unmarshal(t.Payload(), &taskData); err != nil {
				return fmt.Errorf("failed to unmarshal task payload: %w", err)
			}

			// Schedule next run with reset batch processing
			return c.EnqueueCampaignTask(ctx, CampaignTask{
				CampaignID:     taskData.CampaignID,
				CronExpression: taskData.CronExpression,
				BatchSize:      taskData.BatchSize, // Preserve batch size
				Offset:         0,                  // Reset offset for new cron run
			}, time.Second) // 1 second delay
		}))

		c.logger.Info("📅 Scheduling recurring campaign [%s] with cron: %s",
			task.CampaignID, task.CronExpression)

	case !task.ScheduledAt.IsZero():
		// For one-time scheduled campaigns
		processAt := task.ScheduledAt
		if processAt.Before(time.Now()) {
			processAt = time.Now()
		}
		opts = append(opts, asynq.ProcessAt(processAt))
		c.logger.Info("⏰ Scheduling one-time campaign [%s] at: %s",
			task.CampaignID, processAt.Format(time.RFC3339))

	case processIn > 0:
		// For delayed processing
		opts = append(opts, asynq.ProcessIn(processIn))
		c.logger.Info("⌛ Delaying campaign [%s] by: %v",
			task.CampaignID, processIn)

	default:
		// For immediate processing
		c.logger.Info("🚀 Enqueueing campaign [%s] for immediate processing",
			task.CampaignID)
	}

	// Enqueue the task with configured options
	info, err := c.client.EnqueueContext(ctx, t, opts...)
	if err != nil {
		if errors.Is(err, asynq.ErrDuplicateTask) {
			return fmt.Errorf("campaign task already scheduled: %w", err)
		}
		return fmt.Errorf("failed to enqueue campaign task: %w", err)
	}

	// Log successful enqueue with details
	c.logger.Success("✅ Enqueued campaign task [ID: %s] [Queue: %s]", info.ID, info.Queue)
	if info.NextProcessAt.IsZero() {
		c.logger.Info("⚡ Task will process immediately")
	} else {
		c.logger.Info("🕒 Next processing scheduled for: %s",
			info.NextProcessAt.Format(time.RFC3339))
	}

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

	c.logger.Info("Enqueued contact import task [%s] in queue %s for import %s",
		info.ID, info.Queue, task.ImportID)
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
