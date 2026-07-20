-- name: InsertJob :one
-- ON CONFLICT matches the jobs_tenant_dedup_key_active_idx partial unique index
-- (JobHandlerIdempotency). DO NOTHING means no rows are returned when an active
-- Job with the same (tenant_id, dedup_key) already exists; the caller then looks
-- it up with FindActiveJobByDedupKey. lane (ADR-129) is derived by the usecase
-- from kind's registration, never chosen by the enqueue caller.
INSERT INTO jobs (id, tenant_id, kind, lane, status, params, attempts, max_attempts, dedup_key, run_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, 'queued', $5, 0, $6, $7, $8, $9, $9)
ON CONFLICT (tenant_id, dedup_key) WHERE dedup_key IS NOT NULL AND status IN ('queued', 'running')
DO NOTHING
RETURNING id, tenant_id, kind, lane, status, params, result, error, attempts, max_attempts, dedup_key, lease_owner, lease_expires_at, run_at, created_at, updated_at;

-- name: FindActiveJobByDedupKey :one
SELECT id, tenant_id, kind, lane, status, params, result, error, attempts, max_attempts, dedup_key, lease_owner, lease_expires_at, run_at, created_at, updated_at
FROM jobs
WHERE tenant_id = $1 AND dedup_key = $2 AND status IN ('queued', 'running');

-- name: ClaimJobs :many
-- Claimable is either a StatusQueued job whose run_at is due, or a StatusRunning
-- job whose lease already expired (a crashed/drained worker's job, reclaimed for
-- a new attempt without changing its status), restricted to a single lane
-- (ADR-129 lane isolation: a lane's worker never claims another lane's backlog).
-- Both cases increment attempts.
WITH claimable AS (
    SELECT id FROM jobs
    WHERE jobs.lane = $2
      AND ((status = 'queued' AND run_at <= $1)
       OR (status = 'running' AND lease_expires_at IS NOT NULL AND lease_expires_at < $1))
    ORDER BY run_at
    FOR UPDATE SKIP LOCKED
    LIMIT $3
)
UPDATE jobs SET
    status = 'running',
    attempts = jobs.attempts + 1,
    lease_owner = $4,
    lease_expires_at = $5,
    updated_at = $1
FROM claimable
WHERE jobs.id = claimable.id
RETURNING jobs.id, jobs.tenant_id, jobs.kind, jobs.lane, jobs.status, jobs.params, jobs.result, jobs.error, jobs.attempts, jobs.max_attempts, jobs.dedup_key, jobs.lease_owner, jobs.lease_expires_at, jobs.run_at, jobs.created_at, jobs.updated_at;

-- name: HeartbeatJob :one
UPDATE jobs SET lease_expires_at = $4, updated_at = $3
WHERE id = $1 AND lease_owner = $2 AND status = 'running' AND lease_expires_at >= $3
RETURNING lease_expires_at;

-- name: CompleteJob :one
UPDATE jobs SET status = 'succeeded', result = $4, lease_owner = NULL, lease_expires_at = NULL, updated_at = $3
WHERE id = $1 AND lease_owner = $2 AND status = 'running' AND lease_expires_at >= $3
RETURNING id, tenant_id, kind, lane, status, params, result, error, attempts, max_attempts, dedup_key, lease_owner, lease_expires_at, run_at, created_at, updated_at;

-- name: FailJob :one
-- $4 (next_status) is 'queued' (retry) or 'failed' (dead-letter), decided by the
-- caller (usecases.Runner.fail); $6 (next_run_at) is only applied on retry.
UPDATE jobs SET
    status = $4,
    error = $5,
    run_at = CASE WHEN $4 = 'queued' THEN $6 ELSE run_at END,
    lease_owner = NULL,
    lease_expires_at = NULL,
    updated_at = $3
WHERE id = $1 AND lease_owner = $2 AND status = 'running' AND lease_expires_at >= $3
RETURNING id, tenant_id, kind, lane, status, params, result, error, attempts, max_attempts, dedup_key, lease_owner, lease_expires_at, run_at, created_at, updated_at;

-- name: CancelJob :one
UPDATE jobs SET status = 'canceled', lease_owner = NULL, lease_expires_at = NULL, updated_at = $2
WHERE id = $1 AND status IN ('queued', 'running')
RETURNING id, tenant_id, kind, lane, status, params, result, error, attempts, max_attempts, dedup_key, lease_owner, lease_expires_at, run_at, created_at, updated_at;

-- name: GetJob :one
SELECT id, tenant_id, kind, lane, status, params, result, error, attempts, max_attempts, dedup_key, lease_owner, lease_expires_at, run_at, created_at, updated_at
FROM jobs WHERE id = $1;

-- name: LaneDepths :many
-- wi-261 T006: point-in-time queued/running row counts per lane, for the
-- worker's periodic queue-depth/active gauge sampling. Lanes with zero rows
-- of both statuses are simply absent (no GROUP BY output row), matching
-- ports.JobRepository.LaneDepths' documented contract.
SELECT
    lane,
    count(*) FILTER (WHERE status = 'queued')::bigint AS queued,
    count(*) FILTER (WHERE status = 'running')::bigint AS running
FROM jobs
WHERE status IN ('queued', 'running')
GROUP BY lane;
