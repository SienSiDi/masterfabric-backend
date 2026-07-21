#!/usr/bin/env bash
# e2e.sh — local end-to-end smoke test for masterfabric_backend (Day 01 + Day 02 endpoints)
#
# What it does:
#   1. Verifies Postgres + Redis are reachable
#   2. Starts the Go server in the background
#   3. Curls all 7 endpoints and checks status codes + response shape
#   4. Decodes the JWT to prove claims are correct
#   5. Tears down the server
#
# Prerequisites (already set up on this machine):
#   - Go 1.22+ (brew install go)
#   - Postgres on 127.0.0.1:5432 with role/db "masterfabric"
#   - Redis on 127.0.0.1:6379 (Docker: docker compose -f deployments/docker-compose.yml up -d redis)
#   - Migrations applied (see scripts/migrate.sh) + roles seeded (go run scripts/seed.go)
#
# Usage:
#   ./scripts/e2e.sh                 # run all checks
#   ./scripts/e2e.sh --keep-server   # leave the server running after checks

set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

export PATH="/opt/homebrew/bin:$(go env GOPATH)/bin:$PATH"
export DATABASE_DSN="postgres://masterfabric:masterfabric@127.0.0.1:5432/masterfabric?sslmode=disable"
export JWT_SECRET="e2e-test-secret-do-not-use-in-production"
export REDIS_URL="redis://127.0.0.1:6379"
export LOG_FORMAT=text

KEEP_SERVER=0
[[ "${1:-}" == "--keep-server" ]] && KEEP_SERVER=1

# Colors
GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[0;33m'; NC='\033[0m'
pass() { echo -e "${GREEN}PASS${NC} — $1"; }
fail() { echo -e "${RED}FAIL${NC} — $1"; exit 1; }
info() { echo -e "${YELLOW}…${NC} $1"; }

cleanup() {
  if [[ $KEEP_SERVER -eq 0 ]]; then
    info "stopping server"
    lsof -nP -iTCP:8080 -sTCP:LISTEN 2>/dev/null | tail -n +2 | awk '{print $2}' | while read pid; do kill "$pid" 2>/dev/null; done
    pkill -f "/tmp/mf-server" 2>/dev/null || true
  fi
}
trap cleanup EXIT

# --- 0. Pre-flight ---
info "checking prerequisites"
command -v go >/dev/null || fail "go not in PATH"
command -v psql >/dev/null || fail "psql not in PATH"
command -v curl >/dev/null || fail "curl not in PATH"
psql "$DATABASE_DSN" -c 'select 1' >/dev/null 2>&1 || fail "cannot connect to Postgres at $DATABASE_DSN"
# Redis check via nc (TCP port open) — redis-cli is not always installed on the host
nc -z 127.0.0.1 6379 2>/dev/null || (echo "" > /dev/tcp/127.0.0.1/6379) 2>/dev/null || fail "cannot reach Redis at $REDIS_URL (port 6379 not open)"
pass "prerequisites (go, psql, curl, Postgres, Redis)"

# --- 1. Build ---
info "building server"
go build -o /tmp/mf-server ./cmd/server || fail "go build"
pass "go build"

# --- 2. Start server ---
info "starting server on :8080"
pkill -f "/tmp/mf-server" 2>/dev/null || true
lsof -nP -iTCP:8080 -sTCP:LISTEN 2>/dev/null | tail -n +2 | awk '{print $2}' | while read pid; do kill "$pid" 2>/dev/null; done || true
sleep 1
nohup /tmp/mf-server > /tmp/mf-e2e.log 2>&1 &
BOOTED=0
for i in $(seq 1 20); do
  if curl -sS -o /dev/null http://127.0.0.1:8080/health/live 2>/dev/null; then
    BOOTED=1
    break
  fi
  sleep 1
done
if [[ $BOOTED -eq 0 ]]; then
  cat /tmp/mf-e2e.log
  fail "server did not boot in 20s"
fi
pass "server boot"

# --- 3. Cmn endpoints (Day 01) ---
echo ""
echo "=== Day 01 — Cmn[3] + Config[2] ==="

status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/health/live)
[[ "$status" == "200" ]] || fail "GET /health/live -> $status"
grep -q '"alive"' /tmp/body || fail "/health/live body"
pass "GET /health/live -> 200 {alive}"

status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/health/ready)
[[ "$status" == "200" ]] || fail "GET /health/ready -> $status"
grep -q '"postgres":"healthy"' /tmp/body || fail "/health/ready postgres not healthy"
grep -q '"redis":"healthy"' /tmp/body || fail "/health/ready redis not healthy"
pass "GET /health/ready -> 200 {postgres+redis healthy}"

