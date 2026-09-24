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
		INSERT INTO conversation_participants (conversation_id, user_id)
		VALUES ($1, $2)
	`
	if _, err := tx.Exec(ctx, participantQuery, created.ID, creatorID); err != nil {
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
		INSERT INTO conversation_participants (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)
	`
	if _, err := tx.Exec(ctx, participantQuery, created.ID, userID, otherUserID); err != nil {
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
