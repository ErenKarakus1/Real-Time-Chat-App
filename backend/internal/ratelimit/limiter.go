package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	client *redis.Client
}

type Result struct {
	Allowed    bool
	Limit      int64
	Remaining  int64
	RetryAfter time.Duration
}

func NewLimiter(client *redis.Client) *Limiter {
	return &Limiter{client: client}
}

func (l *Limiter) Allow(ctx context.Context, name string, identifier string, limit int64, window time.Duration) (Result, error) {
	key := fmt.Sprintf("rate_limit:%s:%s", name, identifier)

	count, err := l.client.Incr(ctx, key).Result()
	if err != nil {
		return Result{}, err
	}

	if count == 1 {
		if err := l.client.Expire(ctx, key, window).Err(); err != nil {
			return Result{}, err
		}
	}

	ttl, err := l.client.TTL(ctx, key).Result()
	if err != nil {
		return Result{}, err
	}
	if ttl < 0 {
		ttl = window
	}

	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	return Result{
		Allowed:    count <= limit,
		Limit:      limit,
		Remaining:  remaining,
		RetryAfter: ttl,
	}, nil
}
