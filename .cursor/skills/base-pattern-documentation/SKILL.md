---
name: base-pattern-documentation
description: Document Go architectural patterns, base types, and conventions for the masterfabric_backend project. Use when explaining architecture decisions, documenting base patterns, or creating architecture guides.
---

# Base Pattern Documentation Skill

Use this skill when documenting Go architectural patterns or explaining how base types and conventions work in the `masterfabric_backend` project.

## When to Use

- Documenting a new architectural component (handler, use case, repository)
- Explaining Clean Architecture layers and dependency rules
- Creating guides for new contributors
- Documenting the endpoint catalog

## Documentation Template

### For Domain Models

```markdown
## {EntityName}

**Location**: `internal/domain/{context}/model/{entity}.go`

**Purpose**: {What this entity represents in the domain}

**Fields**:
| Field | Type | Description |
|-------|------|-------------|
| ID | uuid.UUID | Unique identifier |

**Business Rules**:
- {Rule 1}

**Related**: {Related entities, events, repositories}
```

### For Use Cases

```markdown
## {UseCaseName}

**Location**: `internal/application/{context}/usecase/{file}.go`

**Input**: `dto.{InputDTO}`
**Output**: `*dto.{OutputDTO}, error`

**Flow**:
1. Validate input
2. Execute business logic
3. Persist changes
4. Publish domain event
5. Return result

**Error Cases**:
- `ErrNotFound`: {when}
- `ErrBadRequest`: {when}
```

### For Handlers

```markdown
## {HandlerName}

**Location**: `internal/infrastructure/http/handler/{context}/handler.go`

**Routes**:
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | /entities | CreateEntity | Create new entity |

**Headers Required**: `Authorization`
```

## Output Format

- Use tables for structured data
- Include code examples from the actual codebase
- Reference specific file paths with `file:line`
- Document the WHY, not just the WHAT
- Keep concise (under 100 lines per component)
