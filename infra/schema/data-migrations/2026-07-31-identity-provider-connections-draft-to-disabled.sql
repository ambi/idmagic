-- wi-309 / IdentityProviderConnectionStatus is simplified from
-- Draft/Active/Disabled to Active/Disabled. Run this once against an existing
-- environment BEFORE applying the updated infra/schema/postgres.sql (its
-- CHECK (status IN ('active', 'disabled')) constraint rejects 'draft' rows).
--
-- Per §5, data migrations are kept out of the declarative schema
-- file and stored as a one-off script instead.
--
-- Idempotent: safe to re-run: no rows match status = 'draft' after the first
-- successful run. This is a one-way migration (draft -> disabled only); a
-- connection only becomes Active again through an explicit administrator
-- activate call, never automatically by this script.
UPDATE identity_provider_connections
SET status = 'disabled', updated_at = now()
WHERE status = 'draft';
