-- name: EnsureDefaultSamlIDPProfile :one
INSERT INTO saml_identity_provider_profiles
    (tenant_id, profile_id, name, mode, is_default)
VALUES ($1, 'default', 'Default', 'shared', TRUE)
ON CONFLICT (tenant_id, profile_id) DO UPDATE SET tenant_id = EXCLUDED.tenant_id
RETURNING tenant_id, profile_id, name, mode, is_default, created_at, updated_at;

-- name: GetSamlIDPProfile :one
SELECT tenant_id, profile_id, name, mode, is_default, created_at, updated_at
FROM saml_identity_provider_profiles
WHERE tenant_id = $1 AND profile_id = $2;

-- name: EnsureSamlIDPProfileRow :exec
-- 行がなければ後続の行ロックは何も固定しないので、先に行を作り、同時の作成の確定を待つ。
-- 値は同じトランザクションの保存が置き換える。
INSERT INTO saml_identity_provider_profiles
    (tenant_id, profile_id, name, mode, is_default, created_at)
VALUES (@tenant_id, @profile_id, '', 'shared', @profile_id::text = 'default', @created_at)
ON CONFLICT (tenant_id, profile_id) DO NOTHING;

-- name: GetSamlIDPProfileForUpdate :one
SELECT tenant_id, profile_id, name, mode, is_default, created_at, updated_at
FROM saml_identity_provider_profiles
WHERE tenant_id = $1 AND profile_id = $2
FOR UPDATE;

-- name: ListSamlIDPProfilesByTenant :many
SELECT tenant_id, profile_id, name, mode, is_default, created_at, updated_at
FROM saml_identity_provider_profiles
WHERE tenant_id = $1
ORDER BY is_default DESC, name, profile_id;

-- name: CountSamlServiceProvidersByIDPProfile :one
SELECT count(*)
FROM saml_service_providers
WHERE tenant_id = $1 AND idp_profile_id = $2;

-- name: CountOtherSamlServiceProvidersByIDPProfile :one
SELECT count(*)
FROM saml_service_providers
WHERE tenant_id = $1 AND idp_profile_id = $2 AND entity_id <> $3;

-- name: UpsertSamlIDPProfile :exec
INSERT INTO saml_identity_provider_profiles
    (tenant_id, profile_id, name, mode, is_default, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (tenant_id, profile_id) DO UPDATE SET
    name = EXCLUDED.name,
    mode = EXCLUDED.mode,
    updated_at = EXCLUDED.updated_at;

-- name: DeleteSamlIDPProfile :exec
DELETE FROM saml_identity_provider_profiles
WHERE tenant_id = $1 AND profile_id = $2;
