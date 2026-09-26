-- name: SaveScimUserRef :exec
INSERT INTO scim_user_refs (tenant_id, scim_id, user_id)
VALUES ($1, $2, $3)
ON CONFLICT (tenant_id, scim_id) DO UPDATE SET
    user_id=EXCLUDED.user_id,
    updated_at=now();

-- name: FindScimUserRefByScimID :one
SELECT tenant_id, scim_id, user_id FROM scim_user_refs WHERE tenant_id = $1 AND scim_id = $2;

-- name: FindScimUserRefByUserID :one
SELECT tenant_id, scim_id, user_id FROM scim_user_refs WHERE tenant_id = $1 AND user_id = $2;

-- name: DeleteScimUserRef :exec
DELETE FROM scim_user_refs WHERE tenant_id = $1 AND scim_id = $2;

-- name: ListScimUserRefsByUserIDs :many
-- 接続は uuid を text として登録しているが、uuid[] の要素はバイナリで符号化されるため、
-- 配列は text[] で受け取ってから uuid[] にする。
SELECT tenant_id, scim_id, user_id FROM scim_user_refs
WHERE tenant_id = sqlc.arg(tenant_id) AND user_id = ANY(sqlc.arg(user_ids)::text[]::uuid[]);
