package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/middleware"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/realtime"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeHandlerConversationService struct {
	listResult     []models.ConversationListItem
	markReadErr    error
	markReadCalled bool
	createRoomErr  error
}

func (s *fakeHandlerConversationService) AddRoomParticipant(ctx context.Context, input services.AddRoomParticipantInput) (models.ConversationParticipant, error) {
	return models.ConversationParticipant{}, nil
}

func (s *fakeHandlerConversationService) CreateDirect(ctx context.Context, input services.CreateDirectInput) (models.Conversation, error) {
	return models.Conversation{}, nil
}

func (s *fakeHandlerConversationService) CreateRoom(ctx context.Context, input services.CreateRoomInput) (models.Conversation, error) {
	if s.createRoomErr != nil {
		return models.Conversation{}, s.createRoomErr
	}
	name := input.Name
	return models.Conversation{ID: uuid.New(), Type: models.ConversationTypeRoom, Name: &name, CreatedBy: &input.CreatorID}, nil
}

func (s *fakeHandlerConversationService) DeleteRoom(ctx context.Context, input services.DeleteRoomInput) error {
	return nil
}

func (s *fakeHandlerConversationService) LeaveRoom(ctx context.Context, input services.LeaveRoomInput) error {
	return nil
}

func (s *fakeHandlerConversationService) ListParticipants(ctx context.Context, input services.ListParticipantsInput) ([]models.ConversationParticipant, error) {
	return nil, nil
}

func (s *fakeHandlerConversationService) ListForUser(ctx context.Context, userID uuid.UUID) ([]models.ConversationListItem, error) {
	return s.listResult, nil
}

func (s *fakeHandlerConversationService) MarkRead(ctx context.Context, input services.MarkConversationReadInput) (models.ConversationParticipant, error) {
	s.markReadCalled = true
	if s.markReadErr != nil {
		return models.ConversationParticipant{}, s.markReadErr
	}
	now := time.Now()
	return models.ConversationParticipant{ConversationID: input.ConversationID, UserID: input.UserID, Role: models.ParticipantRoleMember, LastReadAt: &now}, nil
}

func (s *fakeHandlerConversationService) RemoveParticipant(ctx context.Context, input services.RemoveParticipantInput) error {
	return nil
}

func (s *fakeHandlerConversationService) SearchForUser(ctx context.Context, userID uuid.UUID, query string) ([]models.ConversationListItem, error) {
	return s.listResult, nil
}

func (s *fakeHandlerConversationService) TransferOwnership(ctx context.Context, input services.TransferOwnershipInput) (models.ConversationParticipant, error) {
	return models.ConversationParticipant{}, nil
}

func (s *fakeHandlerConversationService) UpdateRoomName(ctx context.Context, input services.UpdateRoomNameInput) (models.Conversation, error) {
	return models.Conversation{}, nil
}

func (s *fakeHandlerConversationService) UpdateParticipantRole(ctx context.Context, input services.UpdateParticipantRoleInput) (models.ConversationParticipant, error) {
	return models.ConversationParticipant{}, nil
}

type fakeHandlerMessageService struct {
	createErr error
	deleteErr error
	listErr   error
	updateErr error
}

