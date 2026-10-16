-- name: DeletePasswordResetTokensByUser :exec
DELETE FROM password_reset_tokens WHERE user_id=$1;

-- name: InsertPasswordResetToken :exec
INSERT INTO password_reset_tokens (token_hash,id,user_id,purpose,created_at,expires_at)
VALUES ($1,$2,$3,$4,$5,$6);

-- name: FindUnusedPasswordResetToken :one
SELECT token_hash,id,user_id,purpose,created_at,expires_at
FROM password_reset_tokens
WHERE token_hash=$1 AND used_at IS NULL;

-- name: MarkPasswordResetTokenUsed :one
UPDATE password_reset_tokens
SET used_at=sqlc.arg(used_at)::timestamptz
WHERE token_hash=sqlc.arg(token_hash) AND used_at IS NULL
RETURNING token_hash,id,user_id,purpose,created_at,expires_at;
