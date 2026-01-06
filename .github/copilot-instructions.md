# UnifiedUI Chat Service - Copilot Instructions

## Project Overview

**UnifiedUI Chat Service** is a high-performance Go/Gin microservice that serves as a unified abstraction layer between a single frontend and heterogeneous AI agent backends.

### Purpose
- Provide ONE unified chat interface for multiple AI agent systems
- Enable business users to interact with various agents through a consistent UI
- Aggregate and normalize traces from autonomous agents for monitoring/debugging
- Stream responses efficiently using SSE (Server-Sent Events)

### Supported Backend Integrations
- N8N Agent Workflows
- Microsoft Foundry
- Microsoft Copilot
- Custom REST API agents (LangChain, LangGraph, etc.)

---

## Architecture

```
┌─────────────┐     ┌──────────────────────┐     ┌─────────────────────┐
│  Frontend   │────▶│  UnifiedUI Service   │────▶│  Heterogene Backends│
│  (Unified)  │◀SSE─│  (Go/Gin)            │◀────│  N8N, Foundry, etc. │
└─────────────┘     └──────────────────────┘     └─────────────────────┘
                              │
                    ┌─────────┼─────────┐
                    ▼         ▼         ▼
              ┌─────────┐ ┌───────┐ ┌────────┐
              │Document │ │ Cache │ │ Vault  │
              │DB       │ │(Redis)│ │        │
              └─────────┘ └───────┘ └────────┘
```

### Authentication Flow
- Bearer token (MSAL) is passed through to Platform Service
- Platform Service provides config + credentials
- Errors from Platform Service are forwarded 1:1 to client
- Multi-tenancy and security handled entirely by Platform Service

---

## Tech Stack

- **Language**: Go 1.21+
- **Framework**: Gin (HTTP router)
- **Streaming**: SSE (Server-Sent Events)
- **Document DB**: MongoDB / CosmosDB (via Factory Pattern)
- **Cache**: Redis (via Factory Pattern)
- **Vault**: Azure KeyVault / HashiCorp Vault / DotEnvVault (via Factory Pattern)
- **Testing**: Go testing + testify + miniredis + mtest/dockertest

---

## Project Structure

