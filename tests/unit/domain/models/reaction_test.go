package models

import (
	"testing"

	"github.com/unifiedui/agent-service/internal/domain/models"

	"github.com/stretchr/testify/require"
)

func TestNewMessageReaction(t *testing.T) {
	r := models.NewMessageReaction("t1", "c1", "m1", "u1", models.ReactionThumbsUp, "great!")

	require.Equal(t, "t1", r.TenantID)
	require.Equal(t, "c1", r.ConversationID)
	require.Equal(t, "m1", r.MessageID)
	require.Equal(t, "u1", r.UserID)
	require.Equal(t, models.ReactionThumbsUp, r.Reaction)
	require.Equal(t, "great!", r.FeedbackText)
	require.False(t, r.CreatedAt.IsZero())
}

func TestIsValidReactionType(t *testing.T) {
	require.True(t, models.IsValidReactionType(models.ReactionThumbsUp))
	require.True(t, models.IsValidReactionType(models.ReactionThumbsDown))
	require.False(t, models.IsValidReactionType(models.ReactionType("invalid")))
	require.False(t, models.IsValidReactionType(models.ReactionType("")))
}
