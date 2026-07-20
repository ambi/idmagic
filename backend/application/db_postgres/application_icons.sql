-- name: UpsertApplicationIcon :exec
INSERT INTO application_icons (application_id, object_key, content_type, size_bytes, data, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (application_id, object_key) DO UPDATE SET
  content_type = EXCLUDED.content_type,
  size_bytes = EXCLUDED.size_bytes,
  data = EXCLUDED.data,
  updated_at = EXCLUDED.updated_at;

-- name: GetApplicationIcon :one
SELECT a.tenant_id, i.application_id, i.object_key, i.content_type, i.size_bytes, i.data, i.created_at, i.updated_at
FROM application_icons i JOIN applications a ON a.application_id = i.application_id
WHERE a.tenant_id = $1 AND i.application_id = $2 AND i.object_key = $3;

-- name: DeleteApplicationIconsByApplication :exec
DELETE FROM application_icons i WHERE i.application_id = $2
  AND EXISTS (SELECT 1 FROM applications a WHERE a.tenant_id = $1 AND a.application_id = $2);
