-- name: GetClientByID :one
SELECT tenant_id, client_id, application_id, application_protocol_type, client_secret_hash, client_name, client_type, redirect_uris,
  grant_types, response_types, token_endpoint_auth_method, scope, jwks_uri, jwks,
  tls_client_auth_subject_dn, id_token_signed_response_alg,
  require_pushed_authorization_requests, dpop_bound_access_tokens, fapi_profile,
  created_at, updated_at, first_party, claim_policy
FROM oauth2_clients
WHERE tenant_id = $1 AND client_id = $2;

-- name: ListClientsByTenant :many
SELECT tenant_id, client_id, application_id, application_protocol_type, client_secret_hash, client_name, client_type, redirect_uris,
  grant_types, response_types, token_endpoint_auth_method, scope, jwks_uri, jwks,
  tls_client_auth_subject_dn, id_token_signed_response_alg,
  require_pushed_authorization_requests, dpop_bound_access_tokens, fapi_profile,
  created_at, updated_at, first_party, claim_policy
FROM oauth2_clients
WHERE tenant_id = $1
ORDER BY created_at;

-- name: ListClientsByTenantPage :many
-- First page of ListAdminOAuth2Clients keyset pagination (wi-159, ADR-158):
-- client_id order matches the admin handler's pre-existing re-sort of
-- ListClientsByTenant's rows.
SELECT tenant_id, client_id, application_id, application_protocol_type, client_secret_hash, client_name, client_type, redirect_uris,
  grant_types, response_types, token_endpoint_auth_method, scope, jwks_uri, jwks,
  tls_client_auth_subject_dn, id_token_signed_response_alg,
  require_pushed_authorization_requests, dpop_bound_access_tokens, fapi_profile,
  created_at, updated_at, first_party, claim_policy
FROM oauth2_clients
WHERE tenant_id = $1
ORDER BY client_id
LIMIT sqlc.arg(page_limit);

-- name: ListClientsByTenantPageBefore :many
SELECT tenant_id, client_id, application_id, application_protocol_type, client_secret_hash, client_name, client_type, redirect_uris,
  grant_types, response_types, token_endpoint_auth_method, scope, jwks_uri, jwks,
  tls_client_auth_subject_dn, id_token_signed_response_alg,
  require_pushed_authorization_requests, dpop_bound_access_tokens, fapi_profile,
  created_at, updated_at, first_party, claim_policy
FROM oauth2_clients
WHERE tenant_id = $1
  AND client_id < sqlc.arg(before_client_id)::uuid
ORDER BY client_id DESC
LIMIT sqlc.arg(page_limit);

-- name: ListClientsByTenantPageAfter :many
-- Continuation page: resumes strictly after the client_id of the last row
-- the caller saw.
SELECT tenant_id, client_id, application_id, application_protocol_type, client_secret_hash, client_name, client_type, redirect_uris,
  grant_types, response_types, token_endpoint_auth_method, scope, jwks_uri, jwks,
  tls_client_auth_subject_dn, id_token_signed_response_alg,
  require_pushed_authorization_requests, dpop_bound_access_tokens, fapi_profile,
  created_at, updated_at, first_party, claim_policy
FROM oauth2_clients
WHERE tenant_id = $1
  AND client_id > sqlc.arg(after_client_id)::uuid
ORDER BY client_id
LIMIT sqlc.arg(page_limit);

-- name: UpsertClient :exec
INSERT INTO oauth2_clients (
  tenant_id, client_id, client_secret_hash, client_name, client_type, redirect_uris,
  grant_types, response_types, token_endpoint_auth_method, scope, jwks_uri, jwks,
  tls_client_auth_subject_dn, id_token_signed_response_alg,
  require_pushed_authorization_requests, dpop_bound_access_tokens, fapi_profile,
  created_at, updated_at, first_party, claim_policy
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
ON CONFLICT (client_id) DO UPDATE SET
  client_secret_hash = COALESCE(EXCLUDED.client_secret_hash, oauth2_clients.client_secret_hash),
  client_name = EXCLUDED.client_name,
  client_type = EXCLUDED.client_type,
  redirect_uris = EXCLUDED.redirect_uris,
  grant_types = EXCLUDED.grant_types,
  response_types = EXCLUDED.response_types,
  token_endpoint_auth_method = EXCLUDED.token_endpoint_auth_method,
  scope = EXCLUDED.scope,
  jwks_uri = EXCLUDED.jwks_uri,
  jwks = EXCLUDED.jwks,
  tls_client_auth_subject_dn = EXCLUDED.tls_client_auth_subject_dn,
  id_token_signed_response_alg = EXCLUDED.id_token_signed_response_alg,
  require_pushed_authorization_requests = EXCLUDED.require_pushed_authorization_requests,
  dpop_bound_access_tokens = EXCLUDED.dpop_bound_access_tokens,
  fapi_profile = EXCLUDED.fapi_profile,
  first_party = EXCLUDED.first_party,
  claim_policy = EXCLUDED.claim_policy,
  updated_at = EXCLUDED.updated_at;

-- name: DeleteClient :exec
DELETE FROM oauth2_clients WHERE tenant_id = $1 AND client_id = $2;

-- name: ListClientSecretCredentials :many
SELECT id, client_id, secret_hash, created_at, expires_at, revoked_at
FROM oauth2_client_secrets
WHERE client_id = $1
ORDER BY created_at;

-- name: LockClientForSecretIssuance :one
SELECT client_id
FROM oauth2_clients
WHERE client_id = $1
FOR UPDATE;

-- name: InsertClientSecretCredential :exec
INSERT INTO oauth2_client_secrets (id, client_id, secret_hash, created_at, expires_at, revoked_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateClientSecretCredential :exec
UPDATE oauth2_client_secrets
SET expires_at = $2, revoked_at = $3
WHERE id = $1 AND client_id = $4;
