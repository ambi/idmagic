-- name: GetActiveKey :one
SELECT kid,tenant_id,alg,provider,key_usage,scope_id,public_jwk,private_jwk,certificate_der,active,created_at,retired_at,expires_at,archived_at
FROM signing_keys
WHERE active=TRUE AND tenant_id=$1 AND key_usage=$2 AND scope_id=$3 LIMIT 1;

-- name: GetAllKeys :many
SELECT kid,tenant_id,alg,provider,key_usage,scope_id,public_jwk,private_jwk,certificate_der,active,created_at,retired_at,expires_at,archived_at
FROM signing_keys
WHERE archived_at IS NULL AND tenant_id=$1 AND key_usage=$2 AND scope_id=$3 ORDER BY created_at DESC;

-- name: ListPublicKeys :many
SELECT kid,tenant_id,alg,provider,key_usage,scope_id,public_jwk,private_jwk,certificate_der,active,created_at,retired_at,expires_at,archived_at
FROM signing_keys
WHERE archived_at IS NULL AND tenant_id=$1 AND key_usage=$2 AND scope_id=$3 AND (active=TRUE OR expires_at>$4) ORDER BY created_at DESC;

-- name: FindKeyByKID :one
SELECT kid,tenant_id,alg,provider,key_usage,scope_id,public_jwk,private_jwk,certificate_der,active,created_at,retired_at,expires_at,archived_at
FROM signing_keys
WHERE kid=$1 AND tenant_id=$2 AND key_usage=$3 AND scope_id=$4;

-- name: DisableKey :exec
UPDATE signing_keys
SET active=FALSE,archived_at=now(),updated_at=now()
WHERE kid=$1 AND tenant_id=$2;

-- name: ArchiveExpiredKeys :many
UPDATE signing_keys
SET archived_at=$2,updated_at=$2
WHERE archived_at IS NULL AND tenant_id=$1 AND expires_at IS NOT NULL AND expires_at<=$2 AND key_usage=$3 AND scope_id=$4
RETURNING kid,tenant_id,alg,provider,key_usage,scope_id,public_jwk,private_jwk,certificate_der,active,created_at,retired_at,expires_at,archived_at;

-- name: LockSigningKeyRotation :exec
SELECT pg_advisory_xact_lock(hashtext('idmagic-signing-key:'||$1||':'||$2||':'||$3));

-- name: GetActiveKeyCreatedAtForUpdate :one
SELECT created_at
FROM signing_keys
WHERE active=TRUE AND tenant_id=$1 AND key_usage=$2 AND scope_id=$3 LIMIT 1 FOR UPDATE;

-- name: RetireActiveKey :exec
UPDATE signing_keys
SET active=FALSE,retired_at=$4,expires_at=$5,updated_at=$4
WHERE active=TRUE AND tenant_id=$1 AND key_usage=$2 AND scope_id=$3;

-- name: InsertSigningKey :exec
INSERT INTO signing_keys (
  kid, tenant_id, alg, provider, key_usage, scope_id, public_jwk, private_jwk, certificate_der, active
) VALUES (
  $1, $2, 'PS256', 'Database', $3, $4, $5, $6, $7, TRUE
);
