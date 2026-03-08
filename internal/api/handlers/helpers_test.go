package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/unifiedui/agent-service/internal/domain/models"
)

// --- generateMessageID / generateConversationID ---

func TestGenerateMessageID(t *testing.T) {
	id := generateMessageID()
	assert.True(t, len(id) > 4)
	assert.Equal(t, "msg_", id[:4])

	id2 := generateMessageID()
	assert.NotEqual(t, id, id2)
}

func TestGenerateConversationID(t *testing.T) {
	id := generateConversationID()
	assert.True(t, len(id) > 5)
	assert.Equal(t, "conv_", id[:5])

	id2 := generateConversationID()
	assert.NotEqual(t, id, id2)
}

// --- extractBaseURL ---

func TestExtractBaseURL_EmptyString(t *testing.T) {
	assert.Equal(t, "", extractBaseURL(""))
}

func TestExtractBaseURL_HTTPS(t *testing.T) {
	assert.Equal(t, "https://n8n.example.com", extractBaseURL("https://n8n.example.com/webhook/chat"))
}

func TestExtractBaseURL_HTTP(t *testing.T) {
	assert.Equal(t, "http://localhost:5678", extractBaseURL("http://localhost:5678/webhook/chat"))
}

func TestExtractBaseURL_NoPath(t *testing.T) {
	assert.Equal(t, "https://example.com", extractBaseURL("https://example.com"))
}

func TestExtractBaseURL_NoProtocol(t *testing.T) {
	result := extractBaseURL("example.com/path")
	assert.Equal(t, "example.com", result)
}

func TestExtractBaseURL_JustDomain(t *testing.T) {
	assert.Equal(t, "example.com", extractBaseURL("example.com"))
}

// --- getStringFromMap ---

func TestGetStringFromMap_Found(t *testing.T) {
	m := map[string]interface{}{"key": "value"}
	assert.Equal(t, "value", getStringFromMap(m, "key"))
}

func TestGetStringFromMap_NotFound(t *testing.T) {
	m := map[string]interface{}{}
	assert.Equal(t, "", getStringFromMap(m, "key"))
}

func TestGetStringFromMap_WrongType(t *testing.T) {
	m := map[string]interface{}{"key": 123}
	assert.Equal(t, "", getStringFromMap(m, "key"))
}

// --- extractFoundryMetadata ---

func TestExtractFoundryMetadata_WithMessageID(t *testing.T) {
	h := &MessagesHandler{}
	metadata := map[string]interface{}{"message_id": "mid-123"}
	result := h.extractFoundryMetadata(metadata)
	assert.Equal(t, "mid-123", result.ExecutionID)
	assert.Equal(t, "mid-123", result.ExtMessageID)
}

func TestExtractFoundryMetadata_Empty(t *testing.T) {
	h := &MessagesHandler{}
	result := h.extractFoundryMetadata(map[string]interface{}{})
	assert.Equal(t, "", result.ExecutionID)
}

// --- mergeFoundryMetadata ---

func TestMergeFoundryMetadata_AllFields(t *testing.T) {
	h := &MessagesHandler{}
	msg := &models.Message{}
	metadata := map[string]interface{}{
		"response_id": "resp-1",
		"message_id":  "msg-1",
		"model":       "gpt-4",
		"agent_name":  "my-agent",
		"usage": map[string]interface{}{
			"input_tokens":  10,
			"output_tokens": 20,
		},
	}
	h.mergeFoundryMetadata(msg, metadata)
	assert.NotNil(t, msg.Metadata)
	assert.Equal(t, "resp-1", msg.Metadata.ExecutionID)
	assert.Equal(t, "msg-1", msg.Metadata.ExtMessageID)
	assert.Equal(t, "gpt-4", msg.Metadata.Model)
	assert.Equal(t, "my-agent", msg.Metadata.AgentType)
	assert.Equal(t, 10, msg.Metadata.TokensInput)
	assert.Equal(t, 20, msg.Metadata.TokensOutput)
}

func TestMergeFoundryMetadata_ExistingMetadata(t *testing.T) {
	h := &MessagesHandler{}
	msg := &models.Message{
		Metadata: &models.AssistantMetadata{ExecutionID: "existing"},
	}
	h.mergeFoundryMetadata(msg, map[string]interface{}{"response_id": "new"})
	assert.Equal(t, "existing", msg.Metadata.ExecutionID)
}

func TestMergeFoundryMetadata_WorkflowAction(t *testing.T) {
	h := &MessagesHandler{}
	msg := &models.Message{}
	metadata := map[string]interface{}{
		"type":               "workflow_action",
		"kind":               "execute",
		"action_id":          "a1",
		"parent_action_id":   "p1",
		"previous_action_id": "prev1",
		"status":             "completed",
	}
	h.mergeFoundryMetadata(msg, metadata)
	assert.NotNil(t, msg.Metadata)
	assert.Len(t, msg.StatusTraces, 1)
	assert.Equal(t, "workflow_action", msg.StatusTraces[0].Type)
}

// --- convertFilesToFileInputs ---

func TestConvertFilesToFileInputs_Empty(t *testing.T) {
	result := convertFilesToFileInputs(nil)
	assert.Nil(t, result)
}

func TestConvertFilesToFileInputs_EmptySlice(t *testing.T) {
	result := convertFilesToFileInputs([]FileAttachment{})
	assert.Nil(t, result)
}

func TestConvertFilesToFileInputs_WithFiles(t *testing.T) {
	files := []FileAttachment{
		{Type: "image", ImageURL: "data:image/png;base64,abc", Filename: "test.png", MimeType: "image/png", Detail: "low"},
		{Type: "file", FileData: "base64data", Filename: "doc.pdf", MimeType: "application/pdf"},
	}
	result := convertFilesToFileInputs(files)
	assert.Len(t, result, 2)
	assert.Equal(t, "image", result[0].Type)
	assert.Equal(t, "data:image/png;base64,abc", result[0].ImageURL)
	assert.Equal(t, "low", result[0].Detail)
	assert.Equal(t, "file", result[1].Type)
	assert.Equal(t, "base64data", result[1].FileData)
}

// --- toMessageResponse ---

func TestToMessageResponse(t *testing.T) {
	h := &MessagesHandler{}
	msg := &models.Message{
		ID:             "msg-1",
		Type:           models.MessageTypeUser,
		ConversationID: "conv-1",
		ChatAgentID:    "agent-1",
		Content:        "Hello",
		UserID:         "user-1",
	}
	resp := h.toMessageResponse(msg)
	assert.Equal(t, "msg-1", resp.ID)
	assert.Equal(t, models.MessageTypeUser, resp.Type)
	assert.Equal(t, "conv-1", resp.ConversationID)
	assert.Equal(t, "Hello", resp.Content)
}
