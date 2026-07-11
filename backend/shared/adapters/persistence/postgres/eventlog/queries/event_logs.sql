-- name: InsertEventLog :exec
INSERT INTO event_logs (id, tenant_id, type, classification, actor, subject, correlation_id, occurred_at, payload)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetEventLogByID :one
SELECT id, tenant_id, type, classification, actor, subject, correlation_id, occurred_at, created_at, payload
FROM event_logs
WHERE id = $1;

-- name: InsertEventDelivery :exec
INSERT INTO event_deliveries (event_id)
VALUES ($1);

-- name: GetEventDeliveryByID :one
SELECT event_id, status, attempts, last_error, delivered_at, updated_at
FROM event_deliveries
WHERE event_id = $1;
