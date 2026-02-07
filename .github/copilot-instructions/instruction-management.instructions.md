# Instruction Management

## When to Update Instructions

After completing work, evaluate whether any instruction file needs updating.

### Update When

| Change | Update file |
|--------|-------------|
| New infrastructure interface (cache, vault, docdb) | `infrastructure.instructions.md` |
| New handler or endpoint group | `handlers.instructions.md` |
| New route group or auth middleware | `api-routes.instructions.md` |
| New agent backend added | `project-structure.instructions.md` + `infrastructure.instructions.md` |
| Folder structure changed | `project-structure.instructions.md` |
| New test pattern established | `testing.instructions.md` |
| New mock added | `testing.instructions.md` (mock checklist) |
| New domain model added | `project-structure.instructions.md` (entity checklist) |
| Golden rules changed | `copilot-instructions.md` |

### Never Update For

- Bug fixes
- Simple field additions to existing models/DTOs
- One-off handler-specific logic
- Test additions following existing patterns
- Swagger annotation tweaks

---

## Review Checklist

Before finalizing instruction updates:

1. Does the new content match the actual code? (Verify by reading the source.)
2. Is the format consistent with the existing file?
3. Is the information general enough to be useful for future work? (No one-off specifics.)
4. Are code examples minimal but complete?
5. Are references to file paths accurate?

---

## File Responsibilities

| File | Covers |
|------|--------|
| `copilot-instructions.md` | Index, golden rules, naming, quick reference |
| `project-structure.instructions.md` | Folder tree, core/infra pattern, checklists for new entities/backends |
| `api-routes.instructions.md` | Route groups, URL conventions, auth middleware, adding routes |
| `handlers.instructions.md` | Handler struct/method pattern, Swagger annotations, error handling, DTOs |
| `infrastructure.instructions.md` | Cache, docdb, vault, session, platform client, SSE, encryption, agents, trace import |
| `testing.instructions.md` | Test naming, fixtures, helpers, mock pattern, handler test pattern, what to test |
| `instruction-management.instructions.md` | This file — when/how to update docs |
