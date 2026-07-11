-- name: RecentPasswordHistory :many
SELECT encoded, created_at
FROM password_history
WHERE user_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2;

-- name: InsertPasswordHistory :exec
INSERT INTO password_history (user_id, encoded, created_at) VALUES ($1, $2, $3);

-- name: DeletePasswordHistoryForSub :exec
DELETE FROM password_history WHERE user_id = $1;
