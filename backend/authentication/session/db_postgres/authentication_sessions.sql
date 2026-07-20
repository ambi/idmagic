-- name: UpsertAuthenticationSession :exec
INSERT INTO authentication_sessions (
    id, tenant_id, user_id, auth_time, amr, acr, authentication_pending,
    pending_purpose, enrollment_deadline, enrollment_bypass_id, step_up_at,
    expires_at, last_seen_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, now(), now()
)
ON CONFLICT (id) DO UPDATE SET
    amr = EXCLUDED.amr,
    acr = EXCLUDED.acr,
    authentication_pending = EXCLUDED.authentication_pending,
    pending_purpose = EXCLUDED.pending_purpose,
    enrollment_deadline = EXCLUDED.enrollment_deadline,
    enrollment_bypass_id = EXCLUDED.enrollment_bypass_id,
    step_up_at = EXCLUDED.step_up_at,
    expires_at = EXCLUDED.expires_at,
    updated_at = now();

-- name: FindActiveAuthenticationSession :one
-- 認証解決用の fail-closed lookup。tenant_id / revoked_at / expires_at を DB 層で検証し、
-- 別 tenant または失効・期限切れの行を返さない (ADR-126)。
SELECT id, tenant_id, user_id, auth_time, amr, acr, authentication_pending,
       pending_purpose, enrollment_deadline, enrollment_bypass_id, step_up_at,
       expires_at, last_seen_at, revoked_at, revoke_reason
FROM authentication_sessions
WHERE id = $1 AND tenant_id = $2 AND revoked_at IS NULL AND expires_at > $3;

-- name: FindOwnedAuthenticationSession :one
-- revoked/expired を含む所有者確認用 lookup。self-service revoke の idempotency 判定に使う。
SELECT id, tenant_id, user_id, auth_time, amr, acr, authentication_pending,
       pending_purpose, enrollment_deadline, enrollment_bypass_id, step_up_at,
       expires_at, last_seen_at, revoked_at, revoke_reason
FROM authentication_sessions
WHERE id = $1 AND tenant_id = $2 AND user_id = $3;

-- name: RevokeAuthenticationSession :exec
-- revoked_at / revoke_reason は初回だけ確定する idempotent tombstone。
UPDATE authentication_sessions
SET revoked_at = COALESCE(revoked_at, $4),
    revoke_reason = COALESCE(revoke_reason, $3),
    updated_at = now()
WHERE id = $1 AND tenant_id = $2;

-- name: TouchAuthenticationSession :exec
-- LoginSessionTouchInterval 未満の再 touch は更新しない粗粒度な条件更新。
UPDATE authentication_sessions
SET last_seen_at = $3,
    updated_at = now()
WHERE id = $1 AND tenant_id = $2 AND last_seen_at < $4;

-- name: ListActiveAuthenticationSessionsByUser :many
-- keyset pagination を意識した index (tenant_id, user_id, auth_time DESC, id DESC) を使う。
-- 初期実装は先頭ページのみを返す (wi-253 Plan §2)。
SELECT id, tenant_id, user_id, auth_time, amr, acr, authentication_pending,
       pending_purpose, enrollment_deadline, enrollment_bypass_id, step_up_at,
       expires_at, last_seen_at, revoked_at, revoke_reason
FROM authentication_sessions
WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL
  AND expires_at > $3 AND authentication_pending = FALSE
ORDER BY auth_time DESC, id DESC
LIMIT $4;

-- name: DeleteAllAuthenticationSessionsForUser :exec
-- ADR-036 の anonymize cascade から呼ばれる物理削除 (tombstone ではなく erasure)。
DELETE FROM authentication_sessions WHERE tenant_id = $1 AND user_id = $2;

-- name: DeleteExpiredAuthenticationSessionsBatch :execrows
-- housekeeping cleanup。primary key を選んで小 batch で削除する (wi-253 Plan §7)。
DELETE FROM authentication_sessions AS outer_row
WHERE outer_row.id IN (
    SELECT s.id FROM authentication_sessions AS s WHERE s.expires_at < $1 LIMIT $2
);
