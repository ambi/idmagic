-- name: ClaimIdempotencyRecord :one
INSERT INTO http_idempotency_records (id, tenant_id, principal_id, key_hash, fingerprint, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (tenant_id, principal_id, key_hash) DO UPDATE SET
    id = EXCLUDED.id, fingerprint = EXCLUDED.fingerprint,
    created_at = EXCLUDED.created_at, expires_at = EXCLUDED.expires_at,
    response_key_version = 0, response_ciphertext = NULL
WHERE http_idempotency_records.expires_at <= EXCLUDED.created_at
RETURNING *;

-- name: FindIdempotencyRecord :one
SELECT * FROM http_idempotency_records
WHERE tenant_id = $1 AND principal_id = $2 AND key_hash = $3;

-- name: CompleteIdempotencyRecord :execrows
UPDATE http_idempotency_records
SET response_key_version = $2, response_ciphertext = $3
WHERE id = $1 AND response_key_version = 0;

-- name: PurgeIdempotencyRecords :exec
DELETE FROM http_idempotency_records WHERE expires_at <= $1;

-- name: ListIdempotencyResponsesPendingReencryption :many
SELECT * FROM http_idempotency_records
WHERE tenant_id = $1 AND response_key_version > 0 AND response_key_version <> $2
    AND NOT (response_key_version = ANY(sqlc.arg(excluded_versions)::integer[]))
ORDER BY id
LIMIT $3;

-- name: ReplaceIdempotencyResponseCipher :execrows
UPDATE http_idempotency_records SET response_key_version = $2, response_ciphertext = $3
WHERE id = $1 AND response_key_version = sqlc.arg(old_version);

-- name: CountIdempotencyResponsesPendingReencryption :one
SELECT count(*) FROM http_idempotency_records
WHERE tenant_id = $1 AND response_key_version > 0 AND response_key_version <> $2;