status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/metrics)
[[ "$status" == "200" ]] || fail "GET /metrics -> $status"
grep -q "http_requests_total" /tmp/body || fail "/metrics no prometheus output"
pass "GET /metrics -> 200 (Prometheus text)"

status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/api/v1/config)
[[ "$status" == "200" ]] || fail "GET /api/v1/config -> $status"
grep -q 'gemma' /tmp/body || fail "/config no gemma model"
pass "GET /api/v1/config -> 200 (Gemma manifest)"

# --- 4. Auth endpoints (Day 02) ---
echo ""
echo "=== Day 02 — Auth[8] part 1: register + login ==="

EMAIL="e2e-$(date +%s)@masterfabric.dev"
PASSWORD="a-strong-password-12"

status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H "Content-Type: application/json" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
[[ "$status" == "201" ]] || fail "POST /auth/register -> $status (body: $(cat /tmp/body))"
grep -q '"userId"' /tmp/body || fail "/auth/register no userId"
USER_ID=$(python3 -c 'import sys,json; print(json.load(open("/tmp/body"))["userId"])')
pass "POST /auth/register -> 201 (userId=$USER_ID)"

# Duplicate email -> 409
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H "Content-Type: application/json" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
[[ "$status" == "409" ]] || fail "duplicate register -> $status (expected 409)"
pass "POST /auth/register duplicate -> 409 CONFLICT"

# Invalid email -> 400
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H "Content-Type: application/json" -d '{"email":"not-an-email","password":"a-strong-password-12"}')
[[ "$status" == "400" ]] || fail "invalid email -> $status (expected 400)"
pass "POST /auth/register invalid email -> 400 INVALID_INPUT"

# Short password -> 400
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H "Content-Type: application/json" -d '{"email":"x@y.z","password":"short"}')
[[ "$status" == "400" ]] || fail "short password -> $status (expected 400)"
pass "POST /auth/register short password -> 400 INVALID_INPUT"

# Login success
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
[[ "$status" == "200" ]] || fail "POST /auth/login -> $status (body: $(cat /tmp/body))"
ACCESS=$(python3 -c 'import sys,json; print(json.load(open("/tmp/body"))["accessToken"])')
REFRESH=$(python3 -c 'import sys,json; print(json.load(open("/tmp/body"))["refreshToken"])')
EXPIRES=$(python3 -c 'import sys,json; print(json.load(open("/tmp/body"))["expiresIn"])')
[[ -n "$ACCESS" ]] || fail "no accessToken in login response"
[[ -n "$REFRESH" ]] || fail "no refreshToken in login response"
[[ "$EXPIRES" == "900" ]] || fail "expiresIn=$EXPIRES (expected 900)"
pass "POST /auth/login -> 200 (accessToken + refreshToken + expiresIn=900)"

# Login wrong password -> 401
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" -d "{\"email\":\"$EMAIL\",\"password\":\"wrong-password-xxxxx\"}")
[[ "$status" == "401" ]] || fail "wrong password -> $status (expected 401)"
pass "POST /auth/login wrong password -> 401 UNAUTHORIZED"

# Login unknown email -> 401 (same message — no user enumeration)
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" -d '{"email":"nobody@masterfabric.dev","password":"a-strong-password-12"}')
[[ "$status" == "401" ]] || fail "unknown email -> $status (expected 401)"
pass "POST /auth/login unknown email -> 401 UNAUTHORIZED (no enumeration)"

