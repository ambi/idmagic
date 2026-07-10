-- name: GetConsent :one
SELECT user_id, client_id, scopes, created_at, updated_at, granted_at, expires_at, revoked_at
FROM consents
WHERE user_id = $1 AND client_id = $2;

-- name: ListConsentsByTenant :many
SELECT c.user_id, c.client_id, c.scopes, c.created_at, c.updated_at, c.granted_at, c.expires_at, c.revoked_at
FROM consents c
JOIN users u ON c.user_id = u.id
WHERE u.tenant_id = $1
ORDER BY c.user_id, c.client_id;

-- name: UpsertConsent :exec
INSERT INTO consents (user_id, client_id, scopes, granted_at, expires_at, revoked_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, client_id) DO UPDATE SET
  scopes = EXCLUDED.scopes,
  granted_at = EXCLUDED.granted_at,
  expires_at = EXCLUDED.expires_at,
  revoked_at = EXCLUDED.revoked_at,
  updated_at = now();

-- name: RevokeConsent :exec
UPDATE consents SET revoked_at = now(), updated_at = now()
WHERE user_id = $1 AND client_id = $2 AND revoked_at IS NULL;

-- name: DeleteConsentsForSub :exec
DELETE FROM consents WHERE user_id = $1;
