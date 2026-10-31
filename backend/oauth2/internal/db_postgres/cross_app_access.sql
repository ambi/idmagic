-- name: ListCrossAppAccessIssuePolicies :many
SELECT tenant_id, id, client_id, audience, relying_party_client_id, target_client_id, resources, scopes, authorization_detail_types, release_email, created_at
FROM cross_app_access_issue_policies
WHERE tenant_id = $1
ORDER BY client_id, audience;

-- name: GetCrossAppAccessIssuePolicy :one
SELECT tenant_id, id, client_id, audience, relying_party_client_id, target_client_id, resources, scopes, authorization_detail_types, release_email, created_at
FROM cross_app_access_issue_policies
WHERE tenant_id = $1 AND id = $2;

-- name: InsertCrossAppAccessIssuePolicy :execrows
INSERT INTO cross_app_access_issue_policies (tenant_id, id, client_id, audience, relying_party_client_id, target_client_id, resources, scopes, authorization_detail_types, release_email, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (tenant_id, client_id, audience) DO NOTHING;

-- name: DeleteCrossAppAccessIssuePolicy :exec
DELETE FROM cross_app_access_issue_policies WHERE tenant_id = $1 AND id = $2;

-- name: ListCrossAppAccessRedeemPolicies :many
SELECT tenant_id, id, provider_id, client_id, scopes, created_at
FROM cross_app_access_redeem_policies
WHERE tenant_id = $1
ORDER BY provider_id, client_id;

-- name: GetCrossAppAccessRedeemPolicy :one
SELECT tenant_id, id, provider_id, client_id, scopes, created_at
FROM cross_app_access_redeem_policies
WHERE tenant_id = $1 AND id = $2;

-- name: GetCrossAppAccessRedeemPolicyFor :one
SELECT tenant_id, id, provider_id, client_id, scopes, created_at
FROM cross_app_access_redeem_policies
WHERE tenant_id = $1 AND provider_id = $2 AND client_id = $3;

-- name: InsertCrossAppAccessRedeemPolicy :execrows
INSERT INTO cross_app_access_redeem_policies (tenant_id, id, provider_id, client_id, scopes, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tenant_id, provider_id, client_id) DO NOTHING;

-- name: DeleteCrossAppAccessRedeemPolicy :exec
DELETE FROM cross_app_access_redeem_policies WHERE tenant_id = $1 AND id = $2;
