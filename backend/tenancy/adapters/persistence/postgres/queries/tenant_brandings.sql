-- name: FindTenantBrandingByTenant :one
SELECT tenant_id, product_name, logo_object_key, logo_url, favicon_object_key, favicon_url,
       primary_color, accent_color, support_url, legal_url, footer_text, created_at, updated_at
FROM tenant_brandings
WHERE tenant_id = $1;

-- name: SaveTenantBranding :exec
INSERT INTO tenant_brandings (
    tenant_id, product_name, logo_object_key, logo_url, favicon_object_key, favicon_url,
    primary_color, accent_color, support_url, legal_url, footer_text, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (tenant_id) DO UPDATE SET
    product_name = EXCLUDED.product_name,
    logo_object_key = EXCLUDED.logo_object_key,
    logo_url = EXCLUDED.logo_url,
    favicon_object_key = EXCLUDED.favicon_object_key,
    favicon_url = EXCLUDED.favicon_url,
    primary_color = EXCLUDED.primary_color,
    accent_color = EXCLUDED.accent_color,
    support_url = EXCLUDED.support_url,
    legal_url = EXCLUDED.legal_url,
    footer_text = EXCLUDED.footer_text,
    updated_at = EXCLUDED.updated_at;
