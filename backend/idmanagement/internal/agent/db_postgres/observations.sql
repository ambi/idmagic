-- name: LockAgentObservationsTenant :exec
-- 同じテナントの取り込みを直列にし、件数の上限を確かめてから書く。
SELECT pg_advisory_xact_lock(hashtextextended('agent_observations:' || sqlc.arg(tenant_id)::text, 0));

-- name: UpsertAgentObservation :one
-- 保存値より新しい観測だけが属性を更新し、決定の列には触れない。
-- 行を返さなければ変更なしである。
INSERT INTO agent_observations (
  id, tenant_id, issuer, subject, external_registration_id, display_name,
  resources, responsible_candidate, observed_at, source_token_id, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (tenant_id, issuer, subject) DO UPDATE SET
  external_registration_id=EXCLUDED.external_registration_id,
  display_name=EXCLUDED.display_name,
  resources=EXCLUDED.resources,
  responsible_candidate=EXCLUDED.responsible_candidate,
  observed_at=EXCLUDED.observed_at,
  source_token_id=EXCLUDED.source_token_id
WHERE agent_observations.observed_at < EXCLUDED.observed_at
RETURNING (xmax = 0)::boolean AS inserted;

-- name: FindAgentObservationByID :one
SELECT * FROM agent_observations WHERE tenant_id=$1 AND id=$2;

-- name: FindAgentObservationByIDForUpdate :one
SELECT * FROM agent_observations WHERE tenant_id=$1 AND id=$2 FOR UPDATE;

-- name: ListAgentObservationsPage :many
SELECT * FROM agent_observations
WHERE tenant_id=sqlc.arg(tenant_id)
  AND (sqlc.arg(decision)::text = ''
    OR (sqlc.arg(decision)::text = 'confirmed' AND confirmed_agent_id IS NOT NULL)
    OR (sqlc.arg(decision)::text = 'excluded' AND excluded)
    OR (sqlc.arg(decision)::text = 'pending' AND NOT excluded AND confirmed_agent_id IS NULL))
  AND (sqlc.arg(after_id)::text = '' OR (subject, id::text) > (sqlc.arg(after_subject)::text, sqlc.arg(after_id)::text))
ORDER BY subject, id
LIMIT sqlc.arg(page_limit);

-- name: ListAgentObservationsPageBefore :many
SELECT * FROM agent_observations
WHERE tenant_id=sqlc.arg(tenant_id)
  AND (sqlc.arg(decision)::text = ''
    OR (sqlc.arg(decision)::text = 'confirmed' AND confirmed_agent_id IS NOT NULL)
    OR (sqlc.arg(decision)::text = 'excluded' AND excluded)
    OR (sqlc.arg(decision)::text = 'pending' AND NOT excluded AND confirmed_agent_id IS NULL))
  AND (sqlc.arg(before_id)::text = '' OR (subject, id::text) < (sqlc.arg(before_subject)::text, sqlc.arg(before_id)::text))
ORDER BY subject DESC, id DESC
LIMIT sqlc.arg(page_limit);

-- name: CountAgentObservations :one
SELECT count(*) FROM agent_observations
WHERE tenant_id=sqlc.arg(tenant_id)
  AND (sqlc.arg(decision)::text = ''
    OR (sqlc.arg(decision)::text = 'confirmed' AND confirmed_agent_id IS NOT NULL)
    OR (sqlc.arg(decision)::text = 'excluded' AND excluded)
    OR (sqlc.arg(decision)::text = 'pending' AND NOT excluded AND confirmed_agent_id IS NULL));

-- name: ListConfirmedAgentObservations :many
SELECT * FROM agent_observations
WHERE tenant_id=$1 AND confirmed_agent_id IS NOT NULL
ORDER BY subject, id;

-- name: ConfirmAgentObservation :execrows
UPDATE agent_observations
SET confirmed_agent_id=sqlc.arg(agent_id), decided_at=sqlc.arg(decided_at), decided_by_user_id=sqlc.arg(decided_by_user_id)
WHERE tenant_id=sqlc.arg(tenant_id) AND id=sqlc.arg(id) AND NOT excluded AND confirmed_agent_id IS NULL;

-- name: ExcludeAgentObservation :execrows
UPDATE agent_observations
SET excluded=TRUE, decided_at=sqlc.arg(decided_at), decided_by_user_id=sqlc.arg(decided_by_user_id)
WHERE tenant_id=sqlc.arg(tenant_id) AND id=sqlc.arg(id) AND NOT excluded AND confirmed_agent_id IS NULL;

-- name: ReopenAgentObservation :exec
UPDATE agent_observations
SET excluded=FALSE, confirmed_agent_id=NULL, decided_at=NULL, decided_by_user_id=NULL
WHERE tenant_id=$1 AND id=$2;

-- name: PurgeExpiredAgentObservations :execrows
DELETE FROM agent_observations WHERE confirmed_agent_id IS NULL AND observed_at < $1;
