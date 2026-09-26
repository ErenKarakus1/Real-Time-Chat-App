package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/auth"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/presence"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/realtime"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type WebSocketConversationService interface {
	IsParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (bool, error)
}

type WebSocketHandler struct {
	conversations WebSocketConversationService
	hub           *realtime.Hub
	presence      *presence.Service
	jwtSecret     string
	upgrader      websocket.Upgrader
}

type typingEventResponse struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         uuid.UUID `json:"user_id"`
}

func NewWebSocketHandler(conversations WebSocketConversationService, hub *realtime.Hub, presence *presence.Service, jwtSecret string) *WebSocketHandler {
	return &WebSocketHandler{
		conversations: conversations,
		hub:           hub,
		presence:      presence,
		jwtSecret:     jwtSecret,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

func (h *WebSocketHandler) Conversation(c *gin.Context) {
	conversationID, err := uuid.Parse(c.Param("conversation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id must be a valid UUID"})
		return
	}

	claims, err := auth.ParseToken(c.Query("token"), h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization token is invalid"})
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization token is invalid"})
		return
	}

	isParticipant, err := h.conversations.IsParticipant(c, conversationID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not verify conversation access"})
		return
	}

	if !isParticipant {
		c.JSON(http.StatusForbidden, gin.H{"error": "user is not a participant in this conversation"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := realtime.NewClient(conn)
	if err := h.presence.Connect(c, userID); err != nil {
		conn.Close()
		return
	}
	defer h.presence.Disconnect(context.Background(), userID)

	h.hub.Subscribe(c, conversationID, client)
	defer h.hub.Unsubscribe(conversationID, client)
	defer h.publishTypingStopped(context.Background(), conversationID, userID)

	go client.WritePump()
	client.ReadPump(func(event realtime.Event) {
		h.handleIncomingEvent(context.Background(), conversationID, userID, event)
	})
}

func (h *WebSocketHandler) publishTypingStopped(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) {
	if err := h.hub.Publish(ctx, conversationID, realtime.Event{
		Type: realtime.EventTypingStopped,
		Data: typingEventResponse{
			ConversationID: conversationID,
			UserID:         userID,
		},
	}); err != nil {
		log.Printf("broadcast typing stop: %v", err)
	}
}

func (h *WebSocketHandler) handleIncomingEvent(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, event realtime.Event) {
	if event.Type != realtime.EventTypingStarted && event.Type != realtime.EventTypingStopped {
		return
	}

	if err := h.hub.Publish(ctx, conversationID, realtime.Event{
		Type: event.Type,
		Data: typingEventResponse{
			ConversationID: conversationID,
			UserID:         userID,
		},
	}); err != nil {
		log.Printf("broadcast typing event: %v", err)
	}
}
