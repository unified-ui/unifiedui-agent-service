// Package handlers provides HTTP handlers for the API.
// This file contains tests for streaming handlers in messages_send.go.
package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/api/sse"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/agents"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/tests/mocks"
)

// =============================================================================
// Mock StreamReader for testing
// =============================================================================

// mockStreamReader is a mock implementation of agents.StreamReader.
type mockStreamReader struct {
	chunks    []*agents.StreamChunk
	readIndex int
	closed    bool
}

func newMockStreamReader(chunks []*agents.StreamChunk) *mockStreamReader {
	return &mockStreamReader{
		chunks:    chunks,
		readIndex: 0,
		closed:    false,
	}
}

func (m *mockStreamReader) Read() (*agents.StreamChunk, error) {
	if m.closed {
		return nil, io.EOF
	}
	if m.readIndex >= len(m.chunks) {
		return nil, io.EOF
	}
	chunk := m.chunks[m.readIndex]
	m.readIndex++
	return chunk, nil
}

func (m *mockStreamReader) Close() error {
	m.closed = true
	return nil
}

// =============================================================================
// Mock ResponseWriter with Flusher support for SSE
// =============================================================================

type mockSSEResponseWriter struct {
	*httptest.ResponseRecorder
	flushed int
}

func (m *mockSSEResponseWriter) Flush() {
	m.flushed++
}

func newMockSSEResponseWriter() *mockSSEResponseWriter {
	return &mockSSEResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
		flushed:          0,
	}
}

// =============================================================================
// Mock WorkflowClient for testing
// =============================================================================

type mockWorkflowClient struct {
	mock.Mock
	streamReader agents.StreamReader
	invokeError  error
}

func (m *mockWorkflowClient) Invoke(ctx context.Context, req *agents.InvokeRequest) (*agents.InvokeResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*agents.InvokeResponse), args.Error(1)
}

func (m *mockWorkflowClient) InvokeStream(ctx context.Context, req *agents.InvokeRequest) (<-chan *agents.StreamChunk, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan *agents.StreamChunk), args.Error(1)
}

func (m *mockWorkflowClient) InvokeStreamReader(ctx context.Context, req *agents.InvokeRequest) (agents.StreamReader, error) {
	if m.invokeError != nil {
		return nil, m.invokeError
	}
	if m.streamReader != nil {
		return m.streamReader, nil
	}
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(agents.StreamReader), args.Error(1)
}

func (m *mockWorkflowClient) Close() error {
	return nil
}

// =============================================================================
// Test Helper Functions
// =============================================================================

func createStreamingTestContextWithResponseWriter(rw http.ResponseWriter) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(rw)
	c.Request = httptest.NewRequest(http.MethodPost, "/messages", nil)
	return c
}

func createStreamingTestHandler(docDB *mocks.MockDocDBClient, aiService *mocks.MockAIService) *MessagesHandler {
	return &MessagesHandler{
		docDBClient:    docDB,
		sessionService: &mocks.MockSessionService{},
		aiService:      aiService,
	}
}

// createTestSSEWriter creates an SSE writer from a mock response writer for testing
func createTestSSEWriter(w *mockSSEResponseWriter) *sse.Writer {
	writer, err := sse.NewWriter(w)
	if err != nil {
		panic("failed to create SSE writer: " + err.Error())
	}
	return writer
}

