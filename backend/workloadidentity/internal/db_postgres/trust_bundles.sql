-- name: ListWorkloadTrustBundlesByTenant :many
SELECT id,tenant_id,name,profile,trust_domain,issuer,jwks_uri,jwks,accepted_audiences,
max_subject_token_ttl_seconds,status,created_at,updated_at,jwks_cached_at,jwks_cached
FROM workload_trust_bundles WHERE tenant_id=$1 ORDER BY name;

-- name: FindWorkloadTrustBundleByID :one
SELECT id,tenant_id,name,profile,trust_domain,issuer,jwks_uri,jwks,accepted_audiences,
max_subject_token_ttl_seconds,status,created_at,updated_at,jwks_cached_at,jwks_cached
FROM workload_trust_bundles WHERE tenant_id=$1 AND id=$2;

-- name: FindWorkloadTrustBundleByIssuer :one
SELECT id,tenant_id,name,profile,trust_domain,issuer,jwks_uri,jwks,accepted_audiences,
max_subject_token_ttl_seconds,status,created_at,updated_at,jwks_cached_at,jwks_cached
FROM workload_trust_bundles WHERE tenant_id=$1 AND issuer=$2;

-- name: FindSPIFFEWorkloadTrustBundleByTrustDomain :one
SELECT id,tenant_id,name,profile,trust_domain,issuer,jwks_uri,jwks,accepted_audiences,
max_subject_token_ttl_seconds,status,created_at,updated_at,jwks_cached_at,jwks_cached
FROM workload_trust_bundles WHERE tenant_id=$1 AND profile='spiffe' AND trust_domain=$2;

-- 更新では、最後に取得できた鍵 (jwks_cached と jwks_cached_at) を読んだ値で上書きせず、
-- jwks_uri が変わるときだけ消す。並行した取得の記録を古い値へ戻さない。
-- name: SaveWorkloadTrustBundle :exec
INSERT INTO workload_trust_bundles (id,tenant_id,name,profile,trust_domain,issuer,jwks_uri,jwks,
 accepted_audiences,max_subject_token_ttl_seconds,status,created_at,updated_at,jwks_cached_at,jwks_cached)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,jwks_uri=EXCLUDED.jwks_uri,jwks=EXCLUDED.jwks,
 accepted_audiences=EXCLUDED.accepted_audiences,
 max_subject_token_ttl_seconds=EXCLUDED.max_subject_token_ttl_seconds,status=EXCLUDED.status,
 updated_at=EXCLUDED.updated_at,
 jwks_cached_at=CASE WHEN workload_trust_bundles.jwks_uri IS DISTINCT FROM EXCLUDED.jwks_uri
  THEN NULL ELSE workload_trust_bundles.jwks_cached_at END,
 jwks_cached=CASE WHEN workload_trust_bundles.jwks_uri IS DISTINCT FROM EXCLUDED.jwks_uri
  THEN NULL ELSE workload_trust_bundles.jwks_cached END;

-- name: DeleteWorkloadTrustBundle :exec
DELETE FROM workload_trust_bundles WHERE tenant_id=$1 AND id=$2;

-- 取得した jwks_uri が今も行の取得元で、保存済みの取得より新しい場合だけ書く。
-- name: RecordWorkloadTrustBundleJWKSFetched :exec
UPDATE workload_trust_bundles SET jwks_cached_at=sqlc.arg(fetched_at), jwks_cached=sqlc.arg(keys)
WHERE tenant_id=sqlc.arg(tenant_id) AND id=sqlc.arg(id)
 AND jwks_uri IS NOT DISTINCT FROM sqlc.narg(jwks_uri)::text
 AND (jwks_cached_at IS NULL OR jwks_cached_at <= sqlc.arg(fetched_at));
