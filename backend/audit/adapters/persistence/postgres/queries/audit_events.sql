-- name: AppendAuditEvent :exec
INSERT INTO audit_events (id, tenant_id, type, user_id, occurred_at, payload)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO NOTHING;

-- name: AppendAuditEventSearchAttribute :exec
INSERT INTO audit_event_search_attributes (event_id, tenant_id, attr_name, attr_value, occurred_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (event_id, attr_name) DO NOTHING;

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
