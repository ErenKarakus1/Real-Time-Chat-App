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
var ErrInvalidParticipantRole = errors.New("participant role must be admin or member")

type ConversationRepository interface {
	AddParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, role models.ParticipantRole) (models.ConversationParticipant, error)
	CreateDirect(ctx context.Context, conversation models.Conversation, userID uuid.UUID, otherUserID uuid.UUID) (models.Conversation, error)
	CreateRoom(ctx context.Context, conversation models.Conversation, creatorID uuid.UUID) (models.Conversation, error)
	FindByID(ctx context.Context, conversationID uuid.UUID) (models.Conversation, error)
	FindDirectBetweenUsers(ctx context.Context, userID uuid.UUID, otherUserID uuid.UUID) (models.Conversation, error)
	FindParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (models.ConversationParticipant, error)
	IsParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (bool, error)
	ListParticipants(ctx context.Context, conversationID uuid.UUID) ([]models.ConversationParticipant, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error)
	RemoveParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error
	UpdateParticipantRole(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, role models.ParticipantRole) (models.ConversationParticipant, error)
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

type ListParticipantsInput struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
}

type UpdateParticipantRoleInput struct {
	ConversationID uuid.UUID
	ActorID        uuid.UUID
	UserID         uuid.UUID
	Role           models.ParticipantRole
}

type RemoveParticipantInput struct {
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

func (s *ConversationService) ListParticipants(ctx context.Context, input ListParticipantsInput) ([]models.ConversationParticipant, error) {
	isParticipant, err := s.conversations.IsParticipant(ctx, input.ConversationID, input.UserID)
	if err != nil {
		return nil, err
	}

	if !isParticipant {
		return nil, ErrConversationManagementDenied
	}

	return s.conversations.ListParticipants(ctx, input.ConversationID)
}

func (s *ConversationService) UpdateParticipantRole(ctx context.Context, input UpdateParticipantRoleInput) (models.ConversationParticipant, error) {
	if input.UserID == uuid.Nil || input.ActorID == uuid.Nil || input.UserID == input.ActorID {
		return models.ConversationParticipant{}, ErrInvalidParticipant
	}

	if input.Role != models.ParticipantRoleAdmin && input.Role != models.ParticipantRoleMember {
		return models.ConversationParticipant{}, ErrInvalidParticipantRole
	}

	actor, err := s.conversations.FindParticipant(ctx, input.ConversationID, input.ActorID)
	if err != nil {
		return models.ConversationParticipant{}, err
	}

	if actor.Role != models.ParticipantRoleOwner {
		return models.ConversationParticipant{}, ErrConversationManagementDenied
	}

	target, err := s.conversations.FindParticipant(ctx, input.ConversationID, input.UserID)
	if err != nil {
		return models.ConversationParticipant{}, err
	}

	if target.Role == models.ParticipantRoleOwner {
		return models.ConversationParticipant{}, ErrConversationManagementDenied
	}

	return s.conversations.UpdateParticipantRole(ctx, input.ConversationID, input.UserID, input.Role)
}

func (s *ConversationService) RemoveParticipant(ctx context.Context, input RemoveParticipantInput) error {
	if input.UserID == uuid.Nil || input.ActorID == uuid.Nil || input.UserID == input.ActorID {
		return ErrInvalidParticipant
	}

	conversation, err := s.conversations.FindByID(ctx, input.ConversationID)
	if err != nil {
		return err
	}

	if conversation.Type != models.ConversationTypeRoom {
		return ErrConversationManagementDenied
	}

	actor, err := s.conversations.FindParticipant(ctx, input.ConversationID, input.ActorID)
	if err != nil {
		return err
	}

	if actor.Role != models.ParticipantRoleOwner && actor.Role != models.ParticipantRoleAdmin {
		return ErrConversationManagementDenied
	}

	target, err := s.conversations.FindParticipant(ctx, input.ConversationID, input.UserID)
	if err != nil {
		return err
	}

	if target.Role == models.ParticipantRoleOwner {
		return ErrConversationManagementDenied
	}

	if actor.Role == models.ParticipantRoleAdmin && target.Role != models.ParticipantRoleMember {
		return ErrConversationManagementDenied
	}

	return s.conversations.RemoveParticipant(ctx, input.ConversationID, input.UserID)
}
