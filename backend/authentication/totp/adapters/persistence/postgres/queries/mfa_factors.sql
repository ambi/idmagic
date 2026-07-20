-- name: ListMfaFactorsBySub :many
SELECT user_id, type, secret, label, created_at, last_used_at
FROM mfa_factors
WHERE user_id = $1
ORDER BY created_at;

-- name: GetMfaFactor :one
SELECT user_id, type, secret, label, created_at, last_used_at
FROM mfa_factors
WHERE user_id = $1 AND type = $2;

-- name: UpsertMfaFactor :exec
INSERT INTO mfa_factors (user_id, type, secret, label, created_at, last_used_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, type) DO UPDATE SET
    secret = EXCLUDED.secret,
    label = EXCLUDED.label,
    last_used_at = EXCLUDED.last_used_at,
    updated_at = now();

-- name: DeleteMfaFactor :exec
DELETE FROM mfa_factors WHERE user_id = $1 AND type = $2;

-- name: DeleteMfaFactorsForSub :exec
DELETE FROM mfa_factors WHERE user_id = $1;
