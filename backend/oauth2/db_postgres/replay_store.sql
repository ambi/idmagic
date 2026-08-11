-- name: ReserveOauth2ReplayJTI :one
-- SETNX + TTL の写像。live な予約は ON CONFLICT の DO UPDATE ... WHERE が
-- false で 0 行 (ErrNoRows)、期限切れの残骸は上書きして 1 行、未存在は INSERT で 1 行。
-- 行が返れば新規予約成功。kind で dpop / client_assertion を名前空間分けする。
INSERT INTO oauth2_replay_jtis (tenant_id, kind, jti, expires_at)
VALUES (@tenant_id, @kind, @jti, @new_expires_at)
ON CONFLICT (tenant_id, kind, jti) DO UPDATE
    SET expires_at = EXCLUDED.expires_at, created_at = now()
    WHERE oauth2_replay_jtis.expires_at <= @now
RETURNING jti;

-- name: DeleteExpiredOauth2ReplayJTIsBatch :execrows
DELETE FROM oauth2_replay_jtis AS o
WHERE (o.tenant_id, o.kind, o.jti) IN (
    SELECT s.tenant_id, s.kind, s.jti
    FROM oauth2_replay_jtis AS s
    WHERE s.expires_at < @cutoff
    LIMIT @batch_limit
);
