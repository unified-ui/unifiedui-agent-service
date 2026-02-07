# Infrastructure

## Overview

All infrastructure follows the **interface in `internal/core/` → implementation in `internal/infrastructure/` → factory for creation** pattern.

Only `cmd/server/main.go` imports concrete implementations. Everything else depends on interfaces.

---

## Cache (Redis)

### Interface (`internal/core/cache/cache.go`)

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) (bool, error)
    DeletePattern(ctx context.Context, pattern string) (int64, error)
    Ping(ctx context.Context) error
    Close() error
}
```

### Client wraps Cache with additional functionality

```go
type Client interface {
    Cache
    // Additional client-level methods
}
```

### Factory

```go
cache.NewClient(cacheType string, config *CacheConfig) (Client, error)
```

### Configuration

| Env Var | Default | Purpose |
|---------|---------|---------|
| `CACHE_TYPE` | `redis` | Cache implementation |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | (empty) | Redis password |
| `REDIS_DB` | `0` | Redis database index |
| `CACHE_TTL_SECONDS` | `180` | Default TTL (3 minutes) |

---

## Document Database (MongoDB / CosmosDB)

### Client Interface (`internal/core/docdb/client.go`)

```go
type Client interface {
    Database() Database
    Messages() MessagesCollection
    MessagesRaw() Collection
    Traces() TracesCollection
    TracesRaw() Collection
    Ping(ctx context.Context) error
    Close(ctx context.Context) error
    EnsureIndexes(ctx context.Context) error
}
```

### Typed Collections

**TracesCollection** — domain-aware operations:
- `Create(ctx, trace)`, `Get(ctx, id)`, `GetByConversation(ctx, tenantID, convID)`
- `GetByAutonomousAgent(ctx, tenantID, agentID)`, `List(ctx, opts)`, `Count(ctx, opts)`
- `Update(ctx, trace)`, `AddNodes(ctx, id, nodes)`, `AddLogs(ctx, id, logs)`, `Delete(ctx, id)`
- `ListByConversation(ctx, tenantID, convID)`

**MessagesCollection** — message operations:
- `Create(ctx, msg)`, `GetByConversation(ctx, tenantID, convID, opts)`

### Raw Collection

Generic CRUD via `Collection` interface (`FindOne`, `Find`, `InsertOne`, `UpdateOne`, `DeleteOne`, etc.) for cases where typed methods don't suffice.

### MongoDB Indexes

```go
// Created by EnsureIndexes():
{tenantId: 1, conversationId: 1}                    // conversation trace lookup
{tenantId: 1, autonomousAgentId: 1, createdAt: -1}  // agent trace listing
{tenantId: 1, contextType: 1, createdAt: -1}         // filter by context type
```

### Factory

```go
docdb.NewClient(dbType string, config *DocDBConfig) (Client, error)
```

### Configuration

| Env Var | Default | Purpose |
|---------|---------|---------|
| `DOCDB_TYPE` | `mongodb` | Database implementation (mongodb/cosmosdb) |
| `MONGODB_URI` | `mongodb://localhost:27017` | Connection string |
| `MONGODB_DATABASE` | `unifiedui` | Database name |

---

## Vault (Secrets Management)

### Interface (`internal/core/vault/vault.go`)

```go
type Vault interface {
    StoreSecret(ctx context.Context, key string, value string, metadata map[string]string) (string, error)
    GetSecret(ctx context.Context, uri string) (string, error)
    UpdateSecret(ctx context.Context, uri string, value string, metadata map[string]string) (bool, error)
    DeleteSecret(ctx context.Context, uri string) (bool, error)
    Ping(ctx context.Context) error
    Close() error
}
```

### Implementations

| Type | Package | Use case |
|------|---------|----------|
| `dotenv` | `infrastructure/vault/dotenv/` | Local development (reads from .env) |
| `azure` | `infrastructure/vault/azure/` | Production (Azure Key Vault) |
| `hashicorp` | `infrastructure/vault/hashicorp/` | Production (HashiCorp Vault) |

### Configuration

| Env Var | Default | Purpose |
|---------|---------|---------|
| `VAULT_TYPE` | `dotenv` | Vault implementation |
| `AZURE_KEYVAULT_URL` | (empty) | Azure Key Vault URL |
| `HASHICORP_VAULT_ADDR` | (empty) | HashiCorp Vault address |
| `HASHICORP_VAULT_TOKEN` | (empty) | HashiCorp Vault token |
| `SECRETS_ENCRYPTION_KEY` | (empty) | AES-256 key for credential encryption |

---

## Session Service

`internal/services/session/service.go` — manages user session state with encrypted caching.

### Interface

