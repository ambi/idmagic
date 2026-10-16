-- name: SaveAuthorizationRequest :exec
INSERT INTO oauth2_authorization_requests (id, tenant_id, expires_at, payload)
VALUES (@id, @tenant_id, @expires_at, @payload)
ON CONFLICT (id) DO UPDATE
    SET tenant_id = EXCLUDED.tenant_id, expires_at = EXCLUDED.expires_at,
        payload = EXCLUDED.payload, updated_at = now();

-- name: FindAuthorizationRequest :one
-- 期限フィルタなし (parity)。tenant_id は fail-closed 述語。
SELECT payload FROM oauth2_authorization_requests
WHERE id = @id AND tenant_id = @tenant_id;

-- name: LockAuthorizationRequest :one
-- tx 内の read-modify-write を直列化するための行ロック取得。
SELECT payload FROM oauth2_authorization_requests
WHERE id = @id AND tenant_id = @tenant_id
FOR UPDATE;

-- name: UpdateAuthorizationRequestPayload :exec
UPDATE oauth2_authorization_requests
SET payload = @payload, updated_at = now()
WHERE id = @id AND tenant_id = @tenant_id;

-- name: DeleteExpiredAuthorizationRequestsBatch :execrows
DELETE FROM oauth2_authorization_requests AS o
WHERE o.id IN (
    SELECT s.id FROM oauth2_authorization_requests AS s WHERE s.expires_at < @cutoff LIMIT @batch_limit
);
