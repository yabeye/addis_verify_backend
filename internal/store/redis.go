package store

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient initializes a new Redis client and checks connectivity.
func NewRedisClient(addr string, password string, db int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // leave blank if no password set
		DB:       db,       // use 0 by default
	})

	// Test connection with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return rdb, nil
}
