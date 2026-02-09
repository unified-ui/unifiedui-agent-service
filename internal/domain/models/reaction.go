// Package models contains domain models for the UnifiedUI Chat Service.
package models

import (
	"time"
)

// ReactionType represents the type of reaction.
type ReactionType string

const (
	// ReactionThumbsUp represents a positive reaction.
	ReactionThumbsUp ReactionType = "thumbs_up"
	// ReactionThumbsDown represents a negative reaction.
	ReactionThumbsDown ReactionType = "thumbs_down"
)

// MessageReaction represents a user's reaction to an assistant message.
type MessageReaction struct {
	ID             string       `json:"id" bson:"_id"`
	TenantID       string       `json:"tenantId" bson:"tenantId"`
	ConversationID string       `json:"conversationId" bson:"conversationId"`
	MessageID      string       `json:"messageId" bson:"messageId"`
	UserID         string       `json:"userId" bson:"userId"`
	Reaction       ReactionType `json:"reaction" bson:"reaction"`
	FeedbackText   string       `json:"feedbackText,omitempty" bson:"feedbackText,omitempty"`
	CreatedAt      time.Time    `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time    `json:"updatedAt" bson:"updatedAt"`
}

// NewMessageReaction creates a new MessageReaction.
func NewMessageReaction(
	tenantID string,
	conversationID string,
	messageID string,
	userID string,
	reaction ReactionType,
	feedbackText string,
) *MessageReaction {
	now := time.Now().UTC()
	return &MessageReaction{
		TenantID:       tenantID,
		ConversationID: conversationID,
		MessageID:      messageID,
		UserID:         userID,
		Reaction:       reaction,
		FeedbackText:   feedbackText,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IsValidReactionType checks if the given reaction type is valid.
func IsValidReactionType(rt ReactionType) bool {
	return rt == ReactionThumbsUp || rt == ReactionThumbsDown
}