```
unified-ui-agent-service/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── config/
│   │   ├── config.go               # Configuration loading
│   │   └── env.go                  # Environment variable handling
│   ├── core/
│   │   ├── cache/
│   │   │   ├── cache.go            # Cache interface
│   │   │   ├── client.go           # Cache client interface
│   │   │   └── factory.go          # Cache factory
│   │   ├── vault/
│   │   │   ├── vault.go            # Vault interface
│   │   │   ├── client.go           # Vault client interface
│   │   │   └── factory.go          # Vault factory
│   │   └── docdb/
│   │       ├── docdb.go            # Document DB interface
│   │       ├── client.go           # DocDB client interface
│   │       └── factory.go          # DocDB factory
│   ├── infrastructure/
│   │   ├── cache/
│   │   │   └── redis/
│   │   │       ├── cache.go        # Redis cache implementation
│   │   │       └── client.go       # Redis client implementation
│   │   ├── vault/
│   │   │   ├── azure/
│   │   │   │   └── keyvault.go     # Azure KeyVault implementation
│   │   │   ├── hashicorp/
│   │   │   │   └── vault.go        # HashiCorp Vault implementation
│   │   │   └── dotenv/
│   │   │       └── vault.go        # DotEnv vault (development)
│   │   └── docdb/
│   │       ├── mongodb/
│   │       │   ├── client.go       # MongoDB client
│   │       │   └── collections.go  # Collection implementations
│   │       └── cosmosdb/
│   │           └── client.go       # CosmosDB client
│   ├── domain/
│   │   ├── models/
│   │   │   ├── message.go          # Message domain model
│   │   │   ├── trace.go            # Trace domain model
│   │   │   ├── conversation.go     # Conversation domain model
│   │   │   └── session.go          # Session state model
│   │   └── errors/
│   │       └── errors.go           # Domain-specific errors
│   ├── services/
│   │   ├── platform/
│   │   │   ├── client.go           # Platform Service client
│   │   │   └── models.go           # Platform Service DTOs
│   │   ├── agents/
│   │   │   ├── factory.go          # Agent backend factory
│   │   │   ├── n8n/
│   │   │   │   └── client.go       # N8N agent client
│   │   │   ├── foundry/
│   │   │   │   └── client.go       # Microsoft Foundry client
│   │   │   └── langchain/
│   │   │       └── client.go       # LangChain REST client
│   │   ├── chat/
│   │   │   └── service.go          # Chat orchestration service
│   │   └── trace/
│   │       └── service.go          # Trace management service
│   ├── api/
│   │   ├── handlers/
│   │   │   ├── messages.go         # Message handlers
│   │   │   ├── traces.go           # Trace handlers
│   │   │   └── health.go           # Health check handlers
│   │   ├── middleware/
│   │   │   ├── auth.go             # Auth middleware (token forwarding)
│   │   │   ├── tenant.go           # Tenant context middleware
│   │   │   ├── logging.go          # Request logging
│   │   │   └── errors.go           # Error handling middleware
│   │   ├── dto/
│   │   │   ├── requests.go         # Request DTOs
│   │   │   └── responses.go        # Response DTOs
│   │   ├── routes/
│   │   │   └── routes.go           # Route definitions
│   │   └── sse/
│   │       └── writer.go           # SSE response writer
│   └── pkg/
│       ├── encryption/
│       │   └── fernet.go           # Credential encryption
│       └── httpclient/
│           └── client.go           # Reusable HTTP client
├── tests/
│   ├── unit/
│   │   ├── services/
│   │   ├── handlers/
│   │   └── infrastructure/
│   ├── integration/
│   │   └── .gitkeep                # Placeholder for integration tests
│   ├── mocks/
│   │   ├── cache_mock.go
│   │   ├── vault_mock.go
│   │   ├── docdb_mock.go
│   │   └── platform_mock.go
│   └── testutils/
│       ├── fixtures.go             # Test data fixtures
│       └── helpers.go              # Test helper functions
├── .github/
│   └── copilot-instructions.md     # This file
├── .env.example                    # Environment variable template
├── go.mod
├── go.sum
├── Makefile                        # Build and test commands
└── README.md
```

---

## API Endpoints

### Health Endpoints
```
GET  /api/v1/agent-service/health
GET  /api/v1/agent-service/ready
GET  /api/v1/agent-service/live
```

### Message Endpoints
```
GET  /api/v1/agent-service/tenants/{tenantId}/conversation/messages
POST /api/v1/agent-service/tenants/{tenantId}/conversation/messages
GET  /api/v1/agent-service/tenants/{tenantId}/conversation/messages/{messageId}/traces
```

### Traces Endpoints
```
# Core trace operations
POST   /api/v1/agent-service/tenants/{tenantId}/traces
GET    /api/v1/agent-service/tenants/{tenantId}/traces/{traceId}
DELETE /api/v1/agent-service/tenants/{tenantId}/traces/{traceId}
POST   /api/v1/agent-service/tenants/{tenantId}/traces/{traceId}/nodes
POST   /api/v1/agent-service/tenants/{tenantId}/traces/{traceId}/logs

# Conversation context traces
GET /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/traces
PUT /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/traces

# Autonomous agent context traces
GET  /api/v1/agent-service/tenants/{tenantId}/autonomous-agents/traces
GET  /api/v1/agent-service/tenants/{tenantId}/autonomous-agents/{agentId}/traces
PUT  /api/v1/agent-service/tenants/{tenantId}/autonomous-agents/{agentId}/traces
```

---

## Traces Architecture

The traces system imports and manages execution traces from external workflow runs (e.g., N8N, LangGraph, custom agents).

