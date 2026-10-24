-- name: FindDirectoryComputer :one
SELECT * FROM directory_computers WHERE tenant_id = $1 AND connector_id = $2 AND external_id = $3;

-- name: ListEnabledDirectoryComputerHolders :many
SELECT * FROM directory_computers
WHERE tenant_id = sqlc.arg(tenant_id) AND enabled
    AND (object_sid = sqlc.arg(object_sid) OR principal_key = sqlc.arg(principal_key))
ORDER BY connector_id, external_id;

-- name: SaveDirectoryComputer :exec
INSERT INTO directory_computers (
    tenant_id, connector_id, external_id, sam_account_name, object_sid, domain, dns_host_name,
    principal_key, enabled, last_seen_run_id, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (tenant_id, connector_id, external_id) DO UPDATE SET
    sam_account_name = EXCLUDED.sam_account_name,
    object_sid = EXCLUDED.object_sid,
    domain = EXCLUDED.domain,
    dns_host_name = EXCLUDED.dns_host_name,
    principal_key = EXCLUDED.principal_key,
    enabled = EXCLUDED.enabled,
    last_seen_run_id = EXCLUDED.last_seen_run_id,
    updated_at = EXCLUDED.updated_at;

-- name: MarkDirectoryComputerSeen :exec
UPDATE directory_computers SET last_seen_run_id = sqlc.arg(run_id)
WHERE tenant_id = sqlc.arg(tenant_id) AND connector_id = sqlc.arg(connector_id) AND external_id = sqlc.arg(external_id);

-- name: DisableDirectoryComputersNotSeenIn :execrows
UPDATE directory_computers SET enabled = false, updated_at = sqlc.arg(updated_at)
WHERE tenant_id = sqlc.arg(tenant_id) AND connector_id = sqlc.arg(connector_id)
    AND enabled AND last_seen_run_id <> sqlc.arg(run_id);

-- name: FindEnabledDirectoryComputerBySID :one
SELECT * FROM directory_computers WHERE tenant_id = $1 AND object_sid = $2 AND enabled;

-- name: FindEnabledDirectoryComputerByPrincipal :one
SELECT * FROM directory_computers WHERE tenant_id = $1 AND principal_key = $2 AND enabled;
