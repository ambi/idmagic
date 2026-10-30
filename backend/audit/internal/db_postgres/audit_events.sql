-- name: AppendAuditEvent :execrows
INSERT INTO audit_events (id, tenant_id, type, user_id, occurred_at, payload)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO NOTHING;

-- name: AppendAuditEventSearchAttribute :exec
INSERT INTO audit_event_search_attributes (event_id, tenant_id, attr_name, attr_value, occurred_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (event_id, attr_name, attr_value) DO NOTHING;

-- name: LatestAuditEventByAgent :many
SELECT a.attr_value::text AS agent_id, max(e.occurred_at)::timestamptz AS latest
FROM audit_event_search_attributes a
JOIN audit_events e ON e.id = a.event_id
WHERE a.tenant_id = sqlc.arg(tenant_id)
  AND a.attr_name = 'agent.id'
  AND a.attr_value = ANY(sqlc.arg(agent_ids)::text[])
  AND e.type = ANY(sqlc.arg(event_types)::text[])
GROUP BY a.attr_value;

-- name: GetAuditEventByID :one
SELECT id, tenant_id, type, user_id, created_at, occurred_at, payload
FROM audit_events
WHERE id = $1;

-- name: DeleteAuditEventsByTypeBefore :execrows
DELETE FROM audit_events
WHERE type = $1 AND occurred_at < $2;

-- name: DeleteAuditEventsBeforeExceptTypes :execrows
DELETE FROM audit_events
WHERE occurred_at < $1 AND type <> ALL($2::text[]);
