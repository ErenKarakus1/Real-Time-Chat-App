package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/middleware"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ConversationService interface {
	CreateRoom(ctx context.Context, input services.CreateRoomInput) (models.Conversation, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error)
}

type ConversationHandler struct {
	conversations ConversationService
}

type createRoomRequest struct {
	Name string `json:"name"`
}

func NewConversationHandler(conversations ConversationService) *ConversationHandler {
	return &ConversationHandler{
		conversations: conversations,
	}
}

func (h *ConversationHandler) CreateRoom(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user is required"})
		return
	}

	var req createRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	conversation, err := h.conversations.CreateRoom(c, services.CreateRoomInput{
		Name:      req.Name,
		CreatorID: userID,
	})
	if errors.Is(err, services.ErrInvalidRoomName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create room"})
		return
	}

	c.JSON(http.StatusCreated, models.NewConversationResponse(conversation))
}

func (h *ConversationHandler) List(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user is required"})
		return
	}

	conversations, err := h.conversations.ListForUser(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list conversations"})
		return
	}

	c.JSON(http.StatusOK, models.NewConversationResponses(conversations))
}

func authenticatedUserID(c *gin.Context) (uuid.UUID, bool) {
	userIDValue, exists := c.Get(middleware.ContextUserID)
	userIDString, ok := userIDValue.(string)
	if !exists || !ok {
		return uuid.Nil, false
	}

	userID, err := uuid.Parse(userIDString)
	if err != nil {
		return uuid.Nil, false
	}

	return userID, true
}