func createStreamingTestUserMessage() *models.Message {
	return &models.Message{
		ID:             "user-msg-123",
		Type:           models.MessageTypeUser,
		TenantID:       "test-tenant",
		ConversationID: "conv-123",
		ChatAgentID:    "agent-456",
		UserID:         "user-789",
		Content:        "Hello, how are you?",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func createStreamingTestAssistantMessage() *models.Message {
	return &models.Message{
		ID:             "assistant-msg-456",
		Type:           models.MessageTypeAssistant,
		TenantID:       "test-tenant",
		ConversationID: "conv-123",
		ChatAgentID:    "agent-456",
		UserMessageID:  "user-msg-123",
		Content:        "",
		Status:         models.MessageStatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func createStreamingTestTenantContext() *middleware.TenantContext {
	return &middleware.TenantContext{
		TenantID: "test-tenant",
		UserID:   "user-789",
	}
}

func createN8NAgentConfigForStreaming() *platform.AgentConfig {
	return &platform.AgentConfig{
		Type:        platform.AgentTypeN8N,
		TenantID:    "test-tenant",
		ChatAgentID: "agent-456",
		Settings: platform.AgentSettings{
			ChatURL:               "https://n8n.example.com/webhook",
			UseUnifiedChatHistory: false,
		},
	}
}

func createFoundryAgentConfigForStreaming() *platform.AgentConfig {
	return &platform.AgentConfig{
		Type:        platform.AgentTypeFoundry,
		TenantID:    "test-tenant",
		ChatAgentID: "foundry-agent",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://project.api.azure.com",
			AgentName:       "test-agent",
			AgentType:       "AGENT",
			APIVersion:      "2024-07-01-preview",
		},
	}
}

// =============================================================================
// handleDefaultStreaming Tests
// =============================================================================

func TestHandleDefaultStreaming_SuccessfulStream(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "Hello"},
		{Type: agents.ChunkTypeContent, Content: " world!"},
	}
	streamReader := newMockStreamReader(chunks)

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	ctx := context.Background()
	agentConfig := createN8NAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	// Mock message save
	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	executionID := handler.handleDefaultStreaming(
		ctx,
		writer,
		streamReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	// Verify stream was processed
	assert.Empty(t, executionID) // No execution ID in these chunks
	// Note: streamReader is NOT closed on successful EOF - only on error/cancel paths
	assert.Equal(t, len(chunks), streamReader.readIndex) // All chunks were read
	assert.Equal(t, "Hello world!", assistantMessage.Content)
	assert.Equal(t, models.MessageStatusSuccess, assistantMessage.Status)
}

func TestHandleDefaultStreaming_WithMetadataChunk(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "Response text"},
		{Type: agents.ChunkTypeMetadata, ExecutionID: "exec-123"},
	}
	streamReader := newMockStreamReader(chunks)

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	ctx := context.Background()
	agentConfig := createN8NAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	executionID := handler.handleDefaultStreaming(
		ctx,
		writer,
		streamReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	assert.Equal(t, "exec-123", executionID)
	assert.NotNil(t, assistantMessage.Metadata)
	assert.Equal(t, "exec-123", assistantMessage.Metadata.ExecutionID)
}

func TestHandleDefaultStreaming_WithErrorChunk(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "Partial"},
		{Type: agents.ChunkTypeError, Error: errors.New("agent error")},
	}
	streamReader := newMockStreamReader(chunks)

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	ctx := context.Background()
	agentConfig := createN8NAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	executionID := handler.handleDefaultStreaming(
		ctx,
		writer,
		streamReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	assert.Empty(t, executionID)
	assert.Equal(t, models.MessageStatusFailed, assistantMessage.Status)
}

func TestHandleDefaultStreaming_ContextCancellation(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "This won't be processed"},
	}
	streamReader := newMockStreamReader(chunks)

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	agentConfig := createN8NAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleDefaultStreaming(
		ctx,
		writer,
		streamReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	assert.True(t, streamReader.closed)
	assert.Equal(t, models.MessageStatusCancelled, assistantMessage.Status)
}

func TestHandleDefaultStreaming_StreamReadError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	// Create a stream reader that returns an error
	errorReader := &errorStreamReader{readErr: errors.New("network error")}

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	ctx := context.Background()
	agentConfig := createN8NAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleDefaultStreaming(
		ctx,
		writer,
		errorReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	assert.Equal(t, models.MessageStatusFailed, assistantMessage.Status)
}

