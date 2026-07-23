-- name: GetWsFedRelyingParty :one
SELECT tenant_id, wtrealm, application_id, application_protocol_type, display_name, reply_urls, audience, token_type, claim_policy, entra_profile, created_at, updated_at
FROM wsfed_relying_parties WHERE tenant_id = $1 AND wtrealm = $2;

-- name: ListWsFedRelyingPartiesByTenant :many
SELECT tenant_id, wtrealm, application_id, application_protocol_type, display_name, reply_urls, audience, token_type, claim_policy, entra_profile, created_at, updated_at
FROM wsfed_relying_parties WHERE tenant_id = $1 ORDER BY wtrealm;

-- name: UpsertWsFedRelyingParty :exec
INSERT INTO wsfed_relying_parties (tenant_id, wtrealm, display_name, reply_urls, audience, token_type, claim_policy, entra_profile, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (tenant_id,wtrealm) DO UPDATE SET display_name=EXCLUDED.display_name, reply_urls=EXCLUDED.reply_urls,
 audience=EXCLUDED.audience, token_type=EXCLUDED.token_type, claim_policy=EXCLUDED.claim_policy,
 entra_profile=EXCLUDED.entra_profile, updated_at=EXCLUDED.updated_at;

-- name: DeleteWsFedRelyingParty :exec
DELETE FROM wsfed_relying_parties WHERE tenant_id = $1 AND wtrealm = $2;
