-- name: SaveDeviceCode :exec
-- Save / Update 共通の upsert。device_code_hash を PK、(tenant_id,user_code) を UNIQUE 鍵に持つ。
INSERT INTO oauth2_device_codes (device_code_hash, tenant_id, user_code, user_id, state, expires_at, payload)
VALUES (@device_code_hash, @tenant_id, @user_code, @user_id, @state, @expires_at, @payload)
ON CONFLICT (device_code_hash) DO UPDATE
    SET tenant_id = EXCLUDED.tenant_id, user_code = EXCLUDED.user_code, user_id = EXCLUDED.user_id,
        state = EXCLUDED.state, expires_at = EXCLUDED.expires_at, payload = EXCLUDED.payload, updated_at = now();

-- name: FindDeviceCodeByHash :one
-- 期限フィルタなし (parity)。state を read で payload に overlay する。
SELECT payload, state FROM oauth2_device_codes
WHERE device_code_hash = @device_code_hash AND tenant_id = @tenant_id;

-- name: FindDeviceCodeByUserCode :one
SELECT payload, state FROM oauth2_device_codes
WHERE user_code = @user_code AND tenant_id = @tenant_id;

-- name: ExchangeDeviceCode :one
-- 単発 exchange の CAS。state='approved' の行だけを exchanged にして 1 行返す。
UPDATE oauth2_device_codes
SET state = 'exchanged', updated_at = now()
WHERE device_code_hash = @device_code_hash AND tenant_id = @tenant_id AND state = 'approved'
RETURNING payload, state;

-- name: DeleteDeviceCodesForUser :exec
DELETE FROM oauth2_device_codes WHERE tenant_id = @tenant_id AND user_id = @user_id;

-- name: DeleteExpiredDeviceCodesBatch :execrows
DELETE FROM oauth2_device_codes AS o
WHERE o.device_code_hash IN (
    SELECT s.device_code_hash FROM oauth2_device_codes AS s WHERE s.expires_at < @cutoff LIMIT @batch_limit
);
