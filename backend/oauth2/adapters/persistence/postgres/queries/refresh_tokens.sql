-- name: GetRefreshTokenByHash :one
SELECT id::text, hash, family_id::text, COALESCE(parent_id::text, '') AS parent_id, client_id, user_id,
  scopes, issued_at, expires_at, absolute_expires_at, revoked, rotated, sender_constraint,
  COALESCE(sid::text, '') AS sid, resource
FROM refresh_tokens
WHERE hash = $1;

-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (
  id, hash, family_id, parent_id, client_id, user_id, scopes, issued_at,
  expires_at, absolute_expires_at, revoked, rotated, sender_constraint, sid, resource
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NULLIF($13, 'null')::jsonb, $14, $15
);

-- name: RevokeRefreshTokensBySid :exec
UPDATE refresh_tokens
SET revoked = TRUE, updated_at = now()
WHERE sid = $1;

-- name: GetRefreshTokenRotationState :one
SELECT rotated, revoked
FROM refresh_tokens
WHERE id = $1
FOR UPDATE;

-- name: MarkRefreshTokenRotated :exec
UPDATE refresh_tokens
SET rotated = TRUE, updated_at = now()
WHERE id = $1;

-- name: RevokeRefreshTokenFamily :exec
UPDATE refresh_tokens
SET revoked = TRUE, updated_at = now()
WHERE family_id = $1;

-- name: DeleteRefreshTokensForSub :exec
DELETE FROM refresh_tokens
WHERE user_id = $1;
