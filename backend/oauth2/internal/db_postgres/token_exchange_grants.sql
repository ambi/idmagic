-- name: InsertOauth2TokenExchangeGrant :exec
INSERT INTO oauth2_token_exchange_grants (tenant_id, grant_hash, client_id, subject_token_id, resource, expires_at)
VALUES (@tenant_id, @grant_hash, @client_id, @subject_token_id, @resource, @expires_at);

-- name: ClaimOauth2TokenExchangeGrant :one
-- 全項目が一致する行だけを消すので、流用の試みは正当な許可を消費しない。並行した要求の一方だけが行を得る。
DELETE FROM oauth2_token_exchange_grants
WHERE tenant_id = @tenant_id AND grant_hash = @grant_hash AND client_id = @client_id
  AND subject_token_id = @subject_token_id AND resource = @resource AND expires_at > @now
RETURNING grant_hash;

-- name: DeleteExpiredOauth2TokenExchangeGrantsBatch :execrows
DELETE FROM oauth2_token_exchange_grants AS o
WHERE (o.tenant_id, o.grant_hash) IN (
    SELECT s.tenant_id, s.grant_hash
    FROM oauth2_token_exchange_grants AS s
    WHERE s.expires_at < @cutoff
    LIMIT @batch_limit
);
