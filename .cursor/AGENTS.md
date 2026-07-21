# MasterFabric Backend — Agent Conventions

## Project Overview

**masterfabric_backend** is a Go REST backend for the MasterFabric Web-LLM Stack. It has exactly **20 endpoints** grouped as `Cmn[3] + Config[2] + Auth[8] + Web MLC-LLM[7]`. It never calls an LLM — all inference happens in the browser via WebLLM. The backend stores users, sessions, inference events, and decision scores, and exposes `/api/v1/llm/monitoring` for an agent to examine.

## Architecture

- **Pattern:** Domain-Driven Design (DDD) with Clean / Hexagonal Architecture
- **Language:** Go 1.22+
- **Router:** chi v5
- **Database:** PostgreSQL 16 via pgx/v5 + pgxpool
- **Cache:** Redis 7 via go-redis/v9
- **Auth:** JWT (golang-jwt/v5) + bcrypt (added Day 02)
- **Migrations:** goose (SQL files, applied via CLI)
- **Observability:** slog (structured JSON), Prometheus metrics

## Project Structure

```
cmd/server/main.go                       # Entry point, dependency injection
internal/
  domain/                                 # Pure Go, no external deps
    iam/{model, repository, event}        # User, Role, Permission + interfaces
    llm/{model, repository}               # Session, InferenceEvent, DecisionScore + interfaces
    config/{model, repository}            # AppConfig + interface
  application/                            # Use cases: Execute(ctx, req) (resp, error)
    iam/{dto, usecase}
    llm/{dto, usecase}
    config/{dto, usecase}
  infrastructure/
    http/{router, handler/{cmn, config, iam, llm}}
    postgres/{migrations, iam, llm, config}
    redis/
    auth/
  shared/                                 # Cross-cutting concerns
    {config, logger, middleware, errors, response, pagination, validator, events, telemetry}
deployments/                              # Dockerfile, docker-compose.yml, render.yaml
scripts/                                  # migrate.sh, seed.go, test.sh, lint.sh
.cursor/                                  # This folder — agent conventions
```

## Layer Dependency Rule

- `domain` depends on nothing (pure Go).
- `application` depends on `domain`.
- `infrastructure` depends on `domain` + `application`.
- **Never** import infrastructure from domain.

## Key Conventions

### Naming
- Files: `snake_case.go`
- Packages: `lowercase` (single word preferred)
- Interfaces: descriptive name (not `I` prefix) — `UserRepository`, not `IUserRepository`
- Constructors: `NewXxx()` functions
- Errors: `ErrXxx` variables or `domainErr.New(code, message, cause)`

### Use Cases
- `Execute(ctx context.Context, req Req) (Resp, error)`
- Constructor: `NewXxxUseCase(repo, eventBus)` with dependency injection
- Publish domain events after successful state changes (from Day 02)

### HTTP Handlers
- Use chi URL params: `chi.URLParam(r, "paramName")`
- Parse request: `validator.DecodeAndValidate(r, &req)` (or `json.NewDecoder` for thin handlers)
- Return responses: `response.JSON`, `response.Created`, `response.NoContent`, `response.Error`
- Register route in `internal/infrastructure/http/router/router.go`

### Multi-Tenancy
- Not used in MVP (single-user auth). Structure kept for parity with reference repo.

### Error Handling
- Return `error` as last return value
- Use `domainerr.New(code, message, cause)` for domain errors
- Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
- `response.Error` never sends `err.Error()` to the client (CWE-209)

## Build Commands

```bash
go build -o bin/masterfabric-server ./cmd/server
go test ./...
go vet ./...
goose -dir internal/infrastructure/postgres/migrations postgres "$DATABASE_DSN" up
```

## Adding a New Feature (recipe)

1. Define model in `internal/domain/<context>/model/`
2. Define repository interface in `internal/domain/<context>/repository/`
3. Create use case in `internal/application/<context>/usecase/`
4. Create DTO in `internal/application/<context>/dto/`
5. Implement repository in `internal/infrastructure/postgres/<context>/`
6. Create HTTP handler in `internal/infrastructure/http/handler/<context>/`
7. Add route in `internal/infrastructure/http/router/router.go`
8. Wire dependencies in `cmd/server/main.go`
9. Add migration in `internal/infrastructure/postgres/migrations/NNNNN_<name>.sql`

## Reference

- Plan: `trainee/projects/docs/projects/masterfabric_plan/README.md`
- Example backend (style source): https://github.com/gurkanfikretgunak/masterfabric-go
- Tracker repo (`.cursor/` + `render.yaml` source): https://github.com/masterfabric/masterfabric-project-tracker
