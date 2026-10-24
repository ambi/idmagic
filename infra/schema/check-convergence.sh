#!/usr/bin/env bash
# Verifies that infra/schema/postgres.sql converges under psqldef against an
# empty PostgreSQL database: apply -> dry-run (no-op) -> apply -> dry-run
# (no-op). The database is a disposable embedded PostgreSQL started by
# idmagic-dev-infra on a free port, so the check needs no Docker and never
# touches a developer's running dev stack (see wi-308-reconsider-psqldef-adoption
# in work-items/done/ for the bug classes this specifically guards against).
# psqldef is the version pinned in mise.toml.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."

readonly NO_OP_MARKER="-- Nothing is modified --"
readonly SCHEMA="infra/schema/postgres.sql"

WORK_DIR="$(mktemp -d "${TMPDIR:-/tmp}/idmagic-schema-check.XXXXXX")"
READY_FILE="$WORK_DIR/ready.json"
INFRA_PID=

cleanup() {
  if [ -n "$INFRA_PID" ]; then
    kill "$INFRA_PID" 2>/dev/null || true
    wait "$INFRA_PID" 2>/dev/null || true
  fi
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT

go build -o "$WORK_DIR/" ./backend/cmd/idmagic-dev-infra
"$WORK_DIR/idmagic-dev-infra" --empty --postgres-port 0 --ready-file "$READY_FILE" \
  --schema "$SCHEMA" >/dev/null &
INFRA_PID=$!

ready_wait=0
while [ ! -f "$READY_FILE" ]; do
  if ! kill -0 "$INFRA_PID" 2>/dev/null; then
    wait "$INFRA_PID" || true
    INFRA_PID=
    echo "embedded PostgreSQL exited before becoming ready" >&2
    exit 1
  fi
  if [ "$ready_wait" -ge 300 ]; then
    echo "timed out waiting for embedded PostgreSQL" >&2
    exit 1
  fi
  sleep 1
  ready_wait=$((ready_wait + 1))
done

# database_url is postgres://<user>:<password>@<host>:<port>/<database>?sslmode=disable.
database_url="$(jq -r .database_url "$READY_FILE")"
rest="${database_url#postgres://}"
credentials="${rest%%@*}"
rest="${rest#*@}"
host_port="${rest%%/*}"
database="${rest#*/}"
readonly PG_USER="${credentials%%:*}"
readonly PG_HOST="${host_port%:*}"
readonly PG_PORT="${host_port##*:}"
readonly PG_DATABASE="${database%%\?*}"
export PGPASSWORD="${credentials#*:}"

psqldef_run() {
  psqldef -U "$PG_USER" -h "$PG_HOST" -p "$PG_PORT" "$PG_DATABASE" --file "$SCHEMA" "$@"
}

check_converged() {
  local label="$1" out
  out="$(psqldef_run --dry-run)"
  if [ "$out" != "$NO_OP_MARKER" ]; then
    echo "psqldef dry-run produced DDL $label:" >&2
    echo "$out" >&2
    echo "See infra/schema/README.md Rules for known psqldef bug classes (wi-308)." >&2
    exit 1
  fi
}

echo "== apply (1st, empty database) =="
psqldef_run --apply >/dev/null

echo "== dry-run (expect no-op) =="
check_converged "after applying to an empty database"

echo "== apply (2nd, should be a no-op) =="
psqldef_run --apply >/dev/null

echo "== dry-run (expect no-op again) =="
check_converged "after a second, supposedly no-op apply"

echo "schema convergence check passed"
