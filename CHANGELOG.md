# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial open-source release preparation
- CONTRIBUTING.md, SECURITY.md, SPONSORS.md
- docs/adr/ folder with architectural decision records
- Branching strategy documentation in README

### Changed
- Updated README with badges, branching model, and open-source sections
- Improved CI workflow naming consistency

### Fixed
- Workflow runs listing now supports cursor-based pagination — `ListExecutions` accepts and forwards a `cursor` parameter to n8n, and `nextCursor` is returned in the API response via `ListWorkflowRunsResponse`

---

## [0.1.0] - 2026-03-08

### Added
- Core Agent Service implementation
- N8N workflow agent integration
- Microsoft Foundry agent integration
- ReACT agent support via external service
- SSE streaming for real-time responses (22 event types)
- Message and trace management APIs
- Session caching with AES-256 encryption
- Redis cache infrastructure
- MongoDB/CosmosDB document storage
- Azure KeyVault / HashiCorp Vault / DotEnv secret management
- Swagger/OpenAPI documentation
- golangci-lint integration
- Pre-commit hooks
- GitHub Actions CI pipeline
