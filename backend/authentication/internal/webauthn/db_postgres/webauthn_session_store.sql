-- name: SaveWebauthnSession :exec
INSERT INTO webauthn_sessions (tenant_id, session_key, data, expires_at)
VALUES (@tenant_id, @session_key, @data, @expires_at)
ON CONFLICT (tenant_id, session_key) DO UPDATE
    SET data = EXCLUDED.data, expires_at = EXCLUDED.expires_at, updated_at = now();

-- name: TakeWebauthnSession :one
-- GetDel 相当。live な行だけを一度きり取り出す。期限切れ / 未存在は 0 行 (ErrNoRows → nil)。
DELETE FROM webauthn_sessions
WHERE tenant_id = @tenant_id AND session_key = @session_key AND expires_at > @now
RETURNING data;

-- name: DeleteExpiredWebauthnSessionsBatch :execrows
DELETE FROM webauthn_sessions AS o
WHERE (o.tenant_id, o.session_key) IN (
    SELECT s.tenant_id, s.session_key
    FROM webauthn_sessions AS s
    WHERE s.expires_at < @cutoff
    LIMIT @batch_limit
);
