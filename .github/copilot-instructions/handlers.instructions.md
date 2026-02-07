# Handlers

## Handler Pattern

Every handler is a struct with injected dependencies and methods per endpoint.

```go
// TracesHandler handles trace-related HTTP requests.
type TracesHandler struct {
    docDBClient    docdb.Client
    platformClient platform.Client
    importService  *traceimport.ImportService
}

// NewTracesHandler creates a new TracesHandler.
func NewTracesHandler(docDB docdb.Client, platformClient platform.Client, importService *traceimport.ImportService) *TracesHandler {
    return &TracesHandler{
        docDBClient:    docDB,
        platformClient: platformClient,
        importService:  importService,
    }
}
```

---

## Handler Method Template

```go
// CreateTrace handles POST /tenants/{tenantId}/traces
// @Summary Create a new trace
// @Description Creates a new trace for a conversation or autonomous agent
// @Tags Traces
// @Accept json
// @Produce json
// @Param tenantId path string true "Tenant ID"
// @Param request body dto.CreateTraceRequest true "Trace data"
// @Param Authorization header string false "Bearer token (conversation context)"
// @Param X-Unified-UI-Autonomous-Agent-API-Key header string false "API key (agent context)"
// @Success 201 {object} dto.TraceResponse
// @Failure 400 {object} dto.ErrorResponse "Bad request"
// @Failure 401 {object} dto.ErrorResponse "Unauthorized"
// @Failure 409 {object} dto.ErrorResponse "Conflict"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /api/v1/agent-service/tenants/{tenantId}/traces [post]
func (h *TracesHandler) CreateTrace(c *gin.Context) {
    ctx := c.Request.Context()
    tenantID := c.Param("tenantId")

    // 1. Parse request body
    var req dto.CreateTraceRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
        return
    }

    // 2. Validate / authorize
    // 3. Call service/docdb
    // 4. Return response
    c.JSON(http.StatusCreated, dto.TraceToResponse(trace))
}
```

---

## Swagger Annotations (MANDATORY)

Every handler method **must** have these annotations:

| Annotation | Required | Purpose |
|-----------|----------|---------|
| `@Summary` | Yes | One-line summary |
| `@Description` | Yes | Detailed description |
| `@Tags` | Yes | Grouping (Traces, Messages, Health) |
| `@Accept json` | Yes | Request content type |
| `@Produce json` | Yes | Response content type |
| `@Param` | Yes | Path params, query params, request body, headers |
| `@Success` | Yes | Success response with status code + DTO type |
| `@Failure` | Yes | All possible error responses |
| `@Security BearerAuth` | If Bearer auth | Security scheme reference |
| `@Router` | Yes | Full path + HTTP method |

After modifying annotations, regenerate docs:
```bash
swag init -g cmd/server/main.go -o docs
```

---

## Error Handling

Always use `middleware.HandleError(c, err)` with domain errors:

```go
// Domain error constructors (internal/domain/errors/errors.go)
errors.NewNotFoundError("trace", traceID)          // 404
errors.NewValidationError("message", "details")     // 400
errors.NewUnauthorizedError("message")              // 401
errors.NewForbiddenError("message")                 // 403
errors.NewInternalError("message", wrappedErr)      // 500
errors.NewConflictError("message", "details")       // 409
errors.NewBadRequestError("message", "details")     // 400
```

`HandleError` checks for `DomainError` and maps to HTTP status. Unknown errors become 500.

---

## Dual Auth Pattern (Flexible Endpoints)

For endpoints used by both users (Bearer) and autonomous agents (API key):

```go
func (h *TracesHandler) CreateTrace(c *gin.Context) {
    // Check which auth type is present
    token := middleware.GetToken(c)
    apiKey := middleware.GetAutonomousAgentAPIKey(c)

    if isConversationContext(req) {
        // Requires Bearer token → validate via platform service
        userInfo, err := h.platformClient.GetMe(ctx, token)
        h.platformClient.ValidateConversation(ctx, tenantID, req.ConversationID, token)
    } else if isAutonomousAgentContext(req) {
        // Requires API key → validate via platform service
        err := h.platformClient.ValidateAutonomousAgentAPIKey(ctx, tenantID, req.AutonomousAgentID, apiKey)
    }
}
```

---

## DTO Conventions

DTOs live in `internal/api/dto/`:

- **Request DTOs**: `{Action}{Resource}Request` — e.g., `CreateTraceRequest`, `AddNodesRequest`
- **Response DTOs**: `{Resource}Response` — e.g., `TraceResponse`, `ListTracesResponse`
- **Error response**: `ErrorResponse` (defined in `middleware/errors.go`)
- **Conversion helpers**: `TraceToResponse()`, `TracesToResponse()`, `ConvertNodesToModel()`

```go
type CreateTraceRequest struct {
    ApplicationID     string            `json:"applicationId,omitempty"`
    ConversationID    string            `json:"conversationId,omitempty"`
    AutonomousAgentID string            `json:"autonomousAgentId,omitempty"`
    ReferenceID       string            `json:"referenceId"`
    ReferenceName     string            `json:"referenceName"`
    // ...
}
```

---

## Handler Organization by Resource

| File | Resource | Handlers |
|------|----------|----------|
| `health.go` | Health | Health, Ready, Live |
| `messages.go` | Messages | GetMessages, SendMessage |
| `traces.go` | Traces | CreateTrace, GetTrace, DeleteTrace, AddNodes, AddLogs, GetConversationTraces, RefreshConversationTrace, ImportConversationTrace, ListAutonomousAgentTraces, GetAutonomousAgentTraces, RefreshAutonomousAgentTrace, ImportAutonomousAgentTrace |

---

## Helper Methods in Handlers

Extract reusable logic into private methods on the handler struct:

```go
func (h *TracesHandler) getUserID(ctx context.Context, authToken string) (string, error) { ... }
func (h *TracesHandler) resolveUserIDForTrace(ctx context.Context, c *gin.Context, tenantID string, trace *models.Trace) (string, *errors.DomainError) { ... }
func (h *TracesHandler) parseListTracesQueryParams(c *gin.Context) (*docdb.ListTracesOptions, *errors.DomainError) { ... }
```

Keep handler methods focused. If a handler exceeds ~80 lines, extract helpers.