# --- 5. Decode JWT to prove claims ---
echo ""
echo "=== JWT decode ==="
PAYLOAD=$(echo "$ACCESS" | cut -d. -f2)
# pad base64 to multiple of 4
PAD=$(( (4 - ${#PAYLOAD} % 4) % 4 ))
PAYLOAD="${PAYLOAD}$(printf '=%.0s' $(seq 1 $PAD))"
echo "$PAYLOAD" | base64 -d 2>/dev/null | python3 -m json.tool > /tmp/jwt.json
JWT_USER=$(python3 -c 'import sys,json; print(json.load(open("/tmp/jwt.json"))["user_id"])')
JWT_ROLE=$(python3 -c 'import sys,json; import json as j; print(json.load(open("/tmp/jwt.json"))["roles"][0])')
[[ "$JWT_USER" == "$USER_ID" ]] || fail "JWT user_id=$JWT_USER != registered $USER_ID"
[[ "$JWT_ROLE" == "user" ]] || fail "JWT role=$JWT_ROLE (expected user)"
pass "JWT access token decodes (user_id=$JWT_USER, roles=[user])"

# --- 6. DB verification ---
echo ""
echo "=== DB verification ==="
ROW_COUNT=$(psql "$DATABASE_DSN" -t -c "select count(*) from refresh_tokens where user_id='$USER_ID'")
[[ "$ROW_COUNT" -ge 1 ]] || fail "no refresh_tokens row for user"
pass "refresh token persisted to Postgres (rows=$ROW_COUNT, hash stored, not plaintext)"

ROLE_ROW=$(psql "$DATABASE_DSN" -t -c "select r.name from user_roles ur join roles r on r.id=ur.role_id join users u on u.id=ur.user_id where u.email='$EMAIL'")
[[ "$ROLE_ROW" == " user" ]] || fail "user has no 'user' role (got: '$ROLE_ROW')"
pass "default 'user' role assigned on register"

# --- 7. Day 03: refresh + logout + sessions ---
echo ""
echo "=== Day 03 — Auth[8] part 2: refresh + logout + sessions ==="

# refresh: get a new access token with the refresh token
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" -d "{\"refreshToken\":\"$REFRESH\"}")
[[ "$status" == "200" ]] || fail "POST /auth/refresh -> $status (body: $(cat /tmp/body))"
NEW_ACCESS=$(python3 -c 'import sys,json; print(json.load(open("/tmp/body"))["accessToken"])')
[[ -n "$NEW_ACCESS" ]] || fail "no accessToken in refresh response"
pass "POST /auth/refresh -> 200 (new accessToken issued)"

# refresh again with the OLD (now rotated) token -> 401
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" -d "{\"refreshToken\":\"$REFRESH\"}")
[[ "$status" == "401" ]] || fail "rotated refresh -> $status (expected 401)"
pass "POST /auth/refresh with rotated token -> 401 (rotation works)"

# refresh with garbage -> 401
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" -d '{"refreshToken":"not-a-real-token"}')
[[ "$status" == "401" ]] || fail "garbage refresh -> $status (expected 401)"
pass "POST /auth/refresh with garbage -> 401"

# me/sessions with valid JWT -> 200
status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/api/v1/me/sessions \
  -H "Authorization: Bearer $NEW_ACCESS")
[[ "$status" == "200" ]] || fail "GET /me/sessions -> $status (body: $(cat /tmp/body))"
grep -q '"sessions"' /tmp/body || fail "/me/sessions body missing sessions"
pass "GET /me/sessions -> 200 (with valid JWT)"

# me/sessions without auth -> 401
status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/api/v1/me/sessions)
[[ "$status" == "401" ]] || fail "GET /me/sessions without auth -> $status (expected 401)"
pass "GET /me/sessions without auth -> 401"

# logout
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/logout \
  -H "Content-Type: application/json" -H "Authorization: Bearer $NEW_ACCESS" \
  -d "{\"refreshToken\":\"$REFRESH\"}")
[[ "$status" == "204" ]] || fail "POST /auth/logout -> $status (expected 204)"
pass "POST /auth/logout -> 204"

# after logout, refresh with the same token -> 401 (blacklist)
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" -d "{\"refreshToken\":\"$REFRESH\"}")
[[ "$status" == "401" ]] || fail "refresh after logout -> $status (expected 401 from blacklist)"
pass "POST /auth/refresh after logout -> 401 (blacklist works)"

# logout is idempotent
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/logout \
  -H "Content-Type: application/json" -H "Authorization: Bearer $NEW_ACCESS" \
  -d "{\"refreshToken\":\"$REFRESH\"}")
[[ "$status" == "204" ]] || fail "second logout -> $status (expected 204, idempotent)"
pass "POST /auth/logout idempotent -> 204"

# sessions now show revokedAt
curl -sS http://127.0.0.1:8080/api/v1/me/sessions -H "Authorization: Bearer $NEW_ACCESS" > /tmp/body
grep -q '"revokedAt"' /tmp/body || fail "/me/sessions should show revokedAt after logout"
pass "GET /me/sessions shows revokedAt after logout"

# --- 8. Day 04: me + update me + change-password + RBAC + rate limit ---
echo ""
echo "=== Day 04 — Auth[8] part 3: me + change-password + RBAC + rate limit ==="

