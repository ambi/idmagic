-- name: SaveAuthorizationCode :exec
INSERT INTO oauth2_authorization_codes (code, tenant_id, state, redeemed_at, issued_family_id, expires_at, payload)
VALUES (@code, @tenant_id, @state, @redeemed_at, @issued_family_id, @expires_at, @payload)
ON CONFLICT (code) DO UPDATE
    SET tenant_id = EXCLUDED.tenant_id, state = EXCLUDED.state, redeemed_at = EXCLUDED.redeemed_at,
        issued_family_id = EXCLUDED.issued_family_id, expires_at = EXCLUDED.expires_at,
        payload = EXCLUDED.payload, updated_at = now();

-- name: FindAuthorizationCode :one
-- 期限フィルタなし (parity)。state / redeemed_at / issued_family_id は read で payload に overlay。
SELECT payload, state, redeemed_at, issued_family_id FROM oauth2_authorization_codes
WHERE code = @code AND tenant_id = @tenant_id;

-- name: RedeemAuthorizationCode :one
-- 単発 redeem の CAS。state='issued' の行だけを redeemed にして 1 行返す。既 redeemed は 0 行。
UPDATE oauth2_authorization_codes
SET state = 'redeemed', redeemed_at = @redeemed_at, updated_at = now()
WHERE code = @code AND tenant_id = @tenant_id AND state = 'issued'
RETURNING payload, state, redeemed_at, issued_family_id;

-- name: LinkAuthorizationCodeFamily :execrows
UPDATE oauth2_authorization_codes
SET issued_family_id = @issued_family_id, updated_at = now()
WHERE code = @code AND tenant_id = @tenant_id;

-- name: DeleteExpiredAuthorizationCodesBatch :execrows
DELETE FROM oauth2_authorization_codes AS o
WHERE o.code IN (
    SELECT s.code FROM oauth2_authorization_codes AS s WHERE s.expires_at < @cutoff LIMIT @batch_limit
);
