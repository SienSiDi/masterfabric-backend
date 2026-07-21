---
name: masterfabric_backend implementation
overview: Implement the 20-endpoint Go backend for the MasterFabric Web-LLM Stack as a DDD + Clean Architecture modular service. 14-day plan compressed to 2 days — thin implementations, unit tests only, Docker for local infra, optional Render deploy.
todos:
  - id: day-01-scaffold
    content: "BE scaffold + Config[2] + Cmn[3] + Render Blueprint (5/20 EP)"
    status: completed
  - id: day-02-auth-1
    content: "Auth[8] part 1 — register + login + JWT (7/20 EP)"
    status: completed
  - id: day-03-auth-2
    content: "Auth[8] part 2 — refresh + logout + sessions (10/20 EP)"
    status: completed
  - id: day-04-auth-3
    content: "Auth[8] part 3 — me + change-password + RBAC + rate limit (13/20 EP)"
    status: completed
  - id: day-05-llm-1
    content: "Web MLC-LLM[7] part 1 — models + session create/get (16/20 EP)"
    status: completed
  - id: day-06-llm-2
    content: "Web MLC-LLM[7] part 2 — events POST/GET (18/20 EP)"
    status: completed
  - id: day-07-llm-3
    content: "Web MLC-LLM[7] part 3 — score + monitoring aggregate (20/20 EP)"
    status: completed
  - id: day-08-hardening
    content: "BE hardening — security checklist + unit tests (cut: integration tests, Postman)"
    status: pending
  - id: day-09-render-deploy
    content: "BE Render deploy (live) via Render MCP"
    status: pending
isProject: false
---

# masterfabric_backend Implementation Plan

2-day compressed cut of the 14-day plan. See `trainee/projects/docs/projects/masterfabric_plan/spec/roadmap.md` for the full version.

## Phase A — Backend (Day 1 of 2)

### Day 01 — Scaffold + Config + Cmn (5/20 EP)
- DDD skeleton, Chi router, slog, graceful shutdown, Postgres pool, Redis client.
- Migrations: users, roles, user_roles, app_config.
- EP: `/health/live`, `/health/ready`, `/metrics`, `GET /api/v1/config`, `PUT /api/v1/admin/config`.
- `render.yaml` blueprint.

### Day 02 — Auth part 1 (7/20 EP)
- IAM domain, register + login use cases, JWT service, bcrypt.
- Migration: refresh_tokens.

### Day 03 — Auth part 2 (10/20 EP)
- Refresh rotation, logout (Redis blacklist), sessions list.

### Day 04 — Auth part 3 (13/20 EP)
- Me, update me, change-password, RBAC middleware, login rate limit.

### Day 05 — LLM part 1 (16/20 EP)
- LLM domain, models list, session create/get. Migrations: llm_sessions, inference_events, decision_scores.

### Day 06 — LLM part 2 (18/20 EP)
- Record event, list events. Redis rate limit 30/min/user.

### Day 07 — LLM part 3 (20/20 EP)
- Record score (server-side composite recompute), monitoring aggregate.

### Day 08 — Hardening (cut for 2-day)
- Apply SECURITY.md checklist; unit tests for use cases.

### Day 09 — Render deploy
- Apply `render.yaml` via Render MCP; migrate + seed; smoke-test 20 EP.

## Phase B — Frontend (Day 2 of 2)

See `masterfabric_web/` plan in the host repo.

## Cuts for the 2-day timeline

- No integration tests (unit tests for use cases only).
- No Postman collection.
- No `CHANGELOG.md` discipline.
- No `dev.sh` hot-reload wrapper (use `air` directly via `.air.toml`).
- No OpenTelemetry, no Kafka.
- Full security hardening pass deferred (keep the structure; apply checklist on Day 08 if time).
