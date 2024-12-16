package workers

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

type TaskClient struct {
	client *asynq.Client
}

func NewTaskClient(redisAddr string) *TaskClient {
	return &TaskClient{
		client: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr}),
	}
}

func (c *TaskClient) Close() error {
	return c.client.Close()
}

func (c *TaskClient) EnqueueEmailTask(task EmailTask) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}

	_, err = c.client.Enqueue(
		asynq.NewTask(JobTypeEmailSend, payload),
		asynq.Queue("critical"),
		asynq.Timeout(5*time.Minute),
		asynq.MaxRetry(3),
	)
	return err
}

func (c *TaskClient) EnqueueCampaignTask(task CampaignTask, processIn time.Duration) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}

	_, err = c.client.Enqueue(
		asynq.NewTask(JobTypeCampaignProcess, payload),
		asynq.ProcessIn(processIn),
		asynq.Queue("default"),
		asynq.Timeout(30*time.Minute),
		asynq.MaxRetry(5),
	)
	return err
}

func (c *TaskClient) EnqueueWebhookDeliveryTask(task WebhookDeliveryTask) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}
	_, err = c.client.Enqueue(
		asynq.NewTask(JobTypeWebhookDelivery, payload),
		asynq.Queue("default"),
		asynq.MaxRetry(3),
	)
	return err
}

func (c *TaskClient) EnqueueDomainVerificationTask(task DomainVerificationTask) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}
	_, err = c.client.Enqueue(
		asynq.NewTask(JobTypeDomainVerification, payload),
		asynq.Queue("low"),
		asynq.MaxRetry(2),
	)
	return err
}

func (c *TaskClient) EnqueueContactImportTask(task ContactImportTask) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}

	_, err = c.client.Enqueue(
		asynq.NewTask(JobTypeContactImport, payload),
		asynq.Queue("default"),
		asynq.MaxRetry(3),
	)
	return err
}

func (c *TaskClient) EnqueueLLMEmailWriterTask(task LLMEmailWriterTask) error {
	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}
	_, err = c.client.Enqueue(
		asynq.NewTask(JobTypeLLMEmailWriter, payload),
		asynq.Queue("critical"),
		asynq.Timeout(2*time.Minute),
		asynq.MaxRetry(2),
	)
	return err
}
