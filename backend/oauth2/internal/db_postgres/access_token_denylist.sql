-- name: AddOauth2AccessTokenDenylist :exec
-- 失効マーカーを追加する。同一 jti は同一トークンを指し exp も決定的なので冪等 (DO NOTHING)。
INSERT INTO oauth2_access_token_denylist (tenant_id, jti, expires_at)
VALUES (@tenant_id, @jti, @expires_at)
ON CONFLICT (tenant_id, jti) DO NOTHING;

-- name: IsOauth2AccessTokenRevoked :one
-- 期限切れマーカーは revoked とみなさない (正しさは expires_at > now が担保、GC は空間回収のみ)。
SELECT EXISTS(
    SELECT 1 FROM oauth2_access_token_denylist
    WHERE tenant_id = @tenant_id AND jti = @jti AND expires_at > @now
);

-- name: DeleteExpiredOauth2AccessTokenDenylistBatch :execrows
DELETE FROM oauth2_access_token_denylist AS o
WHERE (o.tenant_id, o.jti) IN (
    SELECT s.tenant_id, s.jti
    FROM oauth2_access_token_denylist AS s
    WHERE s.expires_at < @cutoff
    LIMIT @batch_limit
);
