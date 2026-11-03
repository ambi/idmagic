-- name: GetMcpToolReport :one
SELECT tenant_id, resource_server_id, tools, complete, reported_at, previous_tools, previous_complete, changed_at
FROM mcp_tool_reports
WHERE tenant_id = $1 AND resource_server_id = $2;

-- name: UpsertMcpToolReport :exec
-- 指紋が保存済みの行と異なるときだけ、保存済みの一覧を変更前へ移す (REQ-OAUTH2-086)。
-- 比べる値は更新前の行の値なので、並行する報告も一つの文の中で比べて置き換える。
-- 指紋が空の行は列の追加前に保存したもので、内容を比べられないため変更前を作らない。
INSERT INTO mcp_tool_reports (tenant_id, resource_server_id, tools, complete, reported_at, fingerprint)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (resource_server_id) DO UPDATE SET
  tools = EXCLUDED.tools,
  complete = EXCLUDED.complete,
  reported_at = EXCLUDED.reported_at,
  fingerprint = EXCLUDED.fingerprint,
  previous_tools = CASE WHEN mcp_tool_reports.fingerprint NOT IN ('', EXCLUDED.fingerprint)
    THEN mcp_tool_reports.tools ELSE mcp_tool_reports.previous_tools END,
  previous_complete = CASE WHEN mcp_tool_reports.fingerprint NOT IN ('', EXCLUDED.fingerprint)
    THEN mcp_tool_reports.complete ELSE mcp_tool_reports.previous_complete END,
  changed_at = CASE WHEN mcp_tool_reports.fingerprint NOT IN ('', EXCLUDED.fingerprint)
    THEN EXCLUDED.reported_at ELSE mcp_tool_reports.changed_at END
WHERE mcp_tool_reports.tenant_id = EXCLUDED.tenant_id;

-- name: DeleteMcpToolReport :exec
DELETE FROM mcp_tool_reports WHERE tenant_id = $1 AND resource_server_id = $2;
