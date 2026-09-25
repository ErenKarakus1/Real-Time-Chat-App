package db

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis(ctx context.Context, redisURL string) (*redis.Client, error) {
	if redisURL == "" {
		return nil, errors.New("REDIS_URL is required")
	}

	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(options)
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}
