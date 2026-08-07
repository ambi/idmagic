-- name: ListAgentWorkloadBindingsByTrustBundle :many
SELECT id,tenant_id,trust_bundle_id,subject_pattern,agent_id,status,created_at,updated_at,disabled_at
FROM agent_workload_bindings WHERE tenant_id=$1 AND trust_bundle_id=$2 ORDER BY subject_pattern;

-- name: FindAgentWorkloadBindingByID :one
SELECT id,tenant_id,trust_bundle_id,subject_pattern,agent_id,status,created_at,updated_at,disabled_at
FROM agent_workload_bindings WHERE tenant_id=$1 AND id=$2;

-- name: SaveAgentWorkloadBinding :exec
INSERT INTO agent_workload_bindings (id,tenant_id,trust_bundle_id,subject_pattern,agent_id,status,
 created_at,updated_at,disabled_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (id) DO UPDATE SET status=EXCLUDED.status,updated_at=EXCLUDED.updated_at,
 disabled_at=EXCLUDED.disabled_at;

-- name: DeleteAgentWorkloadBinding :exec
DELETE FROM agent_workload_bindings WHERE tenant_id=$1 AND id=$2;
