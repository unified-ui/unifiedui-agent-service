# unified-ui Agent Service

A high-performance Go/Gin microservice that serves as a unified abstraction layer between a single frontend and heterogeneous AI agent backends.

## Overview

UnifiedUI Agent Service provides:
- **Unified Chat Interface** - One API for multiple AI agent systems (N8N, Microsoft Foundry, Copilot, LangChain)
- **Message Streaming** - SSE (Server-Sent Events) for real-time response streaming
- **Agent Tracing** - Capture and store traces from autonomous agents for debugging/monitoring
- **Session Management** - Cached session state with encrypted credentials

## Architecture

```
                      ┌──────────────────────┐
                      │  UnifiedUI           │
                      │  Platform Service    │
                      │  (Auth + Config)     │
                      └──────────────────────┘
                                 ▲
                                 │
                                 ▼
┌───────────────┐     ┌──────────────────────┐     ┌──────────────────────────────────────┐
│  Frontend     │────▶│  UnifiedUI           │────▶│  Heterogene Backends                 │
│  (UnifiedUI)  │◀SSE─│  Agent Service       │◀────│  N8N, Foundry, Custom REST APIs, ... │
└───────────────┘     └──────────────────────┘     └──────────────────────────────────────┘
                                 ▲
                                 │
                       ┌─────────┼─────────┐
                       ▼         ▼         ▼
                 ┌─────────┐ ┌───────┐ ┌────────┐
                 │Document │ │ Cache │ │ Vault  │
                 │DB       │ │       │ │        │
                 └─────────┘ └───────┘ └────────┘
```

## Tech Stack

- **Language**: Go 1.21+
- **Framework**: Gin
- **Document DB**: MongoDB / CosmosDB
- **Cache**: Redis
- **Vault**: Azure KeyVault / HashiCorp Vault / DotEnv (dev)

## Quick Start

### Prerequisites

- Go 1.21+
- Redis
- MongoDB (or CosmosDB)

### Setup

1. Clone the repository:
```bash
git clone <repository-url>
cd unified-ui-agent-service
```

2. Copy environment variables:
```bash
cp .env.example .env
```

3. Install dependencies:
```bash
make deps
```

4. Run the service:
```bash
make run
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-cover

# Run tests with coverage percentage
make test-cover-percent
```

## API Endpoints

### Health Checks

```
GET /health          # Overall health status
GET /health/ready    # Readiness probe
GET /health/live     # Liveness probe
```

### Messages

```
GET  /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/messages
POST /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/messages
```

### Traces

```
GET /api/v1/agent-service/tenants/{tenantId}/conversations/{conversationId}/messages/{messageId}/traces
PUT /api/v1/agent-service/tenants/{tenantId}/autonomous-agents/{agentId}/traces
```

## Project Structure

```
├── cmd/server/            # Application entry point
├── internal/
│   ├── api/               # HTTP handlers, middleware, routes
│   ├── config/            # Configuration management
│   ├── core/              # Interfaces (cache, vault, docdb)
│   ├── domain/            # Domain models and errors
│   ├── infrastructure/    # Interface implementations
│   └── services/          # Business logic
├── tests/
│   ├── unit/              # Unit tests
│   ├── integration/       # Integration tests
│   ├── mocks/             # Mock implementations
│   └── testutils/         # Test utilities and fixtures
└── .github/
    └── copilot-instructions.md  # Copilot development guidelines
```

## Development

### Adding a New Agent Backend

1. Create client in `internal/services/agents/{name}/client.go`
2. Implement the `AgentClient` interface
3. Register in `internal/services/agents/factory.go`
4. Add tests

### Adding a New Infrastructure Component

1. Define interface in `internal/core/{component}/`
2. Implement in `internal/infrastructure/{component}/`
3. Add to factory
4. Add tests

## Configuration

See [.env.example](.env.example) for all available configuration options.

## License

[License information]
