-- name: InsertConnector :execrows
INSERT INTO token_vault_connectors (
  id, tenant_id, name, status, authorization_endpoint, token_endpoint, revocation_endpoint, client_id,
  token_endpoint_auth_method, secret_key_version, secret_ciphertext, allowed_scopes, rules, version, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
ON CONFLICT (tenant_id, name) DO NOTHING;

-- name: UpdateConnector :execrows
UPDATE token_vault_connectors SET
  name=$3, status=$4, authorization_endpoint=$5, token_endpoint=$6, revocation_endpoint=$7, client_id=$8,
  token_endpoint_auth_method=$9, secret_key_version=$10, secret_ciphertext=$11, allowed_scopes=$12, rules=$13,
  updated_at=$14, version=version+1
WHERE tenant_id=$1 AND id=$2 AND version=$15;

-- name: FindConnector :one
SELECT * FROM token_vault_connectors WHERE tenant_id=$1 AND id=$2;

-- name: ListConnectors :many
SELECT * FROM token_vault_connectors WHERE tenant_id=$1 ORDER BY name;

-- name: DeleteConnector :execrows
DELETE FROM token_vault_connectors WHERE tenant_id=$1 AND id=$2;

-- name: FindConnection :one
SELECT * FROM token_vault_connections WHERE tenant_id=$1 AND connector_id=$2 AND user_id=$3;

-- name: ListConnectionsByUser :many
SELECT * FROM token_vault_connections WHERE tenant_id=$1 AND user_id=$2;

-- name: SaveLinkedConnection :exec
INSERT INTO token_vault_connections (
  id, tenant_id, connector_id, user_id, status, granted_scopes, token_key_version,
  access_token_ciphertext, refresh_token_ciphertext, expires_at, token_version, created_at, updated_at
) VALUES ($1,$2,$3,$4,'active',$5,$6,$7,$8,$9,1,$10,$10)
ON CONFLICT (tenant_id, connector_id, user_id) DO UPDATE SET
  status='active', granted_scopes=EXCLUDED.granted_scopes, token_key_version=EXCLUDED.token_key_version,
  access_token_ciphertext=EXCLUDED.access_token_ciphertext, refresh_token_ciphertext=EXCLUDED.refresh_token_ciphertext,
  expires_at=EXCLUDED.expires_at, token_version=token_vault_connections.token_version+1,
  refresh_lease_until=NULL, updated_at=EXCLUDED.updated_at;

-- name: AcquireRefreshLease :execrows
UPDATE token_vault_connections SET refresh_lease_until=$5
WHERE tenant_id=$1 AND id=$2 AND token_version=$3 AND (refresh_lease_until IS NULL OR refresh_lease_until <= $4);

-- name: ReleaseRefreshLease :exec
UPDATE token_vault_connections SET refresh_lease_until=NULL
WHERE tenant_id=$1 AND id=$2 AND token_version=$3;

-- name: StoreRefreshedConnection :execrows
UPDATE token_vault_connections SET
  granted_scopes=$4, token_key_version=$5, access_token_ciphertext=$6, refresh_token_ciphertext=$7, expires_at=$8,
  updated_at=$9, token_version=token_version+1, refresh_lease_until=NULL
WHERE tenant_id=$1 AND id=$2 AND token_version=$3;

-- name: MarkConnectionReauthorizationRequired :execrows
UPDATE token_vault_connections SET
  status='reauthorization_required', token_key_version=NULL, access_token_ciphertext=NULL, refresh_token_ciphertext=NULL,
  expires_at=NULL, refresh_lease_until=NULL, updated_at=$4, token_version=token_version+1
WHERE tenant_id=$1 AND id=$2 AND token_version=$3;

-- name: TouchConnectionLastUsed :exec
UPDATE token_vault_connections SET last_used_at=$3 WHERE tenant_id=$1 AND id=$2;

-- name: DeleteConnection :one
DELETE FROM token_vault_connections WHERE tenant_id=$1 AND connector_id=$2 AND user_id=$3 RETURNING *;

-- name: DeleteExpiredAttempts :exec
DELETE FROM token_vault_attempts WHERE tenant_id=$1 AND expires_at <= $2;

-- name: SaveAttempt :exec
INSERT INTO token_vault_attempts (
  id, tenant_id, connector_id, user_id, state_hash, verifier_key_version, verifier_ciphertext, requested_scopes, expires_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (tenant_id, connector_id, user_id) DO UPDATE SET
  id=EXCLUDED.id, state_hash=EXCLUDED.state_hash, verifier_key_version=EXCLUDED.verifier_key_version,
  verifier_ciphertext=EXCLUDED.verifier_ciphertext, requested_scopes=EXCLUDED.requested_scopes, expires_at=EXCLUDED.expires_at;

-- name: ConsumeAttempt :one
DELETE FROM token_vault_attempts
WHERE tenant_id=$1 AND connector_id=$2 AND user_id=$3 AND state_hash=$4 AND expires_at > $5
RETURNING *;

-- name: ListConnectorsPendingReencryption :many
SELECT * FROM token_vault_connectors WHERE tenant_id=$1 AND secret_key_version <> $2 ORDER BY id LIMIT $3;

-- name: CountConnectorsPendingReencryption :one
SELECT count(*) FROM token_vault_connectors WHERE tenant_id=$1 AND secret_key_version <> $2;

-- name: ReencryptConnectorSecret :execrows
UPDATE token_vault_connectors SET secret_key_version=$3, secret_ciphertext=$4
WHERE tenant_id=$1 AND id=$2 AND version=$5 AND secret_key_version=$6;

-- name: ListConnectionsPendingReencryption :many
SELECT * FROM token_vault_connections
WHERE tenant_id=$1 AND token_key_version IS NOT NULL AND token_key_version <> $2 ORDER BY id LIMIT $3;

-- name: CountConnectionsPendingReencryption :one
SELECT count(*) FROM token_vault_connections
WHERE tenant_id=$1 AND token_key_version IS NOT NULL AND token_key_version <> $2;

-- name: ReencryptConnectionTokens :execrows
UPDATE token_vault_connections SET token_key_version=$3, access_token_ciphertext=$4, refresh_token_ciphertext=$5
WHERE tenant_id=$1 AND id=$2 AND token_version=$6 AND token_key_version=$7;
