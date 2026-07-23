package connector

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/meindokuse/task-service/config"
)

// NewRedis открывает клиент Redis и проверяет соединение через PING.
func NewRedis(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	return client, nil
}
