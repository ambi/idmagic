-- name: ListApplicationsByTenant :many
SELECT tenant_id, application_id, name, kind, status, protocol_type, icon_url, icon_object_key, launch_url, category_ids, created_at, updated_at
FROM applications
WHERE tenant_id = $1
ORDER BY name;

-- name: GetApplicationByID :one
SELECT tenant_id, application_id, name, kind, status, protocol_type, icon_url, icon_object_key, launch_url, category_ids, created_at, updated_at
FROM applications
WHERE tenant_id = $1 AND application_id = $2;

-- name: UpsertApplication :exec
INSERT INTO applications (tenant_id, application_id, name, kind, status, protocol_type, icon_url, icon_object_key, launch_url, category_ids, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (application_id) DO UPDATE SET
  name = EXCLUDED.name,
  status = EXCLUDED.status,
  icon_url = EXCLUDED.icon_url,
  icon_object_key = EXCLUDED.icon_object_key,
  launch_url = EXCLUDED.launch_url,
  category_ids = EXCLUDED.category_ids,
  updated_at = EXCLUDED.updated_at;

-- name: GetApplicationProtocolKey :one
SELECT COALESCE(c.client_id::text, s.entity_id, w.wtrealm, '')::text AS protocol_key
FROM applications a
LEFT JOIN oauth2_clients c ON c.application_id = a.application_id
LEFT JOIN saml_service_providers s ON s.application_id = a.application_id
LEFT JOIN wsfed_relying_parties w ON w.application_id = a.application_id
WHERE a.tenant_id = $1 AND a.application_id = $2;

-- name: FindApplicationByProtocol :one
SELECT a.tenant_id, a.application_id, a.name, a.kind, a.status, a.protocol_type,
       a.icon_url, a.icon_object_key, a.launch_url, a.category_ids, a.created_at, a.updated_at
FROM applications a
LEFT JOIN oauth2_clients c ON c.application_id = a.application_id
LEFT JOIN saml_service_providers s ON s.application_id = a.application_id
LEFT JOIN wsfed_relying_parties w ON w.application_id = a.application_id
WHERE a.tenant_id = $1
  AND a.protocol_type = $2
  AND (
    ($2 = 'oidc' AND c.client_id::text = $3)
    OR ($2 = 'saml' AND s.entity_id = $3)
    OR ($2 = 'wsfed' AND w.wtrealm = $3)
  );

-- name: LinkOAuth2ClientToApplication :exec
UPDATE oauth2_clients
SET application_id = $1
WHERE tenant_id = $2 AND client_id::text = $3 AND application_id IS NULL;

-- name: LinkSamlServiceProviderToApplication :exec
UPDATE saml_service_providers
SET application_id = $1
WHERE tenant_id = $2 AND entity_id = $3 AND application_id IS NULL;

-- name: LinkWsFedRelyingPartyToApplication :exec
UPDATE wsfed_relying_parties
SET application_id = $1
WHERE tenant_id = $2 AND wtrealm = $3 AND application_id IS NULL;

-- name: DeleteApplication :exec
DELETE FROM applications WHERE tenant_id = $1 AND application_id = $2;

-- name: RemoveApplicationCategory :exec
UPDATE applications SET category_ids = array_remove(category_ids, $2), updated_at = now()
WHERE tenant_id = $1 AND $2 = ANY(category_ids);
