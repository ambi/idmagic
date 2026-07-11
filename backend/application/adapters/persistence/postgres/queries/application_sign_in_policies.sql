-- name: GetAppSignInPolicy :one
SELECT a.tenant_id, p.application_id, p.rules, p.created_at, p.updated_at
FROM application_sign_in_policies p JOIN applications a ON a.application_id = p.application_id
WHERE a.tenant_id = $1 AND p.application_id = $2;

-- name: ListAppSignInPoliciesByTenant :many
SELECT a.tenant_id, p.application_id, p.rules, p.created_at, p.updated_at
FROM application_sign_in_policies p JOIN applications a ON a.application_id = p.application_id
WHERE a.tenant_id = $1;

-- name: UpsertAppSignInPolicy :exec
INSERT INTO application_sign_in_policies (application_id, rules, created_at, updated_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (application_id) DO UPDATE SET
  rules = EXCLUDED.rules,
  updated_at = EXCLUDED.updated_at;

-- name: DeleteAppSignInPolicy :exec
DELETE FROM application_sign_in_policies p WHERE p.application_id = $2
  AND EXISTS (SELECT 1 FROM applications a WHERE a.tenant_id = $1 AND a.application_id = $2);

-- name: GetTenantDefaultSignInPolicy :one
SELECT tenant_id, rules, created_at, updated_at
FROM tenant_default_sign_in_policies
WHERE tenant_id = $1;

-- name: UpsertTenantDefaultSignInPolicy :exec
INSERT INTO tenant_default_sign_in_policies (tenant_id, rules, created_at, updated_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id) DO UPDATE SET
  rules = EXCLUDED.rules,
  updated_at = EXCLUDED.updated_at;
