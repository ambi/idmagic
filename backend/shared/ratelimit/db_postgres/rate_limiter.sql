-- name: LockRateLimitCounter :one
-- Allow の read-modify-write を直列化する行ロック取得。
SELECT count, window_expires_at FROM endpoint_rate_limit_counters
WHERE tenant_id = @tenant_id AND policy_id = @policy_id AND key_hash = @key_hash
FOR UPDATE;

-- name: UpsertRateLimitCounter :exec
INSERT INTO endpoint_rate_limit_counters (tenant_id, policy_id, key_hash, count, window_expires_at)
VALUES (@tenant_id, @policy_id, @key_hash, @count, @window_expires_at)
ON CONFLICT (tenant_id, policy_id, key_hash) DO UPDATE
    SET count = EXCLUDED.count, window_expires_at = EXCLUDED.window_expires_at, updated_at = now();

-- name: DeleteExpiredRateLimitCountersBatch :execrows
DELETE FROM endpoint_rate_limit_counters AS o
WHERE (o.tenant_id, o.policy_id, o.key_hash) IN (
    SELECT s.tenant_id, s.policy_id, s.key_hash
    FROM endpoint_rate_limit_counters AS s
    WHERE s.window_expires_at < @cutoff
    LIMIT @batch_limit
);
