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

-- name: ListConsentsByTenantPage :many
-- First page of ListAdminConsents keyset pagination (wi-159, ADR-158): the
-- (user_id, client_id) primary key already backs this range scan.
SELECT c.user_id, c.client_id, c.scopes, c.created_at, c.updated_at, c.granted_at, c.expires_at, c.revoked_at
FROM consents c
JOIN users u ON c.user_id = u.id
WHERE u.tenant_id = $1
ORDER BY c.user_id, c.client_id
LIMIT sqlc.arg(page_limit);

-- name: ListConsentsByTenantPageAfter :many
-- Continuation page: resumes strictly after the (user_id, client_id) keyset
-- of the last row the caller saw.
SELECT c.user_id, c.client_id, c.scopes, c.created_at, c.updated_at, c.granted_at, c.expires_at, c.revoked_at
FROM consents c
JOIN users u ON c.user_id = u.id
WHERE u.tenant_id = $1
  AND (c.user_id, c.client_id) > (sqlc.arg(after_user_id)::uuid, sqlc.arg(after_client_id)::uuid)
ORDER BY c.user_id, c.client_id
LIMIT sqlc.arg(page_limit);

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
