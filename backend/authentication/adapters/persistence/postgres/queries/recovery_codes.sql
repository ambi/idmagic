-- name: ListRecoveryCodesBySub :many
SELECT user_id, code_hash, generated_at, consumed_at
FROM recovery_codes
WHERE user_id = $1
ORDER BY generated_at;

-- name: InsertRecoveryCode :exec
INSERT INTO recovery_codes (user_id, code_hash, generated_at, consumed_at)
VALUES ($1, $2, $3, $4);

-- name: MarkRecoveryCodeConsumed :execrows
UPDATE recovery_codes
SET consumed_at = $3
WHERE user_id = $1 AND code_hash = $2 AND consumed_at IS NULL;

-- name: DeleteRecoveryCodesForSub :exec
DELETE FROM recovery_codes WHERE user_id = $1;
