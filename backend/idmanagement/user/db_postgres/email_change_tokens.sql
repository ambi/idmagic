-- name: DeleteEmailChangeTokensForSub :exec
DELETE FROM email_change_tokens WHERE user_id = $1;

-- name: InsertEmailChangeToken :exec
INSERT INTO email_change_tokens (token_hash, id, user_id, purpose, new_email, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: FindUnusedEmailChangeToken :one
SELECT token_hash, id, user_id, purpose, new_email, created_at, expires_at
FROM email_change_tokens
WHERE token_hash = $1 AND used_at IS NULL;

-- name: MarkEmailChangeTokenUsed :one
UPDATE email_change_tokens
SET used_at = sqlc.arg(used_at)::timestamptz
WHERE token_hash = sqlc.arg(token_hash) AND used_at IS NULL
RETURNING token_hash, id, user_id, purpose, new_email, created_at, expires_at;
