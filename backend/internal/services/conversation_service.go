package services

import (
	"context"
	"errors"
	"strings"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	minRoomNameLength = 3
	maxRoomNameLength = 80
)

var ErrInvalidRoomName = errors.New("room name must be between 3 and 80 characters")
var ErrInvalidDirectConversation = errors.New("direct conversation requires two different users")
var ErrConversationManagementDenied = errors.New("user cannot manage this conversation")
var ErrInvalidParticipant = errors.New("participant must be a different user")

type ConversationRepository interface {
	AddParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, role models.ParticipantRole) (models.ConversationParticipant, error)
	CreateDirect(ctx context.Context, conversation models.Conversation, userID uuid.UUID, otherUserID uuid.UUID) (models.Conversation, error)
	CreateRoom(ctx context.Context, conversation models.Conversation, creatorID uuid.UUID) (models.Conversation, error)
	FindByID(ctx context.Context, conversationID uuid.UUID) (models.Conversation, error)
	FindDirectBetweenUsers(ctx context.Context, userID uuid.UUID, otherUserID uuid.UUID) (models.Conversation, error)
	FindParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (models.ConversationParticipant, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error)
}

type ConversationService struct {
	conversations ConversationRepository
}

type CreateRoomInput struct {
	Name      string
	CreatorID uuid.UUID
}

type CreateDirectInput struct {
	UserID      uuid.UUID
	OtherUserID uuid.UUID
}

type AddRoomParticipantInput struct {
	ConversationID uuid.UUID
	ActorID        uuid.UUID
	UserID         uuid.UUID
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

func (s *ConversationService) CreateDirect(ctx context.Context, input CreateDirectInput) (models.Conversation, error) {
	if input.UserID == uuid.Nil || input.OtherUserID == uuid.Nil || input.UserID == input.OtherUserID {
		return models.Conversation{}, ErrInvalidDirectConversation
	}

	existing, err := s.conversations.FindDirectBetweenUsers(ctx, input.UserID, input.OtherUserID)
	if err == nil {
		return existing, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return models.Conversation{}, err
	}

	conversation := models.Conversation{
		ID:   uuid.New(),
		Type: models.ConversationTypeDirect,
	}

	return s.conversations.CreateDirect(ctx, conversation, input.UserID, input.OtherUserID)
}

func (s *ConversationService) ListForUser(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error) {
	return s.conversations.ListForUser(ctx, userID)
}

func (s *ConversationService) AddRoomParticipant(ctx context.Context, input AddRoomParticipantInput) (models.ConversationParticipant, error) {
	if input.UserID == uuid.Nil || input.ActorID == uuid.Nil || input.UserID == input.ActorID {
		return models.ConversationParticipant{}, ErrInvalidParticipant
	}

	conversation, err := s.conversations.FindByID(ctx, input.ConversationID)
	if err != nil {
		return models.ConversationParticipant{}, err
	}

	if conversation.Type != models.ConversationTypeRoom {
		return models.ConversationParticipant{}, ErrConversationManagementDenied
	}

	actor, err := s.conversations.FindParticipant(ctx, input.ConversationID, input.ActorID)
	if err != nil {
		return models.ConversationParticipant{}, err
	}

	if actor.Role != models.ParticipantRoleOwner && actor.Role != models.ParticipantRoleAdmin {
		return models.ConversationParticipant{}, ErrConversationManagementDenied
	}

	return s.conversations.AddParticipant(ctx, input.ConversationID, input.UserID, models.ParticipantRoleMember)
}
