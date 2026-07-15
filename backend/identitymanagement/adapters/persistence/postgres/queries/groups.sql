-- name: ListGroupsByTenant :many
SELECT id,tenant_id,name,description,roles,membership_type,created_at,updated_at FROM groups
WHERE tenant_id=$1 ORDER BY name;

-- name: FindGroupByID :one
SELECT id,tenant_id,name,description,roles,membership_type,created_at,updated_at FROM groups
WHERE tenant_id=$1 AND id=$2;

-- name: SaveGroup :exec
INSERT INTO groups (id,tenant_id,name,description,roles,membership_type,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,description=EXCLUDED.description,
 roles=EXCLUDED.roles,updated_at=EXCLUDED.updated_at;

-- name: DeleteGroup :exec
DELETE FROM groups WHERE tenant_id=$1 AND id=$2;

-- name: ListGroupMembersByGroup :many
SELECT gm.group_id,gm.user_id,gm.source,gm.rule_version,gm.created_at
FROM group_members gm JOIN groups g ON g.id=gm.group_id
WHERE g.tenant_id=$1 AND gm.group_id=$2 ORDER BY gm.user_id;

-- name: ListGroupsByUser :many
SELECT g.id,g.tenant_id,g.name,g.description,g.roles,g.membership_type,g.created_at,g.updated_at
FROM groups g JOIN group_members gm ON gm.group_id=g.id
LEFT JOIN dynamic_group_rules dgr ON dgr.group_id=g.id AND dgr.tenant_id=g.tenant_id
WHERE g.tenant_id=$1 AND gm.user_id=$2
  AND ((g.membership_type='manual' AND gm.source='manual')
    OR (g.membership_type='dynamic' AND dgr.enabled AND gm.source='dynamic_rule' AND gm.rule_version=dgr.version))
ORDER BY g.name;

-- name: CountGroupMembers :one
SELECT count(*) FROM group_members gm JOIN groups g ON g.id=gm.group_id
WHERE g.tenant_id=$1 AND gm.group_id=$2;

-- name: AddGroupMember :execrows
INSERT INTO group_members (group_id,user_id,source,rule_version,created_at) VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (group_id,user_id) DO NOTHING;

-- name: RemoveGroupMember :execrows
DELETE FROM group_members
WHERE group_id=$2 AND user_id=$3
  AND group_id IN (SELECT id FROM groups WHERE tenant_id=$1 AND id=$2);
