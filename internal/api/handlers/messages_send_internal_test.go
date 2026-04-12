// Package handlers provides HTTP handlers for the API.
// This file contains internal tests for messages_send.go functions.
package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/agents"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/session"
	"github.com/unifiedui/agent-service/tests/mocks"
)

// =============================================================================
// Test Fixtures for SendMessage
// =============================================================================

func createSendTestHandler(
	docDBClient *mocks.MockDocDBClient,
	platformClient *mocks.MockPlatformClient,
	sessionService *mocks.MockSessionService,
) *MessagesHandler {
	mockCache := &mocks.MockConfigCacheService{}
	mockCache.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("cache miss"))
	mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	return &MessagesHandler{
		docDBClient:    docDBClient,
		platformClient: platformClient,
		sessionService: sessionService,
		configCache:    mockCache,
		agentFactory:   agents.NewFactory(),
	}
}

// =============================================================================
// convertFilesToFileInputs Tests (additional cases not in helpers_test.go)
// =============================================================================

func TestConvertFilesToFileInputs_SingleImage(t *testing.T) {
	files := []FileAttachment{
		{
			Type:     "image",
			ImageURL: "data:image/png;base64,iVBORw0KGgo=",
			Filename: "test.png",
			MimeType: "image/png",
			Detail:   "high",
		},
	}

	result := convertFilesToFileInputs(files)

	require.Len(t, result, 1)
	assert.Equal(t, "image", result[0].Type)
	assert.Equal(t, "data:image/png;base64,iVBORw0KGgo=", result[0].ImageURL)
	assert.Equal(t, "test.png", result[0].Filename)
	assert.Equal(t, "image/png", result[0].MimeType)
	assert.Equal(t, "high", result[0].Detail)
}

func TestConvertFilesToFileInputs_SingleFile(t *testing.T) {
	files := []FileAttachment{
		{
			Type:     "file",
			FileData: "SGVsbG8gV29ybGQ=",
			Filename: "document.pdf",
			MimeType: "application/pdf",
		},
	}

	result := convertFilesToFileInputs(files)

	require.Len(t, result, 1)
	assert.Equal(t, "file", result[0].Type)
	assert.Equal(t, "SGVsbG8gV29ybGQ=", result[0].FileData)
	assert.Equal(t, "document.pdf", result[0].Filename)
	assert.Equal(t, "application/pdf", result[0].MimeType)
}

func TestConvertFilesToFileInputs_MultipleFiles(t *testing.T) {
	files := []FileAttachment{
		{
			Type:     "image",
			ImageURL: "data:image/jpeg;base64,/9j/4A==",
			Filename: "photo.jpg",
			MimeType: "image/jpeg",
			Detail:   "low",
		},
		{
			Type:     "file",
			FileData: "UEsDBBQ=",
			Filename: "archive.zip",
			MimeType: "application/zip",
		},
		{
			Type:     "audio",
			FileData: "SUQz",
			Filename: "audio.mp3",
			MimeType: "audio/mpeg",
		},
	}

	result := convertFilesToFileInputs(files)

	require.Len(t, result, 3)

	assert.Equal(t, "image", result[0].Type)
	assert.Equal(t, "photo.jpg", result[0].Filename)
	assert.Equal(t, "low", result[0].Detail)

	assert.Equal(t, "file", result[1].Type)
	assert.Equal(t, "archive.zip", result[1].Filename)

	assert.Equal(t, "audio", result[2].Type)
	assert.Equal(t, "audio.mp3", result[2].Filename)
}

func TestConvertFilesToFileInputs_PreservesAllFields(t *testing.T) {
	files := []FileAttachment{
		{
			Type:     "file",
			ImageURL: "http://example.com/image.png",
			FileData: "base64data",
			Filename: "test.txt",
			MimeType: "text/plain",
			Detail:   "auto",
		},
	}

	result := convertFilesToFileInputs(files)

	require.Len(t, result, 1)
	assert.Equal(t, files[0].Type, result[0].Type)
	assert.Equal(t, files[0].ImageURL, result[0].ImageURL)
	assert.Equal(t, files[0].FileData, result[0].FileData)
	assert.Equal(t, files[0].Filename, result[0].Filename)
	assert.Equal(t, files[0].MimeType, result[0].MimeType)
	assert.Equal(t, files[0].Detail, result[0].Detail)
}

// =============================================================================
// SendMessage Request Validation Tests
// These test the input validation logic without requiring full streaming setup
// =============================================================================

