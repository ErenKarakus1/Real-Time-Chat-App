package models

import (
	"time"

	"github.com/google/uuid"
)

type ConversationType string
type ParticipantRole string

const (
	ConversationTypeRoom   ConversationType = "room"
	ConversationTypeDirect ConversationType = "direct"
)

const (
	ParticipantRoleOwner  ParticipantRole = "owner"
	ParticipantRoleAdmin  ParticipantRole = "admin"
	ParticipantRoleMember ParticipantRole = "member"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Conversation struct {
	ID        uuid.UUID        `json:"id"`
	Type      ConversationType `json:"type"`
	Name      *string          `json:"name"`
	CreatedBy *uuid.UUID       `json:"created_by"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type ConversationParticipant struct {
	ConversationID uuid.UUID       `json:"conversation_id"`
	UserID         uuid.UUID       `json:"user_id"`
	Role           ParticipantRole `json:"role"`
	JoinedAt       time.Time       `json:"joined_at"`
	LastReadAt     *time.Time      `json:"last_read_at"`
}

type Message struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversation_id"`
	SenderID       *uuid.UUID `json:"sender_id"`
	Content        string     `json:"content"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
