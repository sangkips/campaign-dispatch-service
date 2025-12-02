package config

import (
	"errors"
	"os"

	"github.com/rs/zerolog/log"
)

type Config struct {
	DBURL string
	Port  string
	RabbitMQURL string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		DBURL: os.Getenv("DB_URL"),
		Port:  os.Getenv("PORT"),
		RabbitMQURL: os.Getenv("RABBITMQ_URL"),
	}

	if cfg.DBURL == "" {
		log.Error().Msg("DB_URL environment variable is not set")
		return nil, errors.New("DB_URL is required")
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	if cfg.RabbitMQURL == "" {
		cfg.RabbitMQURL = "amqp://guest:guest@localhost:5672/"
		log.Info().Msg("RABBITMQ_URL not set, using default: amqp://guest:guest@localhost:5672/")
	}

	return cfg, nil
}