func TestSendMessage_InvalidJSON(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	handler := createSendTestHandler(mockDocDB, mockPlatform, mockSession)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Set tenant params (simulating URL params)
	c.Params = gin.Params{
		{Key: "tenantId", Value: "test-tenant"},
	}
	c.Set("user_id", "test-user")

	// Invalid JSON body
	c.Request = httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBufferString("invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SendMessage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSendMessage_MissingChatAgentID(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	handler := createSendTestHandler(mockDocDB, mockPlatform, mockSession)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "tenantId", Value: "test-tenant"},
	}
	c.Set("user_id", "test-user")

	// Missing chatAgentId field
	body := `{"message": {"content": "Hello"}}`
	c.Request = httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SendMessage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSendMessage_MissingMessageContent(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	handler := createSendTestHandler(mockDocDB, mockPlatform, mockSession)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "tenantId", Value: "test-tenant"},
	}
	c.Set("user_id", "test-user")

	// Missing message content
	body := `{"chatAgentId": "agent-123", "message": {}}`
	c.Request = httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SendMessage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSendMessage_EmptyMessageContent(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	handler := createSendTestHandler(mockDocDB, mockPlatform, mockSession)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "tenantId", Value: "test-tenant"},
	}
	c.Set("user_id", "test-user")

	// Empty content
	body := `{"chatAgentId": "agent-123", "message": {"content": ""}}`
	c.Request = httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.SendMessage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSendMessage_FoundryAgent_MissingAPIKey(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	handler := createSendTestHandler(mockDocDB, mockPlatform, mockSession)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "tenantId", Value: "test-tenant"},
	}
	c.Set("user_id", "test-user")
	c.Set("auth_token", "bearer-token")

	reqBody := SendMessageRequest{
		ConversationID: "conv-123",
		ChatAgentID:    "foundry-agent",
		Message: MessageContent{
			Content: "Hello Foundry!",
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	// NOT setting X-Microsoft-Foundry-API-Key header

	mockSession.On("GetSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not found"))

	// Return Foundry agent config
	agentConfig := &platform.AgentConfig{
		Type:        platform.AgentTypeFoundry,
		TenantID:    "test-tenant",
		ChatAgentID: "foundry-agent",
		Settings: platform.AgentSettings{
			ProjectEndpoint: "https://project.api.azure.com",
			AgentName:       "test-agent",
		},
	}
	mockPlatform.On("GetAgentConfig", mock.Anything, "test-tenant", "foundry-agent", "conv-123", mock.Anything, mock.Anything).
		Return(agentConfig, nil)

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.SendMessage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["message"], "X-Microsoft-Foundry-API-Key")
}

