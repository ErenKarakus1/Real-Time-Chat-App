package services

import (
	"context"
	"errors"
	"testing"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type fakeConversationRepository struct {
	conversations map[uuid.UUID]models.Conversation
	participants  map[uuid.UUID]map[uuid.UUID]models.ConversationParticipant

	addedUserID       uuid.UUID
	removedUserID     uuid.UUID
	readUserID        uuid.UUID
	listCalled        bool
	searchCalled      bool
	transferOwnerID   uuid.UUID
	transferNewUserID uuid.UUID
	deletedID         uuid.UUID
	updatedName       string
}

func newFakeConversationRepository() *fakeConversationRepository {
	return &fakeConversationRepository{
		conversations: make(map[uuid.UUID]models.Conversation),
		participants:  make(map[uuid.UUID]map[uuid.UUID]models.ConversationParticipant),
	}
}

func (r *fakeConversationRepository) AddParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, role models.ParticipantRole) (models.ConversationParticipant, error) {
	r.addedUserID = userID
	return models.ConversationParticipant{ConversationID: conversationID, UserID: userID, Role: role}, nil
}

func (r *fakeConversationRepository) CreateDirect(ctx context.Context, conversation models.Conversation, userID uuid.UUID, otherUserID uuid.UUID) (models.Conversation, error) {
	return conversation, nil
}

func (r *fakeConversationRepository) CreateRoom(ctx context.Context, conversation models.Conversation, creatorID uuid.UUID) (models.Conversation, error) {
	return conversation, nil
}

func (r *fakeConversationRepository) DeleteConversation(ctx context.Context, conversationID uuid.UUID) error {
	r.deletedID = conversationID
	return nil
}

func (r *fakeConversationRepository) FindByID(ctx context.Context, conversationID uuid.UUID) (models.Conversation, error) {
	conversation, ok := r.conversations[conversationID]
	if !ok {
		return models.Conversation{}, pgx.ErrNoRows
	}
	return conversation, nil
}

func (r *fakeConversationRepository) FindDirectBetweenUsers(ctx context.Context, userID uuid.UUID, otherUserID uuid.UUID) (models.Conversation, error) {
	return models.Conversation{}, pgx.ErrNoRows
}

func (r *fakeConversationRepository) FindParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (models.ConversationParticipant, error) {
	participants := r.participants[conversationID]
	participant, ok := participants[userID]
	if !ok {
		return models.ConversationParticipant{}, pgx.ErrNoRows
	}
	return participant, nil
}

func (r *fakeConversationRepository) IsParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (bool, error) {
	_, ok := r.participants[conversationID][userID]
	return ok, nil
}

func (r *fakeConversationRepository) ListParticipants(ctx context.Context, conversationID uuid.UUID) ([]models.ConversationParticipant, error) {
	participants := make([]models.ConversationParticipant, 0, len(r.participants[conversationID]))
	for _, participant := range r.participants[conversationID] {
		participants = append(participants, participant)
	}
	return participants, nil
}

func (r *fakeConversationRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]models.ConversationListItem, error) {
	r.listCalled = true
	return []models.ConversationListItem{{UnreadCount: 1}}, nil
}

func (r *fakeConversationRepository) MarkRead(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (models.ConversationParticipant, error) {
	r.readUserID = userID
	return r.participants[conversationID][userID], nil
}

func (r *fakeConversationRepository) RemoveParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error {
	r.removedUserID = userID
	return nil
}

func (r *fakeConversationRepository) SearchRoomsForUser(ctx context.Context, userID uuid.UUID, query string) ([]models.ConversationListItem, error) {
	r.searchCalled = true
	return []models.ConversationListItem{{UnreadCount: 2}}, nil
}

func (r *fakeConversationRepository) TransferOwnership(ctx context.Context, conversationID uuid.UUID, currentOwnerID uuid.UUID, newOwnerID uuid.UUID) (models.ConversationParticipant, error) {
	r.transferOwnerID = currentOwnerID
	r.transferNewUserID = newOwnerID
	return models.ConversationParticipant{ConversationID: conversationID, UserID: newOwnerID, Role: models.ParticipantRoleOwner}, nil
}

func (r *fakeConversationRepository) UpdateRoomName(ctx context.Context, conversationID uuid.UUID, name string) (models.Conversation, error) {
	r.updatedName = name
	conversation := r.conversations[conversationID]
	conversation.Name = &name
	return conversation, nil
}

func (r *fakeConversationRepository) UpdateParticipantRole(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, role models.ParticipantRole) (models.ConversationParticipant, error) {
	return models.ConversationParticipant{ConversationID: conversationID, UserID: userID, Role: role}, nil
}

func seedRoom(t *testing.T, repo *fakeConversationRepository) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()

	conversationID := uuid.New()
	ownerID := uuid.New()
	adminID := uuid.New()
	memberID := uuid.New()
	repo.conversations[conversationID] = models.Conversation{ID: conversationID, Type: models.ConversationTypeRoom}
	repo.participants[conversationID] = map[uuid.UUID]models.ConversationParticipant{
		ownerID:  {ConversationID: conversationID, UserID: ownerID, Role: models.ParticipantRoleOwner},
		adminID:  {ConversationID: conversationID, UserID: adminID, Role: models.ParticipantRoleAdmin},
		memberID: {ConversationID: conversationID, UserID: memberID, Role: models.ParticipantRoleMember},
	}

	return conversationID, ownerID, adminID, memberID
}