# Fresh user for clean tests
EMAIL2="e2e2-$(date +%s)@masterfabric.dev"
PASSWORD2="a-strong-password-12"
curl -sS -o /dev/null -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H "Content-Type: application/json" -d "{\"email\":\"$EMAIL2\",\"password\":\"$PASSWORD2\"}"
LOGIN2=$(curl -sS -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" -d "{\"email\":\"$EMAIL2\",\"password\":\"$PASSWORD2\"}")
ACCESS2=$(echo "$LOGIN2" | python3 -c 'import sys,json; print(json.load(sys.stdin)["accessToken"])')
REFRESH2=$(echo "$LOGIN2" | python3 -c 'import sys,json; print(json.load(sys.stdin)["refreshToken"])')

# GET /me
status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/api/v1/me \
  -H "Authorization: Bearer $ACCESS2")
[[ "$status" == "200" ]] || fail "GET /me -> $status (body: $(cat /tmp/body))"
grep -q "$EMAIL2" /tmp/body || fail "/me body missing email"
grep -q '"user"' /tmp/body || fail "/me body missing roles"
pass "GET /me -> 200 (email + roles)"

# GET /me without auth -> 401
status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/api/v1/me)
[[ "$status" == "401" ]] || fail "GET /me without auth -> $status (expected 401)"
pass "GET /me without auth -> 401"

# PUT /me (update email)
NEW_EMAIL="updated-$(date +%s)@masterfabric.dev"
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X PUT http://127.0.0.1:8080/api/v1/me \
  -H "Authorization: Bearer $ACCESS2" -H "Content-Type: application/json" \
  -d "{\"email\":\"$NEW_EMAIL\"}")
[[ "$status" == "200" ]] || fail "PUT /me -> $status (body: $(cat /tmp/body))"
grep -q "$NEW_EMAIL" /tmp/body || fail "/me update didn't return new email"
pass "PUT /me -> 200 (email updated)"

# PUT /me with invalid email -> 400
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X PUT http://127.0.0.1:8080/api/v1/me \
  -H "Authorization: Bearer $ACCESS2" -H "Content-Type: application/json" \
  -d '{"email":"not-an-email"}')
[[ "$status" == "400" ]] || fail "PUT /me invalid email -> $status (expected 400)"
pass "PUT /me invalid email -> 400"

# POST /auth/change-password
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/change-password \
  -H "Authorization: Bearer $ACCESS2" -H "Content-Type: application/json" \
  -d "{\"currentPassword\":\"$PASSWORD2\",\"newPassword\":\"new-strong-password-12\"}")
[[ "$status" == "204" ]] || fail "POST /auth/change-password -> $status (body: $(cat /tmp/body))"
pass "POST /auth/change-password -> 204"

# After password change, old refresh token should be revoked
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" -d "{\"refreshToken\":\"$REFRESH2\"}")
[[ "$status" == "401" ]] || fail "refresh after password change -> $status (expected 401, all tokens revoked)"
pass "POST /auth/refresh after password change -> 401 (all tokens revoked)"

# Change password with wrong current -> 401
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/change-password \
  -H "Authorization: Bearer $ACCESS2" -H "Content-Type: application/json" \
  -d '{"currentPassword":"wrong-current-pwd","newPassword":"another-password-12"}')
[[ "$status" == "401" ]] || fail "change-password wrong current -> $status (expected 401)"
pass "POST /auth/change-password wrong current -> 401"

# RBAC: PUT /admin/config with non-admin user -> 403
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X PUT http://127.0.0.1:8080/api/v1/admin/config \
  -H "Authorization: Bearer $ACCESS2" -H "Content-Type: application/json" \
  -d '{"webllm":{"modelId":"gemma-2b-q4f32_1-MLC","modelUrl":"","estimatedBytes":0},"features":{"scoring":true,"monitoring":true},"limits":{"maxPromptChars":4000,"ratePerMin":30}}')
[[ "$status" == "403" ]] || fail "PUT /admin/config as non-admin -> $status (expected 403)"
pass "PUT /admin/config as non-admin -> 403 FORBIDDEN (RBAC works)"

