package repositories

import (
	"context"
	"time"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const DefaultMessageLimit = 50

type MessageRepository struct {
	db *pgxpool.Pool
}

func NewMessageRepository(db *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{
		db: db,
	}
}

func (r *MessageRepository) Create(ctx context.Context, message models.Message) (models.Message, error) {
	query := `
		INSERT INTO messages (id, conversation_id, sender_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, conversation_id, sender_id, content, created_at, updated_at
	`

	return scanMessage(r.db.QueryRow(ctx, query, message.ID, message.ConversationID, message.SenderID, message.Content))
}

func (r *MessageRepository) ListForConversation(ctx context.Context, conversationID uuid.UUID, before *time.Time, limit int) ([]models.Message, error) {
	if limit <= 0 {
		limit = DefaultMessageLimit
	}

	query := `
		SELECT id, conversation_id, sender_id, content, created_at, updated_at
		FROM messages
		WHERE conversation_id = $1
			AND ($2::timestamptz IS NULL OR created_at < $2)
		ORDER BY created_at DESC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, conversationID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]models.Message, 0)
	for rows.Next() {
		message, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	reverseMessages(messages)
	return messages, nil
}

func scanMessage(row pgx.Row) (models.Message, error) {
	var message models.Message

	err := row.Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderID,
		&message.Content,
		&message.CreatedAt,
		&message.UpdatedAt,
	)

	return message, err
}

func reverseMessages(messages []models.Message) {
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
}
