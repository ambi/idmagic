-- name: ListSsfStreamsByTenant :many
SELECT id,tenant_id,direction,event_types,status,created_at,updated_at,audit_event_types,consecutive_failures,first_failure_at
FROM ssf_streams WHERE tenant_id=$1 ORDER BY created_at;

-- name: FindSsfStreamByID :one
SELECT id,tenant_id,direction,event_types,status,created_at,updated_at,audit_event_types,consecutive_failures,first_failure_at
FROM ssf_streams WHERE tenant_id=$1 AND id=$2;

-- name: SaveSsfStream :exec
INSERT INTO ssf_streams (id,tenant_id,direction,event_types,status,created_at,updated_at,audit_event_types)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (id) DO UPDATE SET
 event_types=EXCLUDED.event_types,status=EXCLUDED.status,updated_at=EXCLUDED.updated_at,
 audit_event_types=EXCLUDED.audit_event_types;

-- name: DeleteSsfStream :exec
DELETE FROM ssf_streams WHERE tenant_id=$1 AND id=$2;

-- name: RecordSsfStreamDeliveryFailure :one
WITH prior AS (
  SELECT status FROM ssf_streams WHERE tenant_id=sqlc.arg(tenant_id) AND id=sqlc.arg(id) FOR UPDATE
)
UPDATE ssf_streams s SET
  consecutive_failures=s.consecutive_failures+1,
  first_failure_at=COALESCE(s.first_failure_at,sqlc.arg(now)::timestamptz),
  status=CASE WHEN s.status='enabled' AND s.consecutive_failures+1>=sqlc.arg(threshold)::int
    AND sqlc.arg(now)::timestamptz-COALESCE(s.first_failure_at,sqlc.arg(now)::timestamptz)>=make_interval(secs=>sqlc.arg(min_elapsed_seconds)::float8)
    THEN 'disabled' ELSE s.status END
FROM prior
WHERE s.tenant_id=sqlc.arg(tenant_id) AND s.id=sqlc.arg(id)
RETURNING prior.status AS previous_status,s.status AS new_status;

-- name: ClearSsfStreamDeliveryFailures :exec
UPDATE ssf_streams SET consecutive_failures=0,first_failure_at=NULL WHERE tenant_id=$1 AND id=$2;
