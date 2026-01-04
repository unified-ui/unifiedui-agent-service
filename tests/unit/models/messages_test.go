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
	assert.Equal(t, content, msg.Message.Content)
	assert.Equal(t, models.MessageTypeUser, msg.Message.Type)
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
	assert.Equal(t, "", msg.Message.Content)
	assert.NotZero(t, msg.CreatedAt)
}

func TestAssistantMessage_SetSuccess(t *testing.T) {
	// Arrange
	msg := models.NewAssistantMessage("tenant", "conv", "user-msg", "app", "", models.MessageStatusPending)
	content := "Final response"

	// Act
	msg.SetSuccess(content)

	// Assert
	assert.Equal(t, models.MessageStatusSuccess, msg.Status)
	assert.Equal(t, content, msg.Message.Content)
	assert.NotZero(t, msg.UpdatedAt)
}

func TestAssistantMessage_SetError(t *testing.T) {
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

func TestAssistantMessage_AddStatusTrace(t *testing.T) {
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

func TestIsUserMessage(t *testing.T) {
	assert.True(t, models.IsUserMessage(models.MessageTypeUser))
	assert.False(t, models.IsUserMessage(models.MessageTypeAssistant))
}

func TestIsAssistantMessage(t *testing.T) {
	assert.True(t, models.IsAssistantMessage(models.MessageTypeAssistant))
	assert.False(t, models.IsAssistantMessage(models.MessageTypeUser))
}