### Context Types
Traces support TWO mutually exclusive context types:

1. **Conversation Context**: Traces linked to a chat conversation
   - Requires: `applicationId` + `conversationId`
   - One trace per conversation (refresh replaces existing)
   - Use case: Tracking agent execution for a specific chat interaction

2. **Autonomous Agent Context**: Traces for scheduled/triggered agent runs
   - Requires: `autonomousAgentId`
   - Multiple traces per agent (list with pagination)
   - Use case: Monitoring background agent executions

### Trace Model
```go
type Trace struct {
    ID                string       `bson:"_id"`
    TenantID          string       `bson:"tenant_id"`
    ContextType       TraceContext `bson:"context_type"` // "conversation" | "autonomous_agent"
    
    // Conversation context (mutually exclusive with AutonomousAgentID)
    ApplicationID     string       `bson:"application_id,omitempty"`
    ConversationID    string       `bson:"conversation_id,omitempty"`
    
    // Autonomous agent context
    AutonomousAgentID string       `bson:"autonomous_agent_id,omitempty"`
    
    // Reference to external workflow
    ReferenceID       string       `bson:"reference_id"`
    ReferenceName     string       `bson:"reference_name"`
    ReferenceMetadata interface{}  `bson:"reference_metadata,omitempty"`
    
    // Hierarchical execution tree
    Nodes             []TraceNode  `bson:"nodes"`
    Logs              []interface{}`bson:"logs"`
    
    // Audit fields
    CreatedAt         time.Time    `bson:"created_at"`
    UpdatedAt         time.Time    `bson:"updated_at"`
    CreatedBy         string       `bson:"created_by"`
    UpdatedBy         string       `bson:"updated_by"`
}

type TraceNode struct {
    ID        string      `bson:"id"`
    Name      string      `bson:"name"`
    Type      NodeType    `bson:"type"`     // llm | tool | agent | chain | other
    Status    NodeStatus  `bson:"status"`   // pending | running | completed | failed
    Input     interface{} `bson:"input,omitempty"`
    Output    interface{} `bson:"output,omitempty"`
    Error     string      `bson:"error,omitempty"`
    StartAt   *time.Time  `bson:"start_at,omitempty"`
    EndAt     *time.Time  `bson:"end_at,omitempty"`
    Duration  float64     `bson:"duration,omitempty"` // seconds
    Metadata  interface{} `bson:"metadata,omitempty"`
    SubNodes  []TraceNode `bson:"sub_nodes,omitempty"` // Nested execution
    CreatedBy string      `bson:"created_by"`
}
```

### TracesCollection Interface
```go
type TracesCollection interface {
    Create(ctx context.Context, trace *models.Trace) error
    Get(ctx context.Context, id string) (*models.Trace, error)
    GetByConversation(ctx context.Context, tenantID, conversationID string) (*models.Trace, error)
    GetByAutonomousAgent(ctx context.Context, tenantID, autonomousAgentID string) (*models.Trace, error)
    List(ctx context.Context, opts *ListTracesOptions) ([]*models.Trace, error)
    Update(ctx context.Context, trace *models.Trace) error
    AddNodes(ctx context.Context, id string, nodes []models.TraceNode) error
    AddLogs(ctx context.Context, id string, logs []interface{}) error
    Delete(ctx context.Context, id string) error
}
```

### Future: Trace Transformers
The architecture supports pluggable transformers for converting external formats:

```go
// Planned interface for trace transformation
type TraceTransformer interface {
    Transform(externalData interface{}) (*models.Trace, error)
    SourceType() string // "n8n", "langchain", "langgraph", etc.
}
```

### MongoDB Indexes
```go
// Compound indexes for efficient queries
{tenant_id: 1, conversation_id: 1}           // unique for conversation traces
{tenant_id: 1, autonomous_agent_id: 1, created_at: -1}  // list agent traces
{tenant_id: 1, context_type: 1, created_at: -1}         // filter by context type
```

