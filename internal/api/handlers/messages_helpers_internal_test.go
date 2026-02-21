// Package handlers provides HTTP handlers for the API.
// This file contains internal tests for message helper functions.
package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/session"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
	"github.com/unifiedui/agent-service/tests/mocks"
)

// --- Test Fixtures ---

func createTestMessagesHandlerInternal(
	docDBClient *mocks.MockDocDBClient,
	sessionService *mocks.MockSessionService,
	importService *traceimport.ImportService,
) *MessagesHandler {
	return &MessagesHandler{
		docDBClient:    docDBClient,
		sessionService: sessionService,
		importService:  importService,
	}
}

func createTestAgentConfig(agentType platform.AgentType) *platform.AgentConfig {
	return &platform.AgentConfig{
		Type:        agentType,
		TenantID:    "test-tenant",
		ChatAgentID: "test-agent",
		Settings: platform.AgentSettings{
			UseUnifiedChatHistory: true,
			ChatHistoryCount:      30,
			ChatURL:               "https://n8n.example.com/webhook/chat",
			ProjectEndpoint:       "https://project.api.azureml.ms",
			APIVersion:            "2025-01-01-preview",
		},
	}
}

func createTestTenantContext() *middleware.TenantContext {
	return &middleware.TenantContext{
		TenantID: "test-tenant",
		UserID:   "test-user",
	}
}

func createTestAssistantMessage() *models.Message {
	return models.NewAssistantMessage(
		"test-tenant",
		"conv-123",
		"user-msg-123",
		"test-agent",
		"",
		models.MessageStatusPending,
	)
}

func createTestUserMessage() *models.Message {
	return models.NewUserMessage(
		"test-tenant",
		"conv-123",
		"test-agent",
		"test-user",
		"Hello, world!",
		nil,
		nil,
	)
}

// --- saveAssistantMessageWithMetadata Tests ---

