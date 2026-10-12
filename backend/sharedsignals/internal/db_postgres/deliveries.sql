-- name: ListSecurityEventDeliveriesByStream :many
SELECT id,tenant_id,stream_id,set_jti,set_payload,status,attempt_count,next_attempt_at,
last_error,created_at,delivered_at
FROM security_event_deliveries WHERE tenant_id=$1 AND stream_id=$2 ORDER BY created_at;

-- name: ListDueSecurityEventDeliveries :many
SELECT id,tenant_id,stream_id,set_jti,set_payload,status,attempt_count,next_attempt_at,
last_error,created_at,delivered_at
FROM security_event_deliveries
WHERE status IN ('pending','failed')
  AND (next_attempt_at IS NULL OR next_attempt_at <= $1)
ORDER BY created_at
LIMIT NULLIF(sqlc.arg(row_limit)::int, 0);

-- name: SaveSecurityEventDelivery :exec
INSERT INTO security_event_deliveries (id,tenant_id,stream_id,set_jti,set_payload,status,
 attempt_count,next_attempt_at,last_error,created_at,delivered_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (id) DO UPDATE SET
 status=EXCLUDED.status,attempt_count=EXCLUDED.attempt_count,
 next_attempt_at=EXCLUDED.next_attempt_at,last_error=EXCLUDED.last_error,
 delivered_at=EXCLUDED.delivered_at;
