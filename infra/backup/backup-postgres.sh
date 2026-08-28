#!/usr/bin/env bash
# Takes a logical (pg_dump custom-format) backup of a PostgreSQL database and
# writes a timestamped dump + sha256 checksum into the given output
# directory (wi-101). This is the portable-export / small-drill
# path; production-scale RPO is met by PITR (base backup + WAL archive),
# documented in docs/runbooks/backup-restore-dr.md and not automated here.
#
# Connects via the standard libpq PGHOST/PGPORT/PGUSER/PGPASSWORD/PGDATABASE
# environment variables (same convention as infra/schema/README.md's
# psqldef workflow). There is no default target: the caller must export
# these explicitly.
#
# Usage: backup-postgres.sh <output-dir>
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <output-dir>" >&2
  exit 1
fi

for var in PGHOST PGPORT PGUSER PGDATABASE; do
  if [ -z "${!var:-}" ]; then
    echo "error: $var must be set (see infra/backup/README.md)" >&2
    exit 1
  fi
done

readonly OUTPUT_DIR="$1"
mkdir -p "$OUTPUT_DIR"

TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
readonly TIMESTAMP
readonly DUMP_FILE="${OUTPUT_DIR}/idmagic-${TIMESTAMP}.dump"

echo "== pg_dump ${PGDATABASE}@${PGHOST}:${PGPORT} -> ${DUMP_FILE} ==" >&2
pg_dump -Fc -f "$DUMP_FILE"

readonly CHECKSUM_FILE="${DUMP_FILE}.sha256"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "$DUMP_FILE" >"$CHECKSUM_FILE"
else
  shasum -a 256 "$DUMP_FILE" >"$CHECKSUM_FILE"
fi

echo "backup complete: ${DUMP_FILE}"
echo "checksum:        ${CHECKSUM_FILE}"