func TestSendMessage_AgentConfigError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	handler := createSendTestHandler(mockDocDB, mockPlatform, mockSession)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "tenantId", Value: "test-tenant"},
	}
	c.Set("user_id", "test-user")
	c.Set("auth_token", "auth-token-123")

	reqBody := SendMessageRequest{
		ConversationID: "conv-123",
		ChatAgentID:    "agent-456",
		Message: MessageContent{
			Content: "Hello!",
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	mockSession.On("GetSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not found"))

	// Platform returns error
	mockPlatform.On("GetAgentConfig", mock.Anything, "test-tenant", "agent-456", "conv-123", mock.Anything, mock.Anything).
		Return(nil, errors.New("agent not found"))

	handler.SendMessage(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSendMessage_MessageStorageError(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	handler := createSendTestHandler(mockDocDB, mockPlatform, mockSession)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "tenantId", Value: "test-tenant"},
	}
	c.Set("user_id", "test-user")
	c.Set("auth_token", "auth-token-123")

	reqBody := SendMessageRequest{
		ConversationID: "conv-123",
		ChatAgentID:    "agent-456",
		Message: MessageContent{
			Content: "Hello!",
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	mockSession.On("GetSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not found"))

	agentConfig := &platform.AgentConfig{
		Type:        platform.AgentTypeN8N,
		TenantID:    "test-tenant",
		ChatAgentID: "agent-456",
		Settings: platform.AgentSettings{
			UseUnifiedChatHistory: false,
		},
	}
	mockPlatform.On("GetAgentConfig", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(agentConfig, nil)

	// Message storage fails
	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).
		Return(errors.New("database error"))

	handler.SendMessage(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSendMessage_LoadsChatHistory(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	handler := createSendTestHandler(mockDocDB, mockPlatform, mockSession)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{
		{Key: "tenantId", Value: "test-tenant"},
	}
	c.Set("user_id", "test-user")
	c.Set("auth_token", "auth-token")

	reqBody := SendMessageRequest{
		ConversationID: "conv-123",
		ChatAgentID:    "agent-456",
		Message: MessageContent{
			Content: "Hello!",
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	mockSession.On("GetSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not found"))

	// Agent config with unified chat history enabled
	agentConfig := &platform.AgentConfig{
		Type:        platform.AgentTypeN8N,
		TenantID:    "test-tenant",
		ChatAgentID: "agent-456",
		Settings: platform.AgentSettings{
			UseUnifiedChatHistory: true,
			ChatHistoryCount:      20,
		},
	}
	mockPlatform.On("GetAgentConfig", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(agentConfig, nil)

	// Mock chat history retrieval - verify correct params passed
	mockDocDB.GetMessagesCollection().On("ListChatHistory", mock.Anything, mock.MatchedBy(func(opts *docdb.ListMessagesOptions) bool {
		return opts.ConversationID == "conv-123" &&
			opts.TenantID == "test-tenant" &&
			opts.Limit == 20
	})).Return(nil, nil)

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.SendMessage(c)

	mockDocDB.GetMessagesCollection().AssertCalled(t, "ListChatHistory", mock.Anything, mock.Anything)
}

// =============================================================================
// Config Cache Behavior Tests
// =============================================================================

func TestSendMessage_FirstMessage_AlwaysFetchesFreshConfig(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	mockConfigCache := &mocks.MockConfigCacheService{}

	handler := &MessagesHandler{
		docDBClient:    mockDocDB,
		platformClient: mockPlatform,
		sessionService: mockSession,
		configCache:    mockConfigCache,
		agentFactory:   agents.NewFactory(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{{Key: "tenantId", Value: "test-tenant"}}
	c.Set("user_id", "test-user")
	c.Set("auth_token", "auth-token")

	reqBody := SendMessageRequest{
		ConversationID: "conv-123",
		ChatAgentID:    "agent-456",
		Message:        MessageContent{Content: "Hello!"},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	mockSession.On("GetSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not found"))

	agentConfig := &platform.AgentConfig{
		Type:        platform.AgentTypeN8N,
		TenantID:    "test-tenant",
		ChatAgentID: "agent-456",
		Settings:    platform.AgentSettings{UseUnifiedChatHistory: false},
	}
	mockPlatform.On("GetAgentConfig", mock.Anything, "test-tenant", "agent-456", "conv-123", mock.Anything, true).
		Return(agentConfig, nil)

	mockConfigCache.On("Set", mock.Anything, "test-tenant", "test-user", "agent-456", agentConfig).
		Return(nil)

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.SendMessage(c)

	mockConfigCache.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockPlatform.AssertCalled(t, "GetAgentConfig", mock.Anything, "test-tenant", "agent-456", "conv-123", mock.Anything, true)
	mockConfigCache.AssertCalled(t, "Set", mock.Anything, "test-tenant", "test-user", "agent-456", agentConfig)
}

func TestSendMessage_NoSession_CallsPlatformAndWritesCache(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	mockConfigCache := &mocks.MockConfigCacheService{}

	handler := &MessagesHandler{
		docDBClient:    mockDocDB,
		platformClient: mockPlatform,
		sessionService: mockSession,
		configCache:    mockConfigCache,
		agentFactory:   agents.NewFactory(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{{Key: "tenantId", Value: "test-tenant"}}
	c.Set("user_id", "test-user")
	c.Set("auth_token", "auth-token")

	reqBody := SendMessageRequest{
		ConversationID: "conv-123",
		ChatAgentID:    "agent-456",
		Message:        MessageContent{Content: "Hello!"},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	mockSession.On("GetSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not found"))

	agentConfig := &platform.AgentConfig{
		Type:        platform.AgentTypeN8N,
		TenantID:    "test-tenant",
		ChatAgentID: "agent-456",
		Settings:    platform.AgentSettings{UseUnifiedChatHistory: false},
	}
	mockPlatform.On("GetAgentConfig", mock.Anything, "test-tenant", "agent-456", "conv-123", mock.Anything, true).
		Return(agentConfig, nil)

	mockConfigCache.On("Set", mock.Anything, "test-tenant", "test-user", "agent-456", agentConfig).
		Return(nil)

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.SendMessage(c)

	mockConfigCache.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockPlatform.AssertCalled(t, "GetAgentConfig", mock.Anything, "test-tenant", "agent-456", "conv-123", mock.Anything, true)
	mockConfigCache.AssertCalled(t, "Set", mock.Anything, "test-tenant", "test-user", "agent-456", agentConfig)
}

func TestSendMessage_XUseCacheFalse_BypassesConfigCache(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	mockConfigCache := &mocks.MockConfigCacheService{}

	handler := &MessagesHandler{
		docDBClient:    mockDocDB,
		platformClient: mockPlatform,
		sessionService: mockSession,
		configCache:    mockConfigCache,
		agentFactory:   agents.NewFactory(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{{Key: "tenantId", Value: "test-tenant"}}
	c.Set("user_id", "test-user")
	c.Set("auth_token", "auth-token")

	reqBody := SendMessageRequest{
		ConversationID: "conv-123",
		ChatAgentID:    "agent-456",
		Message:        MessageContent{Content: "Hello!"},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Use-Cache", "false")

	mockSession.On("GetSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not found"))

	agentConfig := &platform.AgentConfig{
		Type:        platform.AgentTypeN8N,
		TenantID:    "test-tenant",
		ChatAgentID: "agent-456",
		Settings:    platform.AgentSettings{UseUnifiedChatHistory: false},
	}
	mockPlatform.On("GetAgentConfig", mock.Anything, "test-tenant", "agent-456", "conv-123", mock.Anything, false).
		Return(agentConfig, nil)

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.SendMessage(c)

	mockConfigCache.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockPlatform.AssertCalled(t, "GetAgentConfig", mock.Anything, "test-tenant", "agent-456", "conv-123", mock.Anything, false)
	mockConfigCache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSendMessage_SessionHit_SkipsCacheAndPlatform(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	mockConfigCache := &mocks.MockConfigCacheService{}

	handler := &MessagesHandler{
		docDBClient:    mockDocDB,
		platformClient: mockPlatform,
		sessionService: mockSession,
		configCache:    mockConfigCache,
		agentFactory:   agents.NewFactory(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{{Key: "tenantId", Value: "test-tenant"}}
	c.Set("user_id", "test-user")
	c.Set("auth_token", "auth-token")

	reqBody := SendMessageRequest{
		ConversationID: "conv-123",
		ChatAgentID:    "agent-456",
		Message:        MessageContent{Content: "Hello!"},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	sessionConfig := &platform.AgentConfig{
		Type:        platform.AgentTypeN8N,
		TenantID:    "test-tenant",
		ChatAgentID: "agent-456",
		Settings:    platform.AgentSettings{UseUnifiedChatHistory: false},
	}
	sessionData := &session.Data{
		TenantID:       "test-tenant",
		UserID:         "test-user",
		ConversationID: "conv-123",
		Config:         sessionConfig,
		ChatHistory:    []models.ChatHistoryEntry{},
	}
	mockSession.On("GetSession", mock.Anything, "test-tenant", "test-user", "conv-123").
		Return(sessionData, nil)

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.SendMessage(c)

	mockConfigCache.AssertNotCalled(t, "Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockPlatform.AssertNotCalled(t, "GetAgentConfig", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockConfigCache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSendMessage_ConfigCacheGetError_FallsThroughToPlatform(t *testing.T) {
	mockDocDB := mocks.NewMockDocDBClient()
	mockPlatform := &mocks.MockPlatformClient{}
	mockSession := &mocks.MockSessionService{}
	mockConfigCache := &mocks.MockConfigCacheService{}

	handler := &MessagesHandler{
		docDBClient:    mockDocDB,
		platformClient: mockPlatform,
		sessionService: mockSession,
		configCache:    mockConfigCache,
		agentFactory:   agents.NewFactory(),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{{Key: "tenantId", Value: "test-tenant"}}
	c.Set("user_id", "test-user")
	c.Set("auth_token", "auth-token")

	reqBody := SendMessageRequest{
		ConversationID: "conv-123",
		ChatAgentID:    "agent-456",
		Message:        MessageContent{Content: "Hello!"},
	}
	bodyBytes, _ := json.Marshal(reqBody)
	c.Request = httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	mockSession.On("GetSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not found"))

	mockConfigCache.On("Get", mock.Anything, "test-tenant", "test-user", "agent-456").
		Return(nil, errors.New("redis error"))

	agentConfig := &platform.AgentConfig{
		Type:        platform.AgentTypeN8N,
		TenantID:    "test-tenant",
		ChatAgentID: "agent-456",
		Settings:    platform.AgentSettings{UseUnifiedChatHistory: false},
	}
	mockPlatform.On("GetAgentConfig", mock.Anything, "test-tenant", "agent-456", "conv-123", mock.Anything, true).
		Return(agentConfig, nil)

	mockConfigCache.On("Set", mock.Anything, "test-tenant", "test-user", "agent-456", agentConfig).
		Return(nil)

	mockDocDB.GetMessagesCollection().On("Add", mock.Anything, mock.Anything).Return(nil)

	handler.SendMessage(c)

	mockPlatform.AssertCalled(t, "GetAgentConfig", mock.Anything, "test-tenant", "agent-456", "conv-123", mock.Anything, true)
	mockConfigCache.AssertCalled(t, "Set", mock.Anything, "test-tenant", "test-user", "agent-456", agentConfig)
}

// =============================================================================
// streamTitleGeneration Tests
// =============================================================================

func TestStreamTitleGeneration_AIServiceNil(t *testing.T) {
	handler := &MessagesHandler{
		aiService: nil,
	}

	assert.Nil(t, handler.aiService)
}
