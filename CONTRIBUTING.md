# Contributing to unified-ui Agent Service

Thank you for your interest in contributing! 🎉

## Development Setup

```bash
# Clone the repository
git clone https://github.com/unified-ui/unified-ui-agent-service.git
cd unified-ui-agent-service

# Copy environment variables
cp .env.example .env

# Install dependencies
make deps

# Install pre-commit hooks
pre-commit install
pre-commit install --hook-type commit-msg

# Run the service
make dev
```

## Development Workflow

1. **Fork** the repository
2. **Create a branch** following the naming convention: `<type>/<description>`
   - Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`
   - Example: `feat/add-copilot-agent`
3. **Make your changes** and write tests
4. **Run quality checks** locally:
   ```bash
   make lint                 # golangci-lint
   go vet ./...              # Go vet
   make test                 # Run all tests
   make test-cover           # Tests + coverage report
   pre-commit run --all-files  # All pre-commit hooks
   ```
5. **Commit** using [Conventional Commits](https://www.conventionalcommits.org/):
   ```
   feat(agents): add Microsoft Copilot agent adapter
   fix(sse): handle client disconnect gracefully
   ```
6. **Push** and open a Pull Request

## Code Standards

- **Comments**: Only on exported types and functions (Go convention)
- **No inline comments** — code must be self-documenting
- **Interface-driven design** — define interfaces in `internal/core/`, implement in `internal/infrastructure/`
- **Dependency injection** — constructor injection for all services and handlers
- **Domain errors** — use typed errors from `internal/domain/errors/`
- **Swagger annotations** — mandatory on all handler methods
- **Test coverage** must stay above **80%** (target)
- **golangci-lint** must pass with zero warnings
- **Keep files under 300 lines** — split large handlers into helper methods

## Reporting Issues

- Use GitHub Issues
- Include: Go version, OS, steps to reproduce, expected vs. actual behavior

## Pull Request Guidelines

- PRs to `main` must come from `develop` or `hotfix/*` branches
- PRs to `develop` must come from feature/fix branches (`feat/*`, `fix/*`, etc.)
- All CI checks must pass
- Keep PRs focused — one feature or fix per PR

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
