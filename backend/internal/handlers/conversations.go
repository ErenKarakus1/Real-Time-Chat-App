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
	"github.com/jackc/pgx/v5"
)

type ConversationService interface {
	AddRoomParticipant(ctx context.Context, input services.AddRoomParticipantInput) (models.ConversationParticipant, error)
	CreateDirect(ctx context.Context, input services.CreateDirectInput) (models.Conversation, error)
	CreateRoom(ctx context.Context, input services.CreateRoomInput) (models.Conversation, error)
	DeleteRoom(ctx context.Context, input services.DeleteRoomInput) error
	LeaveRoom(ctx context.Context, input services.LeaveRoomInput) error
	ListParticipants(ctx context.Context, input services.ListParticipantsInput) ([]models.ConversationParticipant, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]models.ConversationListItem, error)
	MarkRead(ctx context.Context, input services.MarkConversationReadInput) (models.ConversationParticipant, error)
	RemoveParticipant(ctx context.Context, input services.RemoveParticipantInput) error
	SearchForUser(ctx context.Context, userID uuid.UUID, query string) ([]models.ConversationListItem, error)
	TransferOwnership(ctx context.Context, input services.TransferOwnershipInput) (models.ConversationParticipant, error)
	UpdateRoomName(ctx context.Context, input services.UpdateRoomNameInput) (models.Conversation, error)
	UpdateParticipantRole(ctx context.Context, input services.UpdateParticipantRoleInput) (models.ConversationParticipant, error)
}

type ConversationHandler struct {
	conversations ConversationService
}

type createRoomRequest struct {
	Name string `json:"name"`
}

type createDirectRequest struct {
	OtherUserID string `json:"other_user_id"`
}

type updateRoomNameRequest struct {
	Name string `json:"name"`
}

type addParticipantRequest struct {
	UserID string `json:"user_id"`
}

type updateParticipantRoleRequest struct {
	Role string `json:"role"`
}

type transferOwnershipRequest struct {
	UserID string `json:"user_id"`
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

func (h *ConversationHandler) CreateDirect(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user is required"})
		return
	}

	var req createDirectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	otherUserID, err := uuid.Parse(req.OtherUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "other_user_id must be a valid UUID"})
		return
	}

	conversation, err := h.conversations.CreateDirect(c, services.CreateDirectInput{
		UserID:      userID,
		OtherUserID: otherUserID,
	})
	if errors.Is(err, services.ErrInvalidDirectConversation) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create direct conversation"})
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

	conversations, err := h.conversations.SearchForUser(c, userID, c.Query("q"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list conversations"})
		return
	}

	c.JSON(http.StatusOK, models.NewConversationListItemResponses(conversations))
}

func (h *ConversationHandler) MarkRead(c *gin.Context) {
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

	participant, err := h.conversations.MarkRead(c, services.MarkConversationReadInput{
		ConversationID: conversationID,
		UserID:         userID,
	})
	if errors.Is(err, services.ErrConversationManagementDenied) {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not mark conversation read"})
		return
	}

	c.JSON(http.StatusOK, models.NewParticipantResponse(participant))
}

func (h *ConversationHandler) UpdateRoomName(c *gin.Context) {
	actorID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user is required"})
		return
	}

	conversationID, ok := conversationIDParam(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id must be a valid UUID"})
		return
	}

	var req updateRoomNameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	conversation, err := h.conversations.UpdateRoomName(c, services.UpdateRoomNameInput{
		ConversationID: conversationID,
		ActorID:        actorID,
		Name:           req.Name,
	})
	if errors.Is(err, services.ErrInvalidRoomName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, services.ErrConversationManagementDenied) {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update room"})
		return
	}

	c.JSON(http.StatusOK, models.NewConversationResponse(conversation))
}

