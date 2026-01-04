package messages_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/unifiedui/agent-service/internal/domain/models"
)

func TestNewUserMessage(t *testing.T) {
	// Arrange
	tenantID := "tenant-123"
	userID := "user-456"
	conversationID := "conv-789"
	applicationID := "app-1"
	content := "Hello, world!"
	attachments := []string{"file1.png"}

	// Act
	msg := models.NewUserMessage(tenantID, conversationID, applicationID, userID, content, attachments, nil)

	// Assert
	assert.Equal(t, tenantID, msg.TenantID)
	assert.Equal(t, userID, msg.UserID)
	assert.Equal(t, conversationID, msg.ConversationID)
	assert.Equal(t, applicationID, msg.ApplicationID)
	assert.Equal(t, content, msg.Content)
	assert.Equal(t, models.MessageTypeUser, msg.Type)
	assert.Equal(t, attachments, msg.Attachments)
	assert.NotZero(t, msg.CreatedAt)
	assert.Equal(t, msg.CreatedAt, msg.UpdatedAt)
}

func TestNewAssistantMessage(t *testing.T) {
	// Arrange
	tenantID := "tenant-123"
	conversationID := "conv-789"
	userMessageID := "user-msg-123"
	applicationID := "app-1"
	content := ""
	status := models.MessageStatusPending

	// Act
	msg := models.NewAssistantMessage(tenantID, conversationID, userMessageID, applicationID, content, status)

	// Assert
	assert.Equal(t, tenantID, msg.TenantID)
	assert.Equal(t, conversationID, msg.ConversationID)
	assert.Equal(t, userMessageID, msg.UserMessageID)
	assert.Equal(t, applicationID, msg.ApplicationID)
	assert.Equal(t, models.MessageStatusPending, msg.Status)
	assert.Equal(t, "", msg.Content)
	assert.NotZero(t, msg.CreatedAt)
}

func TestMessage_IsUserMessage(t *testing.T) {
	// Arrange
	userMsg := models.NewUserMessage("tenant", "conv", "app", "user", "Hello", nil, nil)
	assistantMsg := models.NewAssistantMessage("tenant", "conv", "user-msg", "app", "Hi", models.MessageStatusSuccess)

	// Assert
	assert.True(t, userMsg.IsUserMessage())
	assert.False(t, assistantMsg.IsUserMessage())
}

func TestMessage_IsAssistantMessage(t *testing.T) {
	// Arrange
	userMsg := models.NewUserMessage("tenant", "conv", "app", "user", "Hello", nil, nil)
	assistantMsg := models.NewAssistantMessage("tenant", "conv", "user-msg", "app", "Hi", models.MessageStatusSuccess)

	// Assert
	assert.False(t, userMsg.IsAssistantMessage())
	assert.True(t, assistantMsg.IsAssistantMessage())
}

func TestMessage_SetSuccess(t *testing.T) {
	// Arrange
	msg := models.NewAssistantMessage("tenant", "conv", "user-msg", "app", "", models.MessageStatusPending)
	content := "Final response"

	// Act
	msg.SetSuccess(content)

	// Assert
	assert.Equal(t, models.MessageStatusSuccess, msg.Status)
	assert.Equal(t, content, msg.Content)
	assert.NotZero(t, msg.UpdatedAt)
}

func TestMessage_SetError(t *testing.T) {
	// Arrange
	msg := models.NewAssistantMessage("tenant", "conv", "user-msg", "app", "", models.MessageStatusPending)
	errorMessage := "Something went wrong"

	// Act
	msg.SetError(errorMessage)

	// Assert
	assert.Equal(t, models.MessageStatusFailed, msg.Status)
	assert.Equal(t, errorMessage, msg.ErrorMessage)
	assert.NotZero(t, msg.UpdatedAt)
}

func TestMessage_AddStatusTrace(t *testing.T) {
	// Arrange
	msg := models.NewAssistantMessage("tenant", "conv", "user-msg", "app", "", models.MessageStatusPending)

	// Act
	msg.AddStatusTrace("thinking", "Planning", "Analyzing request...", map[string]interface{}{"step": 1})

	// Assert
	assert.Len(t, msg.StatusTraces, 1)
	assert.Equal(t, "thinking", msg.StatusTraces[0].Type)
	assert.Equal(t, "Planning", msg.StatusTraces[0].Name)
	assert.Equal(t, "Analyzing request...", msg.StatusTraces[0].Content)
	assert.Equal(t, 1, msg.StatusTraces[0].Data["step"])
}

func TestMessage_SetMetadata(t *testing.T) {
	// Arrange
	msg := models.NewAssistantMessage("tenant", "conv", "user-msg", "app", "Response", models.MessageStatusSuccess)
	metadata := &models.AssistantMetadata{
		Model:        "gpt-4",
		TokensInput:  100,
		TokensOutput: 50,
		LatencyMs:    500,
	}

	// Act
	msg.SetMetadata(metadata)

	// Assert
	assert.Equal(t, metadata, msg.Metadata)
	assert.NotZero(t, msg.UpdatedAt)
}

func TestUserMessage_ToChatHistoryEntry(t *testing.T) {
	// Arrange
	msg := models.NewUserMessage("tenant", "conv", "app", "user", "What is AI?", nil, nil)
	msg.CreatedAt = time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	// Act
	entry := msg.ToChatHistoryEntry()

	// Assert
	assert.Equal(t, models.MessageTypeUser, entry.Role)
	assert.Equal(t, "What is AI?", entry.Content)
	assert.Equal(t, msg.CreatedAt, entry.Timestamp)
}

func TestAssistantMessage_ToChatHistoryEntry(t *testing.T) {
	// Arrange
	msg := models.NewAssistantMessage("tenant", "conv", "user-msg", "app", "AI is artificial intelligence.", models.MessageStatusSuccess)
	msg.CreatedAt = time.Date(2024, 1, 15, 10, 31, 0, 0, time.UTC)

	// Act
	entry := msg.ToChatHistoryEntry()

	// Assert
	assert.Equal(t, models.MessageTypeAssistant, entry.Role)
	assert.Equal(t, "AI is artificial intelligence.", entry.Content)
	assert.Equal(t, msg.CreatedAt, entry.Timestamp)
}
