package models

import (
	"testing"

	"github.com/unifiedui/agent-service/internal/domain/models"

	"github.com/stretchr/testify/require"
)

func TestNewUserMessage(t *testing.T) {
	req := &models.MessageRequest{
		ChatAgentID: "agent-1",
		Message:     models.MessageRequestContent{Content: "hello"},
	}
	msg := models.NewUserMessage("t1", "c1", "a1", "u1", "hello", []string{"att1"}, req)

	require.Equal(t, models.MessageTypeUser, msg.Type)
	require.Equal(t, "t1", msg.TenantID)
	require.Equal(t, "c1", msg.ConversationID)
	require.Equal(t, "a1", msg.ChatAgentID)
	require.Equal(t, "u1", msg.UserID)
	require.Equal(t, "hello", msg.Content)
	require.Equal(t, []string{"att1"}, msg.Attachments)
	require.Equal(t, req, msg.Request)
	require.False(t, msg.CreatedAt.IsZero())
}

func TestNewAssistantMessage(t *testing.T) {
	msg := models.NewAssistantMessage("t1", "c1", "umsg-1", "a1", "", models.MessageStatusPending)

	require.Equal(t, models.MessageTypeAssistant, msg.Type)
	require.Equal(t, "c1", msg.ConversationID)
	require.Equal(t, "umsg-1", msg.UserMessageID)
	require.Equal(t, "a1", msg.ChatAgentID)
	require.Equal(t, "t1", msg.TenantID)
	require.Equal(t, models.MessageStatusPending, msg.Status)
	require.NotNil(t, msg.StatusTraces)
}

func TestMessage_IsUserMessage(t *testing.T) {
	msg := models.NewUserMessage("t", "c", "a", "u", "hi", nil, nil)
	require.True(t, msg.IsUserMessage())
	require.False(t, msg.IsAssistantMessage())
}

func TestMessage_IsAssistantMessage(t *testing.T) {
	msg := models.NewAssistantMessage("t", "c", "um", "a", "resp", models.MessageStatusSuccess)
	require.True(t, msg.IsAssistantMessage())
	require.False(t, msg.IsUserMessage())
}

func TestMessage_SetError(t *testing.T) {
	msg := models.NewAssistantMessage("t", "c", "um", "a", "", models.MessageStatusPending)
	msg.SetError("something broke")

	require.Equal(t, models.MessageStatusFailed, msg.Status)
	require.Equal(t, "something broke", msg.ErrorMessage)
}

func TestMessage_SetCanceled(t *testing.T) {
	msg := models.NewAssistantMessage("t", "c", "um", "a", "partial", models.MessageStatusPending)
	msg.SetCanceled("partial content")

	require.Equal(t, models.MessageStatusCanceled, msg.Status)
	require.Equal(t, "partial content", msg.Content)
}

func TestMessage_SetSuccess(t *testing.T) {
	msg := models.NewAssistantMessage("t", "c", "um", "a", "", models.MessageStatusPending)
	msg.SetSuccess("full response")

	require.Equal(t, models.MessageStatusSuccess, msg.Status)
	require.Equal(t, "full response", msg.Content)
}

func TestMessage_AddStatusTrace(t *testing.T) {
	msg := models.NewAssistantMessage("t", "c", "um", "a", "", models.MessageStatusPending)
	msg.AddStatusTrace("trace-type", "trace-name", "trace-content", map[string]interface{}{"k": "v"})

	require.Len(t, msg.StatusTraces, 1)
	require.Equal(t, "trace-type", msg.StatusTraces[0].Type)
	require.Equal(t, "trace-name", msg.StatusTraces[0].Name)
	require.Equal(t, "trace-content", msg.StatusTraces[0].Content)
	require.False(t, msg.StatusTraces[0].Timestamp.IsZero())
}

func TestMessage_SetMetadata(t *testing.T) {
	msg := models.NewAssistantMessage("t", "c", "um", "a", "", models.MessageStatusPending)
	meta := &models.AssistantMetadata{Model: "gpt-4", TokensInput: 100}
	msg.SetMetadata(meta)

	require.Equal(t, meta, msg.Metadata)
}

func TestMessage_ToChatHistoryEntry(t *testing.T) {
	msg := models.NewUserMessage("t", "c", "a", "u", "hello", nil, nil)
	entry := msg.ToChatHistoryEntry()

	require.Equal(t, models.MessageTypeUser, entry.Role)
	require.Equal(t, "hello", entry.Content)
	require.Equal(t, msg.CreatedAt, entry.Timestamp)
}
