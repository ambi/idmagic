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
