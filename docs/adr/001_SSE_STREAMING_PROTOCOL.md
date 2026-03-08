# ADR-001: SSE Streaming Protocol

**Status:** Accepted
**Date:** 2026-03-08
**Author:** Enrico Goerlitz

---

## Context

The Agent Service acts as a bridge between the frontend and heterogeneous AI backends (N8N, Microsoft Foundry, Custom REST). AI agents often generate responses token-by-token, requiring real-time streaming to provide responsive user experiences.

Two main options were considered:
1. **WebSockets** — Bi-directional, persistent connections
2. **Server-Sent Events (SSE)** — Unidirectional server-to-client streaming

## Decision

We chose **SSE (Server-Sent Events)** for all agent response streaming.

## Rationale

| Aspect | SSE | WebSockets |
|--------|-----|------------|
| **Direction** | Server → Client (unidirectional) | Bi-directional |
| **Protocol** | HTTP/1.1+ (text/event-stream) | Custom protocol over TCP |
| **Load Balancer** | Works with standard HTTP LBs | Requires sticky sessions or special config |
| **Reconnection** | Built-in auto-reconnect | Manual implementation required |
| **Complexity** | Simple — just HTTP | More complex handshake and state |
| **Use Case Fit** | Stream AI responses (server → client) | Real-time chat (needs both directions) |

For our use case (streaming AI responses from server to client), SSE is simpler, more HTTP-compatible, and sufficient. Bi-directional communication is not required — user messages are sent via standard POST requests.

## SSE Event Types

We define 4 SSE event types:

| Event | Purpose |
|-------|---------|
| `event: message` | Main channel — carries `StreamMessage` JSON |
| `event: trace` | Trace data for monitoring |
| `event: error` | Error notifications |
| `event: done` | Stream completion signal |

The `StreamMessage` type within `event: message` contains 22 subtypes for fine-grained streaming control (TEXT_STREAM, TOOL_CALL_START, REASONING_STREAM, etc.).

## Consequences

**Positive:**
- Simple implementation using `text/event-stream` content type
- Works with existing HTTP infrastructure (proxies, load balancers)
- Built-in browser support via `EventSource` API
- Automatic reconnection on connection drops

**Negative:**
- 6 connections per domain limit in HTTP/1.1 (HTTP/2 multiplexing resolves this)
- Text-only protocol (binary data must be base64-encoded)

## Related

- [design/SSE_VS_WEBSOCKETS_REVIEW.md](../../design/SSE_VS_WEBSOCKETS_REVIEW.md)
- Internal streaming protocol defined in `internal/api/sse/writer.go`
