package workers

import (
	"context"
	"log"
	"time"

	"kori/internal/config"
	"kori/internal/db"
	"kori/internal/models"
)

type Scheduler struct {
	config *config.Config
	server *WorkerServer
	client *TaskClient
	ctx    context.Context
	cancel context.CancelFunc
}

func NewScheduler(cfg *config.Config, server *WorkerServer, redisAddr string) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		config: cfg,
		server: server,
		client: NewTaskClient(redisAddr),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Scheduler) Start() {
	log.Println("Starting scheduler...")

	// Start different scheduling routines
	go s.scheduleCampaigns()
	go s.scheduleEmailRetries()
	go s.scheduleWebhookRetries()
	go s.scheduleDomainVerifications()

	// Wait for context cancellation
	<-s.ctx.Done()
	log.Println("Scheduler stopped")
}

func (s *Scheduler) Stop() {
	s.cancel()
}

func (s *Scheduler) scheduleCampaigns() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			var campaigns []models.Campaign
			err := db.DB.Where("status = ? AND scheduled_for <= ?",
				models.CampaignStatusScheduled, time.Now()).Find(&campaigns).Error
			if err != nil {
				log.Printf("Error fetching scheduled campaigns: %v", err)
				continue
			}

			for _, campaign := range campaigns {
				err := s.client.EnqueueCampaignTask(CampaignTask{
					CampaignID: campaign.ID,
				}, 0)
				if err != nil {
					log.Printf("Failed to enqueue campaign task: %v", err)
				}
			}
		}
	}
}

func (s *Scheduler) scheduleEmailRetries() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			var failedEmails []models.Email
			err := db.DB.Where("status = ? AND updated_at <= ?",
				"FAILED", time.Now().Add(-15*time.Minute)).Find(&failedEmails).Error
			if err != nil {
				log.Printf("Error fetching failed emails: %v", err)
				continue
			}

			for _, email := range failedEmails {
				err := s.client.EnqueueEmailTask(EmailTask{
					UserID: email.ID,
				})
				if err != nil {
					log.Printf("Failed to enqueue email retry task: %v", err)
				}
			}
		}
	}
}

func (s *Scheduler) scheduleWebhookRetries() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			var failedDeliveries []models.Delivery
			err := db.DB.Where("status = ? AND updated_at <= ?",
				"FAILED", time.Now().Add(-15*time.Minute)).Find(&failedDeliveries).Error
			if err != nil {
				log.Printf("Error fetching failed webhook deliveries: %v", err)
				continue
			}

			for _, delivery := range failedDeliveries {
				err := s.client.EnqueueWebhookDeliveryTask(WebhookDeliveryTask{
					WebhookID: delivery.WebhookID,
					Event:     delivery.Event,
					Payload:   string(delivery.Payload),
				})
				if err != nil {
					log.Printf("Failed to enqueue webhook retry task: %v", err)
				}
			}
		}
	}
}

func (s *Scheduler) scheduleDomainVerifications() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			var domains []models.Domain
			err := db.DB.Where("is_verified = ?", false).Find(&domains).Error
			if err != nil {
				log.Printf("Error fetching unverified domains: %v", err)
				continue
			}

			for _, domain := range domains {
				err := s.client.EnqueueDomainVerificationTask(DomainVerificationTask{
					DomainID: domain.ID,
				})
				if err != nil {
					log.Printf("Failed to enqueue domain verification task: %v", err)
				}
			}
		}
	}
}

func (s *Scheduler) scheduleContactImports() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			var pendingImports []models.ContactImport
			err := db.DB.Where("status = ?", "PENDING").Find(&pendingImports).Error
			if err != nil {
				log.Printf("Error fetching pending contact imports: %v", err)
				continue
			}

			for _, importJob := range pendingImports {
				err := s.client.EnqueueContactImportTask(ContactImportTask{
					ImportID: importJob.ID,
				})
				if err != nil {
					log.Printf("Failed to enqueue contact import task: %v", err)
				}
			}
		}
	}
}
