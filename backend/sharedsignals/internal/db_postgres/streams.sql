-- name: ListSsfStreamsByTenant :many
SELECT id,tenant_id,direction,event_types,status,created_at,updated_at
FROM ssf_streams WHERE tenant_id=$1 ORDER BY created_at;

-- name: FindSsfStreamByID :one
SELECT id,tenant_id,direction,event_types,status,created_at,updated_at
FROM ssf_streams WHERE tenant_id=$1 AND id=$2;

-- name: SaveSsfStream :exec
INSERT INTO ssf_streams (id,tenant_id,direction,event_types,status,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (id) DO UPDATE SET
 event_types=EXCLUDED.event_types,status=EXCLUDED.status,updated_at=EXCLUDED.updated_at;

-- name: DeleteSsfStream :exec
DELETE FROM ssf_streams WHERE tenant_id=$1 AND id=$2;
