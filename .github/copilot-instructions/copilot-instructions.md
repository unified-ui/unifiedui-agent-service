---
applyTo: '**'
---

# unified-ui Agent Service — Copilot Instructions

## Project Overview

**unified-ui** is a multi-tenant integration platform for AI agent systems. This agent service is a high-performance Go/Gin microservice that serves as a unified abstraction layer between a single frontend and heterogeneous AI agent backends (N8N, Microsoft Foundry, Copilot, LangChain/Custom REST).

**Tech Stack**: Go 1.24+ · Gin · MongoDB/CosmosDB · Redis · Azure KeyVault / HashiCorp Vault / DotEnv · SSE Streaming · swaggo/swag · testify

---

## Instruction Files Index

Read the relevant instruction file **before** working in that area.

| File | Read when... |
|------|-------------|
| [project-structure.instructions.md](./project-structure.instructions.md) | Understanding folder layout, adding new modules, modifying structure |
| [api-routes.instructions.md](./api-routes.instructions.md) | Adding or modifying API routes, middleware groups, URL conventions |
| [handlers.instructions.md](./handlers.instructions.md) | Implementing handler methods, Swagger annotations, error handling |
| [infrastructure.instructions.md](./infrastructure.instructions.md) | Working with cache, vault, docdb, SSE, session, platform client |
| [testing.instructions.md](./testing.instructions.md) | Writing tests, running tests, understanding mock/fixture patterns |
| [github-pipelines.instructions.md](./github-pipelines.instructions.md) | Working with CI/CD workflows, adding pipelines, coverage thresholds |
| [instruction-management.instructions.md](./instruction-management.instructions.md) | After completing work — decides if/how to update docs |

---

## Golden Rules

1. **Minimal comments** — Comments only on exported types and exported functions (Go convention, those are **mandatory**). No inline comments, no block comments explaining logic. Code must be self-documenting.
2. **Interface-driven design** — Define interfaces in `internal/core/`, implement in `internal/infrastructure/`. Never depend on concrete types.
3. **Factory pattern for infrastructure** — All infrastructure components (cache, vault, docdb) use factory pattern with interfaces in `internal/core/`.
4. **Dependency injection** — Constructor injection for all services and handlers. Never instantiate clients inside handler methods.
5. **Swagger annotations mandatory** — Every handler method MUST have full `swaggo/swag` annotations (`@Summary`, `@Description`, `@Tags`, `@Param`, `@Success`, `@Failure`, `@Router`, `@Security`).
6. **Domain errors** — Use typed errors from `internal/domain/errors/` with `middleware.HandleError(c, err)`. Never write raw `c.JSON(status, ...)` for errors.
7. **`context.Context` first parameter** — All service/repository methods take `context.Context` as first parameter.
8. **Keep files under 300 lines** — Split large handlers into helper methods or separate files.
9. **Handlers are thin** — Parse request → validate → call service/docdb → return response. Complex logic goes into services.
10. **Run tests after changes** — After significant changes: `make test` (runs `go test -v ./...`). Run `go vet ./...` to verify code correctness. Regenerate swagger after handler annotation changes: `swag init -g cmd/server/main.go -o docs`.

---

## Naming Conventions

| What | Pattern | Example |
|------|---------|---------|
| Package | lowercase, single word | `cache`, `vault`, `handlers` |
| Interface | Noun or noun phrase | `Cache`, `Client`, `TracesCollection` |
| Struct | Noun, implementation prefix | `RedisCache`, `ChatService`, `TracesHandler` |
| Constructor | `New{Type}` | `NewTracesHandler`, `NewClient` |
| Method | Verb or verb phrase | `Get`, `SendMessage`, `CreateTrace` |
| Constants | CamelCase | `CacheTypeRedis`, `NodeStatusCompleted` |
| Domain errors | `New{Adj}Error` | `NewNotFoundError`, `NewValidationError` |
| Handler file | `{resource}.go` in `handlers/` | `traces.go`, `messages.go` |
| DTO file | `{purpose}.go` in `dto/` | `requests.go`, `responses.go`, `traces.go` |
| Model file | `{entity}.go` in `domain/models/` | `trace.go`, `message.go`, `session.go` |
| Mock file | `{component}_mock.go` in `tests/mocks/` | `docdb_mock.go`, `platform_mock.go` |
| Test file | `{resource}_test.go` in `tests/unit/{layer}/` | `traces_test.go`, `health_test.go` |
| Test function | `Test{Type}_{Method}_{Scenario}` | `TestTracesHandler_CreateTrace_Success` |

---

## Quick Reference

- **Run**: `make run` (or `go run ./cmd/server`)
- **Build**: `make build`
- **Test**: `make test` (runs `go test -v ./...`)
- **Test with coverage**: `make test-cover`
- **Lint**: `make lint` (requires `golangci-lint`)
- **Swagger docs**: `swag init -g cmd/server/main.go -o docs`
- **Entry point**: `cmd/server/main.go`
- **Config**: `internal/config/config.go` → `Config` struct, loaded from env vars
- **Routes**: `internal/api/routes/routes.go` → `Setup()`
- **Models**: `internal/domain/models/` (trace.go, message.go, session.go, reaction.go)
- **Errors**: `internal/domain/errors/errors.go` → `DomainError` types
- **Core interfaces**: `internal/core/` (cache, vault, docdb)
- **Platform client**: `internal/services/platform/client.go` → `Client` interface
- **AI service**: `internal/services/ai/service.go` → `Service` interface
- **Swagger UI**: `http://localhost:8085/docs/index.html`

---

## Instruction Management (Summary)

After completing work, evaluate whether documentation needs updating. Full rules in [instruction-management.instructions.md](./instruction-management.instructions.md).

**Update docs when:**
- New infrastructure interface added → update `infrastructure.instructions.md`
- New handler or route group added → update `handlers.instructions.md` / `api-routes.instructions.md`
- New test pattern established → update `testing.instructions.md`
- Folder structure changed → update `project-structure.instructions.md`

**Never update docs for:** bug fixes, simple field additions, one-off handler changes.
