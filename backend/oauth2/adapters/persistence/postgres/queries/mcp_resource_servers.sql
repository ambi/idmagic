-- name: GetMcpResourceServer :one
SELECT tenant_id, resource_server_id, resource, name, scopes, state, created_at, updated_at
FROM mcp_resource_servers
WHERE tenant_id = $1 AND resource_server_id = $2;

-- name: GetMcpResourceServerByResource :one
SELECT tenant_id, resource_server_id, resource, name, scopes, state, created_at, updated_at
FROM mcp_resource_servers
WHERE tenant_id = $1 AND resource = $2;

-- name: ListMcpResourceServersByTenant :many
SELECT tenant_id, resource_server_id, resource, name, scopes, state, created_at, updated_at
FROM mcp_resource_servers
WHERE tenant_id = $1
ORDER BY resource;

-- name: UpsertMcpResourceServer :exec
INSERT INTO mcp_resource_servers (tenant_id, resource_server_id, resource, name, scopes, state, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (resource_server_id) DO UPDATE SET
  name = EXCLUDED.name,
  scopes = EXCLUDED.scopes,
  state = EXCLUDED.state,
  updated_at = EXCLUDED.updated_at;

-- name: DeleteMcpResourceServer :exec
DELETE FROM mcp_resource_servers WHERE tenant_id = $1 AND resource_server_id = $2;
