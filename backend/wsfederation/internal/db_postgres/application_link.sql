-- name: LinkApplication :exec
UPDATE wsfed_relying_parties
SET application_id = $1
WHERE tenant_id = $2 AND wtrealm = $3 AND application_id IS NULL;
