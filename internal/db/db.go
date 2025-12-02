package db

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog/log"
)

func ConnectAndMigrate(dbURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to connect to database")
		return nil, err
	}

	if err := db.Ping(); err != nil {
		log.Error().Err(err).Msg("failed to ping database")
		return nil, err
	}

	return db, nil
}
