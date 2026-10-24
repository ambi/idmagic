-- name: CreateDirectoryConnector :exec
INSERT INTO directory_connectors (
    id, tenant_id, display_name, status, enrollment_token_hash, enrollment_expires_at,
    public_key, key_id, created_at, enrolled_at, last_seen_at, disabled_at, revision
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: UpdateDirectoryConnector :execrows
-- 最終接続時刻は TouchDirectoryConnectorLastSeen だけが書くので、ここでは触れない。
UPDATE directory_connectors SET
    display_name = sqlc.arg(display_name),
    status = sqlc.arg(status),
    enrollment_token_hash = sqlc.narg(enrollment_token_hash),
    enrollment_expires_at = sqlc.narg(enrollment_expires_at),
    public_key = sqlc.narg(public_key),
    key_id = sqlc.narg(key_id),
    enrolled_at = sqlc.narg(enrolled_at),
    disabled_at = sqlc.narg(disabled_at),
    revision = sqlc.arg(revision)
WHERE tenant_id = sqlc.arg(tenant_id) AND id = sqlc.arg(id) AND revision = sqlc.arg(revision) - 1;

-- name: FindDirectoryConnector :one
SELECT * FROM directory_connectors WHERE tenant_id = $1 AND id = $2;

-- name: FindDirectoryConnectorByEnrollmentTokenHash :one
SELECT * FROM directory_connectors WHERE tenant_id = $1 AND enrollment_token_hash = $2;

-- name: ListDirectoryConnectors :many
SELECT * FROM directory_connectors WHERE tenant_id = $1 ORDER BY created_at, id;

-- name: TouchDirectoryConnectorLastSeen :exec
UPDATE directory_connectors SET last_seen_at = $3
WHERE tenant_id = $1 AND id = $2 AND status = 'active';
