-- name: FindTenantUserAttributeSchemaByTenant :one
SELECT tenant_id,attributes,created_at,updated_at FROM tenant_user_attribute_schemas
WHERE tenant_id=$1;

-- name: SaveTenantUserAttributeSchema :exec
INSERT INTO tenant_user_attribute_schemas (tenant_id,attributes,created_at,updated_at)
VALUES ($1,$2,$3,$4)
ON CONFLICT (tenant_id) DO UPDATE SET attributes=EXCLUDED.attributes,updated_at=EXCLUDED.updated_at;

-- name: DeleteTenantUserAttributeSchema :exec
DELETE FROM tenant_user_attribute_schemas WHERE tenant_id=$1;

-- name: LockUniqueAttributeValuesExclusive :exec
-- unique の属性の集合を変える保存は排他の固定を取り、交差する User の保存を待たせる。
SELECT pg_advisory_xact_lock(hashtextextended('user_unique_attribute_values:' || sqlc.arg(tenant_id)::text, 0));

-- name: DeleteTenantUniqueAttributeValues :exec
DELETE FROM user_unique_attribute_values WHERE tenant_id = $1;

-- name: CopyUniqueAttributeValues :exec
-- 削除されていない User の、属性 attribute_key の空でない文字列の値を写す。重なりは主キーが拒否する。
INSERT INTO user_unique_attribute_values (tenant_id, attribute_key, value, user_id)
SELECT u.tenant_id, sqlc.arg(attribute_key)::text, u.attributes -> sqlc.arg(attribute_key)::text ->> 'string', u.id
FROM users u
WHERE u.tenant_id = sqlc.arg(tenant_id)
  AND u.lifecycle->>'status' IS DISTINCT FROM 'deleted'
  AND u.attributes -> sqlc.arg(attribute_key)::text ->> 'type' = 'string'
  AND coalesce(u.attributes -> sqlc.arg(attribute_key)::text ->> 'string', '') <> '';
