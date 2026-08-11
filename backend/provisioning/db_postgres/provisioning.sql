-- name: InsertProvisioningConnection :one
INSERT INTO provisioning_connections (
  application_id, tenant_id, status, base_url, credential_id, auth_method, credential_secret,
  credential_created_at, credential_rotated_at, capabilities, feature_flags, scope, group_push,
  attribute_mappings, matching, deprovision_policy, rate_limit_per_minute, max_attempts,
  notification_email, quarantine_after_consecutive_failures, health, consecutive_failure_count,
  last_full_sync_at, quarantined_at, quarantine_reason, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)
ON CONFLICT (application_id) DO NOTHING
RETURNING application_id;

-- name: UpdateProvisioningConnectionWithSecret :exec
UPDATE provisioning_connections SET
  status=$3, base_url=$4, credential_id=$5, auth_method=$6, credential_secret=$7,
  credential_rotated_at=$8, capabilities=$9, feature_flags=$10, scope=$11, group_push=$12,
  attribute_mappings=$13, matching=$14, deprovision_policy=$15, rate_limit_per_minute=$16,
  max_attempts=$17, notification_email=$18, quarantine_after_consecutive_failures=$19, health=$20,
  consecutive_failure_count=$21, last_full_sync_at=$22, quarantined_at=$23, quarantine_reason=$24,
  updated_at=$25
WHERE tenant_id=$1 AND application_id=$2;

-- name: UpdateProvisioningConnection :exec
UPDATE provisioning_connections SET
  status=$3, base_url=$4, capabilities=$5, feature_flags=$6, scope=$7, group_push=$8,
  attribute_mappings=$9, matching=$10, deprovision_policy=$11, rate_limit_per_minute=$12,
  max_attempts=$13, notification_email=$14, quarantine_after_consecutive_failures=$15, health=$16,
  consecutive_failure_count=$17, last_full_sync_at=$18, quarantined_at=$19, quarantine_reason=$20,
  updated_at=$21
WHERE tenant_id=$1 AND application_id=$2;

-- name: FindProvisioningConnection :one
SELECT application_id, tenant_id, status, base_url, credential_id, auth_method,
credential_created_at, credential_rotated_at, capabilities, feature_flags, scope, group_push,
attribute_mappings, matching, deprovision_policy, rate_limit_per_minute, max_attempts,
notification_email, quarantine_after_consecutive_failures, health, consecutive_failure_count,
last_full_sync_at, quarantined_at, quarantine_reason, created_at, updated_at
FROM provisioning_connections WHERE tenant_id=$1 AND application_id=$2;

-- name: GetProvisioningConnectionSecret :one
SELECT credential_secret FROM provisioning_connections WHERE tenant_id=$1 AND application_id=$2;

-- name: DeleteProvisioningConnection :exec
DELETE FROM provisioning_connections WHERE tenant_id=$1 AND application_id=$2;

-- name: ListProvisioningConnectionsByTenant :many
SELECT application_id, tenant_id, status, base_url, credential_id, auth_method,
credential_created_at, credential_rotated_at, capabilities, feature_flags, scope, group_push,
attribute_mappings, matching, deprovision_policy, rate_limit_per_minute, max_attempts,
notification_email, quarantine_after_consecutive_failures, health, consecutive_failure_count,
last_full_sync_at, quarantined_at, quarantine_reason, created_at, updated_at
FROM provisioning_connections WHERE tenant_id=$1 ORDER BY application_id;

-- name: FindRemoteResourceLink :one
SELECT connection_id,tenant_id,source_type,source_id,remote_id,external_id,etag,last_synced_version,updated_at
FROM provisioning_remote_links WHERE connection_id=$1 AND source_type=$2 AND source_id=$3;

