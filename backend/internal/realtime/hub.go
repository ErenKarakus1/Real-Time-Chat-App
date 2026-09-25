package realtime

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type PubSub interface {
	Publish(ctx context.Context, conversationID uuid.UUID, event Event) error
	Subscribe(ctx context.Context, conversationID uuid.UUID) Subscription
}

type Subscription interface {
	Channel() <-chan string
	Close() error
}

type Hub struct {
	mu            sync.RWMutex
	conversations map[uuid.UUID]map[*Client]struct{}
	subscriptions map[uuid.UUID]Subscription
	pubsub        PubSub
}

func NewHub(pubsub PubSub) *Hub {
	return &Hub{
		conversations: make(map[uuid.UUID]map[*Client]struct{}),
		subscriptions: make(map[uuid.UUID]Subscription),
		pubsub:        pubsub,
	}
}

func (h *Hub) Subscribe(ctx context.Context, conversationID uuid.UUID, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.conversations[conversationID] == nil {
		h.conversations[conversationID] = make(map[*Client]struct{})
	}

	h.conversations[conversationID][client] = struct{}{}
	if h.subscriptions[conversationID] == nil {
		subscription := h.pubsub.Subscribe(ctx, conversationID)
		h.subscriptions[conversationID] = subscription
		go h.forwardSubscription(conversationID, subscription)
	}
}

func (h *Hub) Unsubscribe(conversationID uuid.UUID, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients := h.conversations[conversationID]
	if clients == nil {
		return
	}

	if _, exists := clients[client]; exists {
		delete(clients, client)
		client.Close()
	}

	if len(clients) == 0 {
		delete(h.conversations, conversationID)
		if subscription := h.subscriptions[conversationID]; subscription != nil {
			subscription.Close()
			delete(h.subscriptions, conversationID)
		}
	}
}

func (h *Hub) Publish(ctx context.Context, conversationID uuid.UUID, event Event) error {
	return h.pubsub.Publish(ctx, conversationID, event)
}

func (h *Hub) broadcastLocal(conversationID uuid.UUID, event Event) {
	h.mu.RLock()
	clients := h.conversations[conversationID]
	targets := make([]*Client, 0, len(clients))
	for client := range clients {
		targets = append(targets, client)
	}
	h.mu.RUnlock()

	for _, client := range targets {
		select {
		case client.send <- event:
		default:
			h.Unsubscribe(conversationID, client)
		}
	}
}

func (h *Hub) forwardSubscription(conversationID uuid.UUID, subscription Subscription) {
	for payload := range subscription.Channel() {
		event, err := UnmarshalEvent(payload)
		if err != nil {
			continue
		}

		h.broadcastLocal(conversationID, event)
	}
}
