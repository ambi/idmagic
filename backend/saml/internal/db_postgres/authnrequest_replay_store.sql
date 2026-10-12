-- name: ReserveSamlAuthnRequestReplay :one
-- TTL 付き単発予約。live な予約が既にあれば ON CONFLICT の
-- DO UPDATE ... WHERE が false になり 0 行 (ErrNoRows)、期限切れの残骸なら上書きして 1 行、
-- 未存在なら INSERT で 1 行を返す。行が返れば新規予約成功 (重複時は 0 行 = 予約失敗)。
INSERT INTO saml_authnrequest_replays (tenant_id, entity_id, request_id, expires_at)
VALUES (@tenant_id, @entity_id, @request_id, @new_expires_at)
ON CONFLICT (tenant_id, entity_id, request_id) DO UPDATE
    SET expires_at = EXCLUDED.expires_at, created_at = now()
    WHERE saml_authnrequest_replays.expires_at <= @now
RETURNING request_id;

-- name: DeleteExpiredSamlAuthnRequestReplaysBatch :execrows
-- housekeeping cleanup。PK を選んで小 batch で削除する (正しさは read 側の expires_at 述語が担保)。
DELETE FROM saml_authnrequest_replays AS o
WHERE (o.tenant_id, o.entity_id, o.request_id) IN (
    SELECT s.tenant_id, s.entity_id, s.request_id
    FROM saml_authnrequest_replays AS s
    WHERE s.expires_at < @cutoff
    LIMIT @batch_limit
);
