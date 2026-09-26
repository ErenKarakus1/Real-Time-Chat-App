package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/google/uuid"
)

const (
	DefaultMessageLimit = 50
	MaxMessageLimit     = 100
	MaxMessageLength    = 2000
)

var ErrConversationAccessDenied = errors.New("user is not a participant in this conversation")
var ErrInvalidMessageContent = errors.New("message content is required")
var ErrMessageOwnershipDenied = errors.New("user can only manage their own messages")

type MessageRepository interface {
	Create(ctx context.Context, message models.Message) (models.Message, error)
	Delete(ctx context.Context, messageID uuid.UUID) error
	FindByID(ctx context.Context, messageID uuid.UUID) (models.Message, error)
	ListForConversation(ctx context.Context, conversationID uuid.UUID, before *time.Time, limit int) ([]models.Message, error)
	UpdateContent(ctx context.Context, messageID uuid.UUID, content string) (models.Message, error)
}

type MessageConversationRepository interface {
	IsParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (bool, error)
}

type MessageService struct {
	messages      MessageRepository
	conversations MessageConversationRepository
}

type CreateMessageInput struct {
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	Content        string
}

type ListMessagesInput struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	Before         *time.Time
	Limit          int
}

type UpdateMessageInput struct {
	ConversationID uuid.UUID
	MessageID      uuid.UUID
	UserID         uuid.UUID
	Content        string
}

type DeleteMessageInput struct {
	ConversationID uuid.UUID
	MessageID      uuid.UUID
	UserID         uuid.UUID
}

func NewMessageService(messages MessageRepository, conversations MessageConversationRepository) *MessageService {
	return &MessageService{
		messages:      messages,
		conversations: conversations,
	}
}

func (s *MessageService) Create(ctx context.Context, input CreateMessageInput) (models.Message, error) {
	if err := s.ensureParticipant(ctx, input.ConversationID, input.SenderID); err != nil {
		return models.Message{}, err
	}

	content := strings.TrimSpace(input.Content)
	if content == "" {
		return models.Message{}, ErrInvalidMessageContent
	}
	if len(content) > MaxMessageLength {
		return models.Message{}, ErrInvalidMessageContent
	}

	message := models.Message{
		ID:             uuid.New(),
		ConversationID: input.ConversationID,
		SenderID:       &input.SenderID,
		Content:        content,
	}

	return s.messages.Create(ctx, message)
}

func (s *MessageService) ListForConversation(ctx context.Context, input ListMessagesInput) ([]models.Message, error) {
	if err := s.ensureParticipant(ctx, input.ConversationID, input.UserID); err != nil {
		return nil, err
	}

	limit := input.Limit
	if limit <= 0 {
		limit = DefaultMessageLimit
	}
	if limit > MaxMessageLimit {
		limit = MaxMessageLimit
	}

	return s.messages.ListForConversation(ctx, input.ConversationID, input.Before, limit)
}

func (s *MessageService) Update(ctx context.Context, input UpdateMessageInput) (models.Message, error) {
	if err := s.ensureParticipant(ctx, input.ConversationID, input.UserID); err != nil {
		return models.Message{}, err
	}

	message, err := s.messages.FindByID(ctx, input.MessageID)
	if err != nil {
		return models.Message{}, err
	}

	if message.ConversationID != input.ConversationID {
		return models.Message{}, ErrMessageOwnershipDenied
	}

	if message.SenderID == nil || *message.SenderID != input.UserID {
		return models.Message{}, ErrMessageOwnershipDenied
	}

	content := strings.TrimSpace(input.Content)
	if content == "" {
		return models.Message{}, ErrInvalidMessageContent
	}
	if len(content) > MaxMessageLength {
		return models.Message{}, ErrInvalidMessageContent
	}

	return s.messages.UpdateContent(ctx, input.MessageID, content)
}

func (s *MessageService) Delete(ctx context.Context, input DeleteMessageInput) error {
	if err := s.ensureParticipant(ctx, input.ConversationID, input.UserID); err != nil {
		return err
	}

	message, err := s.messages.FindByID(ctx, input.MessageID)
	if err != nil {
		return err
	}

	if message.ConversationID != input.ConversationID {
		return ErrMessageOwnershipDenied
	}

	if message.SenderID == nil || *message.SenderID != input.UserID {
		return ErrMessageOwnershipDenied
	}

	return s.messages.Delete(ctx, input.MessageID)
}

func (s *MessageService) ensureParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error {
	isParticipant, err := s.conversations.IsParticipant(ctx, conversationID, userID)
	if err != nil {
		return err
	}

	if !isParticipant {
		return ErrConversationAccessDenied
	}

	return nil
}
