# ADR-003: Encrypted Session Caching

**Status:** Accepted
**Date:** 2026-03-08
**Author:** Enrico Goerlitz

---

## Context

For every chat message, the Agent Service needs:
1. **Agent Configuration** — Retrieved from Platform Service (HTTP call)
2. **Chat History** — Retrieved from Document DB (database query)

Without caching, every request would require both calls, adding latency and load.

## Decision

Implement **session-based caching** with **AES-256 encryption** for sensitive data.

## Design

### Cache Key Structure

```
session:{tenantID}:{userID}:{conversationID}
```

### Cached Data

```go
type SessionData struct {
    AgentConfig  *platform.ChatAgentConfig
    ChatHistory  []ChatHistoryEntry
    ExpiresAt    time.Time
}
```

### TTL

Default: 180 seconds (3 minutes), configurable via `CACHE_TTL_SECONDS`.

### Encryption

Session data is encrypted before storage using AES-256-GCM (Fernet-style):

```
┌─────────────────────────────────────────────────┐
│  SessionData (struct)                           │
│           │                                     │
│           ▼                                     │
│  JSON Marshal                                   │
│           │                                     │
│           ▼                                     │
│  AES-256-GCM Encrypt (with ENCRYPTION_KEY)     │
│           │                                     │
│           ▼                                     │
│  Base64 Encode                                  │
│           │                                     │
│           ▼                                     │
│  Redis SET with TTL                             │
└─────────────────────────────────────────────────┘
```

### Flow

```
SendMessage Request
    │
    ▼
Check Session Cache (Redis)
    │
    ├── HIT: Decrypt → Use cached AgentConfig + ChatHistory
    │
    └── MISS:
            ├── Platform Service: GetAgentConfig()
            ├── DocDB: ListChatHistory()
            └── Encrypt → Write to Session Cache
    │
    ▼
Process Message (agent call, streaming)
    │
    ▼
Update Session Cache (new history entry)
```

## Rationale

| Aspect | Decision | Reason |
|--------|----------|--------|
| **Encryption** | AES-256-GCM | AgentConfig may contain API keys, credentials |
| **TTL-based expiry** | 180s | Balance between performance and freshness |
| **Per-conversation** | Yes | Different conversations may have different agents |
| **Cache invalidation** | TTL only | Simplicity; stale configs refresh on next request |

## Consequences

**Positive:**
- Reduced latency for follow-up messages in a conversation
- Reduced load on Platform Service and Document DB
- Sensitive data encrypted at rest in Redis

**Negative:**
- Config changes not reflected until TTL expires (acceptable for 3-minute window)
- Encryption key management required (`ENCRYPTION_KEY` env var)

## Future Consideration

A dedicated **Config Cache** (per tenant/user/agent, not per conversation) could further reduce Platform Service calls. See [design/AGENT_CONFIG_CACHE_CONCEPT.md](../../design/AGENT_CONFIG_CACHE_CONCEPT.md).

## Related

- `internal/services/session/service.go` — Session service implementation
- `internal/pkg/encryption/fernet.go` — AES-256 encryption utilities
