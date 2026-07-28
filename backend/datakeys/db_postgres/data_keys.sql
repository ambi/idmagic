-- name: LockDataKeyRotation :exec
SELECT pg_advisory_xact_lock(hashtext('idmagic-datakey:'||$1));

-- name: CountNonDestroyedDataKeys :one
SELECT count(*) FROM tenant_data_encryption_keys
WHERE tenant_id=$1 AND status!='destroyed';

-- name: MaxDataKeyVersion :one
SELECT COALESCE(MAX(version),0)::int FROM tenant_data_encryption_keys
WHERE tenant_id=$1;

-- name: GetActiveDataKey :one
SELECT id,tenant_id,version,status,wrapped_dek,master_key_id,created_at,activated_at,disabled_at,destroyed_at
FROM tenant_data_encryption_keys
WHERE tenant_id=$1 AND status='active' LIMIT 1;

-- name: GetDataKeyByVersion :one
SELECT id,tenant_id,version,status,wrapped_dek,master_key_id,created_at,activated_at,disabled_at,destroyed_at
FROM tenant_data_encryption_keys
WHERE tenant_id=$1 AND version=$2;

-- name: ListDataKeysByTenant :many
SELECT id,tenant_id,version,status,wrapped_dek,master_key_id,created_at,activated_at,disabled_at,destroyed_at
FROM tenant_data_encryption_keys
WHERE tenant_id=$1 ORDER BY version DESC;

-- name: RetireActiveDataKey :exec
UPDATE tenant_data_encryption_keys
SET status='retiring', updated_at=$2
WHERE tenant_id=$1 AND status='active';

-- name: InsertDataKey :exec
INSERT INTO tenant_data_encryption_keys (
  id, tenant_id, version, status, wrapped_dek, master_key_id, created_at, updated_at, activated_at
) VALUES (
  $1, $2, $3, 'active', $4, $5, $6, $6, $6
);

-- name: DisableDataKey :exec
UPDATE tenant_data_encryption_keys
SET status='disabled', disabled_at=$3, updated_at=$3
WHERE tenant_id=$1 AND version=$2;

-- name: DestroyDataKey :exec
UPDATE tenant_data_encryption_keys
SET status='destroyed', wrapped_dek=NULL, destroyed_at=$3, updated_at=$3
WHERE tenant_id=$1 AND version=$2;
