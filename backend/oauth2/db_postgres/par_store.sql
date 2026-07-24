-- name: SavePARRequest :exec
INSERT INTO oauth2_par_requests (request_uri, tenant_id, used, expires_at, payload)
VALUES (@request_uri, @tenant_id, @used, @expires_at, @payload)
ON CONFLICT (request_uri) DO UPDATE
    SET tenant_id = EXCLUDED.tenant_id, used = EXCLUDED.used,
        expires_at = EXCLUDED.expires_at, payload = EXCLUDED.payload, updated_at = now();

-- name: FindPARRequest :one
-- 期限フィルタは付けない (memory adapter とのパリティ: 期限判定は呼び出し側の domain が行う)。
-- tenant_id は fail-closed 述語として必ず含める。used 列を read で payload に overlay する。
SELECT payload, used FROM oauth2_par_requests
WHERE request_uri = @request_uri AND tenant_id = @tenant_id;

-- name: ConsumePARRequest :one
-- 単発消費の CAS。未使用かつ未期限の行だけを used=true にして 1 行返す。それ以外は 0 行。
UPDATE oauth2_par_requests
SET used = TRUE, updated_at = now()
WHERE request_uri = @request_uri AND tenant_id = @tenant_id AND used = FALSE AND expires_at > @now
RETURNING payload;

-- name: DeleteExpiredPARRequestsBatch :execrows
DELETE FROM oauth2_par_requests AS o
WHERE o.request_uri IN (
    SELECT s.request_uri FROM oauth2_par_requests AS s WHERE s.expires_at < @cutoff LIMIT @batch_limit
);
