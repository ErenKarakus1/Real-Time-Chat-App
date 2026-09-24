package services

import (
	"context"
	"errors"
	"strings"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/google/uuid"
)

const (
	minRoomNameLength = 3
	maxRoomNameLength = 80
)

var ErrInvalidRoomName = errors.New("room name must be between 3 and 80 characters")

type ConversationRepository interface {
	CreateRoom(ctx context.Context, conversation models.Conversation, creatorID uuid.UUID) (models.Conversation, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error)
}

type ConversationService struct {
	conversations ConversationRepository
}

type CreateRoomInput struct {
	Name      string
	CreatorID uuid.UUID
}

func NewConversationService(conversations ConversationRepository) *ConversationService {
	return &ConversationService{
		conversations: conversations,
	}
}

func (s *ConversationService) CreateRoom(ctx context.Context, input CreateRoomInput) (models.Conversation, error) {
	name := strings.TrimSpace(input.Name)
	if len(name) < minRoomNameLength || len(name) > maxRoomNameLength {
		return models.Conversation{}, ErrInvalidRoomName
	}

	conversation := models.Conversation{
		ID:   uuid.New(),
		Type: models.ConversationTypeRoom,
		Name: &name,
	}

	return s.conversations.CreateRoom(ctx, conversation, input.CreatorID)
}

func (s *ConversationService) ListForUser(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error) {
	return s.conversations.ListForUser(ctx, userID)
}