func TestConversationServiceRemoveParticipantRoleHierarchy(t *testing.T) {
	repo := newFakeConversationRepository()
	conversationID, _, adminID, memberID := seedRoom(t, repo)
	service := NewConversationService(repo)

	err := service.RemoveParticipant(context.Background(), RemoveParticipantInput{
		ConversationID: conversationID,
		ActorID:        adminID,
		UserID:         memberID,
	})
	if err != nil {
		t.Fatalf("remove member by admin: %v", err)
	}
	if repo.removedUserID != memberID {
		t.Fatalf("removed user = %s, want %s", repo.removedUserID, memberID)
	}

	err = service.RemoveParticipant(context.Background(), RemoveParticipantInput{
		ConversationID: conversationID,
		ActorID:        adminID,
		UserID:         uuid.MustParse(adminID.String()),
	})
	if !errors.Is(err, ErrInvalidParticipant) {
		t.Fatalf("remove self error = %v, want %v", err, ErrInvalidParticipant)
	}
}

func TestConversationServiceAdminCannotRemoveAdmin(t *testing.T) {
	repo := newFakeConversationRepository()
	conversationID, _, adminID, _ := seedRoom(t, repo)
	otherAdminID := uuid.New()
	repo.participants[conversationID][otherAdminID] = models.ConversationParticipant{ConversationID: conversationID, UserID: otherAdminID, Role: models.ParticipantRoleAdmin}
	service := NewConversationService(repo)

	err := service.RemoveParticipant(context.Background(), RemoveParticipantInput{
		ConversationID: conversationID,
		ActorID:        adminID,
		UserID:         otherAdminID,
	})
	if !errors.Is(err, ErrConversationManagementDenied) {
		t.Fatalf("remove admin by admin error = %v, want %v", err, ErrConversationManagementDenied)
	}
}

func TestConversationServiceOwnerActions(t *testing.T) {
	repo := newFakeConversationRepository()
	conversationID, ownerID, _, memberID := seedRoom(t, repo)
	service := NewConversationService(repo)

	if err := service.DeleteRoom(context.Background(), DeleteRoomInput{ConversationID: conversationID, ActorID: memberID}); !errors.Is(err, ErrConversationManagementDenied) {
		t.Fatalf("member delete room error = %v, want %v", err, ErrConversationManagementDenied)
	}
	if err := service.DeleteRoom(context.Background(), DeleteRoomInput{ConversationID: conversationID, ActorID: ownerID}); err != nil {
		t.Fatalf("owner delete room: %v", err)
	}
	if repo.deletedID != conversationID {
		t.Fatalf("deleted room = %s, want %s", repo.deletedID, conversationID)
	}

	if _, err := service.TransferOwnership(context.Background(), TransferOwnershipInput{ConversationID: conversationID, ActorID: memberID, UserID: ownerID}); !errors.Is(err, ErrConversationManagementDenied) {
		t.Fatalf("non-owner transfer error = %v, want %v", err, ErrConversationManagementDenied)
	}
	if _, err := service.TransferOwnership(context.Background(), TransferOwnershipInput{ConversationID: conversationID, ActorID: ownerID, UserID: memberID}); err != nil {
		t.Fatalf("transfer ownership: %v", err)
	}
	if repo.transferOwnerID != ownerID || repo.transferNewUserID != memberID {
		t.Fatalf("transfer owner/new = %s/%s, want %s/%s", repo.transferOwnerID, repo.transferNewUserID, ownerID, memberID)
	}
}

func TestConversationServiceLeaveRoomAndReadTracking(t *testing.T) {
	repo := newFakeConversationRepository()
	conversationID, ownerID, _, memberID := seedRoom(t, repo)
	service := NewConversationService(repo)

	if err := service.LeaveRoom(context.Background(), LeaveRoomInput{ConversationID: conversationID, UserID: ownerID}); !errors.Is(err, ErrConversationManagementDenied) {
		t.Fatalf("owner leave error = %v, want %v", err, ErrConversationManagementDenied)
	}
	if err := service.LeaveRoom(context.Background(), LeaveRoomInput{ConversationID: conversationID, UserID: memberID}); err != nil {
		t.Fatalf("member leave: %v", err)
	}
	if repo.removedUserID != memberID {
		t.Fatalf("leave removed user = %s, want %s", repo.removedUserID, memberID)
	}

	if _, err := service.MarkRead(context.Background(), MarkConversationReadInput{ConversationID: conversationID, UserID: uuid.New()}); !errors.Is(err, ErrConversationManagementDenied) {
		t.Fatalf("non-participant mark read error = %v, want %v", err, ErrConversationManagementDenied)
	}
	if _, err := service.MarkRead(context.Background(), MarkConversationReadInput{ConversationID: conversationID, UserID: memberID}); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if repo.readUserID != memberID {
		t.Fatalf("read user = %s, want %s", repo.readUserID, memberID)
	}
}

func TestConversationServiceSearchForUserFallback(t *testing.T) {
	repo := newFakeConversationRepository()
	service := NewConversationService(repo)
	userID := uuid.New()

	if _, err := service.SearchForUser(context.Background(), userID, "   "); err != nil {
		t.Fatalf("empty search: %v", err)
	}
	if !repo.listCalled || repo.searchCalled {
		t.Fatalf("empty search list/search called = %v/%v, want true/false", repo.listCalled, repo.searchCalled)
	}

	repo.listCalled = false
	if _, err := service.SearchForUser(context.Background(), userID, "dev"); err != nil {
		t.Fatalf("query search: %v", err)
	}
	if repo.listCalled || !repo.searchCalled {
		t.Fatalf("query search list/search called = %v/%v, want false/true", repo.listCalled, repo.searchCalled)
	}
}
