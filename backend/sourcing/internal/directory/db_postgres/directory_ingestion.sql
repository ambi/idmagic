-- name: SaveDirectoryUserCorrelation :exec
INSERT INTO directory_user_correlations (tenant_id, connector_id, external_id, user_id, last_seen_run_id)
VALUES ($1, $2, $3, $4, $5);

-- name: FindDirectoryUserCorrelation :one
SELECT * FROM directory_user_correlations WHERE tenant_id = $1 AND connector_id = $2 AND external_id = $3;

-- name: MarkDirectoryUserCorrelationSeen :exec
UPDATE directory_user_correlations SET last_seen_run_id = sqlc.arg(run_id)
WHERE tenant_id = sqlc.arg(tenant_id) AND connector_id = sqlc.arg(connector_id) AND external_id = sqlc.arg(external_id);

-- name: ListDirectoryUserCorrelationsNotSeenIn :many
SELECT * FROM directory_user_correlations
WHERE tenant_id = sqlc.arg(tenant_id) AND connector_id = sqlc.arg(connector_id) AND last_seen_run_id <> sqlc.arg(run_id)
ORDER BY external_id;

-- name: ListDirectoryUserCorrelations :many
SELECT * FROM directory_user_correlations WHERE tenant_id = $1 AND connector_id = $2 ORDER BY external_id;

-- name: ListDirectoryCorrelatedUserIDs :many
SELECT user_id FROM directory_user_correlations WHERE tenant_id = $1 AND user_id = ANY(sqlc.arg(user_ids)::text[]::uuid[]);

-- name: SupersedeRunningDirectoryIngestionRuns :exec
UPDATE directory_ingestion_runs SET status = 'superseded'
WHERE tenant_id = $1 AND connector_id = $2 AND status IN ('running', 'awaiting_approval');

-- name: CreateDirectoryIngestionRun :exec
INSERT INTO directory_ingestion_runs (id, tenant_id, connector_id, mode, status, started_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: FindDirectoryIngestionRun :one
SELECT * FROM directory_ingestion_runs WHERE tenant_id = $1 AND connector_id = $2 AND id = $3;

-- name: ListDirectoryIngestionRuns :many
SELECT * FROM directory_ingestion_runs WHERE tenant_id = $1 AND connector_id = $2
ORDER BY started_at DESC, id DESC LIMIT $3;

-- name: FindDirectoryIngestionBatch :one
SELECT * FROM directory_ingestion_batches WHERE tenant_id = $1 AND run_id = $2 AND batch_id = $3;

-- name: InsertDirectoryIngestionBatch :execrows
INSERT INTO directory_ingestion_batches (tenant_id, run_id, batch_id, digest, results, recorded_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (run_id, batch_id) DO NOTHING;

-- name: AddDirectoryIngestionRunCounts :exec
UPDATE directory_ingestion_runs SET
    batches = batches + 1,
    items = items + sqlc.arg(items),
    created = created + sqlc.arg(created),
    updated = updated + sqlc.arg(updated),
    unchanged = unchanged + sqlc.arg(unchanged),
    deleted = deleted + sqlc.arg(deleted),
    failed = failed + sqlc.arg(failed)
WHERE tenant_id = sqlc.arg(tenant_id) AND id = sqlc.arg(id);

-- name: ListDirectoryIngestionBatches :many
SELECT * FROM directory_ingestion_batches WHERE tenant_id = $1 AND run_id = $2 ORDER BY recorded_sequence;

-- name: CompleteDirectoryIngestionRun :execrows
UPDATE directory_ingestion_runs SET status = 'completed', completed_at = sqlc.arg(completed_at), disabled = sqlc.arg(disabled)
WHERE tenant_id = sqlc.arg(tenant_id) AND id = sqlc.arg(id) AND status = 'running';

-- name: SaveDirectoryIngestionProgress :exec
UPDATE directory_ingestion_runs SET status = sqlc.arg(status), completed_at = sqlc.arg(completed_at),
 disabled = sqlc.arg(disabled), protection = sqlc.narg(protection)
WHERE tenant_id = sqlc.arg(tenant_id) AND id = sqlc.arg(id);

-- name: HasApplyingDirectoryIngestionRun :one
SELECT EXISTS(SELECT 1 FROM directory_ingestion_runs WHERE tenant_id = $1 AND connector_id = $2 AND status = 'applying');

-- name: TryLockDirectoryIngestion :one
SELECT pg_try_advisory_xact_lock(hashtextextended(sqlc.arg(lock_key)::text, 0))::boolean;