func TestSaveAssistantMessageWithMetadata_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	msg := createTestAssistantMessage()
	msg.Content = "This is the assistant response"
	msg.Status = models.MessageStatusSuccess
	agentConfig := createTestAgentConfig(platform.AgentTypeN8N)
	startTime := time.Now().Add(-100 * time.Millisecond)

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.MatchedBy(func(m *models.Message) bool {
		return m.ID == msg.ID && m.Metadata != nil && m.Metadata.LatencyMs > 0
	})).Return(nil)

	result := handler.saveAssistantMessageWithMetadata(context.Background(), msg, agentConfig, startTime)

	require.NotNil(t, result)
	assert.NotNil(t, result.Metadata)
	assert.Greater(t, result.Metadata.LatencyMs, int64(0))
	assert.Equal(t, string(platform.AgentTypeN8N), result.Metadata.AgentType)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestSaveAssistantMessageWithMetadata_WithExistingMetadata(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	msg := createTestAssistantMessage()
	msg.Metadata = &models.AssistantMetadata{
		Model:        "gpt-4",
		TokensInput:  100,
		TokensOutput: 50,
	}
	agentConfig := createTestAgentConfig(platform.AgentTypeFoundry)
	startTime := time.Now().Add(-200 * time.Millisecond)

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	result := handler.saveAssistantMessageWithMetadata(context.Background(), msg, agentConfig, startTime)

	require.NotNil(t, result)
	assert.Equal(t, "gpt-4", result.Metadata.Model)
	assert.Equal(t, 100, result.Metadata.TokensInput)
	assert.Equal(t, 50, result.Metadata.TokensOutput)
	assert.Greater(t, result.Metadata.LatencyMs, int64(0))
	assert.Equal(t, string(platform.AgentTypeFoundry), result.Metadata.AgentType)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestSaveAssistantMessageWithMetadata_DBError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	msg := createTestAssistantMessage()
	agentConfig := createTestAgentConfig(platform.AgentTypeN8N)
	startTime := time.Now()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(errors.New("database error"))

	result := handler.saveAssistantMessageWithMetadata(context.Background(), msg, agentConfig, startTime)

	assert.Nil(t, result)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

// --- saveFailedAssistantMessage Tests ---

func TestSaveFailedAssistantMessage_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	msg := createTestAssistantMessage()
	errorMsg := "Agent invocation failed: timeout"

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.MatchedBy(func(m *models.Message) bool {
		return m.Status == models.MessageStatusFailed && m.ErrorMessage == errorMsg
	})).Return(nil)

	handler.saveFailedAssistantMessage(context.Background(), msg, errorMsg)

	assert.Equal(t, models.MessageStatusFailed, msg.Status)
	assert.Equal(t, errorMsg, msg.ErrorMessage)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestSaveFailedAssistantMessage_UpdatesMessage(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	msg := createTestAssistantMessage()
	msg.Status = models.MessageStatusPending
	originalUpdatedAt := msg.UpdatedAt

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	time.Sleep(1 * time.Millisecond)
	handler.saveFailedAssistantMessage(context.Background(), msg, "error")

	assert.True(t, msg.UpdatedAt.After(originalUpdatedAt) || msg.UpdatedAt.Equal(originalUpdatedAt))
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestSaveFailedAssistantMessage_DBErrorIgnored(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	msg := createTestAssistantMessage()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(errors.New("db error"))

	// Should not panic even if DB fails
	handler.saveFailedAssistantMessage(context.Background(), msg, "test error")

	assert.Equal(t, models.MessageStatusFailed, msg.Status)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

// --- saveCancelledAssistantMessage Tests ---

func TestSaveCancelledAssistantMessage_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	msg := createTestAssistantMessage()
	partialContent := "This is partial content before cancellation..."
	agentConfig := createTestAgentConfig(platform.AgentTypeN8N)
	startTime := time.Now().Add(-500 * time.Millisecond)

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.MatchedBy(func(m *models.Message) bool {
		return m.Status == models.MessageStatusCancelled &&
			m.Content == partialContent &&
			m.Metadata != nil &&
			m.Metadata.LatencyMs > 0
	})).Return(nil)

	handler.saveCancelledAssistantMessage(msg, partialContent, agentConfig, startTime)

	assert.Equal(t, models.MessageStatusCancelled, msg.Status)
	assert.Equal(t, partialContent, msg.Content)
	assert.NotNil(t, msg.Metadata)
	assert.Greater(t, msg.Metadata.LatencyMs, int64(0))
	assert.Equal(t, string(platform.AgentTypeN8N), msg.Metadata.AgentType)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestSaveCancelledAssistantMessage_WithExistingMetadata(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	msg := createTestAssistantMessage()
	msg.Metadata = &models.AssistantMetadata{
		Model:       "gpt-4",
		ExecutionID: "exec-123",
	}
	agentConfig := createTestAgentConfig(platform.AgentTypeFoundry)
	startTime := time.Now().Add(-300 * time.Millisecond)

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.saveCancelledAssistantMessage(msg, "partial", agentConfig, startTime)

	assert.Equal(t, "gpt-4", msg.Metadata.Model)
	assert.Equal(t, "exec-123", msg.Metadata.ExecutionID)
	assert.Greater(t, msg.Metadata.LatencyMs, int64(0))
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

func TestSaveCancelledAssistantMessage_EmptyContent(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	msg := createTestAssistantMessage()
	agentConfig := createTestAgentConfig(platform.AgentTypeN8N)
	startTime := time.Now()

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.saveCancelledAssistantMessage(msg, "", agentConfig, startTime)

	assert.Equal(t, models.MessageStatusCancelled, msg.Status)
	assert.Equal(t, "", msg.Content)
	mockDocDB.GetMessagesCollection().AssertExpectations(t)
}

// --- updateSessionCache Tests ---

func TestUpdateSessionCache_UseUnifiedHistoryDisabled(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	tenantCtx := createTestTenantContext()
	agentConfig := createTestAgentConfig(platform.AgentTypeN8N)
	agentConfig.Settings.UseUnifiedChatHistory = false
	userMsg := createTestUserMessage()
	assistantMsg := createTestAssistantMessage()

	// Should not call any session service methods
	handler.updateSessionCache(context.Background(), tenantCtx, agentConfig, userMsg, assistantMsg)

	mockSession.AssertNotCalled(t, "GetSession")
	mockSession.AssertNotCalled(t, "SetSession")
	mockSession.AssertNotCalled(t, "UpdateChatHistory")
}

func TestUpdateSessionCache_NewSession(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	tenantCtx := createTestTenantContext()
	agentConfig := createTestAgentConfig(platform.AgentTypeN8N)
	userMsg := createTestUserMessage()
	userMsg.ID = "user-msg-1"
	assistantMsg := createTestAssistantMessage()
	assistantMsg.ID = "assistant-msg-1"
	assistantMsg.Content = "Hello! How can I help you?"

	// GetSession returns nil (no existing session)
	mockSession.On("GetSession", mock.Anything, tenantCtx.TenantID, tenantCtx.UserID, userMsg.ConversationID).
		Return(nil, nil)

	// SetSession should be called with new session
	mockSession.On("SetSession", mock.Anything, mock.MatchedBy(func(s *session.SessionData) bool {
		return s.TenantID == tenantCtx.TenantID &&
			s.UserID == tenantCtx.UserID &&
			s.ConversationID == userMsg.ConversationID &&
			len(s.ChatHistory) == 2
	})).Return(nil)

	handler.updateSessionCache(context.Background(), tenantCtx, agentConfig, userMsg, assistantMsg)

	mockSession.AssertExpectations(t)
}

func TestUpdateSessionCache_ExistingSessionUpdated(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	tenantCtx := createTestTenantContext()
	agentConfig := createTestAgentConfig(platform.AgentTypeN8N)
	userMsg := createTestUserMessage()
	assistantMsg := createTestAssistantMessage()
	assistantMsg.Content = "Response"

	existingSession := &session.SessionData{
		TenantID:       tenantCtx.TenantID,
		UserID:         tenantCtx.UserID,
		ConversationID: userMsg.ConversationID,
		ChatHistory:    []models.ChatHistoryEntry{},
	}

	mockSession.On("GetSession", mock.Anything, tenantCtx.TenantID, tenantCtx.UserID, userMsg.ConversationID).
		Return(existingSession, nil)

	mockSession.On("UpdateChatHistory", mock.Anything, tenantCtx.TenantID, tenantCtx.UserID, userMsg.ConversationID, mock.MatchedBy(func(entries []models.ChatHistoryEntry) bool {
		return len(entries) == 2 &&
			entries[0].Role == models.MessageTypeUser &&
			entries[1].Role == models.MessageTypeAssistant
	})).Return(nil)

	handler.updateSessionCache(context.Background(), tenantCtx, agentConfig, userMsg, assistantMsg)

	mockSession.AssertExpectations(t)
}

func TestUpdateSessionCache_GetSessionError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	tenantCtx := createTestTenantContext()
	agentConfig := createTestAgentConfig(platform.AgentTypeN8N)
	userMsg := createTestUserMessage()
	assistantMsg := createTestAssistantMessage()

	mockSession.On("GetSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("cache error"))

	// Should create a new session when GetSession fails
	mockSession.On("SetSession", mock.Anything, mock.Anything).Return(nil)

	handler.updateSessionCache(context.Background(), tenantCtx, agentConfig, userMsg, assistantMsg)

	mockSession.AssertExpectations(t)
}

// --- updateSessionCacheConfigOnly Tests ---

func TestUpdateSessionCacheConfigOnly_NoExistingSession(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	tenantCtx := createTestTenantContext()
	agentConfig := createTestAgentConfig(platform.AgentTypeFoundry)
	conversationID := "conv-456"

	mockSession.On("GetSession", mock.Anything, tenantCtx.TenantID, tenantCtx.UserID, conversationID).
		Return(nil, nil)

	mockSession.On("SetSession", mock.Anything, mock.MatchedBy(func(s *session.SessionData) bool {
		return s.TenantID == tenantCtx.TenantID &&
			s.UserID == tenantCtx.UserID &&
			s.ConversationID == conversationID &&
			len(s.ChatHistory) == 0
	})).Return(nil)

	handler.updateSessionCacheConfigOnly(context.Background(), tenantCtx, agentConfig, conversationID)

	mockSession.AssertExpectations(t)
}

func TestUpdateSessionCacheConfigOnly_ExistingSessionSkipped(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	tenantCtx := createTestTenantContext()
	agentConfig := createTestAgentConfig(platform.AgentTypeFoundry)
	conversationID := "conv-789"

	existingSession := &session.SessionData{
		TenantID: tenantCtx.TenantID,
	}

	mockSession.On("GetSession", mock.Anything, tenantCtx.TenantID, tenantCtx.UserID, conversationID).
		Return(existingSession, nil)

	// SetSession should NOT be called when session exists
	handler.updateSessionCacheConfigOnly(context.Background(), tenantCtx, agentConfig, conversationID)

	mockSession.AssertExpectations(t)
	mockSession.AssertNotCalled(t, "SetSession")
}

// --- enqueueFoundryTraceImport Tests ---

func TestEnqueueFoundryTraceImport_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)

	// Register a mock importer for Foundry
	mockImporter := mocks.NewMockTraceImporter()
	mockImporter.AgentType = platform.AgentTypeFoundry
	importService.RegisterImporter(mockImporter)

	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	tenantCtx := createTestTenantContext()
	agentConfig := createTestAgentConfig(platform.AgentTypeFoundry)
	userMsg := createTestUserMessage()
	extConversationID := "ext-conv-123"
	foundryAPIKey := "foundry-api-key-secret"

	// Should not panic - enqueue is fire-and-forget
	handler.enqueueFoundryTraceImport(tenantCtx, agentConfig, userMsg, extConversationID, foundryAPIKey)
}

