package workers

import (
	"context"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

type WorkerServer struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

func NewWorkerServer(redisAddr string, redisPassword string, redisUsername string, redisDB int, concurrency int) *WorkerServer {
	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword, Username: redisUsername, DB: redisDB},
		asynq.Config{
			Concurrency: concurrency,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			StrictPriority:  true,
			ShutdownTimeout: time.Second * 30,
		},
	)

	return &WorkerServer{
		server: server,
		mux:    asynq.NewServeMux(),
	}
}

func (w *WorkerServer) Start() error {
	w.registerHandlers()
	log.Printf("Starting worker server with registered handlers...")
	return w.server.Run(w.mux)
}

func (w *WorkerServer) Shutdown(ctx context.Context) error {
	log.Printf("Shutting down worker server...")
	w.server.Shutdown()
	return nil
}

func (w *WorkerServer) registerHandlers() {
	w.mux.HandleFunc(JobTypeEmailSend, HandleEmailSend)
	w.mux.HandleFunc(JobTypeCampaignProcess, HandleCampaignProcess)
	w.mux.HandleFunc(JobTypeContactImport, HandleContactImport)
	w.mux.HandleFunc(JobTypeWebhookDelivery, HandleWebhookDelivery)
	w.mux.HandleFunc(JobTypeLLMEmailWriter, HandleLLMEmailWriter)
	w.mux.HandleFunc(JobTypeDomainVerification, HandleDomainVerification)
}
