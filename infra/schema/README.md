# PostgreSQL schema workflow

`postgres.sql` is the declarative current-state schema for the PostgreSQL
adapter. It is applied with `psqldef`, the PostgreSQL command in the sqldef tool
family.

## Install psqldef

On macOS:

```bash
brew install sqldef/sqldef/psqldef
```

On Linux, download the pre-built `psqldef` binary from the sqldef releases page,
or use the sqldef Docker image in the deploy job. Pin the version in CI/CD jobs
instead of using an unqualified latest binary.

Confirm the installed command:

```bash
psqldef --version
```

## Connection variables

For local dev compose, the equivalent connection is:

```bash
export PGHOST=localhost
export PGPORT=5432
export PGUSER=idmagic
export PGPASSWORD=idmagic
export PGDATABASE=idmagic
```

`psqldef` uses `psql`-style connection options, so production deploy jobs should
map their `DATABASE_URL` secret to `PGHOST` / `PGPORT` / `PGUSER` /
`PGPASSWORD` / `PGDATABASE` before running the workflow.

## Change workflow

1. Edit `infra/schema/postgres.sql` to the desired current schema.
2. If the change needs data movement, add an explicit runbook or purpose-built
   SQL script for the backfill / value conversion. Do not hide data movement in
   the declarative schema file.
3. Generate the planned DDL without applying it:

```bash
psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --dry-run < infra/schema/postgres.sql \
  | tee /tmp/idmagic-schema-plan.sql
```

4. Review `/tmp/idmagic-schema-plan.sql`.
   - Empty output means the database already matches the current schema.
   - `DROP` operations require explicit human review and must not be enabled in
     automation by default.
   - Long-locking operations, type changes, and NOT NULL additions on populated
     tables need a separate rollout plan.
5. Apply the reviewed schema change:

```bash
psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --apply < infra/schema/postgres.sql
```

6. Run dry-run again. The expected result is empty output:

```bash
psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --dry-run < infra/schema/postgres.sql
```

7. Record the generated plan and the final empty dry-run in the WI completion or
   release evidence.

## Local Docker development

The dev compose file has a one-shot `schema` service:

```bash
just dev-compose
```

`schema` waits for PostgreSQL, runs `psqldef --apply --file
/schema/postgres.sql`, exits, and then `idp` starts. The apply step is
idempotent; running compose again should not produce additional DDL after the
database matches `postgres.sql`.

When only the schema changed and the stack is already running, apply it without
recreating the whole stack:

```bash
just schema-compose
```

To inspect the dev database before applying, run dry-run from the host after
installing `psqldef`:

```bash
psqldef -U idmagic -h localhost -p 5432 idmagic \
  --dry-run < infra/schema/postgres.sql
```

## Production deployment

The application does not apply schema changes at startup. Production uses an
explicit deploy step before starting the new application version.

First deployment to an empty database:

```bash
psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --dry-run < infra/schema/postgres.sql
psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --apply < infra/schema/postgres.sql
psqldef -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" "$PGDATABASE" \
  --check < infra/schema/postgres.sql
```

Second and later deployments use the same sequence. `--dry-run` shows the DDL
needed to move the existing database to the new desired schema. After review,
`--apply` performs it, and `--check` must return no pending DDL before the new
application version is promoted.

If the dry-run contains destructive or long-locking changes, stop and create a
separate rollout plan. Do not add `--enable-drop` to automated production jobs
without explicit approval for that release.

## Empty database bootstrap

For a new PostgreSQL database, apply `postgres.sql` directly with the same
`--apply` command. Reference data is not part of this file; the application
converges required rows such as the default tenant at startup.

## Rules

- Keep structural schema in `postgres.sql`.
- Keep data migrations, backfills, and high-risk destructive changes as explicit
  SQL scripts or runbooks outside `infra/schema/`.
- Do not put reference data in this file. The application converges required
  reference data such as the default tenant at startup.
- Do not reintroduce an application startup migration runner. Schema changes are
  a deploy-time operation.