// --- enqueueN8NTraceImport Tests ---

func TestEnqueueN8NTraceImport_Success(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)

	// Register a mock importer for N8N
	mockImporter := mocks.NewMockTraceImporter()
	mockImporter.AgentType = platform.AgentTypeN8N
	importService.RegisterImporter(mockImporter)

	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	tenantCtx := createTestTenantContext()
	agentConfig := createTestAgentConfig(platform.AgentTypeN8N)
	agentConfig.Settings.APICredentials = &platform.Credentials{
		Secret: "n8n-api-key-secret",
	}
	userMsg := createTestUserMessage()
	executionID := "exec-12345"

	// Should not panic - enqueue is fire-and-forget
	handler.enqueueN8NTraceImport(tenantCtx, agentConfig, userMsg, executionID)
}

func TestEnqueueN8NTraceImport_NoAPIKey(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	tenantCtx := createTestTenantContext()
	agentConfig := createTestAgentConfig(platform.AgentTypeN8N)
	agentConfig.Settings.APICredentials = nil
	userMsg := createTestUserMessage()

	// Should return early without panic
	handler.enqueueN8NTraceImport(tenantCtx, agentConfig, userMsg, "exec-123")
}

func TestEnqueueN8NTraceImport_EmptyAPIKey(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	tenantCtx := createTestTenantContext()
	agentConfig := createTestAgentConfig(platform.AgentTypeN8N)
	agentConfig.Settings.APICredentials = &platform.Credentials{
		Secret: "", // Empty
	}
	userMsg := createTestUserMessage()

	// Should return early without panic
	handler.enqueueN8NTraceImport(tenantCtx, agentConfig, userMsg, "exec-123")
}

func TestEnqueueN8NTraceImport_EmptyBaseURL(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	tenantCtx := createTestTenantContext()
	agentConfig := createTestAgentConfig(platform.AgentTypeN8N)
	agentConfig.Settings.ChatURL = "" // Empty URL
	agentConfig.Settings.APICredentials = &platform.Credentials{
		Secret: "api-key",
	}
	userMsg := createTestUserMessage()

	// Should return early without panic
	handler.enqueueN8NTraceImport(tenantCtx, agentConfig, userMsg, "exec-123")
}

func TestEnqueueN8NTraceImport_NoExecutionIDOrConversationID(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockSession := &mocks.MockSessionService{}
	importService := mocks.NewMockImportService(mockDocDB)
	handler := createTestMessagesHandlerInternal(mockDocDB, mockSession, importService)

	tenantCtx := createTestTenantContext()
	agentConfig := createTestAgentConfig(platform.AgentTypeN8N)
	agentConfig.Settings.APICredentials = &platform.Credentials{
		Secret: "api-key",
	}
	userMsg := createTestUserMessage()
	userMsg.ConversationID = "" // Empty

	// Should return early when both executionID and conversationID are empty
	handler.enqueueN8NTraceImport(tenantCtx, agentConfig, userMsg, "")
}
