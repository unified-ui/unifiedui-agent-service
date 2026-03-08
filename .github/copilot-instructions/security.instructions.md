---
applyTo: '**'
---

# Security Guidelines — Agent Service (Go/Gin/MongoDB)

## CRITICAL: Read This First

These rules are **mandatory** for all code generation. Violations cause CodeQL / SAST failures in CI.

---

## 1. NoSQL Injection Prevention

**Threat**: User-controlled values flow into MongoDB queries unsanitized → attackers inject operators like `$gt`, `$regex`, `$where`.

### Rules

- **NEVER** pass raw user input (from `c.Param()`, `c.Query()`, `c.PostForm()`, request body fields) directly into `bson.M{}` filters.
- **ALWAYS** use `sanitizeValue()` from `internal/infrastructure/docdb/mongodb/sanitize.go` for **every** value used in a MongoDB filter.
- **ALWAYS** use `sanitizeRegex()` for values used in `$regex` filters — it calls `regexp.QuoteMeta()`.
- **ALWAYS** use `middleware.SanitizePathParam(c, "paramName")` instead of `c.Param("paramName")` in handlers.
- `sanitizeValue` uses a strict allowlist regex: `^[A-Za-z0-9:._@+\-]{1,512}$`. Values that don't match are replaced with `""`.

### Correct Pattern

```go
// In handler — use SanitizePathParam, NEVER c.Param()
traceID := middleware.SanitizePathParam(c, "traceId")

// In MongoDB layer — sanitize every filter value
filter := bson.M{
    "_id":      sanitizeValue(id),
    "tenantId": sanitizeValue(tenantID),
}

// For regex searches — use sanitizeRegex
filter := bson.M{
    "content": bson.M{"$regex": sanitizeRegex(query), "$options": "i"},
}
```

### Wrong Pattern (CodeQL will flag)

```go
// WRONG — raw c.Param() flows into handler → DB → query
traceID := c.Param("traceId")

// WRONG — unsanitized value in filter
filter := bson.M{"_id": id}

// WRONG — unsanitized regex
filter := bson.M{"content": bson.M{"$regex": query}}
```

---

## 2. SSRF Prevention (Server-Side Request Forgery)

**Threat**: User-controlled URLs/path segments in outbound HTTP requests → attacker redirects server to internal services or cloud metadata endpoints.

### Rules

- **ALWAYS** validate base URLs with `validateHTTPURL()` before making HTTP requests — it enforces `http`/`https` scheme and non-empty host.
- **ALWAYS** validate user-controlled path segments with `validateIdentifier()` — strict allowlist `^[A-Za-z0-9_\-.:]{1,512}$`.
- **ALWAYS** use `url.PathEscape()` for path segments in constructed URLs.
- **NEVER** concatenate raw user input into URL strings.
- **NEVER** trust base URLs from external configuration without validation.

### Correct Pattern

```go
baseURL, err := validateHTTPURL(config.BaseURL)
if err != nil {
    return nil, fmt.Errorf("invalid base URL: %w", err)
}

safePath := url.PathEscape(validateIdentifier(config.ResourceID))
u, err := url.Parse(baseURL + "/api/v1/resources/" + safePath)
```

### Wrong Pattern (CodeQL will flag)

```go
// WRONG — no URL validation
u, _ := url.Parse(config.BaseURL + "/api/resource/" + config.ID)

// WRONG — no path segment validation
u, _ := url.Parse(baseURL + "/api/resource/" + url.PathEscape(config.ID))
```

---

## 3. Path Parameter Validation

**Threat**: Malformed or malicious path parameters bypass middleware and reach DB queries.

### Rules

- **ALWAYS** use `middleware.SanitizePathParam(c, "name")` in handlers — never `c.Param("name")`.
- `SanitizePathParam` validates against `^[A-Za-z0-9_\-]+$` and returns `""` for invalid input.
- Tenant extraction (`GetTenantID`, `GetTenantContext`) already uses `SanitizePathParam` internally.

---

## 4. Secret Management

- **NEVER** hardcode secrets, API keys, or tokens in source code.
- **ALWAYS** use the vault abstraction (`internal/core/vault/`) to retrieve secrets at runtime.
- **NEVER** log secrets — not even at debug level.

---

## 5. HTTP Client Security

- **ALWAYS** set timeouts on HTTP clients (30s default).
- **ALWAYS** close response bodies with `defer func() { _ = resp.Body.Close() }()`.
- **ALWAYS** check response status codes before processing body.
- **NEVER** trust external API responses without validation.

---

## 6. CodeQL Compliance

This project runs CodeQL with `security-extended` and `security-and-quality` query suites. Custom model extensions in `.github/codeql/extensions/` declare sanitization functions as taint barriers.

If you add a new sanitization function:
1. Add it to `.github/codeql/extensions/sanitizers.yml` as a `neutralModel` entry.
2. Verify locally that `go vet ./...` and `go test ./...` pass.
3. The CodeQL scan runs on every PR — all findings must be resolved before merge.

---

## Quick Checklist Before Committing

- [ ] No raw `c.Param()` in handlers — all use `middleware.SanitizePathParam()`
- [ ] All MongoDB filter values go through `sanitizeValue()` or `sanitizeRegex()`
- [ ] All outbound URLs validated with `validateHTTPURL()`
- [ ] All URL path segments validated with `validateIdentifier()`
- [ ] No hardcoded secrets
- [ ] HTTP clients have timeouts and proper error handling