func (s *fakeHandlerMessageService) Create(ctx context.Context, input services.CreateMessageInput) (models.Message, error) {
	if s.createErr != nil {
		return models.Message{}, s.createErr
	}
	return models.Message{ID: uuid.New(), ConversationID: input.ConversationID, SenderID: &input.SenderID, Content: input.Content, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}

func (s *fakeHandlerMessageService) Delete(ctx context.Context, input services.DeleteMessageInput) error {
	return s.deleteErr
}

func (s *fakeHandlerMessageService) ListForConversation(ctx context.Context, input services.ListMessagesInput) ([]models.Message, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return []models.Message{{ID: uuid.New(), ConversationID: input.ConversationID, SenderID: &input.UserID, Content: "hello"}}, nil
}

func (s *fakeHandlerMessageService) Update(ctx context.Context, input services.UpdateMessageInput) (models.Message, error) {
	if s.updateErr != nil {
		return models.Message{}, s.updateErr
	}
	return models.Message{ID: input.MessageID, ConversationID: input.ConversationID, SenderID: &input.UserID, Content: input.Content}, nil
}

type fakePubSub struct {
	events []realtime.Event
}

func (p *fakePubSub) Publish(ctx context.Context, conversationID uuid.UUID, event realtime.Event) error {
	p.events = append(p.events, event)
	return nil
}

func (p *fakePubSub) Subscribe(ctx context.Context, conversationID uuid.UUID) realtime.Subscription {
	return nil
}

func testRouter(userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if userID != uuid.Nil {
		router.Use(func(c *gin.Context) {
			c.Set(middleware.ContextUserID, userID.String())
			c.Next()
		})
	}
	return router
}

func performJSON(router http.Handler, method string, path string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestConversationHandlerListReturnsUnreadCount(t *testing.T) {
	userID := uuid.New()
	conversationID := uuid.New()
	service := &fakeHandlerConversationService{
		listResult: []models.ConversationListItem{{
			Conversation: models.Conversation{ID: conversationID, Type: models.ConversationTypeRoom},
			UnreadCount: 3,
		}},
	}
	handler := NewConversationHandler(service)
	router := testRouter(userID)
	router.GET("/conversations", handler.List)

	recorder := performJSON(router, http.MethodGet, "/conversations?q=dev", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response []models.ConversationListItemResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response) != 1 || response[0].UnreadCount != 3 {
		t.Fatalf("response unread count = %+v, want 3", response)
	}
}

func TestConversationHandlerMarkReadStatusCodes(t *testing.T) {
	userID := uuid.New()
	conversationID := uuid.New()
	service := &fakeHandlerConversationService{markReadErr: services.ErrConversationManagementDenied}
	handler := NewConversationHandler(service)
	router := testRouter(userID)
	router.POST("/conversations/:conversation_id/read", handler.MarkRead)

	recorder := performJSON(router, http.MethodPost, "/conversations/not-a-uuid/read", "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("bad uuid status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	recorder = performJSON(router, http.MethodPost, "/conversations/"+conversationID.String()+"/read", "")
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("denied status = %d, want %d", recorder.Code, http.StatusForbidden)
	}

	service.markReadErr = nil
	recorder = performJSON(router, http.MethodPost, "/conversations/"+conversationID.String()+"/read", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("ok status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !service.markReadCalled {
		t.Fatalf("mark read was not called")
	}
}

func TestConversationHandlerCreateRoomValidation(t *testing.T) {
	handler := NewConversationHandler(&fakeHandlerConversationService{createRoomErr: services.ErrInvalidRoomName})
	router := testRouter(uuid.New())
	router.POST("/conversations/rooms", handler.CreateRoom)

	recorder := performJSON(router, http.MethodPost, "/conversations/rooms", `{"name":"x"}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid room status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestMessageHandlerCreatePublishesEvent(t *testing.T) {
	userID := uuid.New()
	conversationID := uuid.New()
	pubsub := &fakePubSub{}
	handler := NewMessageHandler(&fakeHandlerMessageService{}, realtime.NewHub(pubsub))
	router := testRouter(userID)
	router.POST("/conversations/:conversation_id/messages", handler.Create)

	recorder := performJSON(router, http.MethodPost, "/conversations/"+conversationID.String()+"/messages", `{"content":"hello"}`)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if len(pubsub.events) != 1 || pubsub.events[0].Type != realtime.EventMessageCreated {
		t.Fatalf("events = %+v, want message.created", pubsub.events)
	}
}

func TestMessageHandlerStatusCodes(t *testing.T) {
	userID := uuid.New()
	conversationID := uuid.New()
	messageID := uuid.New()
	handler := NewMessageHandler(&fakeHandlerMessageService{updateErr: services.ErrInvalidMessageContent, deleteErr: services.ErrMessageOwnershipDenied}, realtime.NewHub(&fakePubSub{}))
	router := testRouter(userID)
	router.PATCH("/conversations/:conversation_id/messages/:message_id", handler.Update)
	router.DELETE("/conversations/:conversation_id/messages/:message_id", handler.Delete)
	router.GET("/conversations/:conversation_id/messages", handler.List)

	recorder := performJSON(router, http.MethodPatch, "/conversations/"+conversationID.String()+"/messages/"+messageID.String(), `{"content":""}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid update status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	recorder = performJSON(router, http.MethodDelete, "/conversations/"+conversationID.String()+"/messages/"+messageID.String(), "")
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("delete denied status = %d, want %d", recorder.Code, http.StatusForbidden)
	}

	recorder = performJSON(router, http.MethodGet, "/conversations/"+conversationID.String()+"/messages?limit=0", "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("bad limit status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandlersRequireAuthenticatedUser(t *testing.T) {
	conversationHandler := NewConversationHandler(&fakeHandlerConversationService{})
	messageHandler := NewMessageHandler(&fakeHandlerMessageService{}, realtime.NewHub(&fakePubSub{}))
	router := testRouter(uuid.Nil)
	router.GET("/conversations", conversationHandler.List)
	router.POST("/conversations/:conversation_id/messages", messageHandler.Create)

	if recorder := performJSON(router, http.MethodGet, "/conversations", ""); recorder.Code != http.StatusUnauthorized {
		t.Fatalf("conversation auth status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if recorder := performJSON(router, http.MethodPost, "/conversations/"+uuid.New().String()+"/messages", `{"content":"hello"}`); recorder.Code != http.StatusUnauthorized {
		t.Fatalf("message auth status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestMessageHandlerInternalError(t *testing.T) {
	handler := NewMessageHandler(&fakeHandlerMessageService{listErr: errors.New("boom")}, realtime.NewHub(&fakePubSub{}))
	router := testRouter(uuid.New())
	router.GET("/conversations/:conversation_id/messages", handler.List)

	recorder := performJSON(router, http.MethodGet, "/conversations/"+uuid.New().String()+"/messages", "")
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}
