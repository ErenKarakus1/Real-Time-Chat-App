package repositories

import (
	"context"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConversationRepository struct {
	db *pgxpool.Pool
}

func NewConversationRepository(db *pgxpool.Pool) *ConversationRepository {
	return &ConversationRepository{
		db: db,
	}
}

func (r *ConversationRepository) CreateRoom(ctx context.Context, conversation models.Conversation, creatorID uuid.UUID) (models.Conversation, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Conversation{}, err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO conversations (id, type, name, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, type, name, created_by, created_at, updated_at
	`

	created, err := scanConversation(tx.QueryRow(
		ctx,
		query,
		conversation.ID,
		models.ConversationTypeRoom,
		conversation.Name,
		creatorID,
	))
	if err != nil {
		return models.Conversation{}, err
	}

	participantQuery := `
		INSERT INTO conversation_participants (conversation_id, user_id, role)
		VALUES ($1, $2, $3)
	`
	if _, err := tx.Exec(ctx, participantQuery, created.ID, creatorID, models.ParticipantRoleOwner); err != nil {
		return models.Conversation{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Conversation{}, err
	}

	return created, nil
}

func (r *ConversationRepository) CreateDirect(ctx context.Context, conversation models.Conversation, userID uuid.UUID, otherUserID uuid.UUID) (models.Conversation, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Conversation{}, err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO conversations (id, type, name, created_by)
		VALUES ($1, $2, NULL, $3)
		RETURNING id, type, name, created_by, created_at, updated_at
	`

	created, err := scanConversation(tx.QueryRow(
		ctx,
		query,
		conversation.ID,
		models.ConversationTypeDirect,
		userID,
	))
	if err != nil {
		return models.Conversation{}, err
	}

	participantQuery := `
		INSERT INTO conversation_participants (conversation_id, user_id, role)
		VALUES ($1, $2, $4), ($1, $3, $4)
	`
	if _, err := tx.Exec(ctx, participantQuery, created.ID, userID, otherUserID, models.ParticipantRoleMember); err != nil {
		return models.Conversation{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Conversation{}, err
	}

	return created, nil
}

func (r *ConversationRepository) FindDirectBetweenUsers(ctx context.Context, userID uuid.UUID, otherUserID uuid.UUID) (models.Conversation, error) {
	query := `
		SELECT c.id, c.type, c.name, c.created_by, c.created_at, c.updated_at
		FROM conversations c
		INNER JOIN conversation_participants cp1 ON cp1.conversation_id = c.id
		INNER JOIN conversation_participants cp2 ON cp2.conversation_id = c.id
		WHERE c.type = $1
			AND cp1.user_id = $2
			AND cp2.user_id = $3
	`

	return scanConversation(r.db.QueryRow(ctx, query, models.ConversationTypeDirect, userID, otherUserID))
}

func (r *ConversationRepository) FindByID(ctx context.Context, conversationID uuid.UUID) (models.Conversation, error) {
	query := `
		SELECT id, type, name, created_by, created_at, updated_at
		FROM conversations
		WHERE id = $1
	`

	return scanConversation(r.db.QueryRow(ctx, query, conversationID))
}

func (r *ConversationRepository) UpdateRoomName(ctx context.Context, conversationID uuid.UUID, name string) (models.Conversation, error) {
	query := `
		UPDATE conversations
		SET name = $2,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, type, name, created_by, created_at, updated_at
	`

	return scanConversation(r.db.QueryRow(ctx, query, conversationID, name))
}

func (r *ConversationRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]models.Conversation, error) {
	query := `
		SELECT c.id, c.type, c.name, c.created_by, c.created_at, c.updated_at
		FROM conversations c
		INNER JOIN conversation_participants cp ON cp.conversation_id = c.id
		WHERE cp.user_id = $1
		ORDER BY c.updated_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversations := make([]models.Conversation, 0)
	for rows.Next() {
		conversation, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}

		conversations = append(conversations, conversation)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return conversations, nil
}

func (r *ConversationRepository) IsParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM conversation_participants
			WHERE conversation_id = $1
				AND user_id = $2
		)
	`

	var exists bool
	if err := r.db.QueryRow(ctx, query, conversationID, userID).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (r *ConversationRepository) FindParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) (models.ConversationParticipant, error) {
	query := `
		SELECT conversation_id, user_id, role, joined_at
		FROM conversation_participants
		WHERE conversation_id = $1
			AND user_id = $2
	`

	return scanConversationParticipant(r.db.QueryRow(ctx, query, conversationID, userID))
}

func (r *ConversationRepository) ListParticipants(ctx context.Context, conversationID uuid.UUID) ([]models.ConversationParticipant, error) {
	query := `
		SELECT conversation_id, user_id, role, joined_at
		FROM conversation_participants
		WHERE conversation_id = $1
		ORDER BY joined_at ASC
	`

	rows, err := r.db.Query(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	participants := make([]models.ConversationParticipant, 0)
	for rows.Next() {
		participant, err := scanConversationParticipant(rows)
		if err != nil {
			return nil, err
		}

		participants = append(participants, participant)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return participants, nil
}

func (r *ConversationRepository) AddParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, role models.ParticipantRole) (models.ConversationParticipant, error) {
	query := `
		INSERT INTO conversation_participants (conversation_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (conversation_id, user_id) DO UPDATE
		SET role = conversation_participants.role
		RETURNING conversation_id, user_id, role, joined_at
	`

	return scanConversationParticipant(r.db.QueryRow(ctx, query, conversationID, userID, role))
}

func (r *ConversationRepository) UpdateParticipantRole(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, role models.ParticipantRole) (models.ConversationParticipant, error) {
	query := `
		UPDATE conversation_participants
		SET role = $3
		WHERE conversation_id = $1
			AND user_id = $2
		RETURNING conversation_id, user_id, role, joined_at
	`

	return scanConversationParticipant(r.db.QueryRow(ctx, query, conversationID, userID, role))
}

func (r *ConversationRepository) RemoveParticipant(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error {
	query := `
		DELETE FROM conversation_participants
		WHERE conversation_id = $1
			AND user_id = $2
	`

	_, err := r.db.Exec(ctx, query, conversationID, userID)
	return err
}

func (r *ConversationRepository) TransferOwnership(ctx context.Context, conversationID uuid.UUID, currentOwnerID uuid.UUID, newOwnerID uuid.UUID) (models.ConversationParticipant, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.ConversationParticipant{}, err
	}
	defer tx.Rollback(ctx)

	demoteQuery := `
		UPDATE conversation_participants
		SET role = $3
		WHERE conversation_id = $1
			AND user_id = $2
	`
	if _, err := tx.Exec(ctx, demoteQuery, conversationID, currentOwnerID, models.ParticipantRoleAdmin); err != nil {
		return models.ConversationParticipant{}, err
	}

	promoteQuery := `
		UPDATE conversation_participants
		SET role = $3
		WHERE conversation_id = $1
			AND user_id = $2
		RETURNING conversation_id, user_id, role, joined_at
	`
	newOwner, err := scanConversationParticipant(tx.QueryRow(ctx, promoteQuery, conversationID, newOwnerID, models.ParticipantRoleOwner))
	if err != nil {
		return models.ConversationParticipant{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.ConversationParticipant{}, err
	}

	return newOwner, nil
}

func scanConversation(row pgx.Row) (models.Conversation, error) {
	var conversation models.Conversation

	err := row.Scan(
		&conversation.ID,
		&conversation.Type,
		&conversation.Name,
		&conversation.CreatedBy,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)

	return conversation, err
}

func scanConversationParticipant(row pgx.Row) (models.ConversationParticipant, error) {
	var participant models.ConversationParticipant

	err := row.Scan(
		&participant.ConversationID,
		&participant.UserID,
		&participant.Role,
		&participant.JoinedAt,
	)

	return participant, err
}
