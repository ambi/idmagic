-- name: FindNotificationPreference :one
SELECT * FROM notification_preferences WHERE user_id = $1;

-- name: UpsertNotificationPreference :exec
INSERT INTO notification_preferences (user_id, disabled_categories, updated_at)
VALUES ($1, $2, $3)
ON CONFLICT (user_id) DO UPDATE
    SET disabled_categories = EXCLUDED.disabled_categories,
        updated_at = EXCLUDED.updated_at;

-- InsertKnownSignInDevice は行が無いときだけ挿入し、挿入した行数を返す。1 なら
-- 「初めて見る端末」であり、0 なら既知である。判定を 1 文の upsert に畳まないのは、
-- 同一時刻の 2 回目を新規と誤らないようにするためで、競合しても挿入に成功するのは
-- ちょうど 1 つの呼び出しだけである。
-- name: InsertKnownSignInDevice :execrows
INSERT INTO known_sign_in_devices (user_id, device_hash, label, first_seen_at, last_seen_at)
VALUES ($1, $2, $3, $4, $4)
ON CONFLICT (user_id, device_hash) DO NOTHING;

-- name: TouchKnownSignInDevice :exec
UPDATE known_sign_in_devices
SET last_seen_at = $3, label = COALESCE($4, label)
WHERE user_id = $1 AND device_hash = $2;

-- name: DeleteIdleKnownSignInDevices :execrows
DELETE FROM known_sign_in_devices WHERE last_seen_at < $1;
