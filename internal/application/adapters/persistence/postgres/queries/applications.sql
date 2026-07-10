-- name: ListApplicationsByTenant :many
SELECT tenant_id, application_id, name, kind, status, icon_url, icon_object_key, launch_url, bindings, category_ids, created_at, updated_at
FROM applications
WHERE tenant_id = $1
ORDER BY name;

-- name: GetApplicationByID :one
SELECT tenant_id, application_id, name, kind, status, icon_url, icon_object_key, launch_url, bindings, category_ids, created_at, updated_at
FROM applications
WHERE tenant_id = $1 AND application_id = $2;

-- name: UpsertApplication :exec
INSERT INTO applications (tenant_id, application_id, name, kind, status, icon_url, icon_object_key, launch_url, bindings, category_ids, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (tenant_id, application_id) DO UPDATE SET
  name = EXCLUDED.name,
  kind = EXCLUDED.kind,
  status = EXCLUDED.status,
  icon_url = EXCLUDED.icon_url,
  icon_object_key = EXCLUDED.icon_object_key,
  launch_url = EXCLUDED.launch_url,
  bindings = EXCLUDED.bindings,
  category_ids = EXCLUDED.category_ids,
  updated_at = EXCLUDED.updated_at;

-- name: DeleteApplication :exec
DELETE FROM applications WHERE tenant_id = $1 AND application_id = $2;

-- name: RemoveApplicationCategory :exec
UPDATE applications SET category_ids = array_remove(category_ids, $2), updated_at = now()
WHERE tenant_id = $1 AND $2 = ANY(category_ids);
