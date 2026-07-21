# masterfabric_backend

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)
![Chi](https://img.shields.io/badge/Router-chi%20v5-00ADD8)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white)
![Status](https://img.shields.io/badge/Status-Day%2001%20scaffold-orange)

Go REST backend for the **MasterFabric Web-LLM Stack**. Exactly **20 endpoints** (`Cmn[3] + Config[2] + Auth[8] + Web MLC-LLM[7]`). The backend never calls an LLM — all inference happens in the browser via WebLLM. This service stores users, LLM sessions, inference events, and decision scores, and exposes `/api/v1/llm/monitoring` for an agent to examine.

> Plan: [`trainee/projects/docs/projects/masterfabric_plan/README.md`](../docs/projects/masterfabric_plan/README.md)
> Scaffold preview: [`trainee/projects/docs/projects/masterfabric_plan/scaffold_preview.md`](../docs/projects/masterfabric_plan/scaffold_preview.md)

## Quick start

```bash
# 1. Install the goose CLI for migrations
go install github.com/pressly/goose/v3/cmd/goose@latest

# 2. Start Postgres + Redis (loopback bind)
make docker-up

# 3. Copy env and run migrations
cp .env.example .env
export $(grep -v '^#' .env | xargs)
./scripts/migrate.sh up
go run scripts/seed.go

# 4. Run the server
make run
```

Verify:

```bash
curl http://localhost:8080/health/live
# {"status":"alive"}

curl http://localhost:8080/health/ready
# {"status":"ready","services":{"postgres":"healthy","redis":"healthy"}}

curl http://localhost:8080/api/v1/config
# {"webllm":{"modelId":"gemma-2b-q4f32_1-MLC",...},"features":{...},"limits":{...}}
```

> First time only: run `go mod tidy` to resolve `go.sum` (network required).

## API endpoints (20 total)

| Group | Count | Endpoints |
|-------|------:|-----------|
| **Cmn** | 3 | `GET /health/live`, `GET /health/ready`, `GET /metrics` |
| **Config** | 2 | `GET /api/v1/config`, `PUT /api/v1/admin/config` |
| **Auth** | 8 | `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `GET /me`, `PUT /me`, `POST /auth/change-password`, `GET /me/sessions` |
| **Web MLC-LLM** | 7 | `GET /llm/models`, `POST /llm/sessions`, `GET /llm/sessions/{id}`, `POST /llm/sessions/{id}/events`, `GET /llm/sessions/{id}/events`, `POST /llm/sessions/{id}/score`, `GET /llm/monitoring` |

Full contract: [`docs/projects/masterfabric_plan/spec/api_endpoints.md`](../docs/projects/masterfabric_plan/spec/api_endpoints.md).

**Day 01 status:** 5/20 EP implemented (`Cmn` + `Config`). Auth arrives Day 02-04, LLM Day 05-07.

## Project structure

```
cmd/server/main.go                       # Entry point, DI, graceful shutdown
internal/
  domain/                                 # Pure Go, no external deps
    iam/{model, repository, event}
    llm/{model, repository}
    config/{model, repository}
  application/                            # Use cases: Execute(ctx, req) (resp, error)
    iam/{dto, usecase}
    llm/{dto, usecase}
    config/{dto, usecase}
  infrastructure/
    http/{router, handler/{cmn, config, iam, llm}}
    postgres/{migrations, iam, llm, config}
    redis/
    auth/
  shared/                                 # config, logger, middleware, errors, response, pagination, validator, events, telemetry
deployments/                              # Dockerfile, docker-compose.yml, render.yaml
scripts/                                  # migrate.sh, seed.go, test.sh, lint.sh
.cursor/                                  # AGENTS.md + rules + skills + plans (AI-native)
```

Layer dependency rule: `domain` <- `application` <- `infrastructure` <- `cmd/server`. Domain imports nothing external.

## Make targets

```bash
make build          # build binary to bin/masterfabric-server
make run            # go run ./cmd/server
make test           # go test ./...
make lint           # golangci-lint run
make migrate        # goose up (needs DATABASE_DSN)
make docker-up      # start Postgres + Redis (loopback bind)
make docker-down    # stop Postgres + Redis
make security-scan  # govulncheck + gosec
```

## Configuration

All config via env vars with safe defaults. See [`.env.example`](.env.example). Key vars:

| Variable | Default | Notes |
|----------|---------|-------|
| `DATABASE_DSN` | `postgres://masterfabric:masterfabric@localhost:5432/masterfabric?sslmode=disable` | parsed via `net/url` |
| `REDIS_URL` | `redis://localhost:6379` | |
| `JWT_SECRET` | `change-me-in-production` | startup warns if default |
| `CORS_ALLOWED_ORIGINS` | *(empty)* | comma-separated; credentials off for `*` or empty |
| `MAX_BODY_BYTES` | `1048576` | 1 MiB |
| `LOG_LEVEL` / `LOG_FORMAT` | `info` / `json` | |

## Deploy (Render)

`deployments/render.yaml` is a Render Blueprint. Apply via the Render MCP or dashboard:

- Web service (`mf-masterfabric-backend`, docker runtime, `healthCheckPath: /health/live`)
- Postgres (`mf-masterfabric-pg`)
- Redis keyvalue (`mf-masterfabric-redis`)

See [`docs/projects/masterfabric_plan/spec/mcp_wiring.md`](../docs/projects/masterfabric_plan/spec/mcp_wiring.md) for the MCP-driven deploy flow.

## Security

See [SECURITY.md](SECURITY.md) for the production checklist. Hardening follows the reference repo's [SECURITY.md](https://github.com/gurkanfikretgunak/masterfabric-go/blob/main/SECURITY.md).

## Reference

- Example backend (style source): https://github.com/gurkanfikretgunak/masterfabric-go
- Tracker repo (`.cursor/` + `render.yaml` template): https://github.com/masterfabric/masterfabric-project-tracker
- Course repo (host): https://github.com/masterfabric/one-hundered-days