---

## Swagger API Documentation (MANDATORY)

All HTTP handlers MUST have Swagger annotations. Use `swaggo/swag` for generating OpenAPI docs.

### Swagger Setup
- Swagger UI available at: `/docs/index.html`
- Generate docs: `swag init -g cmd/server/main.go -o docs`
- Import docs in main.go: `_ "github.com/unifiedui/agent-service/docs"`

### Handler Annotation Template
```go
// HandlerName handles the endpoint description.
// @Summary Short summary
// @Description Detailed description of what the endpoint does
// @Tags TagName
// @Accept json
// @Produce json
// @Param paramName path string true "Parameter description"
// @Param request body RequestType true "Request body description"
// @Success 200 {object} ResponseType "Success description"
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /api/v1/agent-service/path [method]
func (h *Handler) HandlerName(c *gin.Context) {
```

### Main.go Swagger Annotations
```go
// @title UnifiedUI Agent Service API
// @version 1.0
// @description Unified abstraction layer for AI agent backends
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url https://github.com/unifiedui/agent-service
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
// @host localhost:8085
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
```

### Regenerate Swagger Docs
After modifying any handler annotations:
```bash
swag init -g cmd/server/main.go -o docs
```

---

## Design Patterns & Principles

### Factory Pattern (MANDATORY)
All infrastructure components MUST use the factory pattern for initialization:

```go
// Example: Cache Factory
type CacheType string

const (
    CacheTypeRedis CacheType = "redis"
)

func NewCacheClient(cacheType CacheType, config *CacheConfig) (CacheClient, error) {
    switch cacheType {
    case CacheTypeRedis:
        return NewRedisCacheClient(config)
    default:
        return nil, fmt.Errorf("unsupported cache type: %s", cacheType)
    }
}
```

### Interface-Driven Design
Always define interfaces in `internal/core/` and implementations in `internal/infrastructure/`:

```go
// internal/core/cache/cache.go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    DeletePattern(ctx context.Context, pattern string) (int64, error)
    Ping(ctx context.Context) error
    Close() error
}
```

### Dependency Injection
Use constructor injection for all services:

```go
func NewChatService(
    cache cache.CacheClient,
    docDB docdb.Client,
    platformClient platform.Client,
) *ChatService {
    return &ChatService{
        cache:          cache,
        docDB:          docDB,
        platformClient: platformClient,
    }
}
```

### Error Handling
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Use custom error types in `internal/domain/errors/`
- Platform Service errors are forwarded directly to client

---

## Caching Strategy

### Session State Caching
- **TTL**: 3 minutes
- **Key Pattern**: `session:{tenantId}:{userId}`
- **Content**: Platform config + encrypted credentials
- Cache is refreshed after each message completion

### Credential Encryption
Credentials from Platform Service MUST be encrypted before caching:
1. Encryption key is retrieved from Vault
2. Use Fernet-style encryption (AES-256)
3. Store encrypted blob in Redis

```go
// Example cache key patterns
"session:{tenantId}:{userId}"           // Session state with encrypted creds
"messages:{tenantId}:{conversationId}"  // Cached messages
```

---

## SSE Streaming

Use Server-Sent Events for streaming responses:

```go
// internal/api/sse/writer.go
func (w *SSEWriter) WriteEvent(event, data string) error {
    _, err := fmt.Fprintf(w.writer, "event: %s\ndata: %s\n\n", event, data)
    if err != nil {
        return err
    }
    w.flusher.Flush()
    return nil
}
```

### Event Types
- `message`: Chat message chunk
- `trace`: Trace update (for autonomous agents)
- `error`: Error event
- `done`: Stream completion

---

## Testing Requirements

### Coverage Target: 85%

### Unit Tests
- Location: `tests/unit/`
- Use `testify` for assertions
- Use `miniredis` for in-memory Redis testing
- Use `mtest` or mocks for MongoDB testing
- Mock all external HTTP calls (Platform Service, Agent backends)

