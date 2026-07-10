-- name: UpsertApplicationIcon :exec
INSERT INTO application_icons (tenant_id, application_id, object_key, content_type, size_bytes, data, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (tenant_id, application_id, object_key) DO UPDATE SET
  content_type = EXCLUDED.content_type,
  size_bytes = EXCLUDED.size_bytes,
  data = EXCLUDED.data,
  updated_at = EXCLUDED.updated_at;

-- name: GetApplicationIcon :one
SELECT tenant_id, application_id, object_key, content_type, size_bytes, data, created_at, updated_at
FROM application_icons
WHERE tenant_id = $1 AND application_id = $2 AND object_key = $3;

-- name: DeleteApplicationIconsByApplication :exec
DELETE FROM application_icons WHERE tenant_id = $1 AND application_id = $2;
