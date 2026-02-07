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

---

## URL Convention

```
/api/v1/agent-service/                              # Base path
/api/v1/agent-service/health                         # Health check
/api/v1/agent-service/tenants/{tenantId}/...          # Tenant-scoped resources

# Messages
GET  /tenants/{tenantId}/conversations/{conversationId}/messages
POST /tenants/{tenantId}/conversations/{conversationId}/messages

# Trace CRUD (flexible auth)
POST   /tenants/{tenantId}/traces
GET    /tenants/{tenantId}/traces/{traceId}
DELETE /tenants/{tenantId}/traces/{traceId}
POST   /tenants/{tenantId}/traces/{traceId}/nodes
POST   /tenants/{tenantId}/traces/{traceId}/logs

# Conversation traces (Bearer auth)
GET /tenants/{tenantId}/conversations/{conversationId}/traces
PUT /tenants/{tenantId}/conversations/{conversationId}/traces

# Autonomous agent traces (Bearer auth)
GET /tenants/{tenantId}/autonomous-agents/traces
GET /tenants/{tenantId}/autonomous-agents/{agentId}/traces
PUT /tenants/{tenantId}/autonomous-agents/{agentId}/traces

# Trace import endpoints (specific auth per handler)
PUT /tenants/{tenantId}/conversations/{conversationId}/traces/import/refresh
PUT /tenants/{tenantId}/autonomous-agents/{agentId}/traces/import/refresh
```

---

## Auth Middleware Detail

### `Authenticate()` — Bearer Token

Extracts `Authorization: Bearer <token>` header. Stores token via `c.Set("auth_token", token)`.

Retrieve in handlers: `middleware.GetToken(c)`

### `AuthenticateAutonomousAgentAPIKey()` — API Key Only

Extracts `X-Unified-UI-Autonomous-Agent-API-Key` header. Stores API key in context.

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
    TracesHandler   *handlers.TracesHandler
    MessagesHandler *handlers.MessagesHandler
    HealthHandler   *handlers.HealthHandler
    AuthMiddleware  *middleware.AuthMiddleware
}
```

All handlers are injected via this config — never instantiate handlers inside route setup.
