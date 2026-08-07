-- name: FindSsfTransmitterConfigByStream :one
SELECT stream_id,tenant_id,delivery_endpoint,audience,delivery_authorization,max_delivery_attempts
FROM ssf_transmitter_configs WHERE tenant_id=$1 AND stream_id=$2;

-- name: SaveSsfTransmitterConfig :exec
INSERT INTO ssf_transmitter_configs (stream_id,tenant_id,delivery_endpoint,audience,delivery_authorization,max_delivery_attempts)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (stream_id) DO UPDATE SET
 delivery_endpoint=EXCLUDED.delivery_endpoint,audience=EXCLUDED.audience,
 delivery_authorization=EXCLUDED.delivery_authorization,max_delivery_attempts=EXCLUDED.max_delivery_attempts;

-- name: DeleteSsfTransmitterConfig :exec
DELETE FROM ssf_transmitter_configs WHERE tenant_id=$1 AND stream_id=$2;
