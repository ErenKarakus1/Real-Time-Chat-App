package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/google/uuid"
)

type fakeMessageRepository struct {
	messages map[uuid.UUID]models.Message

	createdContent string
	updatedContent string
	deletedID      uuid.UUID
	listLimit      int
}

func newFakeMessageRepository() *fakeMessageRepository {
	return &fakeMessageRepository{messages: make(map[uuid.UUID]models.Message)}
}

func (r *fakeMessageRepository) Create(ctx context.Context, message models.Message) (models.Message, error) {
	r.createdContent = message.Content
	r.messages[message.ID] = message
	return message, nil
}

func (r *fakeMessageRepository) Delete(ctx context.Context, messageID uuid.UUID) error {
	r.deletedID = messageID
	return nil
}

func (r *fakeMessageRepository) FindByID(ctx context.Context, messageID uuid.UUID) (models.Message, error) {
	message, ok := r.messages[messageID]
	if !ok {
		return models.Message{}, errors.New("not found")
	}
	return message, nil
}

func (r *fakeMessageRepository) ListForConversation(ctx context.Context, conversationID uuid.UUID, before *time.Time, limit int) ([]models.Message, error) {
	r.listLimit = limit
	return nil, nil
}

func (r *fakeMessageRepository) UpdateContent(ctx context.Context, messageID uuid.UUID, content string) (models.Message, error) {
	r.updatedContent = content
	message := r.messages[messageID]
	message.Content = content
	r.messages[messageID] = message
	return message, nil
}

type fakeMessageConversationRepository struct {
	participants map[uuid.UUID]map[uuid.UUID]bool
}

func newFakeMessageConversationRepository() *fakeMessageConversationRepository {
	return &fakeMessageConversationRepository{participants: make(map[uuid.UUID]map[uuid.UUID]bool)}
}

func (r *fakeMessageConversationRepository) IsParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (bool, error) {
	return r.participants[conversationID][userID], nil
}

func allowMessageParticipant(repo *fakeMessageConversationRepository, conversationID uuid.UUID, userID uuid.UUID) {
	if repo.participants[conversationID] == nil {
		repo.participants[conversationID] = make(map[uuid.UUID]bool)
	}
	repo.participants[conversationID][userID] = true
}

func TestMessageServiceCreateValidatesAndTrimsContent(t *testing.T) {
	messages := newFakeMessageRepository()
	conversations := newFakeMessageConversationRepository()
	service := NewMessageService(messages, conversations)
	conversationID := uuid.New()
	userID := uuid.New()
	allowMessageParticipant(conversations, conversationID, userID)

	if _, err := service.Create(context.Background(), CreateMessageInput{ConversationID: conversationID, SenderID: userID, Content: "   "}); !errors.Is(err, ErrInvalidMessageContent) {
		t.Fatalf("blank content error = %v, want %v", err, ErrInvalidMessageContent)
	}
	if _, err := service.Create(context.Background(), CreateMessageInput{ConversationID: conversationID, SenderID: userID, Content: strings.Repeat("a", MaxMessageLength+1)}); !errors.Is(err, ErrInvalidMessageContent) {
		t.Fatalf("long content error = %v, want %v", err, ErrInvalidMessageContent)
	}
	if _, err := service.Create(context.Background(), CreateMessageInput{ConversationID: conversationID, SenderID: userID, Content: " hello "}); err != nil {
		t.Fatalf("create message: %v", err)
	}
	if messages.createdContent != "hello" {
		t.Fatalf("created content = %q, want %q", messages.createdContent, "hello")
	}
}

func TestMessageServiceRequiresParticipant(t *testing.T) {
	service := NewMessageService(newFakeMessageRepository(), newFakeMessageConversationRepository())

	_, err := service.Create(context.Background(), CreateMessageInput{
		ConversationID: uuid.New(),
		SenderID:       uuid.New(),
		Content:        "hello",
	})
	if !errors.Is(err, ErrConversationAccessDenied) {
		t.Fatalf("create non-participant error = %v, want %v", err, ErrConversationAccessDenied)
	}
}

func TestMessageServiceListCapsLimit(t *testing.T) {
	messages := newFakeMessageRepository()
	conversations := newFakeMessageConversationRepository()
	service := NewMessageService(messages, conversations)
	conversationID := uuid.New()
	userID := uuid.New()
	allowMessageParticipant(conversations, conversationID, userID)

	if _, err := service.ListForConversation(context.Background(), ListMessagesInput{ConversationID: conversationID, UserID: userID, Limit: MaxMessageLimit + 50}); err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if messages.listLimit != MaxMessageLimit {
		t.Fatalf("list limit = %d, want %d", messages.listLimit, MaxMessageLimit)
	}

	if _, err := service.ListForConversation(context.Background(), ListMessagesInput{ConversationID: conversationID, UserID: userID}); err != nil {
		t.Fatalf("list default messages: %v", err)
	}
	if messages.listLimit != DefaultMessageLimit {
		t.Fatalf("default list limit = %d, want %d", messages.listLimit, DefaultMessageLimit)
	}
}

func TestMessageServiceUpdateAndDeleteRequireOwnership(t *testing.T) {
	messages := newFakeMessageRepository()
	conversations := newFakeMessageConversationRepository()
	service := NewMessageService(messages, conversations)
	conversationID := uuid.New()
	ownerID := uuid.New()
	otherID := uuid.New()
	messageID := uuid.New()
	allowMessageParticipant(conversations, conversationID, ownerID)
	allowMessageParticipant(conversations, conversationID, otherID)
	messages.messages[messageID] = models.Message{
		ID:             messageID,
		ConversationID: conversationID,
		SenderID:       &ownerID,
		Content:        "old",
	}

	if _, err := service.Update(context.Background(), UpdateMessageInput{ConversationID: conversationID, MessageID: messageID, UserID: otherID, Content: "new"}); !errors.Is(err, ErrMessageOwnershipDenied) {
		t.Fatalf("update by non-owner error = %v, want %v", err, ErrMessageOwnershipDenied)
	}
	if _, err := service.Update(context.Background(), UpdateMessageInput{ConversationID: conversationID, MessageID: messageID, UserID: ownerID, Content: " new "}); err != nil {
		t.Fatalf("update by owner: %v", err)
	}
	if messages.updatedContent != "new" {
		t.Fatalf("updated content = %q, want %q", messages.updatedContent, "new")
	}

	if err := service.Delete(context.Background(), DeleteMessageInput{ConversationID: conversationID, MessageID: messageID, UserID: otherID}); !errors.Is(err, ErrMessageOwnershipDenied) {
		t.Fatalf("delete by non-owner error = %v, want %v", err, ErrMessageOwnershipDenied)
	}
	if err := service.Delete(context.Background(), DeleteMessageInput{ConversationID: conversationID, MessageID: messageID, UserID: ownerID}); err != nil {
		t.Fatalf("delete by owner: %v", err)
	}
	if messages.deletedID != messageID {
		t.Fatalf("deleted message = %s, want %s", messages.deletedID, messageID)
	}
}
