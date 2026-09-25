package models

import (
	"time"

	"github.com/google/uuid"
)

type ParticipantResponse struct {
	ConversationID uuid.UUID       `json:"conversation_id"`
	UserID         uuid.UUID       `json:"user_id"`
	Role           ParticipantRole `json:"role"`
	JoinedAt       time.Time       `json:"joined_at"`
}

func NewParticipantResponse(participant ConversationParticipant) ParticipantResponse {
	return ParticipantResponse{
		ConversationID: participant.ConversationID,
		UserID:         participant.UserID,
		Role:           participant.Role,
		JoinedAt:       participant.JoinedAt,
	}
}