- Do not use `--enable-drop` in automation without a reviewed migration plan.
- Do not put SQL comments (`--`) in `postgres.sql`. Two independent reasons:
  - Design rationale belongs in `ARCHITECTURE.md` (root or per-context), not in
    the DDL file — a comment restating "why" drifts from the real design record
    the same way a second copy of any decision does. See `ARCHITECTURE.md`
    `## Cross-cutting Concerns` > Database design policy for the column-type
    rules (ADR-084) and the `tenant_id` retention classes (ADR-082, simplified
    by ADR-083); do not restate them here.
  - `psqldef`'s dependency-aware statement ordering (introduced upstream in
    [sqldef/sqldef#1209](https://github.com/sqldef/sqldef/pull/1209)) has shown
    content-sensitive bugs where a comment's exact text — not its meaning —
    changes whether a table is emitted before or after its own indexes on an
    empty database, producing `relation "..." does not exist` during
    `--apply`. [sqldef/sqldef#1121](https://github.com/sqldef/sqldef/issues/1121)
    is a previously-fixed bug in the same family (a comment before `CREATE
    TABLE` made a trailing `CREATE INDEX` run first). Keeping this file
    comment-free sidesteps the whole bug class rather than chasing each
    occurrence.
- Never name a table-level (multi-column) `CONSTRAINT` so it looks exactly
  like PostgreSQL's own default name for an unnamed single-column constraint —
  `<table>_<column>_key` for `UNIQUE`, `<table>_<column>_check` for `CHECK`
  — when `<column>` is a real column of that table but the constraint actually
  covers more than that one column (e.g. `lifecycle_workflows_enabled_revision_check`
  covering both `status` and `enabled_revision`, or an unnamed
  `UNIQUE (tenant_id, group_id)` on `dynamic_group_rules`, which Postgres
  itself would name `dynamic_group_rules_tenant_id_group_id_key`). `psqldef`
  appears to treat a name matching that shape as an implicit/auto-generated
  constraint for comparison purposes; on re-`--apply` this has produced (a)
  spurious rename churn when the table has no explicit name at all, and (b)
  — confirmed against a real, deployed database — the constraint being
  silently `DROP`ped with no replacement `ADD`, permanently losing the
  invariant. Give such constraints an explicit name that does not collide
  with this pattern (e.g. `..._consistency` instead of `..._check`, or add a
  column to the name that isn't a real column).
- Write every `CHECK (col IN (...))` value list in alphabetical order.
  PostgreSQL preserves whatever literal order the schema was written in when
  it reports the constraint back (`pg_get_constraintdef`), but `psqldef`
  normalizes its own "desired state" representation of the same list to
  alphabetical order before comparing. If the source order isn't already
  alphabetical, the two never match and every `--apply` — even a true no-op —
  emits a pointless `DROP CONSTRAINT` / `ADD CONSTRAINT` pair for that column
  (harmless — the recreated constraint is semantically identical — but noisy,
  and it defeats using dry-run output as a reliable "did anything actually
  change" signal). This is the still-open
  [sqldef/sqldef#1295](https://github.com/sqldef/sqldef/issues/1295).
- Conventions this file keeps, because they are about writing SQL rather than
  design (see `ARCHITECTURE.md` for anything beyond these):
  - A table's own identifier is `id`; a reference to a User from another table
    is `user_id` (an owner reference is `owner_user_id`).
  - Every table has `created_at`. Tables whose rows can be updated after
    creation have `updated_at`; insert-only/delete-only rows do not. Domain
    timestamps (`issued_at`, `granted_at`, `occurred_at`, `expires_at`,
    `revoked_at`, `first_seen`, `last_seen`) keep their domain meaning and do
    not replace `created_at`.
  - Second-precision rounding happens only at external protocol boundaries
    (SCIM/SAML/WS-Fed formatting), never in the schema.
  - Go keeps UUID columns as string; `base.go` registers a text codec for the
    uuid OID.
  - Non-FK `tenant_id` columns (`audit_events`, `authentication_event_buckets`)
    stay `TEXT` and hold the UUID as string (`audit_events` also carries a `''`
    tenantless sentinel).
  - `users.lifecycle` is the flagged JSONB normalization candidate.
