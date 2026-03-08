package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/api/handlers"
	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/api/routes"
	"github.com/unifiedui/agent-service/internal/config"
	"github.com/unifiedui/agent-service/internal/core/cache"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/core/vault"
	"github.com/unifiedui/agent-service/internal/services/ai"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// --- Mock implementations ---

type mockCacheClient struct{}

func (m *mockCacheClient) GetCache() cache.Cache                           { return nil }
func (m *mockCacheClient) Ping(_ context.Context) error                    { return nil }
func (m *mockCacheClient) Get(_ context.Context, _ string) ([]byte, error) { return nil, nil }
func (m *mockCacheClient) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}
func (m *mockCacheClient) Delete(_ context.Context, _ string) (bool, error)         { return true, nil }
func (m *mockCacheClient) DeletePattern(_ context.Context, _ string) (int64, error) { return 0, nil }
func (m *mockCacheClient) Close() error                                             { return nil }

type mockDocDBClient struct{}

func (m *mockDocDBClient) Database() docdb.Database              { return nil }
func (m *mockDocDBClient) Messages() docdb.MessagesCollection    { return nil }
func (m *mockDocDBClient) MessagesRaw() docdb.Collection         { return nil }
func (m *mockDocDBClient) Reactions() docdb.ReactionsCollection  { return nil }
func (m *mockDocDBClient) Traces() docdb.TracesCollection        { return nil }
func (m *mockDocDBClient) TracesRaw() docdb.Collection           { return nil }
func (m *mockDocDBClient) Ping(_ context.Context) error          { return nil }
func (m *mockDocDBClient) Close(_ context.Context) error         { return nil }
func (m *mockDocDBClient) EnsureIndexes(_ context.Context) error { return nil }

type mockPlatformClient struct{}

func (m *mockPlatformClient) GetChatAgentConfig(_ context.Context, _, _, _ string, _ bool) (*platform.ChatAgentConfigResponse, error) {
	return nil, nil
}
func (m *mockPlatformClient) GetAgentConfig(_ context.Context, _, _, _, _ string, _ bool) (*platform.AgentConfig, error) {
	return nil, nil
}
func (m *mockPlatformClient) GetAgentConfigFromFile(_ context.Context, _, _ string) (*platform.AgentConfig, error) {
	return nil, nil
}
func (m *mockPlatformClient) GetMe(_ context.Context, _ string) (*platform.UserInfo, error) {
	return nil, nil
}
func (m *mockPlatformClient) GetConversation(_ context.Context, _, _, _ string) (*platform.ConversationResponse, error) {
	return nil, nil
}
func (m *mockPlatformClient) ValidateConversation(_ context.Context, _, _, _ string) error {
	return nil
}
func (m *mockPlatformClient) ValidateAutonomousAgent(_ context.Context, _, _, _ string) error {
	return nil
}
func (m *mockPlatformClient) GetAutonomousAgentConfig(_ context.Context, _, _, _ string) (*platform.AutonomousAgentConfigResponse, error) {
	return nil, nil
}
func (m *mockPlatformClient) GetAutonomousAgentConfigWithBearer(_ context.Context, _, _, _ string) (*platform.AutonomousAgentConfigResponse, error) {
	return nil, nil
}
func (m *mockPlatformClient) ValidateAutonomousAgentAPIKey(_ context.Context, _, _, _ string) error {
	return nil
}
func (m *mockPlatformClient) GetAIModelsByPurpose(_ context.Context, _, _, _ string) ([]platform.AIModelWithSecretResponse, error) {
	return nil, nil
}
func (m *mockPlatformClient) GetCredentialSecret(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}
func (m *mockPlatformClient) UpdateConversationTitle(_ context.Context, _, _, _, _ string) error {
	return nil
}

type mockAIService struct{}

