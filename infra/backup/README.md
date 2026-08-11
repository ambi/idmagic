# PostgreSQL backup / restore scripts

Scripts here implement the pg_dump / logical-restore path of the backup and
DR strategy. Procedures, DR scenarios, and the verification checklist are in
[`infra/runbooks/backup-restore-dr.md`](../runbooks/backup-restore-dr.md);
this README only covers how to run the scripts.

## Requirements

- `pg_dump` / `pg_restore` / `psql` (PostgreSQL 17 client tools, matching the
  server version in `infra/docker/docker-compose.dev.yaml`).
- `psqldef` (see `infra/schema/README.md` for install instructions).
- Go toolchain (for `idmagic-batch restore-consistency-check`, invoked by
  `restore-postgres.sh` via `go run`).

## Connection variables

All scripts connect via the standard libpq environment variables, the same
convention `infra/schema/README.md` uses for `psqldef`:

```bash
export PGHOST=localhost
export PGPORT=5432
export PGUSER=idmagic
export PGPASSWORD=idmagic
export PGDATABASE=idmagic
```

There is no default target — every script requires these to be set
explicitly so a backup or restore never silently targets the wrong
database.

## Usage

```bash
# Take a backup (pg_dump custom format + sha256 checksum).
just backup-postgres <output-dir>

# Restore a backup into an empty target database. The db name must be
# typed out explicitly as a non-production guard.
just restore-postgres <backup-file> <db-name>

# Run the full local backup -> simulated loss -> restore -> consistency
# check drill against a disposable docker compose project.
just restore-drill
```

`restore-postgres.sh` refuses to run against a database that already has
tenant rows (restore into a freshly created, empty database), applies
`infra/schema/postgres.sql` via `psqldef` first, restores data, truncates
the ephemeral UNLOGGED/LOGGED tables, and finishes by running
`idmagic-batch restore-consistency-check`.
