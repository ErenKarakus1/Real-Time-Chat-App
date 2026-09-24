package realtime

import (
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Hub struct {
	mu            sync.RWMutex
	conversations map[uuid.UUID]map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{
		conversations: make(map[uuid.UUID]map[*websocket.Conn]struct{}),
	}
}

func (h *Hub) Subscribe(conversationID uuid.UUID, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.conversations[conversationID] == nil {
		h.conversations[conversationID] = make(map[*websocket.Conn]struct{})
	}

	h.conversations[conversationID][conn] = struct{}{}
}

func (h *Hub) Unsubscribe(conversationID uuid.UUID, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	connections := h.conversations[conversationID]
	if connections == nil {
		return
	}

	delete(connections, conn)
	if len(connections) == 0 {
		delete(h.conversations, conversationID)
	}
}
