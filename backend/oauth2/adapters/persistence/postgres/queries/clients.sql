-- name: GetClientByID :one
SELECT tenant_id, client_id, client_secret_hash, client_name, client_type, redirect_uris,
  grant_types, response_types, token_endpoint_auth_method, scope, jwks_uri, jwks,
  tls_client_auth_subject_dn, id_token_signed_response_alg,
  require_pushed_authorization_requests, dpop_bound_access_tokens, fapi_profile,
  created_at, updated_at, first_party
FROM clients
WHERE tenant_id = $1 AND client_id = $2;

-- name: ListClientsByTenant :many
SELECT tenant_id, client_id, client_secret_hash, client_name, client_type, redirect_uris,
  grant_types, response_types, token_endpoint_auth_method, scope, jwks_uri, jwks,
  tls_client_auth_subject_dn, id_token_signed_response_alg,
  require_pushed_authorization_requests, dpop_bound_access_tokens, fapi_profile,
  created_at, updated_at, first_party
FROM clients
WHERE tenant_id = $1
ORDER BY created_at;

-- name: UpsertClient :exec
INSERT INTO clients (
  tenant_id, client_id, client_secret_hash, client_name, client_type, redirect_uris,
  grant_types, response_types, token_endpoint_auth_method, scope, jwks_uri, jwks,
  tls_client_auth_subject_dn, id_token_signed_response_alg,
  require_pushed_authorization_requests, dpop_bound_access_tokens, fapi_profile,
  created_at, updated_at, first_party
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
ON CONFLICT (client_id) DO UPDATE SET
  client_secret_hash = COALESCE(EXCLUDED.client_secret_hash, clients.client_secret_hash),
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
  updated_at = EXCLUDED.updated_at;

-- name: DeleteClient :exec
DELETE FROM clients WHERE tenant_id = $1 AND client_id = $2;
