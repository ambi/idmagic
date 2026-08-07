-- name: FindSsfReceiverConfigByStream :one
SELECT stream_id,tenant_id,trusted_issuer,jwks_uri,jwks,accepted_audiences
FROM ssf_receiver_configs WHERE tenant_id=$1 AND stream_id=$2;

-- name: SaveSsfReceiverConfig :exec
INSERT INTO ssf_receiver_configs (stream_id,tenant_id,trusted_issuer,jwks_uri,jwks,accepted_audiences)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (stream_id) DO UPDATE SET
 trusted_issuer=EXCLUDED.trusted_issuer,jwks_uri=EXCLUDED.jwks_uri,jwks=EXCLUDED.jwks,
 accepted_audiences=EXCLUDED.accepted_audiences;

-- name: DeleteSsfReceiverConfig :exec
DELETE FROM ssf_receiver_configs WHERE tenant_id=$1 AND stream_id=$2;
