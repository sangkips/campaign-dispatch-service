package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/sangkips/campaign-dispatch-service/internal/config"
	"github.com/sangkips/campaign-dispatch-service/internal/db"
	"github.com/sangkips/campaign-dispatch-service/internal/queue"
	"github.com/sangkips/campaign-dispatch-service/internal/worker"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// Connect to database
	dbConn, err := db.ConnectAndMigrate(cfg.DBURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer dbConn.Close()

	// Connect to RabbitMQ
	rabbitMQ, err := queue.NewRabbitMQ(cfg.RabbitMQURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to RabbitMQ")
	}
	defer rabbitMQ.Close()

	// Initialize dependencies
	sender := worker.NewMockSender(0.95) // 95% success rate
	w := worker.NewWorker(rabbitMQ, dbConn, sender)

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Info().Str("signal", sig.String()).Msg("received signal, shutting down")
		cancel()
	}()

	// Start worker
	if err := w.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("worker failed")
	}

	log.Info().Msg("worker stopped")
}
