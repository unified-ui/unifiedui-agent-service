# ADR-004: Agent Factory Pattern

**Status:** Accepted
**Date:** 2026-03-08
**Author:** Enrico Goerlitz

---

## Context

The Agent Service must communicate with heterogeneous AI backends:
- **N8N** — Webhook-based workflow automation
- **Microsoft Foundry** — Azure AI agent platform
- **ReACT Service** — Custom Python service using unifiedui-sdk
- **Future:** Copilot, LangChain, custom REST APIs

Each backend has different:
- Authentication mechanisms
- API formats
- Streaming protocols
- Trace structures

## Decision

Implement an **Agent Factory** that produces backend-specific clients implementing a common **WorkflowClient** interface.

## Interface

```go
// WorkflowClient is the interface for all agent backends
type WorkflowClient interface {
    // SendMessage streams an agent response
    SendMessage(ctx context.Context, req *SendMessageRequest) (<-chan StreamEvent, error)

    // SupportsStreaming returns whether the backend supports SSE
    SupportsStreaming() bool
}
```

## Factory Implementation

```go
// NewWorkflowClient creates a client for the given agent type
func NewWorkflowClient(
    agentType string,
    config *AgentConfig,
    httpClient *http.Client,
) (WorkflowClient, error) {
    switch agentType {
    case AgentTypeN8N:
        return n8n.NewChatWorkflowClient(config, httpClient)
    case AgentTypeFoundry:
        return foundry.NewWorkflowClient(config, httpClient)
    case AgentTypeReACT:
        return react.NewWorkflowClient(config, httpClient)
    default:
        return nil, fmt.Errorf("unsupported agent type: %s", agentType)
    }
}
```

## Backend Implementations

| Backend | Package | Key Features |
|---------|---------|--------------|
| N8N | `internal/services/agents/n8n/` | Webhook execution, JSON streaming |
| Foundry | `internal/services/agents/foundry/` | Conversation API, workflow actions |
| ReACT | `internal/services/agents/react/` | unifiedui-sdk SSE protocol |

## Adding a New Backend

1. Create package in `internal/services/agents/{name}/`
2. Implement `WorkflowClient` interface
3. Add case in factory switch statement
4. Add agent type constant
5. Add tests in `tests/unit/services/agents/`

## Consequences

**Positive:**
- Unified interface for all backends
- Easy to add new backends without changing handlers
- Backend-specific logic encapsulated
- Testable via mock clients

**Negative:**
- Interface must be general enough for all backends
- Some backend-specific features may not fit the interface cleanly

## Related

- `internal/services/agents/factory.go` — Agent factory implementation
- `internal/services/agents/types.go` — Interface definitions
- `internal/services/agents/{n8n,foundry,react}/` — Backend implementations
