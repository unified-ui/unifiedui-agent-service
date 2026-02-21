---
applyTo: '.github/**'
---

# GitHub Pipelines — Agent Service

## Workflow Naming Convention

All workflow files follow a prefix-based naming convention:

| Prefix | Purpose | Example |
|--------|---------|---------|
| `ci-` | Continuous Integration (lint, test, build) | `ci-tests-and-lint.yml` |
| `cd-` | Continuous Deployment (deploy to environments) | `cd-deploy-staging.yml` |
| `ci-int-tests-` | Integration test suites | `ci-int-tests-redis.yml` |
| `ci-e2e-tests-` | End-to-end test suites | `ci-e2e-tests-api.yml` |

The `name:` field inside each workflow MUST match the filename (without `.yml`).

---

## Current Workflows

### ci-tests-and-lint.yml

**Triggers**: push, pull_request, workflow_dispatch

| Job | What it does |
|-----|-------------|
| **lint** | `go vet` + `staticcheck` + `golangci-lint` |
| **test** | `go test -v -race -coverprofile=coverage.out -covermode=atomic ./...` with Redis and MongoDB services |
| **build** | `go build -v ./cmd/server` |

**CI services**: Redis 7 (alpine), MongoDB 7

**Coverage**: Reported but not enforced yet. Target is 80% — enforce once test coverage reaches that level.

### ci-pr-branch-check.yml

**Triggers**: pull_request to `main`

Validates that PRs to `main` originate from a `release/*` branch.

---

## Adding a New Workflow

1. Choose the correct prefix (`ci-`, `cd-`, `ci-int-tests-`, `ci-e2e-tests-`)
2. Create `.github/workflows/{prefix}{descriptive-name}.yml`
3. Set `name:` to match filename without extension
4. Add appropriate triggers (`on:`)
5. Update this instruction file

---

## Tool Versions

| Tool | Version | Config |
|------|---------|--------|
| Go | 1.24 | `go.mod` |
| staticcheck | latest | `dominikh/staticcheck-action@v1` |
| golangci-lint | latest | `golangci/golangci-lint-action@v6` |

---

## Coverage Policy

- **Target threshold**: 80% (not yet enforced — current coverage is low)
- **Run locally**: `make test-cover` (generates `coverage.html`)
- **Quick check**: `make test-cover-percent`
- Once coverage reaches 80%, add enforcement step in CI:
  ```yaml
  - name: Check coverage threshold
    run: |
      COVERAGE=$(go tool cover -func=coverage.out | grep total: | awk '{print $3}' | tr -d '%')
      echo "Total coverage: ${COVERAGE}%"
      if (( $(echo "$COVERAGE < 80" | bc -l) )); then
        echo "::error::Coverage ${COVERAGE}% is below 80% threshold"
        exit 1
      fi
  ```

---

## Environment Variables (Test Job)

| Variable | Value | Purpose |
|----------|-------|---------|
| `REDIS_HOST` | `localhost` | Redis service |
| `REDIS_PORT` | `6379` | Redis port |
| `MONGODB_URI` | `mongodb://localhost:27017` | MongoDB service |
| `MONGODB_DATABASE` | `unifiedui_test` | Test database name |

---

## Secrets

No GitHub secrets are currently required. If adding Codecov integration later, set `CODECOV_TOKEN` as a repository secret.
