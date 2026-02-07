# Project Structure

## Folder Tree

```
unified-ui-agent-service/
├── cmd/
│   └── server/
│       └── main.go                     # Entry point: config → init infra → wire DI → start server
├── internal/
│   ├── api/
│   │   ├── dto/
│   │   │   ├── requests.go             # Request DTOs (message endpoints)
│   │   │   ├── responses.go            # Response DTOs (message endpoints)
│   │   │   └── traces.go               # Trace-specific DTOs + conversion helpers
│   │   ├── handlers/
│   │   │   ├── health.go               # Health, Ready, Live handlers
│   │   │   ├── messages.go             # GetMessages, SendMessage handlers
│   │   │   └── traces.go               # All trace CRUD + import handlers
│   │   ├── middleware/
│   │   │   ├── auth.go                 # 3 auth middlewares (Bearer, APIKey, Flexible)
│   │   │   ├── errors.go              # HandleError + Recovery middleware
│   │   │   └── logging.go             # Request logging middleware
│   │   ├── routes/
│   │   │   └── routes.go              # Route registration + middleware groups
│   │   └── sse/
│   │       └── writer.go              # SSE streaming writer (event types, flush)
│   ├── config/
│   │   └── config.go                  # Config struct + Load() from env vars
│   ├── core/                          # ── INTERFACES ONLY ──
│   │   ├── cache/
│   │   │   ├── cache.go               # Cache interface (Get/Set/Delete/Ping)
│   │   │   ├── client.go              # Client interface (wraps Cache + health)
│   │   │   └── factory.go             # NewCacheClient factory
│   │   ├── docdb/
│   │   │   ├── docdb.go               # Collection, Database, Cursor, SingleResult interfaces
│   │   │   ├── client.go              # Client interface (Messages/Traces/Ping/EnsureIndexes)
│   │   │   ├── messages.go            # MessagesCollection interface
│   │   │   ├── traces.go              # TracesCollection interface
│   │   │   └── factory.go             # NewDocDBClient factory
│   │   └── vault/
│   │       ├── vault.go               # Vault interface (Store/Get/Update/Delete secrets)
│   │       ├── client.go              # Client interface (wraps Vault + health)
│   │       └── factory.go             # NewVaultClient factory
│   ├── domain/
│   │   ├── errors/
│   │   │   └── errors.go             # DomainError struct + NewXxxError constructors
│   │   └── models/
│   │       ├── trace.go               # Trace, TraceNode, NodeType/Status/ContextType
│   │       ├── message.go             # Message model
│   │       └── session.go             # Session, SessionConfig, ChatHistoryEntry
│   ├── infrastructure/                # ── IMPLEMENTATIONS ──
│   │   ├── cache/
│   │   │   └── redis/
│   │   │       ├── cache.go           # Redis Cache implementation
│   │   │       └── client.go          # Redis Client implementation
│   │   ├── docdb/
│   │   │   ├── mongodb/
│   │   │   │   ├── client.go          # MongoDB client (connect, ping, close, indexes)
│   │   │   │   ├── messages.go        # MongoDB MessagesCollection
│   │   │   │   └── traces.go          # MongoDB TracesCollection
│   │   │   └── cosmosdb/
│   │   │       └── client.go          # CosmosDB client (MongoDB wire protocol)
│   │   └── vault/
│   │       ├── azure/
│   │       │   └── keyvault.go        # Azure Key Vault implementation
│   │       ├── hashicorp/
│   │       │   └── vault.go           # HashiCorp Vault implementation
│   │       └── dotenv/
│   │           └── vault.go           # DotEnv vault (local development)
│   ├── pkg/
│   │   ├── contextformat/
│   │   │   └── formatter.go          # Chat context formatting utilities
│   │   └── encryption/
│   │       └── fernet.go             # Fernet-style AES-256 encryption
│   └── services/
│       ├── agents/
│       │   ├── factory.go             # Agent Factory (N8N, Foundry, Copilot, Custom)
│       │   ├── types.go               # AgentClients, WorkflowClient interface
│       │   ├── foundry/               # Microsoft Foundry agent implementation
│       │   └── n8n/                   # N8N webhook agent implementation
│       ├── platform/
│       │   ├── client.go              # Platform Client interface + HTTP implementation
│       │   └── models.go             # Platform DTOs (AgentConfig, UserInfo, etc.)
│       ├── session/
│       │   └── service.go            # Session Service (cache + encrypt session data)
│       └── traceimport/
│           ├── interfaces.go          # TraceImporter, TraceTransformer interfaces
│           ├── service.go             # ImportService (factory + queue)
│           ├── factory.go             # ImporterFactory (registers per agent type)
│           ├── queue.go               # Background job queue for async imports
│           ├── types.go               # JobType, JobAction, ImportRequest
│           ├── foundry/               # Foundry trace importer + transformer
│           └── n8n/                   # N8N trace importer + transformer
├── tests/
│   ├── mocks/
│   │   ├── cache_mock.go             # MockCacheClient
│   │   ├── docdb_mock.go             # MockDocDBClient + MockTracesCollection + MockMessagesCollection
│   │   ├── encryption_mock.go        # MockEncryptor
│   │   ├── platform_mock.go          # MockPlatformClient
│   │   ├── session_mock.go           # MockSessionService
│   │   ├── traceimport_mock.go       # MockImportService
│   │   └── vault_mock.go             # MockVaultClient
│   ├── testutils/
│   │   ├── fixtures.go               # Test constants + factory functions (NewTestTrace, etc.)
│   │   └── helpers.go                # SetupTestRouter, PerformRequest, AssertStatusCode, ParseJSONResponse
│   └── unit/
│       ├── encryption/
│       ├── handlers/                  # Handler tests (traces_test.go, health_test.go)
│       ├── infrastructure/
│       ├── models/
│       ├── pkg/
│       ├── services/
│       │   ├── agents/
│       │   ├── platform/
│       │   └── traceimport/
│       └── session/
├── docs/
│   ├── docs.go                        # Generated Swagger
│   ├── swagger.json
│   └── swagger.yaml
├── poc/                               # Proof of concept code (foundry, langchain, n8n)
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Core vs Infrastructure Pattern

The central architectural principle: **interfaces in `internal/core/`, implementations in `internal/infrastructure/`**.

```
internal/core/cache/cache.go       → Cache interface (Get, Set, Delete, Ping, Close)
internal/infrastructure/cache/redis/ → Redis implementation