func (m *mockAIService) GenerateTitle(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}
func (m *mockAIService) GenerateDescription(_ context.Context, _, _, _, _ string, _ map[string]interface{}) (string, error) {
	return "", nil
}
func (m *mockAIService) AnalyzeTrace(_ context.Context, _ string, _ ai.AnalyzeTraceInput) (string, error) {
	return "", nil
}
func (m *mockAIService) SummarizeTrace(_ context.Context, _ string, _ ai.SummarizeTraceInput) (string, error) {
	return "", nil
}
func (m *mockAIService) TraceChat(_ context.Context, _ string, _ ai.TraceChatInput) (string, error) {
	return "", nil
}
func (m *mockAIService) TestModel(_ context.Context, _ string, _, _ map[string]interface{}) (*ai.TestModelResult, error) {
	return nil, nil
}
func (m *mockAIService) GetCapabilities(_ context.Context, _ string) (*ai.Capabilities, error) {
	return nil, nil
}

type mockVaultClient struct{}

func (m *mockVaultClient) GetVault() vault.Vault          { return nil }
func (m *mockVaultClient) BuildSecretURI(_ string) string { return "" }
func (m *mockVaultClient) StoreSecret(_ context.Context, _, _ string, _ map[string]string) (string, error) {
	return "", nil
}
func (m *mockVaultClient) GetSecret(_ context.Context, _ string, _ bool) (string, error) {
	return "", nil
}
func (m *mockVaultClient) UpdateSecret(_ context.Context, _, _ string, _ map[string]string) (bool, error) {
	return false, nil
}
func (m *mockVaultClient) DeleteSecret(_ context.Context, _ string) (bool, error) { return false, nil }
func (m *mockVaultClient) Ping(_ context.Context) error                           { return nil }
func (m *mockVaultClient) Close() error                                           { return nil }

// --- Test helpers ---

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func createTestConfig() *routes.Config {
	healthHandler := handlers.NewHealthHandler(&mockCacheClient{}, &mockDocDBClient{})
	authMiddleware := middleware.NewAuthMiddleware("http://test-platform-service")

	return &routes.Config{
		HealthHandler:    healthHandler,
		MessagesHandler:  nil,
		ReactionsHandler: nil,
		TracesHandler:    nil,
		DataHandler:      nil,
		AIHandler:        nil,
		AuthMiddleware:   authMiddleware,
		ServiceKeyMw:     nil,
	}
}

// --- Setup Tests ---

func TestSetup_ReturnsNonNilRouter(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()

	routes.Setup(router, cfg)

	assert.NotNil(t, router)
}

func TestSetup_RegistersHealthRoutes(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()
	routes.Setup(router, cfg)

	testCases := []struct {
		name   string
		path   string
		method string
	}{
		{"health endpoint", "/api/v1/agent-service/health", http.MethodGet},
		{"ready endpoint", "/api/v1/agent-service/ready", http.MethodGet},
		{"live endpoint", "/api/v1/agent-service/live", http.MethodGet},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should not return 404 (route exists)
			assert.NotEqual(t, http.StatusNotFound, w.Code, "route %s should exist", tc.path)
		})
	}
}

func TestSetup_RegistersTenantRoutes(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()
	routes.Setup(router, cfg)

	// These routes require auth so they'll return 401, but not 404
	testCases := []struct {
		name   string
		path   string
		method string
	}{
		{"conversation messages GET", "/api/v1/agent-service/tenants/test-tenant/conversation/messages", http.MethodGet},
		{"conversation messages POST", "/api/v1/agent-service/tenants/test-tenant/conversation/messages", http.MethodPost},
		{"conversation traces", "/api/v1/agent-service/tenants/test-tenant/conversations/conv-123/traces", http.MethodGet},
		{"autonomous agent traces list", "/api/v1/agent-service/tenants/test-tenant/autonomous-agents/traces", http.MethodGet},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should return 401 (unauthorized) not 404 (route exists but needs auth)
			assert.NotEqual(t, http.StatusNotFound, w.Code, "route %s should exist", tc.path)
		})
	}
}

func TestSetup_RegistersTraceRoutes(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()
	routes.Setup(router, cfg)

	testCases := []struct {
		name   string
		path   string
		method string
	}{
		{"get trace", "/api/v1/agent-service/tenants/test-tenant/traces/trace-123", http.MethodGet},
		{"delete trace", "/api/v1/agent-service/tenants/test-tenant/traces/trace-123", http.MethodDelete},
		{"create trace", "/api/v1/agent-service/tenants/test-tenant/traces", http.MethodPost},
		{"add nodes", "/api/v1/agent-service/tenants/test-tenant/traces/trace-123/nodes", http.MethodPost},
		{"add logs", "/api/v1/agent-service/tenants/test-tenant/traces/trace-123/logs", http.MethodPost},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should not return 404
			assert.NotEqual(t, http.StatusNotFound, w.Code, "route %s should exist", tc.path)
		})
	}
}

