#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
MODE=${1:-durable}
RUN_DIR=$(mktemp -d "${TMPDIR:-/tmp}/idmagic-dev.XXXXXX")
# Persist the embedded PostgreSQL cluster across runs so `initdb` only pays its
# cost once. The schema is reset on every start (devinfra DROP SCHEMA + reapply),
# so the persistent directory only caches the initialized cluster, never state.
PG_DATA_DIR=${IDMAGIC_DEV_PG_DIR:-$ROOT_DIR/.dev/postgres}
DEV_API_ADDR=${ADDR:-:8081}
DEV_ISSUER=${ISSUER:-http://localhost:5173}
API_PID=
WORKER_PID=
INFRA_PID=
UI_PID=

cleanup() {
  # The stack is disposable, so tear it down fast. API/worker/UI hold no durable
  # state and are killed immediately. The dev-infra process must stop gracefully:
  # a SIGKILL would orphan the embedded postmaster it spawned and leave the
  # PostgreSQL port bound for the next run.
  for pid in "$UI_PID" "$API_PID" "$WORKER_PID"; do
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
      kill -9 "$pid" 2>/dev/null || true
    fi
  done
  if [ -n "$INFRA_PID" ] && kill -0 "$INFRA_PID" 2>/dev/null; then
    kill "$INFRA_PID" 2>/dev/null || true
    wait "$INFRA_PID" 2>/dev/null || true
  fi
  rm -rf "$RUN_DIR"
}
trap cleanup EXIT INT TERM

if [ "$MODE" != "durable" ] && [ "$MODE" != "memory" ]; then
  echo "usage: $0 [durable|memory]" >&2
  exit 2
fi

command -v go >/dev/null || {
  echo "go is required" >&2
  exit 1
}
command -v bun >/dev/null || {
  echo "bun is required" >&2
  exit 1
}

if [ ! -d "$ROOT_DIR/frontend/node_modules" ]; then
  echo "Installing UI dependencies..."
  (cd "$ROOT_DIR/frontend" && bun install --frozen-lockfile)
fi

echo "Building local development binaries..."
build_pkgs=(./backend/cmd/idmagic)
if [ "$MODE" = "durable" ]; then
  build_pkgs+=(./backend/cmd/idmagic-worker ./backend/cmd/idmagic-dev-infra)
fi
(
  cd "$ROOT_DIR"
  go build -o "$RUN_DIR/" "${build_pkgs[@]}"
)

# Start the UI first: it does not depend on the database, so its dev-server
# warm-up overlaps with the embedded infrastructure bring-up below.
echo "Starting UI at http://localhost:5173"
echo "Demo credentials: alice / demo-password-1234"
(
  cd "$ROOT_DIR/frontend"
  exec bun ./node_modules/vite/bin/vite.js
) &
UI_PID=$!

DATABASE_URL=
if [ "$MODE" = "durable" ]; then
  READY_FILE="$RUN_DIR/infra-ready.json"
  mkdir -p "$(dirname "$PG_DATA_DIR")"
  echo "Starting embedded PostgreSQL development endpoint"
  (
    cd "$ROOT_DIR"
    exec "$RUN_DIR/idmagic-dev-infra" --ready-file "$READY_FILE" --data-dir "$PG_DATA_DIR"
  ) &
  INFRA_PID=$!

  ready_wait=0
  while [ ! -f "$READY_FILE" ]; do
    if ! kill -0 "$INFRA_PID" 2>/dev/null; then
      wait "$INFRA_PID" || true
      echo "Development infrastructure exited before becoming ready" >&2
      exit 1
    fi
    if [ "$ready_wait" -ge 300 ]; then
      echo "Timed out waiting for development infrastructure" >&2
      exit 1
    fi
    sleep 1
    ready_wait=$((ready_wait + 1))
  done

  DATABASE_URL="postgres://idmagic:idmagic@127.0.0.1:55432/idmagic?sslmode=disable"
else
  echo "Starting lightweight memory mode: durable jobs and CSV import are unavailable"
fi

echo "Starting idmagic API at $DEV_API_ADDR"
(
  cd "$ROOT_DIR"
  : "${WEBAUTHN_RP_ID:=localhost}"
  : "${WEBAUTHN_RP_ORIGINS:=http://localhost:5173}"
  : "${WEBAUTHN_RP_DISPLAY_NAME:=IdMagic Local}"
  : "${DEMO_CLIENT_SECRET:=demo-client-secret}"
  : "${DEMO_USER_PASSWORD:=demo-password-1234}"
  export WEBAUTHN_RP_ID WEBAUTHN_RP_ORIGINS WEBAUTHN_RP_DISPLAY_NAME
  export DEMO_CLIENT_SECRET DEMO_USER_PASSWORD
  export ADDR="$DEV_API_ADDR" ISSUER="$DEV_ISSUER" EVENT_SINK=console SEED_ENVIRONMENT=development SEED_PROFILE=development
  if [ "$MODE" = "durable" ]; then
    export PERSISTENCE=postgres DATABASE_URL
  else
    export PERSISTENCE=memory
  fi
  exec "$RUN_DIR/idmagic"
) &
API_PID=$!

if [ "$MODE" = "durable" ]; then
  echo "Starting idmagic worker"
  (
    cd "$ROOT_DIR"
    export PERSISTENCE=postgres DATABASE_URL EVENT_SINK=console
    exec "$RUN_DIR/idmagic-worker"
  ) &
  WORKER_PID=$!
fi

while :; do
  for process in "infra:$INFRA_PID" "api:$API_PID" "worker:$WORKER_PID" "ui:$UI_PID"; do
    name=${process%%:*}
    pid=${process#*:}
    if [ -n "$pid" ] && ! kill -0 "$pid" 2>/dev/null; then
      status=0
      wait "$pid" || status=$?
      echo "$name process exited (status $status); stopping local development stack" >&2
      exit "$status"
    fi
  done
  sleep 1
done
