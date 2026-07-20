-- name: SaveScimToken :exec
INSERT INTO scim_tokens (id, tenant_id, token_hash, description, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE SET
    token_hash=EXCLUDED.token_hash,
    description=EXCLUDED.description,
    expires_at=EXCLUDED.expires_at,
    updated_at=now();

-- name: FindScimTokenByHash :one
SELECT id, tenant_id, token_hash, description, created_at, expires_at
FROM scim_tokens WHERE token_hash = $1;

-- name: ListScimTokensByTenant :many
SELECT id, tenant_id, token_hash, description, created_at, expires_at
FROM scim_tokens WHERE tenant_id = $1;

-- name: DeleteScimToken :exec
DELETE FROM scim_tokens WHERE tenant_id = $1 AND id = $2;