internal/core/docdb/client.go      → Client interface (Messages, Traces, Ping)
internal/infrastructure/docdb/mongodb/ → MongoDB implementation

internal/core/vault/vault.go       → Vault interface (StoreSecret, GetSecret, DeleteSecret)
internal/infrastructure/vault/azure/   → Azure Key Vault implementation
internal/infrastructure/vault/dotenv/  → DotEnv development implementation
```

Handlers and services depend ONLY on `internal/core/` interfaces. Never import from `internal/infrastructure/` except in `cmd/server/main.go` for wiring.

---

## Adding a New Entity / Domain Model

1. **Model**: `internal/domain/models/{entity}.go` — struct + constructor + enums
2. **DocDB interface**: `internal/core/docdb/{entity}.go` — typed collection interface
3. **DocDB implementation**: `internal/infrastructure/docdb/mongodb/{entity}.go` — MongoDB collection
4. **DTOs**: `internal/api/dto/{entity}.go` — request/response structs + conversion helpers
5. **Handler**: `internal/api/handlers/{entity}.go` — handler struct + methods with Swagger annotations
6. **Routes**: register in `internal/api/routes/routes.go` → appropriate auth group
7. **Mocks**: `tests/mocks/{entity}_mock.go` — testify mock for collection
8. **Fixtures**: add factory function to `tests/testutils/fixtures.go`
9. **Tests**: `tests/unit/handlers/{entity}_test.go`

---

## Adding a New Agent Backend

1. Create directory: `internal/services/agents/{name}/`
2. Implement WorkflowClient interface (see `internal/services/agents/types.go`)
3. Register in `internal/services/agents/factory.go` → `CreateClients()` switch
4. Create trace importer: `internal/services/traceimport/{name}/`
5. Implement `TraceImporter` interface from `internal/services/traceimport/interfaces.go`
6. Register importer in `cmd/server/main.go` → `importService.RegisterImporter()`
7. Add `platform.AgentType{Name}` constant in platform models
8. Write tests in `tests/unit/services/agents/` and `tests/unit/services/traceimport/`

---

## Adding a New Infrastructure Component

1. Define interface in `internal/core/{component}/{component}.go`
2. Define client interface in `internal/core/{component}/client.go`
3. Create factory in `internal/core/{component}/factory.go`
4. Implement in `internal/infrastructure/{component}/{impl}/`
5. Wire in `cmd/server/main.go` → `create{Component}Client()` helper
6. Add mock in `tests/mocks/{component}_mock.go`
7. Test factory + implementation in `tests/unit/infrastructure/`

---

## Dependency Flow

```
cmd/server/main.go
  → config.Load()
  → creates infrastructure clients (cache, vault, docdb)
  → creates services (platform.Client, session.Service, agents.Factory, traceimport.ImportService)
  → creates handlers (TracesHandler, MessagesHandler, HealthHandler)
  → passes to routes.Setup()

routes.Setup()
  → groups routes by auth middleware
  → registers handler methods per route
```

Only `main.go` knows about concrete implementations. Everything else uses interfaces.
