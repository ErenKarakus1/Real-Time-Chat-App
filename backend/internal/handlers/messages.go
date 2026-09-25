package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/realtime"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MessageService interface {
	Create(ctx context.Context, input services.CreateMessageInput) (models.Message, error)
	ListForConversation(ctx context.Context, input services.ListMessagesInput) ([]models.Message, error)
}

type MessageHandler struct {
	messages MessageService
	hub      *realtime.Hub
}

type createMessageRequest struct {
	Content string `json:"content"`
}

func NewMessageHandler(messages MessageService, hub *realtime.Hub) *MessageHandler {
	return &MessageHandler{
		messages: messages,
		hub:      hub,
	}
}

func (h *MessageHandler) Create(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user is required"})
		return
	}

	conversationID, ok := conversationIDParam(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id must be a valid UUID"})
		return
	}

	var req createMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	message, err := h.messages.Create(c, services.CreateMessageInput{
		ConversationID: conversationID,
		SenderID:       userID,
		Content:        req.Content,
	})
	if errors.Is(err, services.ErrConversationAccessDenied) {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, services.ErrInvalidMessageContent) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create message"})
		return
	}

	response := models.NewMessageResponse(message)
	if err := h.hub.Publish(c, conversationID, realtime.Event{
		Type: realtime.EventMessageCreated,
		Data: response,
	}); err != nil {
		log.Printf("broadcast message: %v", err)
	}

	c.JSON(http.StatusCreated, response)
}

func (h *MessageHandler) List(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user is required"})
		return
	}

	conversationID, ok := conversationIDParam(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id must be a valid UUID"})
		return
	}

	limit, ok := limitQuery(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
		return
	}

	messages, err := h.messages.ListForConversation(c, services.ListMessagesInput{
		ConversationID: conversationID,
		UserID:         userID,
		Limit:          limit,
	})
	if errors.Is(err, services.ErrConversationAccessDenied) {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list messages"})
		return
	}

	c.JSON(http.StatusOK, models.NewMessageResponses(messages))
}

func conversationIDParam(c *gin.Context) (uuid.UUID, bool) {
	conversationID, err := uuid.Parse(c.Param("conversation_id"))
	return conversationID, err == nil
}

func limitQuery(c *gin.Context) (int, bool) {
	value := c.Query("limit")
	if value == "" {
		return 0, true
	}

	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return 0, false
	}

	return limit, true
}
