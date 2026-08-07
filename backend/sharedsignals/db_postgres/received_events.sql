-- name: ExistsReceivedSecurityEventByJTI :one
SELECT EXISTS (
  SELECT 1 FROM received_security_events WHERE tenant_id=$1 AND stream_id=$2 AND set_jti=$3
) AS exists;

-- name: SaveReceivedSecurityEvent :exec
INSERT INTO received_security_events (id,tenant_id,stream_id,set_jti,event_type,subject,
 verification_result,received_at,reflected_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (stream_id, set_jti) DO NOTHING;
