package postgres

import (
	"context"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
)

// Config contains PostgreSQL connection settings.
type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Open creates and validates a PostgreSQL connection pool and returns the raw sqlx handle.
func Open(ctx context.Context, config Config) (*sqlx.DB, error) {
	db, err := sqlx.ConnectContext(ctx, "postgres", config.DSN)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	}

	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}

	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	}

	return db, nil
}
