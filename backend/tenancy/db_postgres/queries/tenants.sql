-- name: FindTenantByID :one
SELECT id,realm,display_name,status,created_at,updated_at,disabled_at FROM tenants
WHERE id=$1;

-- name: FindTenantByRealm :one
SELECT id,realm,display_name,status,created_at,updated_at,disabled_at FROM tenants
WHERE realm=$1;

-- name: FindAllTenants :many
SELECT id,realm,display_name,status,created_at,updated_at,disabled_at FROM tenants
ORDER BY id;

-- name: SaveTenant :exec
INSERT INTO tenants (id,realm,display_name,status,created_at,updated_at,disabled_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (id) DO UPDATE SET realm=EXCLUDED.realm,display_name=EXCLUDED.display_name,
status=EXCLUDED.status,updated_at=EXCLUDED.updated_at,disabled_at=EXCLUDED.disabled_at;
