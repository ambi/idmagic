-- name: ListApplicationCategoriesByTenant :many
SELECT tenant_id, category_id, name, position, created_at, updated_at
FROM application_categories
WHERE tenant_id = $1
ORDER BY position, name;

-- name: GetApplicationCategoryByID :one
SELECT tenant_id, category_id, name, position, created_at, updated_at
FROM application_categories
WHERE tenant_id = $1 AND category_id = $2;

-- name: UpsertApplicationCategory :exec
INSERT INTO application_categories (tenant_id, category_id, name, position, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tenant_id, category_id) DO UPDATE SET
  name = EXCLUDED.name,
  position = EXCLUDED.position,
  updated_at = EXCLUDED.updated_at;

-- name: DeleteApplicationCategory :exec
DELETE FROM application_categories WHERE tenant_id = $1 AND category_id = $2;
