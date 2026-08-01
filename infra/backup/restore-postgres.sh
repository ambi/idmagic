#!/usr/bin/env bash
# Restores a pg_dump custom-format backup into a target PostgreSQL database
# and runs the post-restore consistency check (wi-101, ADR-153).
#
# Connects via the standard libpq PGHOST/PGPORT/PGUSER/PGPASSWORD/PGDATABASE
# environment variables (same convention as infra/schema/README.md's
# psqldef workflow). psqldef, psql, pg_restore, and idmagic-batch all read
# these directly; DATABASE_URL (derived below) is only for
# idmagic-batch restore-consistency-check.
#
# Order (ADR-153): refuse a non-empty target -> apply schema
# (infra/schema/postgres.sql) -> pg_restore data -> truncate ephemeral
# UNLOGGED/LOGGED tables -> idmagic-batch restore-consistency-check.
#
# Usage: restore-postgres.sh <backup-file> --yes-restore-into-this-database <db-name>
set -euo pipefail

if [ "$#" -ne 3 ] || [ "$2" != "--yes-restore-into-this-database" ]; then
  echo "usage: $0 <backup-file> --yes-restore-into-this-database <db-name>" >&2
  exit 1
fi

readonly BACKUP_FILE="$1"
readonly CONFIRM_DB_NAME="$3"

for var in PGHOST PGPORT PGUSER PGDATABASE; do
  if [ -z "${!var:-}" ]; then
    echo "error: $var must be set (see infra/backup/README.md)" >&2
    exit 1
  fi
done

# Non-production guard: the operator must type the exact target database
# name, forcing a deliberate look at what is about to be overwritten.
if [ "$CONFIRM_DB_NAME" != "$PGDATABASE" ]; then
  echo "non-production guard: --yes-restore-into-this-database must exactly match \$PGDATABASE (got '${CONFIRM_DB_NAME}', target is '${PGDATABASE}')" >&2
  exit 1
fi

if [ ! -f "$BACKUP_FILE" ]; then
  echo "backup file not found: $BACKUP_FILE" >&2
  exit 1
fi

if [ -f "${BACKUP_FILE}.sha256" ]; then
  echo "== verifying checksum ==" >&2
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$(dirname "$BACKUP_FILE")" && sha256sum -c "$(basename "${BACKUP_FILE}.sha256")")
  else
    (cd "$(dirname "$BACKUP_FILE")" && shasum -a 256 -c "$(basename "${BACKUP_FILE}.sha256")")
  fi
else
  echo "warning: no checksum file next to ${BACKUP_FILE}; skipping verification" >&2
fi

cd "$(dirname "${BASH_SOURCE[0]}")/../.."

echo "== refusing to restore into a non-empty database ==" >&2
EXISTING_TENANTS="$(psql -Atqc "SELECT count(*) FROM tenants" 2>/dev/null || echo "0")"
if [ "$EXISTING_TENANTS" != "0" ]; then
  echo "target database ${PGDATABASE} already has ${EXISTING_TENANTS} tenant row(s); restore into a freshly created, empty database instead." >&2
  exit 1
fi

echo "== applying schema (infra/schema/postgres.sql) ==" >&2
psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --apply <infra/schema/postgres.sql

echo "== pg_restore <- ${BACKUP_FILE} ==" >&2
# --data-only: schema was just applied by psqldef above from the declarative
# infra/schema/postgres.sql, so only rows are restored from the dump, not a
# second (conflicting) copy of the DDL. --disable-triggers skips per-row FK
# trigger checks so tables can load out of dependency order; it requires
# owning the tables, which the restoring role does since psqldef just
# created them.
pg_restore -d "$PGDATABASE" --data-only --disable-triggers --no-owner --no-privileges "$BACKUP_FILE"

echo "== truncating ephemeral UNLOGGED/LOGGED tables ==" >&2
psql -Atqc "
  TRUNCATE
    oauth2_authorization_requests,
    oauth2_authorization_codes,
    oauth2_par_requests,
    oauth2_device_codes,
    oauth2_replay_jtis,
    oauth2_access_token_denylist,
    webauthn_sessions,
    login_throttle_counters,
    saml_authnrequest_replays
"

echo "== running post-restore consistency check ==" >&2
DATABASE_URL="postgres://${PGUSER}:${PGPASSWORD:-}@${PGHOST}:${PGPORT}/${PGDATABASE}?sslmode=disable" \
  go run ./backend/cmd/idmagic-batch restore-consistency-check

echo "restore complete and consistent: ${PGDATABASE}@${PGHOST}:${PGPORT}"
