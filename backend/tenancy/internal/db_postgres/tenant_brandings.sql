-- name: FindTenantBrandingByTenant :one
SELECT tenant_id, product_name, logo_object_key, logo_url, favicon_object_key, favicon_url,
       primary_color, accent_color, footer_link_1_label, footer_link_1_url,
       footer_link_2_label, footer_link_2_url, footer_text, created_at, updated_at
FROM tenant_brandings
WHERE tenant_id = $1;

-- name: SaveTenantBranding :exec
INSERT INTO tenant_brandings (
    tenant_id, product_name, logo_object_key, logo_url, favicon_object_key, favicon_url,
    primary_color, accent_color, footer_link_1_label, footer_link_1_url,
    footer_link_2_label, footer_link_2_url, footer_text, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
ON CONFLICT (tenant_id) DO UPDATE SET
    product_name = EXCLUDED.product_name,
    logo_object_key = EXCLUDED.logo_object_key,
    logo_url = EXCLUDED.logo_url,
    favicon_object_key = EXCLUDED.favicon_object_key,
    favicon_url = EXCLUDED.favicon_url,
    primary_color = EXCLUDED.primary_color,
    accent_color = EXCLUDED.accent_color,
    footer_link_1_label = EXCLUDED.footer_link_1_label,
    footer_link_1_url = EXCLUDED.footer_link_1_url,
    footer_link_2_label = EXCLUDED.footer_link_2_label,
    footer_link_2_url = EXCLUDED.footer_link_2_url,
    footer_text = EXCLUDED.footer_text,
    updated_at = EXCLUDED.updated_at;
