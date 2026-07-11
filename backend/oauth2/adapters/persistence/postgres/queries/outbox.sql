-- name: AppendOutboxEvent :exec
INSERT INTO outbox (event_type, topic, payload)
VALUES ($1, $2, $3);