### Test Naming Convention
```go
func TestServiceName_MethodName_Scenario(t *testing.T) {
    // Arrange
    // Act
    // Assert
}
```

### Factory & Initialization Tests (MANDATORY)
Every factory and service initialization MUST be tested:

```go
func TestNewCacheClient_Redis_Success(t *testing.T) { ... }
func TestNewCacheClient_UnsupportedType_Error(t *testing.T) { ... }
func TestNewVaultClient_AzureKeyVault_Success(t *testing.T) { ... }
```

### Mocking Strategy
```go
// tests/mocks/cache_mock.go
type MockCacheClient struct {
    mock.Mock
}

func (m *MockCacheClient) Get(ctx context.Context, key string) ([]byte, error) {
    args := m.Called(ctx, key)
    return args.Get(0).([]byte), args.Error(1)
}
```

---

## Code Style Guidelines

### Naming Conventions
- **Packages**: lowercase, single word (e.g., `cache`, `vault`, `handlers`)
- **Interfaces**: noun or noun phrase (e.g., `Cache`, `VaultClient`)
- **Structs**: noun (e.g., `RedisCache`, `ChatService`)
- **Methods**: verb or verb phrase (e.g., `Get`, `SendMessage`)
- **Constants**: CamelCase (e.g., `CacheTypeRedis`)

### File Organization
- One primary type per file
- Test files: `*_test.go` in same package OR `tests/unit/`
- Keep files under 300 lines

### Context Usage
Always pass `context.Context` as first parameter:

```go
func (s *ChatService) SendMessage(ctx context.Context, req *SendMessageRequest) (*Message, error)
```

### Error Messages
- Lowercase, no punctuation
- Include relevant identifiers

```go
return fmt.Errorf("failed to get message: tenantId=%s, messageId=%s: %w", tenantId, messageId, err)
```

---

## Environment Variables

```bash
# Server
SERVER_PORT=8080

# Cache
CACHE_TYPE=redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
CACHE_TTL_SECONDS=180

# Document DB
DOCDB_TYPE=mongodb
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=unifiedui

# Vault
VAULT_TYPE=dotenv  # Options: dotenv, azure, hashicorp
AZURE_KEYVAULT_URL=https://myvault.vault.azure.net
HASHICORP_VAULT_ADDR=http://localhost:8200
HASHICORP_VAULT_TOKEN=

# Encryption
SECRETS_ENCRYPTION_KEY=  # 32-byte base64 encoded key

# Platform Service
PLATFORM_SERVICE_URL=http://localhost:8081
```

---

## Development Workflow

### Running Locally
```bash
make run          # Start server
make test         # Run all tests
make test-cover   # Run tests with coverage report
make lint         # Run linter
```

### Adding a New Agent Backend
1. Create client in `internal/services/agents/{name}/client.go`
2. Implement the `AgentClient` interface
3. Register in `internal/services/agents/factory.go`
4. Add tests in `tests/unit/services/agents/`

### Adding a New Infrastructure Component
1. Define interface in `internal/core/{component}/`
2. Implement in `internal/infrastructure/{component}/`
3. Add to factory in `internal/core/{component}/factory.go`
4. Add tests for factory and implementation

---

## Important Reminders for Copilot

1. **Always use interfaces** defined in `internal/core/`
2. **Always use factory pattern** for infrastructure components
3. **Always write tests** - target 85% coverage
4. **Always use context.Context** as first parameter
5. **Always encrypt credentials** before caching
6. **Forward Platform Service errors** directly to client
7. **Use SSE** for streaming responses
8. **Use miniredis** for cache testing
9. **Mock external HTTP calls** in tests
10. **Keep files under 300 lines**

---

## References

- Attached Python implementations from `unified-ui-backend` for pattern reference
- See `internal/core/` for interface definitions
- See `tests/mocks/` for mock implementations
