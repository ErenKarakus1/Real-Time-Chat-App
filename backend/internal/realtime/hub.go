package realtime

import (
	"sync"

	"github.com/google/uuid"
)

type Hub struct {
	mu            sync.RWMutex
	conversations map[uuid.UUID]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		conversations: make(map[uuid.UUID]map[*Client]struct{}),
	}
}

func (h *Hub) Subscribe(conversationID uuid.UUID, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.conversations[conversationID] == nil {
		h.conversations[conversationID] = make(map[*Client]struct{})
	}

	h.conversations[conversationID][client] = struct{}{}
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
	}
}

func (h *Hub) Broadcast(conversationID uuid.UUID, event Event) {
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
