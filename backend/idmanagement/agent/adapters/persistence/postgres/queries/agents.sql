-- name: ListAgentsByTenant :many
SELECT id,tenant_id,name,description,kind,owner_user_id,status,roles,
created_at,updated_at,disabled_at,killed_at FROM agents
WHERE tenant_id=$1 ORDER BY name;

-- name: FindAgentByID :one
SELECT id,tenant_id,name,description,kind,owner_user_id,status,roles,
created_at,updated_at,disabled_at,killed_at FROM agents
WHERE tenant_id=$1 AND id=$2;

-- name: FindAgentByClientID :one
SELECT id,tenant_id,name,description,kind,owner_user_id,status,roles,
created_at,updated_at,disabled_at,killed_at FROM agents
WHERE tenant_id=$1 AND id IN (
  SELECT agent_id FROM agent_credential_bindings WHERE client_id=$2
) LIMIT 1;

-- name: SaveAgent :exec
INSERT INTO agents (id,tenant_id,name,description,kind,owner_user_id,status,roles,
 created_at,updated_at,disabled_at,killed_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,description=EXCLUDED.description,
 kind=EXCLUDED.kind,owner_user_id=EXCLUDED.owner_user_id,status=EXCLUDED.status,roles=EXCLUDED.roles,
 updated_at=EXCLUDED.updated_at,disabled_at=EXCLUDED.disabled_at,killed_at=EXCLUDED.killed_at;

-- name: DeleteAgent :exec
DELETE FROM agents WHERE tenant_id=$1 AND id=$2;

-- name: ListAgentBindingsByAgent :many
SELECT b.agent_id,b.client_id,b.created_at
FROM agent_credential_bindings b JOIN agents a ON a.id=b.agent_id
WHERE a.tenant_id=$1 AND b.agent_id=$2 ORDER BY b.client_id;

-- name: AddAgentBinding :execrows
INSERT INTO agent_credential_bindings (agent_id,client_id,created_at)
VALUES ($1,$2,$3)
ON CONFLICT DO NOTHING;

-- name: RemoveAgentBinding :execrows
DELETE FROM agent_credential_bindings
WHERE agent_id=$2 AND client_id=$3
  AND agent_id IN (SELECT id FROM agents WHERE tenant_id=$1 AND id=$2);
