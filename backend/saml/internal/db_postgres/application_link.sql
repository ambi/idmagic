-- name: LinkApplication :exec
UPDATE saml_service_providers
SET application_id = $1
WHERE tenant_id = $2 AND entity_id = $3 AND application_id IS NULL;
