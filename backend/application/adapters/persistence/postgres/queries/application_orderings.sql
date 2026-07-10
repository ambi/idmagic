-- name: GetApplicationOrdering :one
SELECT user_id, application_ids, created_at, updated_at
FROM application_orderings
WHERE user_id = $1;

-- name: UpsertApplicationOrdering :exec
INSERT INTO application_orderings (user_id, application_ids, created_at, updated_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE SET
  application_ids = EXCLUDED.application_ids,
  updated_at = EXCLUDED.updated_at;
