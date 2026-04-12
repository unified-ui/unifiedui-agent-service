# API Routes

## Route Setup

All routes are registered in `internal/api/routes/routes.go` → `Setup()`.

Routes are organized into **groups by authentication type** using Gin middleware.

```go
func Setup(r *gin.Engine, cfg *Config) {
    base := r.Group("/api/v1/agent-service")

    // Group 1: No auth (health endpoints)
    health := base.Group("")

    // Group 2: Bearer token auth
    protected := base.Group("")
    protected.Use(cfg.AuthMiddleware.Authenticate())

    // Group 3: API key auth only
    agentImport := base.Group("")
    agentImport.Use(cfg.AuthMiddleware.AuthenticateAutonomousAgentAPIKey())

    // Group 4: Bearer OR API key (flexible)
    flexibleAuth := base.Group("")
    flexibleAuth.Use(cfg.AuthMiddleware.AuthenticateFlexible())

    // Group 5: Service key auth (S2S only)
    serviceKey := base.Group("")
    serviceKey.Use(cfg.ServiceKeyMw.Authenticate())
}
```

---

## Route Groups & When to Use

| Group | Middleware | Use for |
|-------|-----------|---------|
| `health` | None | `/health`, `/ready`, `/live` |
| `protected` | `Authenticate()` | All user-facing endpoints (messages, conversation traces) |
| `agentImport` | `AuthenticateAutonomousAgentAPIKey()` | External agent callbacks (API key in header) |
| `flexibleAuth` | `AuthenticateFlexible()` | Endpoints usable by both users and agents (create trace, add nodes/logs) |
| `serviceKey` | `ServiceKeyMw.Authenticate()` | Internal S2S endpoints (data cleanup by platform service) |

---

## URL Convention

```
/api/v1/agent-service/                              # Base path
/api/v1/agent-service/health                         # Health check
/api/v1/agent-service/tenants/{tenantId}/...          # Tenant-scoped resources

# Messages
GET  /tenants/{tenantId}/conversations/{conversationId}/messages
POST /tenants/{tenantId}/conversations/{conversationId}/messages
PUT  /tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}
DELETE /tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}

# Message Reactions
GET    /tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}/reactions
POST   /tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}/reactions
DELETE /tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}/reactions/{reactionType}

# Trace CRUD (flexible auth)
POST   /tenants/{tenantId}/traces
GET    /tenants/{tenantId}/traces/{traceId}
DELETE /tenants/{tenantId}/traces/{traceId}
POST   /tenants/{tenantId}/traces/{traceId}/nodes
POST   /tenants/{tenantId}/traces/{traceId}/logs

# Conversation traces (Bearer auth)
GET /tenants/{tenantId}/conversations/{conversationId}/traces
PUT /tenants/{tenantId}/conversations/{conversationId}/traces
PUT /tenants/{tenantId}/conversations/{conversationId}/traces/import/refresh

# AI endpoints (Bearer auth)
POST /tenants/{tenantId}/ai/generate-description
POST /tenants/{tenantId}/ai/analyze-trace
POST /tenants/{tenantId}/ai/summarize-trace
POST /tenants/{tenantId}/ai/test-model
GET  /tenants/{tenantId}/ai/capabilities

# Autonomous agent traces (Bearer auth)
GET /tenants/{tenantId}/autonomous-agents/traces
GET /tenants/{tenantId}/autonomous-agents/{agentId}/traces
PUT /tenants/{tenantId}/autonomous-agents/{agentId}/traces

# Trace import endpoints (API key auth)
PUT /tenants/{tenantId}/autonomous-agents/{agentId}/traces/import
PUT /tenants/{tenantId}/autonomous-agents/{agentId}/traces/{traceId}/import/refresh

# Data cleanup endpoints (Service key auth)
DELETE /tenants/{tenantId}/conversations/{conversationId}/data
DELETE /tenants/{tenantId}/autonomous-agents/{agentId}/data
```

---

## Auth Middleware Detail

### `Authenticate()` — Bearer Token

Extracts `Authorization: Bearer <token>` header. Stores token via `c.Set("auth_token", token)`.

Retrieve in handlers: `middleware.GetToken(c)`

### `AuthenticateAutonomousAgentAPIKey()` — API Key Only

Extracts `X-Unified-UI-Workflow-API-Key` header. Stores API key in context.

Retrieve in handlers: `middleware.GetAutonomousAgentAPIKey(c)`

### `AuthenticateFlexible()` — Bearer OR API Key

Tries Bearer first. If no Bearer token, falls back to API key extraction. Stores whichever is found.

Used for endpoints that autonomous agents (API key) and users (Bearer) both need to call.

Handlers using flexible auth must check which auth type was provided:
```go
token := middleware.GetToken(c)
apiKey := middleware.GetAutonomousAgentAPIKey(c)
if token != "" {
    // Bearer auth path
} else if apiKey != "" {
    // API key auth path
}
```

---

## Global Middleware

`SetupWithMiddleware()` adds global middleware to the Gin engine:
- `LoggingMiddleware` — request/response logging
- `ErrorMiddleware.Recovery()` — panic recovery
- `gin.Recovery()` — Gin's default recovery

---

## Adding a New Route

1. Choose the correct auth group (see table above)
2. Register in `Setup()` function in `routes.go`
3. Follow URL convention: `/tenants/{tenantId}/...`
4. Handler method must have full Swagger annotations
5. If new route group is needed, add middleware and document in this file

---

## Route Config

```go
type Config struct {
    HealthHandler    *handlers.HealthHandler
    MessagesHandler  *handlers.MessagesHandler
    ReactionsHandler *handlers.ReactionsHandler
    TracesHandler    *handlers.TracesHandler
    DataHandler      *handlers.DataHandler
    AIHandler        *handlers.AIHandler
    AuthMiddleware   *middleware.AuthMiddleware
    ServiceKeyMw     *middleware.ServiceKeyMiddleware
}
```

All handlers are injected via this config — never instantiate handlers inside route setup.
