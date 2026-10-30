-- name: FindAgentRevocationEpoch :one
SELECT agent_id,tenant_id,epoch,reason,advanced_at,source_event_id
FROM agent_revocation_epochs WHERE tenant_id=$1 AND agent_id=$2;

-- name: AdvanceAgentRevocationEpoch :one
INSERT INTO agent_revocation_epochs (agent_id,tenant_id,epoch,reason,advanced_at,source_event_id)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (agent_id) DO UPDATE SET
 epoch=EXCLUDED.epoch,reason=EXCLUDED.reason,advanced_at=EXCLUDED.advanced_at,
 source_event_id=EXCLUDED.source_event_id
 WHERE EXCLUDED.epoch > agent_revocation_epochs.epoch
RETURNING agent_id,tenant_id,epoch,reason,advanced_at,source_event_id;

-- name: FindUserRevocationEpoch :one
SELECT user_id,tenant_id,epoch,reason,advanced_at,source_event_id
FROM user_revocation_epochs WHERE tenant_id=$1 AND user_id=$2;

-- name: AdvanceUserRevocationEpoch :one
INSERT INTO user_revocation_epochs (user_id,tenant_id,epoch,reason,advanced_at,source_event_id)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (user_id) DO UPDATE SET
 epoch=EXCLUDED.epoch,reason=EXCLUDED.reason,advanced_at=EXCLUDED.advanced_at,
 source_event_id=EXCLUDED.source_event_id
 WHERE EXCLUDED.epoch > user_revocation_epochs.epoch
RETURNING user_id,tenant_id,epoch,reason,advanced_at,source_event_id;