func TestSetup_RegistersAutonomousAgentRoutes(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()
	routes.Setup(router, cfg)

	testCases := []struct {
		name   string
		path   string
		method string
	}{
		{"agent traces GET", "/api/v1/agent-service/tenants/test-tenant/autonomous-agents/agent-123/traces", http.MethodGet},
		{"agent traces PUT", "/api/v1/agent-service/tenants/test-tenant/autonomous-agents/agent-123/traces", http.MethodPut},
		{"agent import traces", "/api/v1/agent-service/tenants/test-tenant/autonomous-agents/agent-123/traces/import", http.MethodPut},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should not return 404
			assert.NotEqual(t, http.StatusNotFound, w.Code, "route %s should exist", tc.path)
		})
	}
}

func TestSetup_HealthEndpointsAccessibleWithoutAuth(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()
	routes.Setup(router, cfg)

	// Health endpoints should be accessible without authentication
	testCases := []struct {
		name string
		path string
	}{
		{"health", "/api/v1/agent-service/health"},
		{"ready", "/api/v1/agent-service/ready"},
		{"live", "/api/v1/agent-service/live"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, http.NoBody)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should return 200 (mocked services return healthy)
			assert.Equal(t, http.StatusOK, w.Code, "health endpoint %s should be accessible", tc.path)
		})
	}
}

func TestSetup_ProtectedRoutesRequireAuth(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()
	routes.Setup(router, cfg)

	// These routes should require authorization
	protectedPaths := []struct {
		name   string
		path   string
		method string
	}{
		{"messages GET", "/api/v1/agent-service/tenants/test-tenant/conversation/messages", http.MethodGet},
		{"messages POST", "/api/v1/agent-service/tenants/test-tenant/conversation/messages", http.MethodPost},
	}

	for _, tc := range protectedPaths {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should return 401 without auth header
			assert.Equal(t, http.StatusUnauthorized, w.Code, "route %s should require auth", tc.path)
		})
	}
}

