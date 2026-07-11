-- name: UpsertTenantBrandingAsset :exec
INSERT INTO tenant_branding_assets (tenant_id, kind, object_key, content_type, size_bytes, data, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (tenant_id, kind, object_key) DO UPDATE SET
    content_type = EXCLUDED.content_type,
    size_bytes = EXCLUDED.size_bytes,
    data = EXCLUDED.data,
    updated_at = EXCLUDED.updated_at;

-- name: GetTenantBrandingAsset :one
SELECT tenant_id, kind, object_key, content_type, size_bytes, data, created_at, updated_at
FROM tenant_branding_assets
WHERE tenant_id = $1 AND kind = $2 AND object_key = $3;

-- name: DeleteTenantBrandingAssetsByKind :exec
DELETE FROM tenant_branding_assets WHERE tenant_id = $1 AND kind = $2;