-- name: UpsertRemoteResourceLink :exec
INSERT INTO provisioning_remote_links (connection_id, tenant_id, source_type, source_id, remote_id, external_id, etag, last_synced_version, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (connection_id, source_type, source_id) DO UPDATE SET
  remote_id=EXCLUDED.remote_id, external_id=EXCLUDED.external_id, etag=EXCLUDED.etag,
  last_synced_version=EXCLUDED.last_synced_version, updated_at=EXCLUDED.updated_at;

-- name: InsertProvisioningDelivery :one
INSERT INTO provisioning_deliveries (id, tenant_id, connection_id, source_type, source_id, source_version, operation, status, job_id, last_error, created_at, updated_at, completed_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (tenant_id, connection_id, source_type, source_id, source_version) DO NOTHING
RETURNING id;

-- name: FindProvisioningDelivery :one
SELECT id, tenant_id, connection_id, source_type, source_id, source_version, operation, status, job_id, last_error, created_at, updated_at, completed_at
FROM provisioning_deliveries WHERE tenant_id=$1 AND id=$2;

-- name: ListProvisioningDeliveriesByConnectionAndStatus :many
SELECT id, tenant_id, connection_id, source_type, source_id, source_version, operation, status, job_id, last_error, created_at, updated_at, completed_at
FROM provisioning_deliveries WHERE tenant_id=$1 AND connection_id=$2 AND status=$3 ORDER BY created_at DESC LIMIT $4;

-- name: ListProvisioningDeliveriesByConnection :many
SELECT id, tenant_id, connection_id, source_type, source_id, source_version, operation, status, job_id, last_error, created_at, updated_at, completed_at
FROM provisioning_deliveries WHERE tenant_id=$1 AND connection_id=$2 ORDER BY created_at DESC LIMIT $3;

-- name: ListProvisioningDeliveriesByConnectionPage :many
-- First page of ListProvisioningDeliveries keyset pagination (wi-159).
-- Empty status/source_type arguments disable that filter.
SELECT id, tenant_id, connection_id, source_type, source_id, source_version, operation, status, job_id, last_error, created_at, updated_at, completed_at
FROM provisioning_deliveries
WHERE tenant_id=sqlc.arg(tenant_id) AND connection_id=sqlc.arg(connection_id)
  AND (sqlc.arg(filter_status)::text = '' OR status=sqlc.arg(filter_status)::text)
  AND (sqlc.arg(filter_source_type)::text = '' OR source_type=sqlc.arg(filter_source_type)::text)
ORDER BY created_at DESC, id DESC LIMIT sqlc.arg(page_limit);

-- name: ListProvisioningDeliveriesByConnectionPageAfter :many
-- Continuation page: resumes strictly after the (created_at, id) keyset of
-- the last row the caller saw.
SELECT id, tenant_id, connection_id, source_type, source_id, source_version, operation, status, job_id, last_error, created_at, updated_at, completed_at
FROM provisioning_deliveries WHERE tenant_id=$1 AND connection_id=$2
  AND (sqlc.arg(filter_status)::text = '' OR status=sqlc.arg(filter_status)::text)
  AND (sqlc.arg(filter_source_type)::text = '' OR source_type=sqlc.arg(filter_source_type)::text)
  AND (created_at, id) < (sqlc.arg(after_created_at)::timestamptz, sqlc.arg(after_id)::uuid)
ORDER BY created_at DESC, id DESC LIMIT sqlc.arg(page_limit);

-- name: ListProvisioningDeliveriesByConnectionPageBefore :many
SELECT id, tenant_id, connection_id, source_type, source_id, source_version, operation, status, job_id, last_error, created_at, updated_at, completed_at
FROM provisioning_deliveries WHERE tenant_id=$1 AND connection_id=$2
  AND (sqlc.arg(filter_status)::text = '' OR status=sqlc.arg(filter_status)::text)
  AND (sqlc.arg(filter_source_type)::text = '' OR source_type=sqlc.arg(filter_source_type)::text)
  AND (created_at, id) > (sqlc.arg(before_created_at)::timestamptz, sqlc.arg(before_id)::uuid)
ORDER BY created_at ASC, id ASC LIMIT sqlc.arg(page_limit);

-- name: ListUnenqueuedProvisioningDeliveries :many
SELECT id, tenant_id, connection_id, source_type, source_id, source_version, operation, status, job_id, last_error, created_at, updated_at, completed_at
FROM provisioning_deliveries WHERE status='pending' AND job_id IS NULL ORDER BY created_at LIMIT $1;

-- name: AttachProvisioningDeliveryJob :execrows
UPDATE provisioning_deliveries SET job_id=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2 AND job_id IS NULL;

-- name: UpdateProvisioningDeliveryStatus :exec
UPDATE provisioning_deliveries SET status=$3, last_error=$4, updated_at=now(),
  completed_at = CASE WHEN $3 IN ('succeeded','dead_letter') THEN now() ELSE completed_at END
WHERE tenant_id=$1 AND id=$2;

-- name: RetryDeadLetterProvisioningDelivery :execrows
UPDATE provisioning_deliveries SET status='pending', job_id=NULL, last_error=NULL, updated_at=now()
WHERE tenant_id=$1 AND id=$2 AND status='dead_letter';
