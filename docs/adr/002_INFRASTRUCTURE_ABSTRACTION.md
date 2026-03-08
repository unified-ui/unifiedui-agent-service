# ADR-002: Infrastructure Abstraction via Interfaces

**Status:** Accepted
**Date:** 2026-03-08
**Author:** Enrico Goerlitz

---

## Context

The Agent Service needs to work with multiple infrastructure providers:
- **Document DB:** MongoDB (dev/prod) and CosmosDB (Azure)
- **Cache:** Redis
- **Vault:** Azure KeyVault, HashiCorp Vault, DotEnv (.env for local dev)

Directly coupling to concrete implementations would make testing difficult and limit deployment flexibility.

## Decision

All infrastructure components are accessed through **interfaces defined in `internal/core/`**, with concrete implementations in `internal/infrastructure/`.

## Structure

```
internal/core/         ← Interfaces ONLY
├── cache/
│   ├── cache.go       # Cache interface (Get/Set/Delete/Ping)
│   ├── client.go      # Client interface (wraps Cache + health)
│   └── factory.go     # NewCacheClient factory
├── docdb/
│   ├── docdb.go       # Collection, Database interfaces
│   ├── client.go      # Client interface
│   ├── messages.go    # MessagesCollection interface
│   ├── traces.go      # TracesCollection interface
│   └── factory.go     # NewDocDBClient factory
└── vault/
    ├── vault.go       # Vault interface (Store/Get/Update/Delete)
    ├── client.go      # Client interface (wraps Vault + health)
    └── factory.go     # NewVaultClient factory

internal/infrastructure/   ← Implementations
├── cache/redis/
├── docdb/mongodb/
├── docdb/cosmosdb/
└── vault/{azure,hashicorp,dotenv}/
```

## Factory Pattern

Each infrastructure type has a factory function that selects the implementation based on configuration:

```go
// NewVaultClient creates a vault client based on config
func NewVaultClient(cfg *config.Config) (Client, error) {
    switch cfg.VaultType {
    case VaultTypeAzure:
        return azure.NewClient(cfg)
    case VaultTypeHashiCorp:
        return hashicorp.NewClient(cfg)
    case VaultTypeDotEnv:
        return dotenv.NewClient(cfg)
    default:
        return nil, fmt.Errorf("unknown vault type: %s", cfg.VaultType)
    }
}
```

## Consequences

**Positive:**
- **Testability** — Mock interfaces in tests without real infrastructure
- **Flexibility** — Switch implementations via config (e.g., MongoDB → CosmosDB)
- **Separation** — Business logic in `services/`, infrastructure details isolated
- **Dependency Injection** — All dependencies passed via constructors

**Negative:**
- Additional boilerplate for interfaces and factories
- Must ensure interface contracts are honored by all implementations

## Related

- `internal/core/` — All interface definitions
- `internal/infrastructure/` — All implementations
- `tests/mocks/` — Mock implementations for testing
