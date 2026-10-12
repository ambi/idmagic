-- name: SaveApiToken :exec
INSERT INTO api_tokens (
    id, tenant_id, user_id, jti, client_id, scopes, audience, dpop_jkt,
    description, created_at, expires_at, revoked_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (id) DO UPDATE SET
    user_id=EXCLUDED.user_id,
    jti=EXCLUDED.jti,
    client_id=EXCLUDED.client_id,
    scopes=EXCLUDED.scopes,
    audience=EXCLUDED.audience,
    dpop_jkt=EXCLUDED.dpop_jkt,
    description=EXCLUDED.description,
    expires_at=EXCLUDED.expires_at,
    revoked_at=EXCLUDED.revoked_at,
    updated_at=now();

-- name: FindApiTokenByJTI :one
SELECT id, tenant_id, user_id, jti, client_id, scopes, audience, dpop_jkt,
       description, created_at, expires_at, revoked_at
FROM api_tokens WHERE tenant_id = $1 AND jti = $2;

-- name: ListApiTokensByTenant :many
SELECT id, tenant_id, user_id, jti, client_id, scopes, audience, dpop_jkt,
       description, created_at, expires_at, revoked_at
FROM api_tokens WHERE tenant_id = $1 ORDER BY created_at, id;

-- name: RevokeApiToken :exec
UPDATE api_tokens SET revoked_at = COALESCE(revoked_at, $3), updated_at = now()
WHERE tenant_id = $1 AND id = $2;

-- name: RevokeApiTokenByJTI :exec
UPDATE api_tokens SET revoked_at = COALESCE(revoked_at, $3), updated_at = now()
WHERE tenant_id = $1 AND jti = $2;
