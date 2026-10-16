-- name: ListMfaFactorsPendingReencryption :many
-- Rows with secret material (plaintext or ciphertext) not yet on
-- activeVersion: legacy plaintext (secret_key_version NULL) or an older DEK
-- version. Rows with neither secret nor secret_ciphertext (factor types that
-- carry no reversible secret) are excluded — there is nothing to migrate.
SELECT m.user_id, m.type, m.secret, m.secret_key_version, m.secret_ciphertext
FROM mfa_factors m
JOIN users u ON u.id = m.user_id
WHERE u.tenant_id = $1
  AND (m.secret IS NOT NULL OR m.secret_ciphertext IS NOT NULL)
  AND (m.secret_key_version IS NULL OR m.secret_key_version <> $2)
ORDER BY m.user_id, m.type
LIMIT $3;

-- name: CountMfaFactorsPendingReencryption :one
SELECT count(*) FROM mfa_factors m
JOIN users u ON u.id = m.user_id
WHERE u.tenant_id = $1
  AND (m.secret IS NOT NULL OR m.secret_ciphertext IS NOT NULL)
  AND (m.secret_key_version IS NULL OR m.secret_key_version <> $2);

-- name: UpdateMfaFactorCiphertext :exec
-- Writes back a re-encrypted secret without touching label/created_at/
-- last_used_at, and always clears the legacy plaintext column.
UPDATE mfa_factors
SET secret = NULL, secret_key_version = $3, secret_ciphertext = $4, updated_at = now()
WHERE user_id = $1 AND type = $2;
