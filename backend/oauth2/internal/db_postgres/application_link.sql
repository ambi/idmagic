-- name: LinkApplication :exec
UPDATE oauth2_clients
SET application_id = $1
WHERE tenant_id = $2 AND client_id::text = $3 AND application_id IS NULL;
