# Tooling Guide — Agent Service

This document describes the development tooling, workflows, and quality gates for the unified-ui Agent Service.

## Prerequisites

| Tool | Version | Installation |
|------|---------|--------------|
| Go | 1.24+ | [go.dev/dl](https://go.dev/dl/) |
| Air | latest | `go install github.com/air-verse/air@latest` |
| golangci-lint | 1.62+ | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |
| swag | latest | `go install github.com/swaggo/swag/cmd/swag@latest` |
| pre-commit | latest | `pip install pre-commit` |
| commitlint | latest | `npm install -g @commitlint/cli @commitlint/config-conventional` |

## Quick Commands

```bash
# Development
make dev          # Start with hot reload (Air)
make run          # Start without hot reload
make build        # Build binary

# Testing
make test         # Run all tests
make test-cover   # Run tests with coverage report

# Code Quality
make lint         # Run golangci-lint
go vet ./...      # Run go vet

# Swagger
swag init -g cmd/server/main.go -o docs   # Regenerate API docs

# Dependencies
make deps         # Download dependencies

# Docker
docker compose -f docker/docker-compose.yml --profile dev up    # Dev mode
docker compose -f docker/docker-compose.yml --profile prod up   # Prod mode
```

## Pre-commit Hooks

Install hooks once per clone:

```bash
pre-commit install
pre-commit install --hook-type commit-msg
```

Hooks run automatically on `git commit`. Manual run:

```bash
pre-commit run --all-files
```

## Commit Convention

Commits must follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`

**Examples**:
```
feat(handlers): add message reactions endpoint
fix(sse): handle client disconnect gracefully
docs(api): update swagger annotations for traces
chore(deps): update go-redis to v9.4.0
```

## Code Quality Gates

### Linting (golangci-lint)

Configuration: `.golangci.yml`

Enabled linters:
- `errcheck` — error handling
- `gosimple` — simplification suggestions
- `govet` — suspicious constructs
- `staticcheck` — static analysis
- `unused` — unused code
- `gofmt` / `goimports` — formatting
- `gosec` — security issues
- `gocritic` — advanced checks

### Testing

- Minimum coverage: **80%**
- Test location: `tests/` directory
- Naming: `Test{Type}_{Method}_{Scenario}`
- Parallelization: `-parallel` flag enabled

## CI/CD Workflows

| Workflow | Trigger | Job |
|----------|---------|-----|
| `ci-tests-and-lint.yml` | push/PR | Tests, lint, vet |
| `ci-pr-branch-check.yml` | PR | Branch naming check |
| `codeql.yml` | push/PR/weekly | Security scanning |

## Security

- **Dependabot** updates dependencies weekly (Mondays 09:00 CET)
- **CodeQL** scans for vulnerabilities on every push and weekly
- **gosec** checks run via golangci-lint

## IDE Configuration

### VS Code

Recommended extensions:
- `golang.go`
- `EditorConfig.EditorConfig`

Settings (`.vscode/settings.json`):
```json
{
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--config=.golangci.yml"],
  "editor.formatOnSave": true,
  "go.useLanguageServer": true
}
```

### GoLand

- Enable golangci-lint integration
- Set goimports as formatter
- Enable EditorConfig support
