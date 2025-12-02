package main

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
	"github.com/sangkips/campaign-dispatch-service/internal/config"
	"github.com/sangkips/campaign-dispatch-service/internal/db"
	"github.com/sangkips/campaign-dispatch-service/internal/domains/campaigns"
	"github.com/sangkips/campaign-dispatch-service/internal/domains/customers"
	"github.com/sangkips/campaign-dispatch-service/internal/domains/messages"
	"github.com/sangkips/campaign-dispatch-service/internal/health"
	"github.com/sangkips/campaign-dispatch-service/internal/queue"
	"github.com/sangkips/campaign-dispatch-service/internal/worker"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	db, err := db.ConnectAndMigrate(cfg.DBURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// Initialize RabbitMQ
	rabbitMQ, err := queue.NewRabbitMQ(cfg.RabbitMQURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to RabbitMQ")
	}
	defer rabbitMQ.Close()

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	customerHandler := customers.NewHandler(db)
	r.Route("/customers", func(r chi.Router) {
		customerHandler.RegisterCustomerRoutes(r)
	})

	campaignHandler := campaigns.NewHandler(db, rabbitMQ)
	r.Route("/campaigns", func(r chi.Router) {
		campaignHandler.RegisterCampaignRoutes(r)
	})

	healthHandler := health.NewHandler(db, rabbitMQ)
	r.Get("/health", healthHandler.Health)

	// Initialize repositories for scheduler
	campaignRepo := campaigns.NewRepository(db)
	messagesRepo := messages.NewRepository(db)

	// Start Scheduler
	scheduler := worker.NewScheduler(campaignRepo, messagesRepo, rabbitMQ, 10*time.Second)
	go scheduler.Start()
	defer scheduler.Stop()

	log.Info().Msg("server starting on :" + cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}

}