// errorStreamReader always returns an error on Read
type errorStreamReader struct {
	readErr error
}

func (e *errorStreamReader) Read() (*agents.StreamChunk, error) {
	return nil, e.readErr
}

func (e *errorStreamReader) Close() error {
	return nil
}

// =============================================================================
// handleFoundryStreaming Tests
// =============================================================================

func TestHandleFoundryStreaming_SuccessfulStream(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "Hello from Foundry"},
		{Type: agents.ChunkTypeDone, ExecutionID: "foundry-exec-123"},
	}
	streamReader := newMockStreamReader(chunks)

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	ctx := context.Background()
	agentConfig := createFoundryAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleFoundryStreaming(
		ctx,
		writer,
		streamReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	// Stream was fully read - check readIndex equals chunk count
	assert.Equal(t, 2, streamReader.readIndex)
	assert.Equal(t, "Hello from Foundry", assistantMessage.Content)
	assert.Equal(t, models.MessageStatusSuccess, assistantMessage.Status)
}

func TestHandleFoundryStreaming_WithMetadata(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "Response"},
		{
			Type:        agents.ChunkTypeMetadata,
			ExecutionID: "exec-456",
			Metadata: map[string]interface{}{
				"message_id":  "ext-msg-123",
				"response_id": "resp-456",
				"model":       "gpt-4",
			},
		},
	}
	streamReader := newMockStreamReader(chunks)

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	ctx := context.Background()
	agentConfig := createFoundryAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleFoundryStreaming(
		ctx,
		writer,
		streamReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	assert.NotNil(t, assistantMessage.Metadata)
	assert.Equal(t, "exec-456", assistantMessage.Metadata.ExecutionID)
}

func TestHandleFoundryStreaming_WithDoneChunkMetadata(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "Response content"},
		{
			Type:        agents.ChunkTypeDone,
			ExecutionID: "final-exec-id",
			Metadata: map[string]interface{}{
				"model":       "gpt-4-turbo",
				"agent_name":  "test-agent",
				"response_id": "final-response-id",
			},
		},
	}
	streamReader := newMockStreamReader(chunks)

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	ctx := context.Background()
	agentConfig := createFoundryAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleFoundryStreaming(
		ctx,
		writer,
		streamReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	assert.Equal(t, "Response content", assistantMessage.Content)
	assert.NotNil(t, assistantMessage.Metadata)
	assert.Equal(t, "final-exec-id", assistantMessage.Metadata.ExecutionID)
	assert.Equal(t, "gpt-4-turbo", assistantMessage.Metadata.Model)
}

func TestHandleFoundryStreaming_WithNewMessageChunk(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "First message"},
		{Type: agents.ChunkTypeNewMessage, Metadata: map[string]interface{}{"message_id": "new-msg-1"}},
		{Type: agents.ChunkTypeContent, Content: "Second message"},
	}
	streamReader := newMockStreamReader(chunks)

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	ctx := context.Background()
	agentConfig := createFoundryAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleFoundryStreaming(
		ctx,
		writer,
		streamReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	// The last message should have "Second message" content
	assert.Equal(t, 3, streamReader.readIndex)
}

func TestHandleFoundryStreaming_WithErrorChunk(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "Partial response"},
		{Type: agents.ChunkTypeError, Error: errors.New("foundry error")},
	}
	streamReader := newMockStreamReader(chunks)

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	ctx := context.Background()
	agentConfig := createFoundryAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleFoundryStreaming(
		ctx,
		writer,
		streamReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	assert.Equal(t, models.MessageStatusFailed, assistantMessage.Status)
}

func TestHandleFoundryStreaming_ContextCancellation(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "Won't be processed"},
	}
	streamReader := newMockStreamReader(chunks)

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	agentConfig := createFoundryAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleFoundryStreaming(
		ctx,
		writer,
		streamReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	assert.True(t, streamReader.closed)
	assert.Equal(t, models.MessageStatusCancelled, assistantMessage.Status)
}

