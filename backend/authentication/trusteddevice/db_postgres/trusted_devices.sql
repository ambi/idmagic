-- name: FindTrustedDeviceBySelector :one
SELECT id, tenant_id, user_id, selector, verifier_hash, label,
       created_at, last_used_at, expires_at, revoked_at, revoke_reason
FROM trusted_devices
WHERE tenant_id = $1 AND selector = $2;

-- name: FindTrustedDeviceByID :one
SELECT id, tenant_id, user_id, selector, verifier_hash, label,
       created_at, last_used_at, expires_at, revoked_at, revoke_reason
FROM trusted_devices
WHERE tenant_id = $1 AND user_id = $2 AND id = $3;

-- name: ListActiveTrustedDevicesByUser :many
SELECT id, tenant_id, user_id, selector, verifier_hash, label,
       created_at, last_used_at, expires_at, revoked_at, revoke_reason
FROM trusted_devices
WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > $3
ORDER BY last_used_at DESC;

-- name: UpsertTrustedDevice :exec
INSERT INTO trusted_devices (
    id, tenant_id, user_id, selector, verifier_hash, label,
    created_at, last_used_at, expires_at, revoked_at, revoke_reason
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (id) DO UPDATE SET
    selector = EXCLUDED.selector,
    verifier_hash = EXCLUDED.verifier_hash,
    label = EXCLUDED.label,
    last_used_at = EXCLUDED.last_used_at,
    expires_at = EXCLUDED.expires_at,
    revoked_at = EXCLUDED.revoked_at,
    revoke_reason = EXCLUDED.revoke_reason;

-- name: RevokeTrustedDevicesForUser :many
UPDATE trusted_devices
SET revoked_at = $3, revoke_reason = $4
WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL
RETURNING id, tenant_id, user_id, selector, verifier_hash, label,
          created_at, last_used_at, expires_at, revoked_at, revoke_reason;

-- name: DeleteTrustedDevicesForSub :exec
DELETE FROM trusted_devices WHERE user_id = $1;
