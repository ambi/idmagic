-- name: FindUserBySub :one
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes FROM users
WHERE id=$1 AND (lifecycle->>'status' IS DISTINCT FROM 'deleted');

-- name: FindUserBySubIncludingDeleted :one
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes FROM users
WHERE id=$1;

-- name: FindUserByUsername :one
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes FROM users
WHERE tenant_id=$1 AND preferred_username=$2 AND (lifecycle->>'status' IS DISTINCT FROM 'deleted');

-- name: FindUserByEmail :one
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes FROM users
WHERE tenant_id=$1 AND lower(email)=lower($2) AND (lifecycle->>'status' IS DISTINCT FROM 'deleted')
LIMIT 1;

-- name: ListUsersByTenant :many
SELECT id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes FROM users
WHERE tenant_id=$1 AND (lifecycle->>'status' IS DISTINCT FROM 'deleted')
ORDER BY preferred_username;

-- name: SaveUser :exec
INSERT INTO users (id,tenant_id,preferred_username,password_hash,name,given_name,family_name,email,
 email_verified,mfa_enrolled,created_at,updated_at,roles,lifecycle,attributes)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
ON CONFLICT (id) DO UPDATE SET preferred_username=EXCLUDED.preferred_username,
 password_hash=EXCLUDED.password_hash,name=EXCLUDED.name,given_name=EXCLUDED.given_name,
 family_name=EXCLUDED.family_name,email=EXCLUDED.email,email_verified=EXCLUDED.email_verified,
 mfa_enrolled=EXCLUDED.mfa_enrolled,roles=EXCLUDED.roles,lifecycle=EXCLUDED.lifecycle,
 attributes=EXCLUDED.attributes,updated_at=EXCLUDED.updated_at;
