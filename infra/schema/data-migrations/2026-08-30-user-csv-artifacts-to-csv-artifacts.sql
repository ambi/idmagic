-- wi-350 / The CSV artifact store is no longer User-specific: User and Group CSV
-- both keep their upload payloads and their result error pages in it. Run this
-- once against an existing environment BEFORE applying the updated
-- infra/schema/postgres.sql.
--
-- psqldef compares a desired schema against the live one and cannot see a rename,
-- so applying postgres.sql without this script first would DROP the old tables and
-- CREATE empty new ones, discarding every stored artifact. Renaming here keeps the
-- rows, so an import job queued before the deploy still reads its payload after it.
--
-- Per §5, data migrations are kept out of the declarative schema file and stored as
-- a one-off script instead.
--
-- Idempotent: safe to re-run. `ALTER TABLE IF EXISTS ... RENAME TO` does nothing once
-- the old name is gone, and the constraint/index renames are guarded the same way.
-- Reversible: swap the two names in every statement to roll the deploy back.
ALTER TABLE IF EXISTS user_csv_artifacts RENAME TO csv_artifacts;
ALTER TABLE IF EXISTS user_csv_artifact_chunks RENAME TO csv_artifact_chunks;

ALTER TABLE IF EXISTS csv_artifacts
    RENAME CONSTRAINT user_csv_artifacts_tenant_id_fkey TO csv_artifacts_tenant_id_fkey;
ALTER TABLE IF EXISTS csv_artifact_chunks
    RENAME CONSTRAINT user_csv_artifact_chunks_artifact_id_fkey TO csv_artifact_chunks_artifact_id_fkey;

ALTER INDEX IF EXISTS user_csv_artifacts_tenant_created_idx
    RENAME TO csv_artifacts_tenant_created_idx;
