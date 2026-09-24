package models

import (
	"time"

	"github.com/google/uuid"
)

type ConversationResponse struct {
	ID        uuid.UUID        `json:"id"`
	Type      ConversationType `json:"type"`
	Name      *string          `json:"name"`
	CreatedBy *uuid.UUID       `json:"created_by"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

func NewConversationResponse(conversation Conversation) ConversationResponse {
	return ConversationResponse{
		ID:        conversation.ID,
		Type:      conversation.Type,
		Name:      conversation.Name,
		CreatedBy: conversation.CreatedBy,
		CreatedAt: conversation.CreatedAt,
		UpdatedAt: conversation.UpdatedAt,
	}
}

func NewConversationResponses(conversations []Conversation) []ConversationResponse {
	responses := make([]ConversationResponse, 0, len(conversations))
	for _, conversation := range conversations {
		responses = append(responses, NewConversationResponse(conversation))
	}

	return responses
}
