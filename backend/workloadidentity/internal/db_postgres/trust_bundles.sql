-- name: ListWorkloadTrustBundlesByTenant :many
SELECT id,tenant_id,name,trust_domain,issuer,jwks_uri,jwks,accepted_audiences,
max_subject_token_ttl_seconds,status,created_at,updated_at,jwks_cached_at
FROM workload_trust_bundles WHERE tenant_id=$1 ORDER BY name;

-- name: FindWorkloadTrustBundleByID :one
SELECT id,tenant_id,name,trust_domain,issuer,jwks_uri,jwks,accepted_audiences,
max_subject_token_ttl_seconds,status,created_at,updated_at,jwks_cached_at
FROM workload_trust_bundles WHERE tenant_id=$1 AND id=$2;

-- name: FindWorkloadTrustBundleByIssuer :one
SELECT id,tenant_id,name,trust_domain,issuer,jwks_uri,jwks,accepted_audiences,
max_subject_token_ttl_seconds,status,created_at,updated_at,jwks_cached_at
FROM workload_trust_bundles WHERE tenant_id=$1 AND issuer=$2;

-- name: SaveWorkloadTrustBundle :exec
INSERT INTO workload_trust_bundles (id,tenant_id,name,trust_domain,issuer,jwks_uri,jwks,
 accepted_audiences,max_subject_token_ttl_seconds,status,created_at,updated_at,jwks_cached_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,jwks_uri=EXCLUDED.jwks_uri,jwks=EXCLUDED.jwks,
 accepted_audiences=EXCLUDED.accepted_audiences,
 max_subject_token_ttl_seconds=EXCLUDED.max_subject_token_ttl_seconds,status=EXCLUDED.status,
 updated_at=EXCLUDED.updated_at,jwks_cached_at=EXCLUDED.jwks_cached_at;

-- name: DeleteWorkloadTrustBundle :exec
DELETE FROM workload_trust_bundles WHERE tenant_id=$1 AND id=$2;
