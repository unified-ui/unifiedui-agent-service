# Testing

## Overview

- **Framework**: Go `testing` package + `testify` (assert, require, mock)
- **Run all tests**: `make test` → `go test -v ./...`
- **Run with coverage**: `make test-cover` → generates `coverage.html`
- **Coverage target**: 85%

---

## Directory Structure

```
tests/
├── mocks/                              # testify mocks for all interfaces
│   ├── ai_service_mock.go              # MockAIService
│   ├── cache_mock.go                   # MockCacheClient
│   ├── docdb_mock.go                   # MockDocDBClient + MockTracesCollection + MockMessagesCollection
│   ├── encryption_mock.go              # MockEncryptor
│   ├── platform_mock.go               # MockPlatformClient
│   ├── session_mock.go                # MockSessionService
│   ├── traceimport_mock.go            # MockImportService
│   └── vault_mock.go                  # MockVaultClient
├── testutils/
│   ├── fixtures.go                    # Test constants + factory functions
│   └── helpers.go                     # Router setup, HTTP helpers, assertions
└── unit/
    ├── encryption/                    # Encryption tests
    ├── handlers/                      # Handler tests
    │   ├── traces_test.go
    │   └── health_test.go
    ├── infrastructure/                # Infrastructure implementation tests
    ├── models/                        # Model tests
    ├── pkg/                           # Package utility tests
    ├── services/
    │   ├── agents/                    # Agent factory/client tests
    │   ├── platform/                  # Platform client tests
    │   └── traceimport/               # Trace import tests
    └── session/                       # Session service tests
```

---

## Test Naming Convention

```go
func TestTypeName_MethodName_Scenario(t *testing.T) {
    // Arrange
    // Act
    // Assert
}
```

Examples:
- `TestTracesHandler_CreateTrace_Success`
- `TestTracesHandler_CreateTrace_MixedContext_Error`
- `TestTracesHandler_CreateTrace_ConversationAlreadyExists_Conflict`
- `TestTracesHandler_AddNodes_TraceNotFound`
- `TestHealthHandler_Health_Success`

---

## Test Fixtures (`tests/testutils/fixtures.go`)

### Constants

```go
const (
    TestTenantID       = "test-tenant-id"
    TestConversationID = "test-conversation-id"
    TestMessageID      = "test-message-id"
    TestApplicationID  = "test-application-id"
    TestUserID         = "test-user-id"
    TestTraceID        = "test-trace-id"
)
```

### Factory Functions

```go
func NewTestUserMessage() *models.Message { ... }
func NewTestAssistantMessage() *models.Message { ... }
func NewTestTrace() *models.Trace { ... }
func NewTestToolTrace() *models.Trace { ... }
func NewTestTraces(count int) []*models.Trace { ... }
func NewTestSession() *models.Session { ... }
```

Always use fixtures for test data. Never hardcode IDs or create models inline.

---

## Test Helpers (`tests/testutils/helpers.go`)

### Router Setup

```go
func SetupTestRouter() *gin.Engine {
    gin.SetMode(gin.TestMode)
    return gin.New()
}
```

### HTTP Request Execution

```go
func PerformRequest(router *gin.Engine, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder
```

This marshals the body to JSON, creates the request, adds headers, and executes against the router.

### Response Assertions

```go
func AssertStatusCode(t *testing.T, expected int, w *httptest.ResponseRecorder)
func ParseJSONResponse(t *testing.T, w *httptest.ResponseRecorder, v interface{})
```

### Context Helpers

```go
func NewTestContext() (*gin.Context, *httptest.ResponseRecorder)
func NewTestContextWithRequest(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder)
func SetPathParams(c *gin.Context, params map[string]string)
func SetAuthToken(c *gin.Context, token string)
func TestContext() context.Context  // returns context.Background()
```

---

## Mock Pattern

All mocks use `testify/mock`. Located in `tests/mocks/`.

### MockPlatformClient Example

```go
type MockPlatformClient struct {
    mock.Mock
}

func (m *MockPlatformClient) GetMe(ctx context.Context, authToken string) (*platform.UserInfo, error) {
    args := m.Called(ctx, authToken)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*platform.UserInfo), args.Error(1)
}

// Interface compliance check
var _ platform.Client = (*MockPlatformClient)(nil)
```

### MockDocDBClient

Special pattern — exposes typed sub-collection mocks:

```go
mockDocDB := mocks.NewMockDocDBClient()

// Set up expectations on sub-collections:
mockDocDB.GetTracesCollection().On("Get", mock.Anything, mock.Anything).Return(existingTrace, nil)
mockDocDB.GetTracesCollection().On("Create", mock.Anything, mock.Anything).Return(nil)

// Verify expectations:
mockDocDB.GetTracesCollection().AssertExpectations(t)
```

---

## Handler Test Pattern

