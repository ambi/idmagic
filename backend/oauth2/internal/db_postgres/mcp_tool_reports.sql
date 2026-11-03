-- name: GetMcpToolReport :one
SELECT tenant_id, resource_server_id, tools, complete, reported_at
FROM mcp_tool_reports
WHERE tenant_id = $1 AND resource_server_id = $2;

-- name: UpsertMcpToolReport :exec
INSERT INTO mcp_tool_reports (tenant_id, resource_server_id, tools, complete, reported_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (resource_server_id) DO UPDATE SET
  tools = EXCLUDED.tools,
  complete = EXCLUDED.complete,
  reported_at = EXCLUDED.reported_at
WHERE mcp_tool_reports.tenant_id = EXCLUDED.tenant_id;

-- name: DeleteMcpToolReport :exec
DELETE FROM mcp_tool_reports WHERE tenant_id = $1 AND resource_server_id = $2;