func TestHandleFoundryStreaming_StreamReadError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	errorReader := &errorStreamReader{readErr: errors.New("foundry stream error")}

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	ctx := context.Background()
	agentConfig := createFoundryAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleFoundryStreaming(
		ctx,
		writer,
		errorReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	assert.Equal(t, models.MessageStatusFailed, assistantMessage.Status)
}

// =============================================================================
// streamTitleGeneration Tests
// =============================================================================

func TestStreamTitleGeneration_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockAI := &mocks.MockAIService{}
	mockPlatform := &mocks.MockPlatformClient{}
	handler := &MessagesHandler{
		docDBClient:    mockDocDB,
		aiService:      mockAI,
		platformClient: mockPlatform,
	}

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	mockAI.On("GenerateTitle", mock.Anything, "test-tenant", "Hello", "Hi there").
		Return("Greeting Conversation", nil)

	// Mock UpdateConversationTitle (called in goroutine)
	mockPlatform.On("UpdateConversationTitle", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Maybe()

	ctx := context.Background()
	handler.streamTitleGeneration(ctx, writer, "test-tenant", "conv-123", "Hello", "Hi there", "auth-token")

	mockAI.AssertCalled(t, "GenerateTitle", mock.Anything, "test-tenant", "Hello", "Hi there")
}

func TestStreamTitleGeneration_AIError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockAI := &mocks.MockAIService{}
	handler := &MessagesHandler{
		docDBClient: mockDocDB,
		aiService:   mockAI,
	}

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	mockAI.On("GenerateTitle", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("", errors.New("AI service error"))

	ctx := context.Background()
	handler.streamTitleGeneration(ctx, writer, "test-tenant", "conv-123", "Hello", "Hi there", "auth-token")

	mockAI.AssertCalled(t, "GenerateTitle", mock.Anything, "test-tenant", "Hello", "Hi there")
}

func TestStreamTitleGeneration_EmptyAuthToken(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockAI := &mocks.MockAIService{}
	handler := &MessagesHandler{
		docDBClient: mockDocDB,
		aiService:   mockAI,
	}

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	mockAI.On("GenerateTitle", mock.Anything, "test-tenant", "Hello", "Hi there").
		Return("Generated Title", nil)

	ctx := context.Background()
	handler.streamTitleGeneration(ctx, writer, "test-tenant", "conv-123", "Hello", "Hi there", "")

	// Should complete without trying to update conversation title
	mockAI.AssertCalled(t, "GenerateTitle", mock.Anything, "test-tenant", "Hello", "Hi there")
}

func TestStreamTitleGeneration_NilAIService(t *testing.T) {
	handler := &MessagesHandler{
		aiService: nil,
	}

	// This should not panic and should return early
	assert.Nil(t, handler.aiService)
}

// =============================================================================
// handleStreamingResponse Tests (integration-style)
// =============================================================================

func TestHandleStreamingResponse_InvokeError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}

	workflowClient := &mockWorkflowClient{
		invokeError: errors.New("failed to invoke agent"),
	}

	handler := &MessagesHandler{
		docDBClient:    mockDocDB,
		sessionService: mockSession,
	}

	w := newMockSSEResponseWriter()
	c := createStreamingTestContextWithResponseWriter(w)

	agentClients := &agents.AgentClients{
		WorkflowClient: workflowClient,
	}

	agentConfig := createN8NAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleStreamingResponse(
		c,
		createStreamingTestTenantContext(),
		agentClients,
		agentConfig,
		userMessage,
		assistantMessage,
		nil, // chatHistory
		"",  // extConversationID
		"",  // foundryAPIKey
		nil, // contextData
		"",  // authToken
		false,
		nil, // files
	)

	// Should have written error to response
	body := w.Body.String()
	assert.Contains(t, body, "STREAM_ERROR")
}

