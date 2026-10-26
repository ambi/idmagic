-- name: ListGroupsByTenant :many
SELECT id,tenant_id,name,description,email,attributes,roles,membership_type,created_at,updated_at,name_key,upstream_managed,role_delegation FROM groups
WHERE tenant_id=$1 ORDER BY name;

-- name: ListGroupsByTenantPage :many
-- First page of ListGroups keyset pagination (wi-159): stable sort by
-- (name, id) so admins see the pre-existing alphabetical order.
SELECT id,tenant_id,name,description,email,attributes,roles,membership_type,created_at,updated_at,name_key,upstream_managed,role_delegation FROM groups
WHERE tenant_id=$1
ORDER BY name, id
LIMIT sqlc.arg(page_limit);

-- name: ListGroupsByTenantPageBefore :many
SELECT id,tenant_id,name,description,email,attributes,roles,membership_type,created_at,updated_at,name_key,upstream_managed,role_delegation FROM groups
WHERE tenant_id=$1
  AND (name, id) < (sqlc.arg(before_name)::text, sqlc.arg(before_id)::uuid)
ORDER BY name DESC, id DESC
LIMIT sqlc.arg(page_limit);

-- name: ListGroupsByTenantPageAfter :many
-- Continuation page: resumes strictly after the (name, id) keyset of the last
-- row the caller saw.
SELECT id,tenant_id,name,description,email,attributes,roles,membership_type,created_at,updated_at,name_key,upstream_managed,role_delegation FROM groups
WHERE tenant_id=$1
  AND (name, id) > (sqlc.arg(after_name)::text, sqlc.arg(after_id)::uuid)
ORDER BY name, id
LIMIT sqlc.arg(page_limit);

-- name: ListGroupsByTenantPageEnd :many
SELECT id,tenant_id,name,description,email,attributes,roles,membership_type,created_at,updated_at,name_key,upstream_managed,role_delegation FROM groups
WHERE tenant_id=$1
ORDER BY name DESC, id DESC
LIMIT sqlc.arg(page_limit);

-- name: CountGroupsByTenant :one
SELECT count(*) FROM groups WHERE tenant_id=$1;

-- name: FindGroupByID :one
SELECT id,tenant_id,name,description,email,attributes,roles,membership_type,created_at,updated_at,name_key,upstream_managed,role_delegation FROM groups
WHERE tenant_id=$1 AND id=$2;

-- name: SaveGroup :exec
INSERT INTO groups (id,tenant_id,name,description,email,attributes,roles,membership_type,created_at,updated_at,name_key,upstream_managed,role_delegation)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,name_key=EXCLUDED.name_key,description=EXCLUDED.description,
 email=EXCLUDED.email,attributes=EXCLUDED.attributes,
 roles=EXCLUDED.roles,updated_at=EXCLUDED.updated_at,
 upstream_managed=groups.upstream_managed OR EXCLUDED.upstream_managed;

-- name: SetGroupRoleDelegation :execrows
UPDATE groups SET role_delegation=$3, updated_at=$4 WHERE tenant_id=$1 AND id=$2;

-- name: DeleteGroup :exec
DELETE FROM groups WHERE tenant_id=$1 AND id=$2;

-- name: ListGroupMembersByGroup :many
SELECT gm.group_id,gm.user_id,gm.source,gm.rule_version,gm.created_at
FROM group_members gm JOIN groups g ON g.id=gm.group_id
WHERE g.tenant_id=$1 AND gm.group_id=$2 ORDER BY gm.user_id;

-- name: ListGroupsByUser :many
SELECT sqlc.embed(g), COALESCE(dgr.expression, '')::text AS dynamic_expression
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

-- name: FindDynamicGroupRule :one
SELECT group_id,tenant_id,expression,enabled,version,referenced_attributes,created_at,updated_at FROM dynamic_group_rules WHERE tenant_id=$1 AND group_id=$2;

-- name: ListDynamicGroupRules :many
SELECT group_id FROM dynamic_group_rules WHERE tenant_id=$1 ORDER BY group_id;

-- name: SaveDynamicGroupRule :exec
INSERT INTO dynamic_group_rules (group_id,tenant_id,expression,enabled,version,referenced_attributes,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (group_id) DO UPDATE SET expression=EXCLUDED.expression,enabled=EXCLUDED.enabled,version=EXCLUDED.version,referenced_attributes=EXCLUDED.referenced_attributes,updated_at=EXCLUDED.updated_at;
