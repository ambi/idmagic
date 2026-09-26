-- name: SaveScimGroupRef :exec
INSERT INTO scim_group_refs (tenant_id, scim_id, group_id)
VALUES ($1, $2, $3)
ON CONFLICT (tenant_id, scim_id) DO UPDATE SET
    group_id=EXCLUDED.group_id,
    updated_at=now();

-- name: FindScimGroupRefByScimID :one
SELECT tenant_id, scim_id, group_id FROM scim_group_refs WHERE tenant_id = $1 AND scim_id = $2;

-- name: FindScimGroupRefByGroupID :one
SELECT tenant_id, scim_id, group_id FROM scim_group_refs WHERE tenant_id = $1 AND group_id = $2;

-- name: DeleteScimGroupRef :exec
DELETE FROM scim_group_refs WHERE tenant_id = $1 AND scim_id = $2;

-- name: ListScimGroupRefsByGroupIDs :many
-- 接続は uuid を text として登録しているが、uuid[] の要素はバイナリで符号化されるため、
-- 配列は text[] で受け取ってから uuid[] にする。
SELECT tenant_id, scim_id, group_id FROM scim_group_refs
WHERE tenant_id = sqlc.arg(tenant_id) AND group_id = ANY(sqlc.arg(group_ids)::text[]::uuid[]);