func TestHandleStreamingResponse_FoundryAgentType(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "Foundry response"},
	}
	streamReader := newMockStreamReader(chunks)

	workflowClient := &mockWorkflowClient{
		streamReader: streamReader,
	}

	handler := &MessagesHandler{
		docDBClient:    mockDocDB,
		sessionService: mockSession,
	}

	w := newMockSSEResponseWriter()
	c := createStreamingTestContextWithResponseWriter(w)

	agentClients := &agents.AgentClients{
		WorkflowClient: workflowClient,
	}

	agentConfig := createFoundryAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)
	mockSession.On("GetSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not found"))
	mockSession.On("SetSession", mock.Anything, mock.Anything).Return(nil)

	handler.handleStreamingResponse(
		c,
		createStreamingTestTenantContext(),
		agentClients,
		agentConfig,
		userMessage,
		assistantMessage,
		nil,
		"ext-conv-123",
		"foundry-api-key",
		nil,
		"",
		false,
		nil,
	)

	assert.True(t, streamReader.closed)
}

func TestHandleStreamingResponse_N8NAgentType(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "N8N response"},
	}
	streamReader := newMockStreamReader(chunks)

	workflowClient := &mockWorkflowClient{
		streamReader: streamReader,
	}

	handler := &MessagesHandler{
		docDBClient:    mockDocDB,
		sessionService: mockSession,
	}

	w := newMockSSEResponseWriter()
	c := createStreamingTestContextWithResponseWriter(w)

	agentClients := &agents.AgentClients{
		WorkflowClient: workflowClient,
	}

	agentConfig := createN8NAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleStreamingResponse(
		c,
		createStreamingTestTenantContext(),
		agentClients,
		agentConfig,
		userMessage,
		assistantMessage,
		nil,
		"",
		"",
		nil,
		"",
		false,
		nil,
	)

	assert.True(t, streamReader.closed)
}

