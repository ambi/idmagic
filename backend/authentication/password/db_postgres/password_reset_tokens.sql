-- name: DeletePasswordResetTokensByUser :exec
DELETE FROM password_reset_tokens WHERE user_id=$1;

-- name: InsertPasswordResetToken :exec
INSERT INTO password_reset_tokens (token_hash,user_id,created_at,expires_at) VALUES ($1,$2,$3,$4);

-- name: ConsumePasswordResetToken :one
DELETE FROM password_reset_tokens
WHERE token_hash=$1
RETURNING user_id,token_hash,created_at,expires_at;
