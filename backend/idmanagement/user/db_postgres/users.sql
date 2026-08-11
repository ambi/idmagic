-- name: FindUserBySub :one
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes,search_text FROM users
WHERE id=$1 AND (lifecycle->>'status' IS DISTINCT FROM 'deleted');

-- name: FindUserBySubIncludingDeleted :one
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes,search_text FROM users
WHERE id=$1;

-- name: FindUserByUsername :one
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes,search_text FROM users
WHERE tenant_id=$1 AND preferred_username=$2 AND (lifecycle->>'status' IS DISTINCT FROM 'deleted');

-- name: FindUserByEmail :one
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes,search_text FROM users
WHERE tenant_id=$1 AND lower(email)=lower($2) AND (lifecycle->>'status' IS DISTINCT FROM 'deleted')
LIMIT 1;

-- name: ListUsersByTenant :many
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes,search_text FROM users
WHERE tenant_id=$1 AND (lifecycle->>'status' IS DISTINCT FROM 'deleted')
ORDER BY preferred_username;

-- name: ListUsersByTenantPage :many
-- First page of ListAdminUsers keyset pagination (wi-159): stable
-- sort by (preferred_username, id) so admins see the pre-existing alphabetical
-- order, with id as tie-break for uniqueness.
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes,search_text FROM users
WHERE tenant_id=$1 AND (lifecycle->>'status' IS DISTINCT FROM 'deleted')
ORDER BY preferred_username, id
LIMIT $2;

-- name: ListUsersByTenantPageAfter :many
-- Continuation page: resumes strictly after the (preferred_username, id) keyset
-- of the last row the caller saw.
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes,search_text FROM users
WHERE tenant_id=$1 AND (lifecycle->>'status' IS DISTINCT FROM 'deleted')
  AND (preferred_username, id) > (sqlc.arg(after_username)::text, sqlc.arg(after_id)::uuid)
ORDER BY preferred_username, id
LIMIT sqlc.arg(page_limit);

-- name: ListUsersByTenantPageFiltered :many
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes,search_text FROM users
WHERE tenant_id=sqlc.arg(tenant_id) AND (lifecycle->>'status' IS DISTINCT FROM 'deleted')
  AND (sqlc.arg(filter_query)::text = '' OR search_text ILIKE '%' || lower(sqlc.arg(filter_query)::text) || '%' ESCAPE '\')
  AND (sqlc.arg(filter_status)::text = '' OR coalesce(lifecycle->>'status', 'active') = sqlc.arg(filter_status)::text)
ORDER BY preferred_username, id
LIMIT sqlc.arg(page_limit);

-- name: ListUsersByTenantPageAfterFiltered :many
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes,search_text FROM users
WHERE tenant_id=sqlc.arg(tenant_id) AND (lifecycle->>'status' IS DISTINCT FROM 'deleted')
  AND (sqlc.arg(filter_query)::text = '' OR search_text ILIKE '%' || lower(sqlc.arg(filter_query)::text) || '%' ESCAPE '\')
  AND (sqlc.arg(filter_status)::text = '' OR coalesce(lifecycle->>'status', 'active') = sqlc.arg(filter_status)::text)
  AND (preferred_username, id) > (sqlc.arg(after_username)::text, sqlc.arg(after_id)::uuid)
ORDER BY preferred_username, id
LIMIT sqlc.arg(page_limit);

-- name: ListUsersByTenantPageBeforeFiltered :many
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes,search_text FROM users
WHERE tenant_id=sqlc.arg(tenant_id) AND (lifecycle->>'status' IS DISTINCT FROM 'deleted')
  AND (sqlc.arg(filter_query)::text = '' OR search_text ILIKE '%' || lower(sqlc.arg(filter_query)::text) || '%' ESCAPE '\')
  AND (sqlc.arg(filter_status)::text = '' OR coalesce(lifecycle->>'status', 'active') = sqlc.arg(filter_status)::text)
  AND (preferred_username, id) < (sqlc.arg(before_username)::text, sqlc.arg(before_id)::uuid)
ORDER BY preferred_username DESC, id DESC
LIMIT sqlc.arg(page_limit);

-- name: ListUsersByTenantPageBefore :many
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes,search_text FROM users
WHERE tenant_id=$1 AND (lifecycle->>'status' IS DISTINCT FROM 'deleted')
  AND (preferred_username, id) < (sqlc.arg(before_username)::text, sqlc.arg(before_id)::uuid)
ORDER BY preferred_username DESC, id DESC
LIMIT sqlc.arg(page_limit);

-- name: ListUsersByTenantPageEnd :many
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes,search_text FROM users
WHERE tenant_id=sqlc.arg(tenant_id) AND (lifecycle->>'status' IS DISTINCT FROM 'deleted')
ORDER BY preferred_username DESC, id DESC
LIMIT sqlc.arg(page_limit);

-- name: ListUsersByTenantPageEndFiltered :many
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes,search_text FROM users
WHERE tenant_id=sqlc.arg(tenant_id) AND (lifecycle->>'status' IS DISTINCT FROM 'deleted')
  AND (sqlc.arg(filter_query)::text = '' OR search_text ILIKE '%' || lower(sqlc.arg(filter_query)::text) || '%' ESCAPE '\')
  AND (sqlc.arg(filter_status)::text = '' OR coalesce(lifecycle->>'status', 'active') = sqlc.arg(filter_status)::text)
ORDER BY preferred_username DESC, id DESC
LIMIT sqlc.arg(page_limit);

-- name: CountUsersByTenant :one
SELECT count(*) FROM users
WHERE tenant_id=$1 AND (lifecycle->>'status' IS DISTINCT FROM 'deleted');

-- name: CountUsersByTenantFiltered :one
SELECT count(*) FROM users
WHERE tenant_id=sqlc.arg(tenant_id) AND (lifecycle->>'status' IS DISTINCT FROM 'deleted')
  AND (sqlc.arg(filter_query)::text = '' OR search_text ILIKE '%' || lower(sqlc.arg(filter_query)::text) || '%' ESCAPE '\')
  AND (sqlc.arg(filter_status)::text = '' OR coalesce(lifecycle->>'status', 'active') = sqlc.arg(filter_status)::text);

-- name: SaveUser :exec
INSERT INTO users (id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
 email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
ON CONFLICT (id) DO UPDATE SET preferred_username=EXCLUDED.preferred_username,
 password_hash=EXCLUDED.password_hash,name=EXCLUDED.name,given_name=EXCLUDED.given_name,
 family_name=EXCLUDED.family_name,email=EXCLUDED.email,email_verified=EXCLUDED.email_verified,
 mfa_enrolled=EXCLUDED.mfa_enrolled,roles=EXCLUDED.roles,lifecycle=EXCLUDED.lifecycle,
 attributes=EXCLUDED.attributes,updated_at=EXCLUDED.updated_at;
