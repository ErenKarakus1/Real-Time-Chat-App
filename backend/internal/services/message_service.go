package services

import (
	"context"
	"errors"
	"strings"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/google/uuid"
)

const DefaultMessageLimit = 50

var ErrConversationAccessDenied = errors.New("user is not a participant in this conversation")
var ErrInvalidMessageContent = errors.New("message content is required")

type MessageRepository interface {
	Create(ctx context.Context, message models.Message) (models.Message, error)
	ListForConversation(ctx context.Context, conversationID uuid.UUID, limit int) ([]models.Message, error)
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
	Limit          int
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

	return s.messages.ListForConversation(ctx, input.ConversationID, limit)
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
