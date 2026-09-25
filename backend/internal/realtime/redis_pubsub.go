package realtime

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisPubSub struct {
	client *redis.Client
}

type RedisSubscription struct {
	pubsub *redis.PubSub
	ch     chan string
}

func NewRedisPubSub(client *redis.Client) *RedisPubSub {
	return &RedisPubSub{
		client: client,
	}
}

func (p *RedisPubSub) Publish(ctx context.Context, conversationID uuid.UUID, event Event) error {
	payload, err := event.Marshal()
	if err != nil {
		return err
	}

	return p.client.Publish(ctx, conversationChannel(conversationID), payload).Err()
}

func (p *RedisPubSub) Subscribe(ctx context.Context, conversationID uuid.UUID) Subscription {
	pubsub := p.client.Subscribe(ctx, conversationChannel(conversationID))
	subscription := &RedisSubscription{
		pubsub: pubsub,
		ch:     make(chan string),
	}

	go subscription.forward()
	return subscription
}

func (s *RedisSubscription) Channel() <-chan string {
	return s.ch
}

func (s *RedisSubscription) Close() error {
	return s.pubsub.Close()
}

func (s *RedisSubscription) forward() {
	defer close(s.ch)

	for message := range s.pubsub.Channel() {
		s.ch <- message.Payload
	}
}

func conversationChannel(conversationID uuid.UUID) string {
	return fmt.Sprintf("conversation:%s:events", conversationID)
}