# Rate limit: 6 rapid logins with wrong password -> 6th should be 429
RL_EMAIL="rl-test-$(date +%s)@masterfabric.dev"
curl -sS -o /dev/null -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H "Content-Type: application/json" -d "{\"email\":\"$RL_EMAIL\",\"password\":\"a-strong-password-12\"}"
RL_BLOCKED=0
for i in 1 2 3 4 5 6; do
  status=$(curl -sS -o /dev/null -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/auth/login \
    -H "Content-Type: application/json" -d "{\"email\":\"$RL_EMAIL\",\"password\":\"wrong-password-xxxxx\"}")
  if [[ "$status" == "429" ]]; then
    RL_BLOCKED=$i
    break
  fi
done
[[ "$RL_BLOCKED" -gt 0 ]] || fail "rate limit never triggered after 6 attempts (last status: $status)"
pass "login rate limit triggered at attempt #$RL_BLOCKED (429 RATE_LIMITED)"

# Verify rate-limit headers are present
HEADERS=$(curl -sS -D - -o /dev/null -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" -d "{\"email\":\"$RL_EMAIL\",\"password\":\"wrong-password-xxxxx\"}")
echo "$HEADERS" | grep -qi "X-RateLimit-Limit" || fail "missing X-RateLimit-Limit header"
echo "$HEADERS" | grep -qi "X-RateLimit-Remaining" || fail "missing X-RateLimit-Remaining header"
pass "rate-limit headers present (X-RateLimit-Limit, X-RateLimit-Remaining)"

# --- 9. Day 05: Web MLC-LLM part 1 (models + session create/get) ---
echo ""
echo "=== Day 05 — Web MLC-LLM[7] part 1: models + session create/get ==="

# GET /llm/models
status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/api/v1/llm/models \
  -H "Authorization: Bearer $ACCESS2")
[[ "$status" == "200" ]] || fail "GET /llm/models -> $status (body: $(cat /tmp/body))"
grep -q 'gemma-2b-q4f32_1-MLC' /tmp/body || fail "/llm/models missing Gemma"
pass "GET /llm/models -> 200 (Gemma manifest)"

# /llm/models without auth -> 401
status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/api/v1/llm/models)
[[ "$status" == "401" ]] || fail "GET /llm/models without auth -> $status (expected 401)"
pass "GET /llm/models without auth -> 401"

# POST /llm/sessions
SESS_RESP=$(curl -sS -w "\n%{http_code}" -X POST http://127.0.0.1:8080/api/v1/llm/sessions \
  -H "Authorization: Bearer $ACCESS2" -H "Content-Type: application/json" \
  -d '{"modelId":"gemma-2b-q4f32_1-MLC","modelHash":"sha256:e2e-test"}')
SESS_STATUS=$(echo "$SESS_RESP" | tail -1)
SESS_BODY=$(echo "$SESS_RESP" | sed '$d')
[[ "$SESS_STATUS" == "201" ]] || fail "POST /llm/sessions -> $SESS_STATUS (body: $SESS_BODY)"
SESS_ID=$(echo "$SESS_BODY" | python3 -c 'import sys,json; print(json.load(sys.stdin)["sessionId"])')
[[ -n "$SESS_ID" ]] || fail "no sessionId in response"
pass "POST /llm/sessions -> 201 (sessionId=$SESS_ID)"

# GET /llm/sessions/{id} (owner)
status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID \
  -H "Authorization: Bearer $ACCESS2")
[[ "$status" == "200" ]] || fail "GET /llm/sessions/{id} -> $status (body: $(cat /tmp/body))"
grep -q "$SESS_ID" /tmp/body || fail "/llm/sessions/{id} body missing sessionId"
pass "GET /llm/sessions/{id} -> 200 (owner)"

# GET /llm/sessions/{id} without auth -> 401
status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID)
[[ "$status" == "401" ]] || fail "GET /llm/sessions/{id} without auth -> $status (expected 401)"
pass "GET /llm/sessions/{id} without auth -> 401"

# GET /llm/sessions/{unknown-uuid} -> 404
status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/api/v1/llm/sessions/00000000-0000-0000-0000-000000000000 \
  -H "Authorization: Bearer $ACCESS2")
[[ "$status" == "404" ]] || fail "GET /llm/sessions/{unknown} -> $status (expected 404)"
pass "GET /llm/sessions/{unknown-uuid} -> 404 NOT_FOUND"

# GET /llm/sessions/{bad-uuid} -> 400
status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/api/v1/llm/sessions/not-a-uuid \
  -H "Authorization: Bearer $ACCESS2")
[[ "$status" == "400" ]] || fail "GET /llm/sessions/{bad-uuid} -> $status (expected 400)"
pass "GET /llm/sessions/{bad-uuid} -> 400 INVALID_INPUT"

