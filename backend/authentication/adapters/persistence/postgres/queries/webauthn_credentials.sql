-- name: ListWebAuthnCredentialsBySub :many
SELECT credential_id, user_id, public_key, sign_count, transports, aaguid, label,
    backup_eligible, backup_state, created_at, last_used_at
FROM webauthn_credentials
WHERE user_id = $1
ORDER BY created_at;

-- name: GetWebAuthnCredentialByID :one
SELECT credential_id, user_id, public_key, sign_count, transports, aaguid, label,
    backup_eligible, backup_state, created_at, last_used_at
FROM webauthn_credentials
WHERE credential_id = $1;

-- name: UpsertWebAuthnCredential :exec
INSERT INTO webauthn_credentials (credential_id, user_id, public_key, sign_count, transports, aaguid, label,
    backup_eligible, backup_state, created_at, last_used_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (credential_id) DO UPDATE SET
    sign_count = EXCLUDED.sign_count,
    label = EXCLUDED.label,
    last_used_at = EXCLUDED.last_used_at,
    updated_at = now();

-- name: UpdateWebAuthnCredentialSignCount :exec
UPDATE webauthn_credentials
SET sign_count = $2, last_used_at = $3, updated_at = now()
WHERE credential_id = $1;

-- name: DeleteWebAuthnCredential :exec
DELETE FROM webauthn_credentials WHERE user_id = $1 AND credential_id = $2;

-- name: DeleteWebAuthnCredentialsForSub :exec
DELETE FROM webauthn_credentials WHERE user_id = $1;
