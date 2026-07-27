-- name: SaveIdentityProviderConnection :exec
INSERT INTO identity_provider_connections (
  tenant_id, provider_id, display_name, protocol, status, issuer, client_id, secret_reference,
  authorization_endpoint, token_endpoint, jwks_uri, saml_sso_url, saml_entity_id,
  saml_signing_certificates, claim_mapping, linking_policy, jit_provisioning,
  allowed_email_domains, metadata_refreshed_at, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7::text,''),NULLIF($8::text,''),NULLIF($9::text,''),NULLIF($10::text,''),
  NULLIF($11::text,''),NULLIF($12::text,''),NULLIF($13::text,''),$14,$15,$16,$17,$18,$19,$20,$21)
ON CONFLICT (tenant_id, provider_id) DO UPDATE SET
  display_name=EXCLUDED.display_name, protocol=EXCLUDED.protocol, status=EXCLUDED.status,
  issuer=EXCLUDED.issuer, client_id=EXCLUDED.client_id,
  secret_reference=COALESCE(EXCLUDED.secret_reference, identity_provider_connections.secret_reference),
  authorization_endpoint=EXCLUDED.authorization_endpoint, token_endpoint=EXCLUDED.token_endpoint,
  jwks_uri=EXCLUDED.jwks_uri, saml_sso_url=EXCLUDED.saml_sso_url,
  saml_entity_id=EXCLUDED.saml_entity_id,
  saml_signing_certificates=EXCLUDED.saml_signing_certificates,
  claim_mapping=EXCLUDED.claim_mapping, linking_policy=EXCLUDED.linking_policy,
  jit_provisioning=EXCLUDED.jit_provisioning, allowed_email_domains=EXCLUDED.allowed_email_domains,
  metadata_refreshed_at=EXCLUDED.metadata_refreshed_at, updated_at=EXCLUDED.updated_at;

-- name: FindIdentityProviderConnection :one
SELECT tenant_id, provider_id, display_name, protocol, status, issuer,
COALESCE(client_id,'') AS client_id, COALESCE(secret_reference,'') AS secret_reference, COALESCE(authorization_endpoint,'') AS authorization_endpoint,
COALESCE(token_endpoint,'') AS token_endpoint, COALESCE(jwks_uri,'') AS jwks_uri, COALESCE(saml_sso_url,'') AS saml_sso_url,
COALESCE(saml_entity_id,'') AS saml_entity_id, saml_signing_certificates, claim_mapping, linking_policy,
jit_provisioning, allowed_email_domains, metadata_refreshed_at, created_at, updated_at
FROM identity_provider_connections WHERE tenant_id=$1 AND provider_id=$2;

-- name: ListIdentityProviderConnections :many
SELECT tenant_id, provider_id, display_name, protocol, status, issuer,
COALESCE(client_id,'') AS client_id, COALESCE(secret_reference,'') AS secret_reference, COALESCE(authorization_endpoint,'') AS authorization_endpoint,
COALESCE(token_endpoint,'') AS token_endpoint, COALESCE(jwks_uri,'') AS jwks_uri, COALESCE(saml_sso_url,'') AS saml_sso_url,
COALESCE(saml_entity_id,'') AS saml_entity_id, saml_signing_certificates, claim_mapping, linking_policy,
jit_provisioning, allowed_email_domains, metadata_refreshed_at, created_at, updated_at
FROM identity_provider_connections WHERE tenant_id=$1 ORDER BY display_name, provider_id;

-- name: DeleteIdentityProviderConnection :exec
DELETE FROM identity_provider_connections WHERE tenant_id=$1 AND provider_id=$2;

-- name: CreateFederatedIdentity :exec
INSERT INTO federated_identities
  (tenant_id,provider_id,external_subject,local_user_id,linked_at,last_login_at)
VALUES ($1,$2,$3,$4,$5,$6);

-- name: FindFederatedIdentityBySubject :one
SELECT tenant_id,provider_id,external_subject,local_user_id,linked_at,last_login_at
FROM federated_identities WHERE tenant_id=$1 AND provider_id=$2 AND external_subject=$3;

-- name: FindFederatedIdentityByUserProvider :one
SELECT tenant_id,provider_id,external_subject,local_user_id,linked_at,last_login_at
FROM federated_identities WHERE tenant_id=$1 AND provider_id=$2 AND local_user_id=$3;

-- name: ListFederatedIdentitiesByUser :many
SELECT tenant_id,provider_id,external_subject,local_user_id,linked_at,last_login_at
FROM federated_identities WHERE tenant_id=$1 AND local_user_id=$2 ORDER BY provider_id;

-- name: DeleteFederatedIdentity :exec
DELETE FROM federated_identities
WHERE tenant_id=$1 AND provider_id=$2 AND local_user_id=$3;

-- name: SaveFederatedLoginAttempt :exec
INSERT INTO federated_login_attempts
  (tenant_id,state,provider_id,protocol,nonce,pkce_verifier,request_id,return_to,link_user_id,
   created_at,expires_at,consumed_at)
VALUES ($1,$2,$3,$4,NULLIF($5::text,''),NULLIF($6::text,''),NULLIF($7::text,''),NULLIF($8::text,''),NULLIF($9::text,'')::uuid,
  $10,$11,$12);

-- name: ConsumeFederatedLoginAttempt :one
UPDATE federated_login_attempts SET consumed_at=$3
WHERE tenant_id=$1 AND state=$2 AND consumed_at IS NULL AND expires_at>$3
RETURNING state,tenant_id,provider_id,protocol,COALESCE(nonce,'') AS nonce,COALESCE(pkce_verifier,'') AS pkce_verifier,
  COALESCE(request_id,'') AS request_id,COALESCE(return_to,'') AS return_to,COALESCE(link_user_id::text,'') AS link_user_id,
  created_at,expires_at,consumed_at;

-- name: GetFederatedLoginAttemptConsumedAt :one
SELECT consumed_at FROM federated_login_attempts
WHERE tenant_id=$1 AND state=$2;

-- name: DeleteExpiredReplays :exec
DELETE FROM federated_response_replays
WHERE tenant_id=$1 AND response_id=$2 AND expires_at<=now();

-- name: ReserveReplay :one
INSERT INTO federated_response_replays
  (tenant_id,response_id,expires_at) VALUES ($1,$2,$3)
ON CONFLICT DO NOTHING RETURNING response_id;
