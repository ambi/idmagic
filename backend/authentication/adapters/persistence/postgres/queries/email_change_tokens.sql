-- name: DeleteEmailChangeTokensForSub :exec
DELETE FROM email_change_tokens WHERE user_id = $1;

-- name: InsertEmailChangeToken :exec
INSERT INTO email_change_tokens (token_hash, user_id, new_email, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5);

-- name: ConsumeEmailChangeToken :one
DELETE FROM email_change_tokens
WHERE token_hash = $1
RETURNING user_id, token_hash, new_email, created_at, expires_at;
