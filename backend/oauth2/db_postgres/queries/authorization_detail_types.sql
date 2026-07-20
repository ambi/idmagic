-- name: GetAuthorizationDetailType :one
SELECT tenant_id, type, description, schema, display_template, state, created_at, updated_at
FROM authorization_detail_types
WHERE tenant_id = $1 AND type = $2;

-- name: ListAuthorizationDetailTypesByTenant :many
SELECT tenant_id, type, description, schema, display_template, state, created_at, updated_at
FROM authorization_detail_types
WHERE tenant_id = $1
ORDER BY type;

-- name: UpsertAuthorizationDetailType :exec
INSERT INTO authorization_detail_types (tenant_id, type, description, schema, display_template, state, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (tenant_id, type) DO UPDATE SET
  description = EXCLUDED.description,
  schema = EXCLUDED.schema,
  display_template = EXCLUDED.display_template,
  state = EXCLUDED.state,
  updated_at = EXCLUDED.updated_at;

-- name: DeleteAuthorizationDetailType :exec
DELETE FROM authorization_detail_types WHERE tenant_id = $1 AND type = $2;