Complete test structure for handler tests:

```go
func TestTracesHandler_CreateTrace_Conversation_Success(t *testing.T) {
    // 1. Setup mocks
    mockDocDB := mocks.NewMockDocDBClient()
    mockPlatform := &mocks.MockPlatformClient{}

    // 2. Create request DTO
    createReq := dto.CreateTraceRequest{
        ApplicationID:  testutils.TestApplicationID,
        ConversationID: testutils.TestConversationID,
        ReferenceID:    "workflow-123",
        ReferenceName:  "Test Workflow",
    }

    // 3. Set mock expectations
    mockPlatform.On("GetMe", mock.Anything, mock.Anything).Return(&platform.UserInfo{ID: testutils.TestUserID}, nil)
    mockPlatform.On("ValidateConversation", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
    mockDocDB.GetTracesCollection().On("GetByConversation", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
    mockDocDB.GetTracesCollection().On("Create", mock.Anything, mock.Anything).Return(nil)

    // 4. Create handler with mocks
    handler := createTestTracesHandler(mockDocDB, mockPlatform)

    // 5. Setup test router with middleware
    router := testutils.SetupTestRouter()
    router.Use(flexibleAuthTestMiddleware())  // Simulates auth middleware
    router.POST("/tenants/:tenantId/traces", handler.CreateTrace)

    // 6. Execute request
    headers := map[string]string{"Authorization": "Bearer test-token"}
    w := testutils.PerformRequest(router, "POST", "/tenants/"+testutils.TestTenantID+"/traces", createReq, headers)

    // 7. Assert response
    testutils.AssertStatusCode(t, http.StatusCreated, w)

    var response dto.TraceResponse
    testutils.ParseJSONResponse(t, w, &response)
    assert.Equal(t, testutils.TestTenantID, response.TenantID)

    // 8. Verify mock expectations
    mockDocDB.GetTracesCollection().AssertExpectations(t)
    mockPlatform.AssertExpectations(t)
}
```

---

## Test Auth Middleware Helpers

Tests use simplified middleware that simulates auth without calling platform service:

```go
func flexibleAuthTestMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if token := c.GetHeader("Authorization"); token != "" {
            c.Set("auth_token", strings.TrimPrefix(token, "Bearer "))
        }
        if apiKey := c.GetHeader("X-Unified-UI-Autonomous-Agent-API-Key"); apiKey != "" {
            c.Set("autonomous_agent_api_key", apiKey)
        }
        c.Next()
    }
}

func autonomousAgentAPIKeyMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        apiKey := c.GetHeader("X-Unified-UI-Autonomous-Agent-API-Key")
        c.Set("autonomous_agent_api_key", apiKey)
        c.Next()
    }
}
```

---

## Test Helper Function

Each handler test file has a helper to create the handler with mocks:

```go
func createTestTracesHandler(mockDocDB *mocks.MockDocDBClient, mockPlatform *mocks.MockPlatformClient) *handlers.TracesHandler {
    return handlers.NewTracesHandler(mockDocDB, mockPlatform, nil)
}

func createTestMessagesHandler(mockDocDB *mocks.MockDocDBClient, mockPlatform *mocks.MockPlatformClient, mockSession *mocks.MockSessionService, mockAI *mocks.MockAIService) *handlers.MessagesHandler {
    return handlers.NewMessagesHandler(mockDocDB, mockPlatform, nil, mockSession, nil, mockAI)
}
```

---

## What to Test

### Handler Tests

For each handler method, test:
- **Success case** — valid input, expected response
- **Validation errors** — missing/invalid fields → 400
- **Not found** — resource doesn't exist → 404
- **Conflict** — duplicate creation → 409
- **Auth variations** — Bearer vs API key for flexible endpoints
- **Missing auth** — no token/key → 401

### Service Tests

- Business logic paths (success, edge cases, error handling)
- Factory creation (valid types, unsupported types)
- Interface compliance (`var _ Interface = (*Mock)(nil)`)

### Infrastructure Tests

- Factory pattern (create valid, unsupported type error)
- Connection/ping/close lifecycle

---

## Mock Creation Checklist

When adding a new interface:

1. Create `tests/mocks/{component}_mock.go`
2. Embed `mock.Mock` in struct
3. Implement all interface methods using `m.Called(...)` pattern
4. Add interface compliance check: `var _ SomeInterface = (*MockSomething)(nil)`
5. For DocDB sub-collections: add getter on `MockDocDBClient`

---

## Running Tests

```bash
# All tests
make test

# Specific package
go test -v ./tests/unit/handlers/...

# Specific test function
go test -v -run TestTracesHandler_CreateTrace_Success ./tests/unit/handlers/

# With coverage
make test-cover
# Opens coverage.html with per-file coverage

# Coverage percentage only
make test-cover-percent
```
