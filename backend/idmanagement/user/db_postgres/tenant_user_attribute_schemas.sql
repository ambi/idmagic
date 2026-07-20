-- name: FindTenantUserAttributeSchemaByTenant :one
SELECT tenant_id,attributes,created_at,updated_at FROM tenant_user_attribute_schemas
WHERE tenant_id=$1;

-- name: SaveTenantUserAttributeSchema :exec
INSERT INTO tenant_user_attribute_schemas (tenant_id,attributes,created_at,updated_at)
VALUES ($1,$2,$3,$4)
ON CONFLICT (tenant_id) DO UPDATE SET attributes=EXCLUDED.attributes,updated_at=EXCLUDED.updated_at;

-- name: DeleteTenantUserAttributeSchema :exec
DELETE FROM tenant_user_attribute_schemas WHERE tenant_id=$1;