func TestSetup_NonExistentRouteReturns404(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()
	routes.Setup(router, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent-service/non-existent-route", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- SetupWithMiddleware Tests ---

func TestSetupWithMiddleware_AppliesMiddleware(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()

	loggingMw := middleware.NewLoggingMiddlewareWithLogger(zerolog.Nop())
	errorMw := middleware.NewErrorMiddleware()

	routes.SetupWithMiddleware(router, cfg, loggingMw, errorMw)

	// Verify router is configured and routes work
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent-service/health", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSetupWithMiddleware_RecoverFromPanic(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()

	loggingMw := middleware.NewLoggingMiddlewareWithLogger(zerolog.Nop())
	errorMw := middleware.NewErrorMiddleware()

	routes.SetupWithMiddleware(router, cfg, loggingMw, errorMw)

	// Add a route that panics to test recovery
	router.GET("/test-panic", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/test-panic", http.NoBody)
	w := httptest.NewRecorder()

	// Should not panic, middleware should recover
	require.NotPanics(t, func() {
		router.ServeHTTP(w, req)
	})

	// Should return 500 after recovery
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSetupWithMiddleware_RoutesStillWork(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()

	loggingMw := middleware.NewLoggingMiddlewareWithLogger(zerolog.Nop())
	errorMw := middleware.NewErrorMiddleware()

	routes.SetupWithMiddleware(router, cfg, loggingMw, errorMw)

	// Test that all route groups are still registered after middleware setup
	testCases := []struct {
		name           string
		path           string
		method         string
		expectedNot404 bool
	}{
		{"health", "/api/v1/agent-service/health", http.MethodGet, true},
		{"ready", "/api/v1/agent-service/ready", http.MethodGet, true},
		{"live", "/api/v1/agent-service/live", http.MethodGet, true},
		{"tenant routes", "/api/v1/agent-service/tenants/test/conversation/messages", http.MethodGet, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tc.expectedNot404 {
				assert.NotEqual(t, http.StatusNotFound, w.Code, "route %s should exist", tc.path)
			}
		})
	}
}

// --- Route Group Registration Tests ---

func TestSetup_DoesNotRegisterAIRoutesWhenHandlerNil(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()
	cfg.AIHandler = nil // Explicitly nil

	routes.Setup(router, cfg)

	// AI routes should return 404 when AIHandler is nil
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-service/tenants/test-tenant/ai/generate-description", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "AI routes should not be registered when handler is nil")
}

func TestSetup_DoesNotRegisterReactionRoutesWhenHandlerNil(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()
	cfg.ReactionsHandler = nil // Explicitly nil

	routes.Setup(router, cfg)

	// Reaction routes should return 404 when ReactionsHandler is nil
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent-service/tenants/test-tenant/conversations/conv-123/messages/msg-123/reactions", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "Reaction routes should not be registered when handler is nil")
}

func TestSetup_DoesNotRegisterServiceKeyRoutesWhenMiddlewareNil(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()
	cfg.ServiceKeyMw = nil
	cfg.DataHandler = nil

	routes.Setup(router, cfg)

	// Service key routes should return 404
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/agent-service/tenants/test-tenant/conversations/conv-123/data", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code, "Service key routes should not be registered when middleware/handler is nil")
}

func TestSetup_RegistersReactionRoutesWhenHandlerProvided(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()
	cfg.ReactionsHandler = handlers.NewReactionsHandler(&mockDocDBClient{}, &mockPlatformClient{})

	routes.Setup(router, cfg)

	testCases := []struct {
		name   string
		path   string
		method string
	}{
		{"upsert reaction", "/api/v1/agent-service/tenants/test-tenant/conversations/conv-123/messages/msg-123/reactions", http.MethodPost},
		{"delete reaction", "/api/v1/agent-service/tenants/test-tenant/conversations/conv-123/messages/msg-123/reactions", http.MethodDelete},
		{"get reactions", "/api/v1/agent-service/tenants/test-tenant/conversations/conv-123/messages/msg-123/reactions", http.MethodGet},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.NotEqual(t, http.StatusNotFound, w.Code, "reaction route %s should be registered", tc.path)
		})
	}
}

func TestSetup_RegistersAIRoutesWhenHandlerProvided(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()
	cfg.AIHandler = handlers.NewAIHandler(&mockAIService{}, &mockPlatformClient{})

	routes.Setup(router, cfg)

	testCases := []struct {
		name   string
		path   string
		method string
	}{
		{"generate description", "/api/v1/agent-service/tenants/test-tenant/ai/generate-description", http.MethodPost},
		{"analyze trace", "/api/v1/agent-service/tenants/test-tenant/ai/analyze-trace", http.MethodPost},
		{"summarize trace", "/api/v1/agent-service/tenants/test-tenant/ai/summarize-trace", http.MethodPost},
		{"trace chat", "/api/v1/agent-service/tenants/test-tenant/ai/trace-chat", http.MethodPost},
		{"test model", "/api/v1/agent-service/tenants/test-tenant/ai/test-model", http.MethodPost},
		{"get capabilities", "/api/v1/agent-service/tenants/test-tenant/ai/capabilities", http.MethodGet},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.NotEqual(t, http.StatusNotFound, w.Code, "AI route %s should be registered", tc.path)
		})
	}
}

func TestSetup_RegistersServiceKeyRoutesWhenProvided(t *testing.T) {
	router := setupTestRouter()
	cfg := createTestConfig()
	cfg.ServiceKeyMw = middleware.NewServiceKeyMiddleware(&mockVaultClient{}, config.AppVaultConfig{})
	cfg.DataHandler = handlers.NewDataHandler(&mockDocDBClient{})

	routes.Setup(router, cfg)

	testCases := []struct {
		name   string
		path   string
		method string
	}{
		{"delete conversation data", "/api/v1/agent-service/tenants/test-tenant/conversations/conv-123/data", http.MethodDelete},
		{"delete agent data", "/api/v1/agent-service/tenants/test-tenant/autonomous-agents/agent-123/data", http.MethodDelete},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.NotEqual(t, http.StatusNotFound, w.Code, "service key route %s should be registered", tc.path)
		})
	}
}