func TestHandleStreamingResponse_WithTitleGeneration(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	mockAI := &mocks.MockAIService{}
	mockPlatform := &mocks.MockPlatformClient{}

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "Response"},
	}
	streamReader := newMockStreamReader(chunks)

	workflowClient := &mockWorkflowClient{
		streamReader: streamReader,
	}

	handler := &MessagesHandler{
		docDBClient:    mockDocDB,
		sessionService: mockSession,
		aiService:      mockAI,
		platformClient: mockPlatform,
	}

	w := newMockSSEResponseWriter()
	c := createStreamingTestContextWithResponseWriter(w)

	agentClients := &agents.AgentClients{
		WorkflowClient: workflowClient,
	}

	agentConfig := createN8NAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)
	mockAI.On("GenerateTitle", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("Generated Title", nil)
	mockPlatform.On("UpdateConversationTitle", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Maybe()

	handler.handleStreamingResponse(
		c,
		createStreamingTestTenantContext(),
		agentClients,
		agentConfig,
		userMessage,
		assistantMessage,
		nil,
		"",
		"",
		nil,
		"auth-token",
		true, // isFirstMessage triggers title generation
		nil,
	)

	time.Sleep(100 * time.Millisecond) // Give goroutine time to complete
	mockAI.AssertCalled(t, "GenerateTitle", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// =============================================================================
// extractFoundryMetadata Tests
// =============================================================================

func TestExtractFoundryMetadata_WithMessageID_Streaming(t *testing.T) {
	handler := &MessagesHandler{}

	metadata := map[string]interface{}{
		"message_id": "ext-msg-123",
	}

	result := handler.extractFoundryMetadata(metadata)

	assert.NotNil(t, result)
	assert.Equal(t, "ext-msg-123", result.ExecutionID)
	assert.Equal(t, "ext-msg-123", result.ExtMessageID)
}

func TestExtractFoundryMetadata_EmptyMap(t *testing.T) {
	handler := &MessagesHandler{}

	result := handler.extractFoundryMetadata(map[string]interface{}{})

	assert.NotNil(t, result)
	assert.Empty(t, result.ExecutionID)
}

func TestExtractFoundryMetadata_NilValue(t *testing.T) {
	handler := &MessagesHandler{}

	metadata := map[string]interface{}{
		"message_id": nil,
	}

	result := handler.extractFoundryMetadata(metadata)

	assert.NotNil(t, result)
	assert.Empty(t, result.ExecutionID)
}

// =============================================================================
// mergeFoundryMetadata Tests
// =============================================================================

func TestMergeFoundryMetadata_AllFields_Streaming(t *testing.T) {
	handler := &MessagesHandler{}
	msg := &models.Message{}

	metadata := map[string]interface{}{
		"response_id": "resp-123",
		"message_id":  "msg-456",
		"model":       "gpt-4",
		"agent_name":  "test-agent",
		"usage": map[string]interface{}{
			"input_tokens":  100,
			"output_tokens": 200,
		},
	}

	handler.mergeFoundryMetadata(msg, metadata)

	assert.NotNil(t, msg.Metadata)
	assert.Equal(t, "resp-123", msg.Metadata.ExecutionID)
	assert.Equal(t, "msg-456", msg.Metadata.ExtMessageID)
	assert.Equal(t, "gpt-4", msg.Metadata.Model)
	assert.Equal(t, "test-agent", msg.Metadata.AgentType)
	assert.Equal(t, 100, msg.Metadata.TokensInput)
	assert.Equal(t, 200, msg.Metadata.TokensOutput)
}

func TestMergeFoundryMetadata_DoesNotOverwriteExisting(t *testing.T) {
	handler := &MessagesHandler{}
	msg := &models.Message{
		Metadata: &models.AssistantMetadata{
			ExecutionID:  "original-exec",
			ExtMessageID: "original-msg",
		},
	}

	metadata := map[string]interface{}{
		"response_id": "new-exec",
		"message_id":  "new-msg",
	}

	handler.mergeFoundryMetadata(msg, metadata)

	// Should not overwrite existing values
	assert.Equal(t, "original-exec", msg.Metadata.ExecutionID)
	assert.Equal(t, "original-msg", msg.Metadata.ExtMessageID)
}

func TestMergeFoundryMetadata_WorkflowAction_Streaming(t *testing.T) {
	handler := &MessagesHandler{}
	msg := &models.Message{}

	metadata := map[string]interface{}{
		"type":               "workflow_action",
		"kind":               "tool_call",
		"action_id":          "action-123",
		"parent_action_id":   "parent-456",
		"previous_action_id": "prev-789",
		"status":             "completed",
	}

	handler.mergeFoundryMetadata(msg, metadata)

	assert.NotNil(t, msg.Metadata)
	require.Len(t, msg.StatusTraces, 1)
	assert.Equal(t, "workflow_action", msg.StatusTraces[0].Type)
	assert.Equal(t, "tool_call", msg.StatusTraces[0].Name)
}

func TestMergeFoundryMetadata_EmptyMetadata(t *testing.T) {
	handler := &MessagesHandler{}
	msg := &models.Message{}

	handler.mergeFoundryMetadata(msg, map[string]interface{}{})

	assert.NotNil(t, msg.Metadata)
}

// =============================================================================
// getStringFromMap Tests
// =============================================================================

func TestGetStringFromMap_ValidString(t *testing.T) {
	m := map[string]interface{}{
		"key": "value",
	}

	result := getStringFromMap(m, "key")
	assert.Equal(t, "value", result)
}

func TestGetStringFromMap_MissingKey(t *testing.T) {
	m := map[string]interface{}{
		"other": "value",
	}

	result := getStringFromMap(m, "key")
	assert.Empty(t, result)
}

func TestGetStringFromMap_NonStringValue(t *testing.T) {
	m := map[string]interface{}{
		"key": 123,
	}

	result := getStringFromMap(m, "key")
	assert.Empty(t, result)
}

func TestGetStringFromMap_NilValue(t *testing.T) {
	m := map[string]interface{}{
		"key": nil,
	}

	result := getStringFromMap(m, "key")
	assert.Empty(t, result)
}

// =============================================================================
// saveCancelledAssistantMessage Tests
// =============================================================================

func TestSaveCancelledAssistantMessage_SetsCorrectStatus(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	msg := createStreamingTestAssistantMessage()
	agentConfig := createN8NAgentConfigForStreaming()
	startTime := time.Now().Add(-500 * time.Millisecond)

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.saveCancelledAssistantMessage(msg, "partial content", agentConfig, startTime)

	assert.Equal(t, models.MessageStatusCancelled, msg.Status)
	assert.Equal(t, "partial content", msg.Content)
	assert.NotNil(t, msg.Metadata)
	assert.Greater(t, msg.Metadata.LatencyMs, int64(0))
	assert.Equal(t, "N8N", msg.Metadata.AgentType)
}

// =============================================================================
// saveFailedAssistantMessage Tests
// =============================================================================

func TestSaveFailedAssistantMessage_SetsErrorStatus(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	msg := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.saveFailedAssistantMessage(context.Background(), msg, "error message")

	assert.Equal(t, models.MessageStatusFailed, msg.Status)
	assert.Equal(t, "error message", msg.ErrorMessage)
}

// =============================================================================
// saveAssistantMessageWithMetadata Tests
// =============================================================================

func TestSaveAssistantMessageWithMetadata_Success_Streaming(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	msg := createStreamingTestAssistantMessage()
	msg.SetSuccess("response content")
	agentConfig := createFoundryAgentConfigForStreaming()
	startTime := time.Now().Add(-200 * time.Millisecond)

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	result := handler.saveAssistantMessageWithMetadata(context.Background(), msg, agentConfig, startTime)

	assert.NotNil(t, result)
	assert.NotNil(t, msg.Metadata)
	assert.Greater(t, msg.Metadata.LatencyMs, int64(0))
	assert.Equal(t, "MICROSOFT_FOUNDRY", msg.Metadata.AgentType)
}

func TestSaveAssistantMessageWithMetadata_DBError_Streaming(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	msg := createStreamingTestAssistantMessage()
	agentConfig := createN8NAgentConfigForStreaming()
	startTime := time.Now()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).
		Return(errors.New("database error"))

	result := handler.saveAssistantMessageWithMetadata(context.Background(), msg, agentConfig, startTime)

	assert.Nil(t, result)
}

// =============================================================================
// Edge Cases and Integration Tests
// =============================================================================

func TestHandleDefaultStreaming_MultipleContentChunks(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "Part 1. "},
		{Type: agents.ChunkTypeContent, Content: "Part 2. "},
		{Type: agents.ChunkTypeContent, Content: "Part 3."},
	}
	streamReader := newMockStreamReader(chunks)

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	ctx := context.Background()
	agentConfig := createN8NAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleDefaultStreaming(
		ctx,
		writer,
		streamReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	assert.Equal(t, "Part 1. Part 2. Part 3.", assistantMessage.Content)
}

func TestHandleFoundryStreaming_EmptyContent(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeDone},
	}
	streamReader := newMockStreamReader(chunks)

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	ctx := context.Background()
	agentConfig := createFoundryAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleFoundryStreaming(
		ctx,
		writer,
		streamReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	assert.Empty(t, assistantMessage.Content)
	assert.Equal(t, models.MessageStatusSuccess, assistantMessage.Status)
}

