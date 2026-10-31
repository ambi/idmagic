-- name: ListEffectiveAgentGuardrailPolicies :many
SELECT tenant_id, scope_key, limits, version, updated_at
FROM agent_guardrail_policies
WHERE tenant_id = $1 AND scope_key = ANY($2::text[])
ORDER BY scope_key;

-- name: GetAgentGuardrailPolicy :one
SELECT tenant_id, scope_key, limits, version, updated_at
FROM agent_guardrail_policies
WHERE tenant_id = $1 AND scope_key = $2;

-- name: InsertAgentGuardrailPolicy :execrows
INSERT INTO agent_guardrail_policies (tenant_id, scope_key, agent_id, limits, version, updated_at)
VALUES ($1, $2, sqlc.narg(agent_id), $3, $4, $5)
ON CONFLICT (tenant_id, scope_key) DO NOTHING;

-- name: UpdateAgentGuardrailPolicy :execrows
UPDATE agent_guardrail_policies
SET limits = $3, version = $4, updated_at = $5
WHERE tenant_id = $1 AND scope_key = $2 AND version = sqlc.arg(expected_version);

-- name: DeleteAgentGuardrailPolicy :exec
DELETE FROM agent_guardrail_policies WHERE tenant_id = $1 AND scope_key = $2;
