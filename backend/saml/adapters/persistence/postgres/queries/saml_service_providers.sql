-- name: GetSamlServiceProvider :one
SELECT tenant_id, entity_id, display_name, acs_urls, slo_url, audience, claim_policy, sign_assertion, sign_response,
want_authn_requests_signed, authn_request_signing_certificate_pem, created_at, updated_at
FROM saml_service_providers WHERE tenant_id = $1 AND entity_id = $2;

-- name: ListSamlServiceProvidersByTenant :many
SELECT tenant_id, entity_id, display_name, acs_urls, slo_url, audience, claim_policy, sign_assertion, sign_response,
want_authn_requests_signed, authn_request_signing_certificate_pem, created_at, updated_at
FROM saml_service_providers WHERE tenant_id = $1 ORDER BY entity_id;

-- name: UpsertSamlServiceProvider :exec
INSERT INTO saml_service_providers (tenant_id, entity_id, display_name, acs_urls, slo_url, audience, claim_policy, sign_assertion, sign_response,
want_authn_requests_signed, authn_request_signing_certificate_pem, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (tenant_id,entity_id) DO UPDATE SET display_name=EXCLUDED.display_name,
acs_urls=EXCLUDED.acs_urls, slo_url=EXCLUDED.slo_url, audience=EXCLUDED.audience, claim_policy=EXCLUDED.claim_policy,
sign_assertion=EXCLUDED.sign_assertion, sign_response=EXCLUDED.sign_response, want_authn_requests_signed=EXCLUDED.want_authn_requests_signed,
authn_request_signing_certificate_pem=EXCLUDED.authn_request_signing_certificate_pem, updated_at=EXCLUDED.updated_at;

-- name: DeleteSamlServiceProvider :exec
DELETE FROM saml_service_providers WHERE tenant_id = $1 AND entity_id = $2;
