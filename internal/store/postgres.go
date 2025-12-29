package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool creates a connection pool and pings the database to verify connectivity.
func NewPostgresPool(dsn string, maxConns int) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse config to allow for more granular control later (max connections, etc.)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// Example: setting max connections
	// config.MaxConns = int32(maxConns)

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return pool, nil
}
