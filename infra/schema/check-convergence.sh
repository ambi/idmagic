#!/usr/bin/env bash
# Verifies that infra/schema/postgres.sql converges under psqldef against an
# empty PostgreSQL database: apply -> dry-run (no-op) -> apply -> dry-run
# (no-op). Runs against an isolated, disposable compose project so it never
# touches a developer's running dev stack (see wi-308-reconsider-psqldef-adoption
# in work-items/done/ for the bug classes this specifically guards against).
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."

readonly NO_OP_MARKER="-- Nothing is modified --"

compose() {
  docker compose -p idmagic-schema-check \
    -f infra/docker/docker-compose.dev.yaml \
    -f infra/docker/docker-compose.schema-check.yaml \
    "$@"
}

cleanup() {
  compose down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

apply() {
  compose run --rm schema
}

dry_run() {
  compose run --rm schema \
    -U idmagic -h postgres -p 5432 idmagic --dry-run --file /schema/postgres.sql
}

check_converged() {
  local label="$1" out
  out="$(dry_run)"
  if [ "$out" != "$NO_OP_MARKER" ]; then
    echo "psqldef dry-run produced DDL $label:" >&2
    echo "$out" >&2
    echo "See infra/schema/README.md Rules for known psqldef bug classes (wi-308)." >&2
    exit 1
  fi
}

compose up -d postgres

echo "== apply (1st, empty database) =="
apply >/dev/null

echo "== dry-run (expect no-op) =="
check_converged "after applying to an empty database"

echo "== apply (2nd, should be a no-op) =="
apply >/dev/null

echo "== dry-run (expect no-op again) =="
check_converged "after a second, supposedly no-op apply"

echo "schema convergence check passed"