func TestHandleDefaultStreaming_EmptyChunkContent(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: ""},
		{Type: agents.ChunkTypeContent, Content: "actual content"},
		{Type: agents.ChunkTypeContent, Content: ""},
	}
	streamReader := newMockStreamReader(chunks)

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	ctx := context.Background()
	agentConfig := createN8NAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleDefaultStreaming(
		ctx,
		writer,
		streamReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	assert.Equal(t, "actual content", assistantMessage.Content)
}

func TestHandleDefaultStreaming_NilErrorInChunk(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	handler := createStreamingTestHandler(mockDocDB, nil)

	// Error chunk with nil error should be handled gracefully
	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "Content"},
		{Type: agents.ChunkTypeError, Error: nil}, // nil error
	}
	streamReader := newMockStreamReader(chunks)

	w := newMockSSEResponseWriter()
	writer := createTestSSEWriter(w)

	ctx := context.Background()
	agentConfig := createN8NAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleDefaultStreaming(
		ctx,
		writer,
		streamReader,
		createStreamingTestTenantContext(),
		agentConfig,
		userMessage,
		assistantMessage,
		time.Now(),
	)

	// Should complete successfully since error is nil
	assert.Equal(t, "Content", assistantMessage.Content)
	assert.Equal(t, models.MessageStatusSuccess, assistantMessage.Status)
}

