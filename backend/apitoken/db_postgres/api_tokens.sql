-- name: SaveApiToken :exec
INSERT INTO api_tokens (id, tenant_id, token_hash, scopes, description, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO UPDATE SET
    token_hash=EXCLUDED.token_hash,
    scopes=EXCLUDED.scopes,
    description=EXCLUDED.description,
    expires_at=EXCLUDED.expires_at,
    updated_at=now();

-- name: FindApiTokenByHash :one
SELECT id, tenant_id, token_hash, scopes, description, created_at, expires_at
FROM api_tokens WHERE token_hash = $1;

-- name: ListApiTokensByTenant :many
SELECT id, tenant_id, token_hash, scopes, description, created_at, expires_at
FROM api_tokens WHERE tenant_id = $1 ORDER BY created_at, id;

-- name: DeleteApiToken :exec
DELETE FROM api_tokens WHERE tenant_id = $1 AND id = $2;