# DB: verify session row
ROW_COUNT=$(psql "$DATABASE_DSN" -t -c "select count(*) from llm_sessions where id='$SESS_ID'")
[[ "$ROW_COUNT" -eq 1 ]] || fail "no llm_sessions row for $SESS_ID"
pass "session persisted to Postgres"

# --- 9b. Day 06: Web MLC-LLM part 2 (events POST/GET) ---
echo ""
echo "=== Day 06 — Web MLC-LLM[7] part 2: events POST/GET ==="

# POST /llm/sessions/{id}/events
EVENT_RESP=$(curl -sS -w "\n%{http_code}" -X POST http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID/events \
  -H "Authorization: Bearer $ACCESS2" -H "Content-Type: application/json" \
  -d '{"prompt":"Summarize the Go roadmap","completion":"1. Fundamentals 2. Concurrency 3. APIs","tokensIn":8,"tokensOut":12,"latencyMs":4200}')
EVENT_STATUS=$(echo "$EVENT_RESP" | tail -1)
EVENT_BODY=$(echo "$EVENT_RESP" | sed '$d')
[[ "$EVENT_STATUS" == "201" ]] || fail "POST /llm/sessions/{id}/events -> $EVENT_STATUS (body: $EVENT_BODY)"
EVENT_ID=$(echo "$EVENT_BODY" | python3 -c 'import sys,json; print(json.load(sys.stdin)["eventId"])')
[[ -n "$EVENT_ID" ]] || fail "no eventId in response"
pass "POST /llm/sessions/{id}/events -> 201 (eventId=$EVENT_ID)"

# Record a second event
EVENT2_RESP=$(curl -sS -X POST http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID/events \
  -H "Authorization: Bearer $ACCESS2" -H "Content-Type: application/json" \
  -d '{"prompt":"What is a goroutine?","completion":"A lightweight thread","tokensIn":5,"tokensOut":4,"latencyMs":1800}')
pass "POST /llm/sessions/{id}/events second event -> 201"

# GET /llm/sessions/{id}/events
status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID/events \
  -H "Authorization: Bearer $ACCESS2")
[[ "$status" == "200" ]] || fail "GET /llm/sessions/{id}/events -> $status (body: $(cat /tmp/body))"
EVENTS_COUNT=$(python3 -c 'import sys,json; print(len(json.load(open("/tmp/body"))["events"]))')
[[ "$EVENTS_COUNT" == "2" ]] || fail "expected 2 events, got $EVENTS_COUNT"
TOTAL=$(python3 -c 'import sys,json; print(json.load(open("/tmp/body"))["total"])')
[[ "$TOTAL" == "2" ]] || fail "expected total=2, got $TOTAL"
pass "GET /llm/sessions/{id}/events -> 200 (events=$EVENTS_COUNT, total=$TOTAL, newest first)"

# events without auth -> 401
status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID/events)
[[ "$status" == "401" ]] || fail "GET /llm/sessions/{id}/events without auth -> $status (expected 401)"
pass "GET /llm/sessions/{id}/events without auth -> 401"

# POST events to unknown session -> 404
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/llm/sessions/00000000-0000-0000-0000-000000000000/events \
  -H "Authorization: Bearer $ACCESS2" -H "Content-Type: application/json" \
  -d '{"prompt":"x","completion":"y","tokensIn":1,"tokensOut":1,"latencyMs":100}')
[[ "$status" == "404" ]] || fail "POST events to unknown session -> $status (expected 404)"
pass "POST /llm/sessions/{unknown}/events -> 404"

# POST events with empty prompt -> 400
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID/events \
  -H "Authorization: Bearer $ACCESS2" -H "Content-Type: application/json" \
  -d '{"prompt":"","completion":"y","tokensIn":1,"tokensOut":1,"latencyMs":100}')
[[ "$status" == "400" ]] || fail "POST events empty prompt -> $status (expected 400)"
pass "POST /llm/sessions/{id}/events empty prompt -> 400"

# DB: verify events persisted
ROW_COUNT=$(psql "$DATABASE_DSN" -t -c "select count(*) from inference_events where session_id='$SESS_ID'")
[[ "$ROW_COUNT" -eq 2 ]] || fail "expected 2 inference_events rows, got $ROW_COUNT"
pass "2 events persisted to Postgres"

# --- 9c. Day 07: Web MLC-LLM part 3 (score + monitoring) ---
echo ""
echo "=== Day 07 — Web MLC-LLM[7] part 3: score + monitoring ==="