```go
type Service interface {
    GetSession(ctx context.Context, tenantID, userID, conversationID string) (*SessionData, error)
    SetSession(ctx context.Context, session *SessionData) error
    UpdateChatHistory(ctx context.Context, tenantID, userID, convID string, entries []models.ChatHistoryEntry) error
    DeleteSession(ctx context.Context, tenantID, userID, conversationID string) error
    BuildCacheKey(tenantID, userID, conversationID string) string
}
```

### Session Flow

1. User sends message → check cache for existing session
2. Cache miss → call platform service for config → encrypt credentials → cache session (3 min TTL)
3. Cache hit → decrypt credentials → use cached config
4. After message completion → update session chat history in cache

### Key Pattern

`session:{tenantId}:{userId}:{conversationId}`

---

## Platform Service Client

`internal/services/platform/client.go` — HTTP client for the platform-service.

### Interface

```go
type Client interface {
    GetApplicationConfig(ctx context.Context, tenantID, applicationID, authToken string) (*ApplicationConfigResponse, error)
    GetAgentConfig(ctx context.Context, tenantID, applicationID, conversationID, authToken string) (*AgentConfig, error)
    GetAgentConfigFromFile(ctx context.Context, tenantID, applicationID string) (*AgentConfig, error)
    GetMe(ctx context.Context, authToken string) (*UserInfo, error)
    GetConversation(ctx context.Context, tenantID, conversationID, authToken string) (*ConversationResponse, error)
    ValidateConversation(ctx context.Context, tenantID, conversationID, authToken string) error
    ValidateAutonomousAgent(ctx context.Context, tenantID, autonomousAgentID, authToken string) error
    GetAutonomousAgentConfig(ctx context.Context, tenantID, autonomousAgentID, apiKey string) (*AutonomousAgentConfigResponse, error)
    ValidateAutonomousAgentAPIKey(ctx context.Context, tenantID, autonomousAgentID, apiKey string) error
}
```

### Authentication Modes

- **Bearer forwarding**: User's MSAL token is passed through to platform service
- **API key validation**: Autonomous agent API key validated against platform service
- **Service key**: `X-Service-Key` header for service-to-service auth (from `X_AGENT_SERVICE_KEY` env var)

Platform service errors are forwarded 1:1 to the client.

---

## SSE Streaming

`internal/api/sse/writer.go` — Server-Sent Events for real-time chat responses.

### Writer

```go
writer, err := sse.NewWriter(c.Writer)  // Sets SSE headers + gets Flusher
writer.WriteEvent(sse.EventMessage, data)
writer.WriteStreamMessage(sse.StreamTypeTextStream, "chunk text")
writer.WriteDone()
```

### Event Types

| Event | Purpose |
|-------|---------|
| `message` | Chat message chunk |
| `trace` | Trace update |
| `error` | Error event |
| `done` | Stream completion |

### Stream Message Types

| Type | Purpose |
|------|---------|
| `STREAM_START` | Stream begins |
| `TEXT_STREAM` | Text content chunk |
| `STREAM_NEW_MESSAGE` | New message in multi-message response |
| `MESSAGE_COMPLETE` | Complete message with metadata |
| `STREAM_END` | Stream ends |
| `ERROR` | Error in stream |

---

## Encryption

`internal/pkg/encryption/fernet.go` — Fernet-style AES-256 encryption for credential caching.

```go
type Encryptor interface {
    Encrypt(plaintext string) (string, error)
    Decrypt(ciphertext string) (string, error)
}
```

Used by session service to encrypt credentials before storing in Redis cache.

---

## Agent Factory

`internal/services/agents/factory.go` — creates agent clients based on config type.

```go
type Factory struct{}

func (f *Factory) CreateClients(config *AgentConfig) (*AgentClients, error)
func (f *Factory) CreateFoundryClients(config *AgentConfig, apiToken string) (*AgentClients, error)
```

Supported backends: N8N, Microsoft Foundry. Copilot and Custom are defined but not yet implemented.

---

## Trace Import Service

`internal/services/traceimport/` — imports and transforms traces from external platforms.

### Components

| Component | Purpose |
|-----------|---------|
| `ImportService` | Orchestrator: factory + job queue |
| `ImporterFactory` | Maps agent type → importer |
| `TraceImporter` | Interface: `Import(ctx, req) (traceID, error)` |
| `TraceTransformer` | Interface: `Transform(items, createdBy) []TraceNode` |
| `JobQueue` | Background worker queue for async imports |

### Sync Import

```go
traceID, err := importService.Import(ctx, agentType, importRequest)
```

### Async Import

```go
importService.EnqueueImport(agentType, importRequest)
```

### Adding a New Importer

1. Create `internal/services/traceimport/{name}/` directory
2. Implement `TraceImporter` interface (fetches from external API)
3. Implement transformer (converts external format → `[]TraceNode`)
4. Register in `main.go`: `importService.RegisterImporter(newImporter)`
5. Add `JobType{Name}` constant in `types.go`