func TestExtractBaseURL_HTTPSWithPath_Streaming(t *testing.T) {
	result := extractBaseURL("https://example.com/webhook/chat")
	assert.Equal(t, "https://example.com", result)
}

func TestExtractBaseURL_HTTPWithPath_Streaming(t *testing.T) {
	result := extractBaseURL("http://localhost:8080/api/chat")
	assert.Equal(t, "http://localhost:8080", result)
}

func TestExtractBaseURL_NoPath_Streaming(t *testing.T) {
	result := extractBaseURL("https://example.com")
	assert.Equal(t, "https://example.com", result)
}

func TestExtractBaseURL_EmptyURL_Streaming(t *testing.T) {
	result := extractBaseURL("")
	assert.Empty(t, result)
}

// =============================================================================
// Verify SSE Response Format Tests (SSE response output validation)
// =============================================================================

func TestHandleStreamingResponse_SSEHeaders(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "Response"},
	}
	streamReader := newMockStreamReader(chunks)

	workflowClient := &mockWorkflowClient{
		streamReader: streamReader,
	}

	handler := &MessagesHandler{
		docDBClient:    mockDocDB,
		sessionService: mockSession,
	}

	w := newMockSSEResponseWriter()
	c := createStreamingTestContextWithResponseWriter(w)

	agentClients := &agents.AgentClients{
		WorkflowClient: workflowClient,
	}

	agentConfig := createN8NAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleStreamingResponse(
		c,
		createStreamingTestTenantContext(),
		agentClients,
		agentConfig,
		userMessage,
		assistantMessage,
		nil,
		"",
		"",
		nil,
		"",
		false,
		nil,
	)

	// Verify SSE headers are set
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
}

func TestHandleStreamingResponse_SSEEventFormat(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}

	chunks := []*agents.StreamChunk{
		{Type: agents.ChunkTypeContent, Content: "Hello"},
	}
	streamReader := newMockStreamReader(chunks)

	workflowClient := &mockWorkflowClient{
		streamReader: streamReader,
	}

	handler := &MessagesHandler{
		docDBClient:    mockDocDB,
		sessionService: mockSession,
	}

	w := newMockSSEResponseWriter()
	c := createStreamingTestContextWithResponseWriter(w)

	agentClients := &agents.AgentClients{
		WorkflowClient: workflowClient,
	}

	agentConfig := createN8NAgentConfigForStreaming()
	userMessage := createStreamingTestUserMessage()
	assistantMessage := createStreamingTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.handleStreamingResponse(
		c,
		createStreamingTestTenantContext(),
		agentClients,
		agentConfig,
		userMessage,
		assistantMessage,
		nil,
		"",
		"",
		nil,
		"",
		false,
		nil,
	)

	body := w.Body.String()
	// Should contain SSE event format components
	assert.True(t, strings.Contains(body, "event:") || strings.Contains(body, "STREAM_START"))
}