func (h *ConversationHandler) DeleteRoom(c *gin.Context) {
	actorID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user is required"})
		return
	}

	conversationID, ok := conversationIDParam(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id must be a valid UUID"})
		return
	}

	err := h.conversations.DeleteRoom(c, services.DeleteRoomInput{
		ConversationID: conversationID,
		ActorID:        actorID,
	})
	if errors.Is(err, services.ErrConversationManagementDenied) {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete room"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ConversationHandler) AddParticipant(c *gin.Context) {
	actorID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user is required"})
		return
	}

	conversationID, ok := conversationIDParam(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id must be a valid UUID"})
		return
	}

	var req addParticipantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id must be a valid UUID"})
		return
	}

	participant, err := h.conversations.AddRoomParticipant(c, services.AddRoomParticipantInput{
		ConversationID: conversationID,
		ActorID:        actorID,
		UserID:         userID,
	})
	if errors.Is(err, services.ErrInvalidParticipant) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, services.ErrConversationManagementDenied) {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation or participant not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not add participant"})
		return
	}

	c.JSON(http.StatusCreated, models.NewParticipantResponse(participant))
}

func (h *ConversationHandler) ListParticipants(c *gin.Context) {
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

	participants, err := h.conversations.ListParticipants(c, services.ListParticipantsInput{
		ConversationID: conversationID,
		UserID:         userID,
	})
	if errors.Is(err, services.ErrConversationManagementDenied) {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list participants"})
		return
	}

	c.JSON(http.StatusOK, models.NewParticipantResponses(participants))
}

func (h *ConversationHandler) UpdateParticipantRole(c *gin.Context) {
	actorID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user is required"})
		return
	}

	conversationID, ok := conversationIDParam(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id must be a valid UUID"})
		return
	}

	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id must be a valid UUID"})
		return
	}

	var req updateParticipantRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	participant, err := h.conversations.UpdateParticipantRole(c, services.UpdateParticipantRoleInput{
		ConversationID: conversationID,
		ActorID:        actorID,
		UserID:         userID,
		Role:           models.ParticipantRole(req.Role),
	})
	if errors.Is(err, services.ErrInvalidParticipant) || errors.Is(err, services.ErrInvalidParticipantRole) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, services.ErrConversationManagementDenied) {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation or participant not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update participant role"})
		return
	}

	c.JSON(http.StatusOK, models.NewParticipantResponse(participant))
}

func (h *ConversationHandler) RemoveParticipant(c *gin.Context) {
	actorID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user is required"})
		return
	}

	conversationID, ok := conversationIDParam(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id must be a valid UUID"})
		return
	}

	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id must be a valid UUID"})
		return
	}

	err = h.conversations.RemoveParticipant(c, services.RemoveParticipantInput{
		ConversationID: conversationID,
		ActorID:        actorID,
		UserID:         userID,
	})
	if errors.Is(err, services.ErrInvalidParticipant) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, services.ErrConversationManagementDenied) {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation or participant not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not remove participant"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ConversationHandler) LeaveRoom(c *gin.Context) {
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

	err := h.conversations.LeaveRoom(c, services.LeaveRoomInput{
		ConversationID: conversationID,
		UserID:         userID,
	})
	if errors.Is(err, services.ErrInvalidParticipant) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, services.ErrConversationManagementDenied) {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation or participant not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not leave room"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ConversationHandler) TransferOwnership(c *gin.Context) {
	actorID, ok := authenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user is required"})
		return
	}

	conversationID, ok := conversationIDParam(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id must be a valid UUID"})
		return
	}

	var req transferOwnershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id must be a valid UUID"})
		return
	}

	participant, err := h.conversations.TransferOwnership(c, services.TransferOwnershipInput{
		ConversationID: conversationID,
		ActorID:        actorID,
		UserID:         userID,
	})
	if errors.Is(err, services.ErrInvalidParticipant) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, services.ErrConversationManagementDenied) {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation or participant not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not transfer ownership"})
		return
	}

	c.JSON(http.StatusOK, models.NewParticipantResponse(participant))
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