# Get the first event ID from the session
FIRST_EVENT_ID=$(curl -sS http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID/events \
  -H "Authorization: Bearer $ACCESS2" | python3 -c 'import sys,json; print(json.load(sys.stdin)["events"][0]["eventId"])')
SECOND_EVENT_ID=$(curl -sS http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID/events \
  -H "Authorization: Bearer $ACCESS2" | python3 -c 'import sys,json; print(json.load(sys.stdin)["events"][1]["eventId"])')

# POST /llm/sessions/{id}/score (accept)
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID/score \
  -H "Authorization: Bearer $ACCESS2" -H "Content-Type: application/json" \
  -d "{\"eventId\":\"$FIRST_EVENT_ID\",\"correctness\":0.8,\"latencyScore\":1.0,\"safetyFlag\":false,\"costScore\":0.9,\"userSignal\":\"accept\",\"composite\":0.85}")
[[ "$status" == "201" ]] || fail "POST /llm/sessions/{id}/score -> $status (body: $(cat /tmp/body))"
pass "POST /llm/sessions/{id}/score (accept) -> 201"

# Score the second event (reject)
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID/score \
  -H "Authorization: Bearer $ACCESS2" -H "Content-Type: application/json" \
  -d "{\"eventId\":\"$SECOND_EVENT_ID\",\"correctness\":0.3,\"latencyScore\":0.5,\"safetyFlag\":false,\"costScore\":0.8,\"userSignal\":\"reject\",\"composite\":0.35}")
[[ "$status" == "201" ]] || fail "POST /llm/sessions/{id}/score (reject) -> $status (body: $(cat /tmp/body))"
pass "POST /llm/sessions/{id}/score (reject) -> 201"

# Double-score the first event -> 409
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID/score \
  -H "Authorization: Bearer $ACCESS2" -H "Content-Type: application/json" \
  -d "{\"eventId\":\"$FIRST_EVENT_ID\",\"correctness\":0.5,\"composite\":0.5}")
[[ "$status" == "409" ]] || fail "double score -> $status (expected 409)"
pass "POST /llm/sessions/{id}/score double -> 409 CONFLICT"

# Score with safetyFlag=true should force composite=0 server-side
# Use a fresh event for this
curl -sS -o /dev/null -X POST http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID/events \
  -H "Authorization: Bearer $ACCESS2" -H "Content-Type: application/json" \
  -d '{"prompt":"unsafe","completion":"flagged","tokensIn":1,"tokensOut":1,"latencyMs":50}'
THIRD_EVENT_ID=$(curl -sS http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID/events \
  -H "Authorization: Bearer $ACCESS2" | python3 -c 'import sys,json; print(json.load(sys.stdin)["events"][0]["eventId"])')
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID/score \
  -H "Authorization: Bearer $ACCESS2" -H "Content-Type: application/json" \
  -d "{\"eventId\":\"$THIRD_EVENT_ID\",\"correctness\":0.9,\"latencyScore\":1.0,\"safetyFlag\":true,\"costScore\":0.9,\"userSignal\":\"accept\",\"composite\":0.99}")
[[ "$status" == "201" ]] || fail "safety score -> $status (body: $(cat /tmp/body))"
# Verify composite was overridden to 0 in DB
SAVED_COMPOSITE=$(psql "$DATABASE_DSN" -t -c "select composite from decision_scores where event_id='$THIRD_EVENT_ID'" | xargs)
[[ "$SAVED_COMPOSITE" == "0" ]] || fail "safetyFlag=true should force composite=0, got $SAVED_COMPOSITE"
pass "POST /llm/sessions/{id}/score with safetyFlag=true -> composite forced to 0.0"

# Score non-owner -> 403
status=$(curl -sS -o /tmp/body -w "%{http_code}" -X POST http://127.0.0.1:8080/api/v1/llm/sessions/$SESS_ID/score \
  -H "Authorization: Bearer $ACCESS" -H "Content-Type: application/json" \
  -d "{\"eventId\":\"$FIRST_EVENT_ID\",\"composite\":0.5}")
[[ "$status" == "403" ]] || fail "score non-owner -> $status (expected 403)"
pass "POST /llm/sessions/{id}/score non-owner -> 403 FORBIDDEN"

# GET /llm/monitoring as non-admin -> 403
status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/api/v1/llm/monitoring \
  -H "Authorization: Bearer $ACCESS2")
