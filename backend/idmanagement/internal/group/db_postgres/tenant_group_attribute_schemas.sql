-- name: FindTenantGroupAttributeSchemaByTenant :one
SELECT tenant_id,attributes,created_at,updated_at FROM tenant_group_attribute_schemas
WHERE tenant_id=$1;

-- name: SaveTenantGroupAttributeSchema :exec
INSERT INTO tenant_group_attribute_schemas (tenant_id,attributes,created_at,updated_at)
VALUES ($1,$2,$3,$4)
ON CONFLICT (tenant_id) DO UPDATE SET attributes=EXCLUDED.attributes,updated_at=EXCLUDED.updated_at;

-- name: DeleteTenantGroupAttributeSchema :exec
DELETE FROM tenant_group_attribute_schemas WHERE tenant_id=$1;
