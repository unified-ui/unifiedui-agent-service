package agents

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/agents/foundry"
	"github.com/unifiedui/agent-service/internal/services/agents/n8n"
)

// =============================================================================
// toN8NFileInputs Tests
// =============================================================================

func TestToN8NFileInputs_EmptySlice(t *testing.T) {
	result := toN8NFileInputs([]FileInput{})

	assert.NotNil(t, result)
	assert.Empty(t, result)
	assert.Len(t, result, 0)
}

func TestToN8NFileInputs_SingleFile(t *testing.T) {
	input := []FileInput{
		{
			Type:     "image",
			ImageURL: "data:image/png;base64,iVBORw0KGgo=",
			FileData: "",
			Filename: "test.png",
			MimeType: "image/png",
			Detail:   "auto",
		},
	}

	result := toN8NFileInputs(input)

	require.Len(t, result, 1)
	assert.Equal(t, "image", result[0].Type)
	assert.Equal(t, "data:image/png;base64,iVBORw0KGgo=", result[0].ImageURL)
	assert.Empty(t, result[0].FileData)
	assert.Equal(t, "test.png", result[0].Filename)
	assert.Equal(t, "image/png", result[0].MimeType)
	assert.Equal(t, "auto", result[0].Detail)
}

func TestToN8NFileInputs_MultipleFiles(t *testing.T) {
	input := []FileInput{
		{
			Type:     "image",
			ImageURL: "https://example.com/image.jpg",
			FileData: "",
			Filename: "external.jpg",
			MimeType: "image/jpeg",
			Detail:   "high",
		},
		{
			Type:     "file",
			ImageURL: "",
			FileData: "SGVsbG8gV29ybGQ=",
			Filename: "document.pdf",
			MimeType: "application/pdf",
			Detail:   "",
		},
		{
			Type:     "audio",
			ImageURL: "",
			FileData: "YXVkaW9kYXRh",
			Filename: "recording.mp3",
			MimeType: "audio/mpeg",
			Detail:   "",
		},
	}

	result := toN8NFileInputs(input)

	require.Len(t, result, 3)

	// First file - image
	assert.Equal(t, "image", result[0].Type)
	assert.Equal(t, "https://example.com/image.jpg", result[0].ImageURL)
	assert.Empty(t, result[0].FileData)
	assert.Equal(t, "external.jpg", result[0].Filename)
	assert.Equal(t, "image/jpeg", result[0].MimeType)
	assert.Equal(t, "high", result[0].Detail)

	// Second file - document
	assert.Equal(t, "file", result[1].Type)
	assert.Empty(t, result[1].ImageURL)
	assert.Equal(t, "SGVsbG8gV29ybGQ=", result[1].FileData)
	assert.Equal(t, "document.pdf", result[1].Filename)
	assert.Equal(t, "application/pdf", result[1].MimeType)
	assert.Empty(t, result[1].Detail)

	// Third file - audio
	assert.Equal(t, "audio", result[2].Type)
	assert.Empty(t, result[2].ImageURL)
	assert.Equal(t, "YXVkaW9kYXRh", result[2].FileData)
	assert.Equal(t, "recording.mp3", result[2].Filename)
	assert.Equal(t, "audio/mpeg", result[2].MimeType)
	assert.Empty(t, result[2].Detail)
}

