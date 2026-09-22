package database

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

// DB0 is singleton Redis client for URL storage
var DB0 *redis.Client

// DB1 is the singleton Redis client for rate limiting
var DB1 *redis.Client

func Init() error {
	addr := os.Getenv("DB_ADDR")
	pass := os.Getenv("DB_PASS")

	DB0 = redis.NewClient(&redis.Options{
		Addr: 		addr,
		Password: pass,
		DB: 			0,
	})
	DB1 = redis.NewClient(&redis.Options{
		Addr: 		addr,
		Password: pass,
		DB:				1,
	})

	ctx := context.Background()
	if err := DB0.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis DB0 ping failed: %w", err)
	}
	if err := DB1.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis DB1 ping failed: %w", err)
	}
	return nil
}