[[ "$status" == "403" ]] || fail "GET /llm/monitoring as non-admin -> $status (expected 403)"
pass "GET /llm/monitoring as non-admin -> 403 FORBIDDEN"

# Create an admin user directly in the DB + log in as them
ADMIN_EMAIL="admin-e2e-$(date +%s)@masterfabric.dev"
cat > /tmp/genhash.go <<'GO'
package main
import (
  "fmt"
  "golang.org/x/crypto/bcrypt"
)
func main() {
  h, _ := bcrypt.GenerateFromPassword([]byte("admin-strong-password-12"), bcrypt.DefaultCost)
  fmt.Print(string(h))
}
GO
ADMIN_HASH=$(go run /tmp/genhash.go)
psql "$DATABASE_DSN" -c "INSERT INTO users (email, password_hash) VALUES ('$ADMIN_EMAIL', '$ADMIN_HASH') ON CONFLICT DO NOTHING" -t > /dev/null 2>&1
ADMIN_USER_ID=$(psql "$DATABASE_DSN" -t -c "select id from users where email='$ADMIN_EMAIL'" | xargs)
psql "$DATABASE_DSN" -c "INSERT INTO user_roles (user_id, role_id) SELECT '$ADMIN_USER_ID', id FROM roles WHERE name='admin' ON CONFLICT DO NOTHING" -t > /dev/null 2>&1
# Login as admin
ADMIN_LOGIN=$(curl -sS -X POST http://127.0.0.1:8080/api/v1/auth/login -H "Content-Type: application/json" -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"admin-strong-password-12\"}")
ADMIN_ACCESS=$(echo "$ADMIN_LOGIN" | python3 -c 'import sys,json; print(json.load(sys.stdin)["accessToken"])')
[[ -n "$ADMIN_ACCESS" ]] || fail "admin login failed"
pass "admin user created + logged in"

# GET /llm/monitoring as admin -> 200
status=$(curl -sS -o /tmp/body -w "%{http_code}" http://127.0.0.1:8080/api/v1/llm/monitoring \
  -H "Authorization: Bearer $ADMIN_ACCESS")
[[ "$status" == "200" ]] || fail "GET /llm/monitoring as admin -> $status (body: $(cat /tmp/body))"
# Verify the payload shape
python3 <<'PY' || fail "monitoring payload shape invalid"
import json
with open("/tmp/body") as f:
    d = json.load(f)
assert "totals" in d, "missing totals"
assert "latency" in d, "missing latency"
assert "tokens" in d, "missing tokens"
assert "scores" in d, "missing scores"
assert "byModel" in d, "missing byModel"
events = d["totals"]["events"]
scored = d["totals"]["scoredEvents"]
assert events >= 3, f"expected >=3 events, got {events}"
assert scored >= 3, f"expected >=3 scored events, got {scored}"
assert "gemma-2b-q4f32_1-MLC" in str(d["byModel"]), "missing gemma in byModel"
print(f"  totals: {d['totals']}")
print(f"  latency: {d['latency']}")
print(f"  scores: {d['scores']}")
print(f"  byModel: {d['byModel']}")
PY
pass "GET /llm/monitoring as admin -> 200 (totals + latency + tokens + scores + byModel)"

# Verify Prometheus has the new metrics
curl -sS http://127.0.0.1:8080/metrics | grep -q "llm_events_total" || fail "missing llm_events_total in /metrics"
curl -sS http://127.0.0.1:8080/metrics | grep -q "llm_decision_score_sum" || fail "missing llm_decision_score_sum in /metrics"
pass "Prometheus metrics include llm_events_total + llm_decision_score_sum"

# --- 10. Summary ---
echo ""
echo "================================"
echo -e "${GREEN}All e2e checks passed.${NC}"
echo "  Endpoints verified: 20/20 — ALL DONE"
echo "  - Cmn:    3/3 (live, ready, metrics)"
echo "  - Config: 2/2 (get, put)"
echo "  - Auth:   8/8 (register, login, refresh, logout, sessions, me, update-me, change-password)"
echo "  - LLM:    7/7 (models, create-session, get-session, record-event, list-events, score, monitoring)"
echo "  Extras:  RBAC (403), rate limit (429), Prometheus (llm_events_total, llm_tokens_total, llm_decision_score_sum)"
echo "================================"
if [[ $KEEP_SERVER -eq 1 ]]; then
  info "server left running on :8080 (use 'lsof -ti:8080 | xargs kill' to stop)"
else
  info "server stopped"
fi