func TestToN8NFileInputs_PreservesNilSlice(t *testing.T) {
	// When passing nil, toN8NFileInputs creates an empty slice (not nil)
	result := toN8NFileInputs(nil)

	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

// =============================================================================
// toFoundryFileInputs Tests
// =============================================================================

func TestToFoundryFileInputs_EmptySlice(t *testing.T) {
	result := toFoundryFileInputs([]FileInput{})

	assert.NotNil(t, result)
	assert.Empty(t, result)
	assert.Len(t, result, 0)
}

func TestToFoundryFileInputs_SingleFile(t *testing.T) {
	input := []FileInput{
		{
			Type:     "image",
			ImageURL: "data:image/png;base64,iVBORw0KGgo=",
			FileData: "",
			Filename: "test.png",
			MimeType: "image/png",
			Detail:   "low",
		},
	}

	result := toFoundryFileInputs(input)

	require.Len(t, result, 1)
	assert.Equal(t, "image", result[0].Type)
	assert.Equal(t, "data:image/png;base64,iVBORw0KGgo=", result[0].ImageURL)
	assert.Empty(t, result[0].FileData)
	assert.Equal(t, "test.png", result[0].Filename)
	assert.Equal(t, "image/png", result[0].MimeType)
	assert.Equal(t, "low", result[0].Detail)
}

func TestToFoundryFileInputs_MultipleFiles(t *testing.T) {
	input := []FileInput{
		{
			Type:     "image",
			ImageURL: "https://example.com/photo.png",
			FileData: "",
			Filename: "photo.png",
			MimeType: "image/png",
			Detail:   "auto",
		},
		{
			Type:     "file",
			ImageURL: "",
			FileData: "cGRmY29udGVudA==",
			Filename: "report.pdf",
			MimeType: "application/pdf",
			Detail:   "",
		},
		{
			Type:     "audio",
			ImageURL: "",
			FileData: "d2F2ZGF0YQ==",
			Filename: "voice.wav",
			MimeType: "audio/wav",
			Detail:   "",
		},
	}

	result := toFoundryFileInputs(input)

	require.Len(t, result, 3)

	// First file - image
	assert.Equal(t, "image", result[0].Type)
	assert.Equal(t, "https://example.com/photo.png", result[0].ImageURL)
	assert.Empty(t, result[0].FileData)
	assert.Equal(t, "photo.png", result[0].Filename)
	assert.Equal(t, "image/png", result[0].MimeType)
	assert.Equal(t, "auto", result[0].Detail)

	// Second file - document
	assert.Equal(t, "file", result[1].Type)
	assert.Empty(t, result[1].ImageURL)
	assert.Equal(t, "cGRmY29udGVudA==", result[1].FileData)
	assert.Equal(t, "report.pdf", result[1].Filename)
	assert.Equal(t, "application/pdf", result[1].MimeType)
	assert.Empty(t, result[1].Detail)

	// Third file - audio
	assert.Equal(t, "audio", result[2].Type)
	assert.Empty(t, result[2].ImageURL)
	assert.Equal(t, "d2F2ZGF0YQ==", result[2].FileData)
	assert.Equal(t, "voice.wav", result[2].Filename)
	assert.Equal(t, "audio/wav", result[2].MimeType)
	assert.Empty(t, result[2].Detail)
}

func TestToFoundryFileInputs_PreservesNilSlice(t *testing.T) {
	// When passing nil, toFoundryFileInputs creates an empty slice (not nil)
	result := toFoundryFileInputs(nil)

	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

// =============================================================================
// convertN8NChunk Tests
// =============================================================================

func TestConvertN8NChunk_ValidContentChunk(t *testing.T) {
	n8nChunk := &n8n.StreamChunk{
		Type:        n8n.ChunkTypeContent,
		Content:     "Hello, world!",
		ExecutionID: "exec-123",
		Metadata:    map[string]interface{}{"key": "value"},
		Error:       nil,
	}

	result := convertN8NChunk(n8nChunk)

	require.NotNil(t, result)
	assert.Equal(t, ChunkTypeContent, result.Type)
	assert.Equal(t, "Hello, world!", result.Content)
	assert.Equal(t, "exec-123", result.ExecutionID)
	assert.Equal(t, "value", result.Metadata["key"])
	assert.Nil(t, result.Error)
}

func TestConvertN8NChunk_MetadataChunk(t *testing.T) {
	metadata := map[string]interface{}{
		"model":       "gpt-4",
		"temperature": 0.7,
	}
	n8nChunk := &n8n.StreamChunk{
		Type:        n8n.ChunkTypeMetadata,
		Content:     "",
		ExecutionID: "exec-456",
		Metadata:    metadata,
		Error:       nil,
	}

	result := convertN8NChunk(n8nChunk)

	require.NotNil(t, result)
	assert.Equal(t, ChunkTypeMetadata, result.Type)
	assert.Empty(t, result.Content)
	assert.Equal(t, "exec-456", result.ExecutionID)
	assert.Equal(t, "gpt-4", result.Metadata["model"])
	assert.Equal(t, 0.7, result.Metadata["temperature"])
	assert.Nil(t, result.Error)
}

func TestConvertN8NChunk_ErrorChunk(t *testing.T) {
	testError := errors.New("something went wrong")
	n8nChunk := &n8n.StreamChunk{
		Type:        n8n.ChunkTypeError,
		Content:     "",
		ExecutionID: "exec-789",
		Metadata:    nil,
		Error:       testError,
	}

	result := convertN8NChunk(n8nChunk)

	require.NotNil(t, result)
	assert.Equal(t, ChunkTypeError, result.Type)
	assert.Empty(t, result.Content)
	assert.Equal(t, "exec-789", result.ExecutionID)
	assert.Nil(t, result.Metadata)
	assert.Equal(t, testError, result.Error)
	assert.EqualError(t, result.Error, "something went wrong")
}

func TestConvertN8NChunk_FinalChunk(t *testing.T) {
	// Done/EOF chunk indicates end of stream
	n8nChunk := &n8n.StreamChunk{
		Type:        n8n.ChunkTypeDone,
		Content:     "",
		ExecutionID: "exec-final",
		Metadata:    map[string]interface{}{"total_tokens": 150},
		Error:       nil,
	}

	result := convertN8NChunk(n8nChunk)

	require.NotNil(t, result)
	assert.Equal(t, ChunkTypeDone, result.Type)
	assert.Empty(t, result.Content)
	assert.Equal(t, "exec-final", result.ExecutionID)
	assert.Equal(t, 150, result.Metadata["total_tokens"])
	assert.Nil(t, result.Error)
}

func TestConvertN8NChunk_EmptyChunk(t *testing.T) {
	n8nChunk := &n8n.StreamChunk{
		Type:        n8n.ChunkTypeContent,
		Content:     "",
		ExecutionID: "",
		Metadata:    nil,
		Error:       nil,
	}

	result := convertN8NChunk(n8nChunk)

	require.NotNil(t, result)
	assert.Equal(t, ChunkTypeContent, result.Type)
	assert.Empty(t, result.Content)
	assert.Empty(t, result.ExecutionID)
	assert.Nil(t, result.Metadata)
	assert.Nil(t, result.Error)
}

// =============================================================================
// convertFoundryChunk Tests
// =============================================================================

func TestConvertFoundryChunk_ValidContentChunk(t *testing.T) {
	foundryChunk := &foundry.StreamChunk{
		Type:        foundry.ChunkTypeContent,
		Content:     "Greetings from Foundry!",
		ExecutionID: "foundry-exec-123",
		Metadata:    map[string]interface{}{"source": "foundry"},
		Error:       nil,
	}

	result := convertFoundryChunk(foundryChunk)

	require.NotNil(t, result)
	assert.Equal(t, ChunkTypeContent, result.Type)
	assert.Equal(t, "Greetings from Foundry!", result.Content)
	assert.Equal(t, "foundry-exec-123", result.ExecutionID)
	assert.Equal(t, "foundry", result.Metadata["source"])
	assert.Nil(t, result.Error)
}

func TestConvertFoundryChunk_MetadataChunk(t *testing.T) {
	metadata := map[string]interface{}{
		"agent_name": "test-agent",
		"version":    "1.0.0",
	}
	foundryChunk := &foundry.StreamChunk{
		Type:        foundry.ChunkTypeMetadata,
		Content:     "",
		ExecutionID: "foundry-exec-456",
		Metadata:    metadata,
		Error:       nil,
	}

	result := convertFoundryChunk(foundryChunk)

	require.NotNil(t, result)
	assert.Equal(t, ChunkTypeMetadata, result.Type)
	assert.Empty(t, result.Content)
	assert.Equal(t, "foundry-exec-456", result.ExecutionID)
	assert.Equal(t, "test-agent", result.Metadata["agent_name"])
	assert.Equal(t, "1.0.0", result.Metadata["version"])
	assert.Nil(t, result.Error)
}

func TestConvertFoundryChunk_ErrorChunk(t *testing.T) {
	testError := errors.New("foundry service unavailable")
	foundryChunk := &foundry.StreamChunk{
		Type:        foundry.ChunkTypeError,
		Content:     "",
		ExecutionID: "foundry-exec-789",
		Metadata:    nil,
		Error:       testError,
	}

	result := convertFoundryChunk(foundryChunk)

	require.NotNil(t, result)
	assert.Equal(t, ChunkTypeError, result.Type)
	assert.Empty(t, result.Content)
	assert.Equal(t, "foundry-exec-789", result.ExecutionID)
	assert.Nil(t, result.Metadata)
	assert.Equal(t, testError, result.Error)
	assert.EqualError(t, result.Error, "foundry service unavailable")
}

func TestConvertFoundryChunk_FinalChunk(t *testing.T) {
	// Done/EOF chunk indicates end of stream
	foundryChunk := &foundry.StreamChunk{
		Type:        foundry.ChunkTypeDone,
		Content:     "",
		ExecutionID: "foundry-exec-final",
		Metadata:    map[string]interface{}{"execution_time_ms": 1500},
		Error:       nil,
	}

	result := convertFoundryChunk(foundryChunk)

	require.NotNil(t, result)
	assert.Equal(t, ChunkTypeDone, result.Type)
	assert.Empty(t, result.Content)
	assert.Equal(t, "foundry-exec-final", result.ExecutionID)
	assert.Equal(t, 1500, result.Metadata["execution_time_ms"])
	assert.Nil(t, result.Error)
}

func TestConvertFoundryChunk_NewMessageChunk(t *testing.T) {
	// Foundry supports new_message chunk type
	foundryChunk := &foundry.StreamChunk{
		Type:        foundry.ChunkTypeNewMessage,
		Content:     "",
		ExecutionID: "foundry-exec-new",
		Metadata:    map[string]interface{}{"message_id": "msg-001"},
		Error:       nil,
	}

	result := convertFoundryChunk(foundryChunk)

	require.NotNil(t, result)
	assert.Equal(t, ChunkTypeNewMessage, result.Type)
	assert.Empty(t, result.Content)
	assert.Equal(t, "foundry-exec-new", result.ExecutionID)
	assert.Equal(t, "msg-001", result.Metadata["message_id"])
	assert.Nil(t, result.Error)
}

func TestConvertFoundryChunk_EmptyChunk(t *testing.T) {
	foundryChunk := &foundry.StreamChunk{
		Type:        foundry.ChunkTypeContent,
		Content:     "",
		ExecutionID: "",
		Metadata:    nil,
		Error:       nil,
	}

	result := convertFoundryChunk(foundryChunk)

	require.NotNil(t, result)
	assert.Equal(t, ChunkTypeContent, result.Type)
	assert.Empty(t, result.Content)
	assert.Empty(t, result.ExecutionID)
	assert.Nil(t, result.Metadata)
	assert.Nil(t, result.Error)
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestToN8NFileInputs_LargeFileData(t *testing.T) {
	// Test with large file data (simulating a larger file)
	largeData := make([]byte, 1024*100) // 100KB of data
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	input := []FileInput{
		{
			Type:     "file",
			FileData: string(largeData),
			Filename: "large_file.bin",
			MimeType: "application/octet-stream",
		},
	}

	result := toN8NFileInputs(input)

	require.Len(t, result, 1)
	assert.Equal(t, len(largeData), len(result[0].FileData))
	assert.Equal(t, "large_file.bin", result[0].Filename)
}

func TestToFoundryFileInputs_SpecialCharacters(t *testing.T) {
	// Test with special characters in filename and metadata
	input := []FileInput{
		{
			Type:     "file",
			FileData: "dGVzdA==",
			Filename: "test-file_2024 (1).pdf",
			MimeType: "application/pdf",
		},
	}

	result := toFoundryFileInputs(input)

	require.Len(t, result, 1)
	assert.Equal(t, "test-file_2024 (1).pdf", result[0].Filename)
}

func TestConvertN8NChunk_WithComplexMetadata(t *testing.T) {
	metadata := map[string]interface{}{
		"nested": map[string]interface{}{
			"key": "value",
		},
		"array":   []interface{}{1, 2, 3},
		"boolean": true,
		"number":  42.5,
	}
	n8nChunk := &n8n.StreamChunk{
		Type:        n8n.ChunkTypeMetadata,
		ExecutionID: "exec-complex",
		Metadata:    metadata,
	}

	result := convertN8NChunk(n8nChunk)

	require.NotNil(t, result)
	assert.Equal(t, ChunkTypeMetadata, result.Type)
	assert.NotNil(t, result.Metadata)
	assert.NotNil(t, result.Metadata["nested"])
	assert.NotNil(t, result.Metadata["array"])
	assert.Equal(t, true, result.Metadata["boolean"])
	assert.Equal(t, 42.5, result.Metadata["number"])
}

func TestConvertFoundryChunk_WithComplexMetadata(t *testing.T) {
	metadata := map[string]interface{}{
		"agent_config": map[string]interface{}{
			"name":    "foundry-agent",
			"version": "2.0",
		},
		"tags":     []interface{}{"production", "v2"},
		"enabled":  true,
		"priority": 1,
	}
	foundryChunk := &foundry.StreamChunk{
		Type:        foundry.ChunkTypeMetadata,
		ExecutionID: "foundry-complex",
		Metadata:    metadata,
	}

	result := convertFoundryChunk(foundryChunk)

	require.NotNil(t, result)
	assert.Equal(t, ChunkTypeMetadata, result.Type)
	assert.NotNil(t, result.Metadata)
	assert.NotNil(t, result.Metadata["agent_config"])
	assert.NotNil(t, result.Metadata["tags"])
	assert.Equal(t, true, result.Metadata["enabled"])
	assert.Equal(t, 1, result.Metadata["priority"])
}
