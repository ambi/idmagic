-- name: SaveMfaEnrollmentBypass :exec
INSERT INTO mfa_enrollment_bypasses (id, tenant_id, user_id, issued_by, issued_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: FindActiveMfaEnrollmentBypass :one
SELECT id, tenant_id, user_id, issued_by, issued_at, expires_at, consumed_at, revoked_at, expired_at
FROM mfa_enrollment_bypasses
WHERE tenant_id = $1 AND user_id = $2 AND consumed_at IS NULL AND revoked_at IS NULL AND expired_at IS NULL AND expires_at > $3
ORDER BY issued_at DESC LIMIT 1;

-- name: ConsumeActiveMfaEnrollmentBypass :one
UPDATE mfa_enrollment_bypasses
SET consumed_at = $3
WHERE id = (
    SELECT b.id FROM mfa_enrollment_bypasses b
    WHERE b.tenant_id = $1 AND b.user_id = $2 AND b.consumed_at IS NULL AND b.revoked_at IS NULL AND b.expired_at IS NULL AND b.expires_at > $3
    ORDER BY b.issued_at DESC LIMIT 1 FOR UPDATE SKIP LOCKED
)
RETURNING id, tenant_id, user_id, issued_by, issued_at, expires_at, consumed_at, revoked_at, expired_at;

-- name: RevokeActiveMfaEnrollmentBypass :one
UPDATE mfa_enrollment_bypasses
SET revoked_at = $3
WHERE id = (
    SELECT b.id FROM mfa_enrollment_bypasses b
    WHERE b.tenant_id = $1 AND b.user_id = $2 AND b.consumed_at IS NULL AND b.revoked_at IS NULL AND b.expired_at IS NULL
    ORDER BY b.issued_at DESC LIMIT 1 FOR UPDATE SKIP LOCKED
)
RETURNING id, tenant_id, user_id, issued_by, issued_at, expires_at, consumed_at, revoked_at, expired_at;

-- name: ExpireOpenMfaEnrollmentBypass :one
UPDATE mfa_enrollment_bypasses
SET expired_at = $3
WHERE id = (
    SELECT b.id FROM mfa_enrollment_bypasses b
    WHERE b.tenant_id = $1 AND b.user_id = $2 AND b.consumed_at IS NULL AND b.revoked_at IS NULL AND b.expired_at IS NULL AND b.expires_at <= $3
    ORDER BY b.issued_at DESC LIMIT 1 FOR UPDATE SKIP LOCKED
)
RETURNING id, tenant_id, user_id, issued_by, issued_at, expires_at, consumed_at, revoked_at, expired_at;
