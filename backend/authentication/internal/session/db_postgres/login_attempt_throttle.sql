-- name: GetThrottleLock :one
-- TryAcquire 用の read-only lookup。locked_until だけ見て allowed / retry を判定する。
SELECT locked_until FROM login_throttle_counters
WHERE tenant_id = @tenant_id AND kind = @kind AND identifier_hash = @identifier_hash;

-- name: LockThrottleCounter :one
-- RecordFailure の read-modify-write を直列化する行ロック取得。
SELECT failures, window_expires_at, locked_until FROM login_throttle_counters
WHERE tenant_id = @tenant_id AND kind = @kind AND identifier_hash = @identifier_hash
FOR UPDATE;

-- name: UpsertThrottleCounter :exec
INSERT INTO login_throttle_counters (tenant_id, kind, identifier_hash, failures, window_expires_at, locked_until)
VALUES (@tenant_id, @kind, @identifier_hash, @failures, @window_expires_at, @locked_until)
ON CONFLICT (tenant_id, kind, identifier_hash) DO UPDATE
    SET failures = EXCLUDED.failures, window_expires_at = EXCLUDED.window_expires_at,
        locked_until = EXCLUDED.locked_until, updated_at = now();

-- name: DeleteThrottleCounter :exec
DELETE FROM login_throttle_counters
WHERE tenant_id = @tenant_id AND kind = @kind AND identifier_hash = @identifier_hash;

-- name: DeleteExpiredThrottleCountersBatch :execrows
-- window と lock の双方が過ぎた行だけを回収する (fail-closed 前提を GC で崩さない)。
DELETE FROM login_throttle_counters AS o
WHERE (o.tenant_id, o.kind, o.identifier_hash) IN (
    SELECT s.tenant_id, s.kind, s.identifier_hash
    FROM login_throttle_counters AS s
    WHERE s.window_expires_at < @cutoff AND (s.locked_until IS NULL OR s.locked_until < @cutoff)
    LIMIT @batch_limit
);
