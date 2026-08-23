#!/usr/bin/env bash
# Runs a full local backup -> simulated db loss -> restore ->
# consistency-check drill against a disposable, isolated docker compose
# project, and prints elapsed time as a local RPO/RTO estimate
# (wi-101). Mirrors the disposable-compose-project + trap cleanup
# pattern from infra/schema/check-convergence.sh so it never touches a
# developer's running dev stack.
#
# This drill covers the pg_dump / logical-restore path only. PITR and a
# Vault Transit key-provider drill are documented in
# docs/operations/backup-restore-dr.md as follow-up work: this repo's dev
# compose has no Vault/OpenBao service and no staging environment to run
# them against.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."

readonly PROJECT="idmagic-backup-drill"
readonly WORKDIR="$(mktemp -d)"

export PGHOST=localhost
export PGPORT=15432
export PGUSER=idmagic
export PGPASSWORD=idmagic
export PGDATABASE=idmagic

compose() {
  docker compose -p "$PROJECT" \
    -f infra/docker/docker-compose.dev.yaml \
    -f infra/docker/docker-compose.backup-drill.yaml \
    "$@"
}

cleanup() {
  compose down -v --remove-orphans >/dev/null 2>&1 || true
  rm -rf "$WORKDIR"
}
trap cleanup EXIT

echo "== starting disposable postgres + applying schema ==" >&2
compose up -d postgres
compose run --rm schema

echo "== seeding representative data ==" >&2
DATABASE_URL="postgres://${PGUSER}:${PGPASSWORD}@${PGHOST}:${PGPORT}/${PGDATABASE}?sslmode=disable" \
PERSISTENCE=postgres \
DEMO_CLIENT_SECRET=demo-client-secret \
DEMO_USER_PASSWORD=demo-password-1234 \
  go run ./backend/cmd/idmagic-seed --environment development --profile development --mode apply

# The development seed manifest provisions tenants/users/clients but does
# not issue a token, so no signing key exists yet (SigningKeys bootstraps
# lazily on first use, like DataKeys). A real production tenant will have
# issued at least one token by the time it is backed up; insert a
# representative active signing key so the drill exercises the JWKS
# continuity check the same way a live restore would.
echo "== seeding a representative active signing key per tenant ==" >&2
psql -Atqc "
  INSERT INTO signing_keys (kid, tenant_id, alg, provider, key_usage, scope_id, public_jwk, private_jwk, active, created_at, updated_at)
  SELECT 'drill-' || t.id, t.id, 'PS256', 'Postgres', 'Signing', 'default', '{}', '{}', true, now(), now()
  FROM tenants t
"

echo "== backup ==" >&2
readonly BACKUP_START="$(date +%s)"
./infra/backup/backup-postgres.sh "$WORKDIR"
readonly BACKUP_FILE="$(ls "$WORKDIR"/*.dump)"
readonly BACKUP_END="$(date +%s)"

echo "== simulating db loss (drop and recreate the database) ==" >&2
psql -d postgres -Atqc "DROP DATABASE ${PGDATABASE}"
psql -d postgres -Atqc "CREATE DATABASE ${PGDATABASE}"

echo "== restore ==" >&2
readonly RESTORE_START="$(date +%s)"
./infra/backup/restore-postgres.sh "$BACKUP_FILE" --yes-restore-into-this-database "$PGDATABASE"
readonly RESTORE_END="$(date +%s)"

echo
echo "== drill summary =="
echo "backup duration:  $((BACKUP_END - BACKUP_START))s"
echo "restore duration: $((RESTORE_END - RESTORE_START))s (local RTO estimate, includes consistency check)"
echo "backup artifact:  ${BACKUP_FILE}"
echo "checksum:         ${BACKUP_FILE}.sha256"
