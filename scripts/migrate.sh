#!/usr/bin/env bash
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/internal/infrastructure/postgres/migrations"
DSN="${DATABASE_DSN:?set DATABASE_DSN}"

cmd="${1:-status}"
case "$cmd" in
  up)
    goose -dir "$DIR" postgres "$DSN" up
    ;;
  down)
    goose -dir "$DIR" postgres "$DSN" down
    ;;
  status)
    goose -dir "$DIR" postgres "$DSN" status
    ;;
  create)
    name="${2:?usage: migrate.sh create NAME}"
    if [[ ! "$name" =~ ^[a-zA-Z0-9_]+$ ]]; then
      echo "error: NAME must match [a-zA-Z0-9_]" >&2
      exit 1
    fi
    goose -dir "$DIR" create "$name" sql
    ;;
  *)
    echo "usage: migrate.sh {up|down|status|create NAME}" >&2
    exit 1
    ;;
esac
