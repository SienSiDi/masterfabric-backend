# Security

Production checklist for `masterfabric_backend`. Based on the reference repo's [SECURITY.md](https://github.com/gurkanfikretgunak/masterfabric-go/blob/main/SECURITY.md).

## Before exposing on a public network

- [ ] Set a strong, random **`JWT_SECRET`** (never use the default `change-me-in-production`).
- [ ] Set explicit **`CORS_ALLOWED_ORIGINS`** (avoid `*`); credentials are auto-disabled for `*` or empty.
- [ ] Enable **`DB_SSLMODE=require`** (or stricter) in `DATABASE_DSN`.
- [ ] Restrict **`/metrics`** and **`/health/*`** at the network edge (they are unauthenticated).
- [ ] Replace default database credentials in any non-local deployment.
- [ ] Set `MAX_BODY_BYTES` to a sensible limit for your traffic (default 1 MiB).

## Implemented controls (Day 01)

| Area | Control | CWE |
|------|---------|-----|
| Body limit | `MAX_BODY_BYTES` middleware via `http.MaxBytesReader` | CWE-400 |
| CORS | Allow-list; credentials disabled for `*` or empty | CWE-942 |
| DSN parsing | `net/url` parse (passwords with `@:/?#%` safe) | CWE-116 |
| Pagination | `page` clamped to `MaxPage` (1,000,000) | CWE-190 |
| Config parsing | `envOrDefaultInt32` with 32-bit bounds | gosec G115 |
| Error responses | Generic client message; full detail to `slog` only | CWE-209 |
| Health probes | `/health/ready` returns generic `unhealthy` markers | CWE-209 |
| JWT default | Startup `slog.Warn` when `JWT_SECRET` is the default | — |
| Migration names | `migrate.sh create NAME` sanitizes to `[a-zA-Z0-9_]` | CWE-22 |
| Container | Multi-stage Dockerfile, `alpine:3.24` runtime, non-root `appuser` | — |
| Local compose | Postgres + Redis bound to `127.0.0.1` | CWE-942 |

## Pending (Day 08 hardening pass)

- Outbound HTTP client: no-redirect + 30s timeout + 1 MiB body cap (CWE-522) — if any outbound client is added.
- Full `golangci-lint`, `govulncheck`, `gosec` clean run.
- Integration tests with `testcontainers-go` (cut from 2-day timeline; add later).

## Verification

```bash
go build ./... && go vet ./... && go test ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
go run github.com/securego/gosec/v2/cmd/gosec@latest -quiet ./...
```

## Secrets policy

- **Never commit** `.env`, `*.token`, `*.key`, or any file containing `JWT_SECRET`, `DATABASE_DSN`, `REDIS_URL`.
- Commit only `.env.example` with placeholder values.
- Render env vars are set via the Render MCP or dashboard, never in git.
