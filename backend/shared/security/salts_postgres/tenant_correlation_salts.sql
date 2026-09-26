-- name: InsertTenantCorrelationSalt :exec
INSERT INTO tenant_correlation_salts (tenant_id, salt) VALUES ($1, $2) ON CONFLICT (tenant_id) DO NOTHING;

-- name: FindTenantCorrelationSalt :one
SELECT salt FROM tenant_correlation_salts WHERE tenant_id = $1;
