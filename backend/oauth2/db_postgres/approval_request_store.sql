-- name: SaveApprovalRequest :exec
INSERT INTO oauth2_approval_requests (id, tenant_id, auth_req_id_hash, client_id, user_id, state, interval_seconds, last_polled_at, expires_at, payload)
VALUES (@id, @tenant_id, @auth_req_id_hash, @client_id, @user_id, @state, @interval_seconds, @last_polled_at, @expires_at, @payload);

-- name: FindApprovalRequestByID :one
SELECT payload, state, interval_seconds, last_polled_at FROM oauth2_approval_requests
WHERE id = @id AND tenant_id = @tenant_id;

-- name: FindApprovalRequestByAuthReqIDHash :one
SELECT payload, state, interval_seconds, last_polled_at FROM oauth2_approval_requests
WHERE auth_req_id_hash = @auth_req_id_hash AND tenant_id = @tenant_id;

-- name: ListPendingApprovalRequestsForUser :many
SELECT payload, state, interval_seconds, last_polled_at FROM oauth2_approval_requests
WHERE tenant_id = @tenant_id AND user_id = @user_id AND state = 'pending' AND expires_at > @now
ORDER BY created_at ASC;

-- name: RecordApprovalRequestPoll :one
WITH current_request AS (
    SELECT id,
        state = 'pending' AND last_polled_at IS NOT NULL
            AND @now < last_polled_at + make_interval(secs => interval_seconds) AS too_fast
    FROM oauth2_approval_requests
    WHERE oauth2_approval_requests.auth_req_id_hash = @auth_req_id_hash
      AND oauth2_approval_requests.tenant_id = @tenant_id
    FOR UPDATE
)
UPDATE oauth2_approval_requests AS request
SET interval_seconds = CASE
        WHEN current_request.too_fast THEN request.interval_seconds + 5
        ELSE request.interval_seconds
    END,
    last_polled_at = CASE WHEN request.state = 'pending' THEN @now ELSE request.last_polled_at END,
    updated_at = now()
FROM current_request
WHERE request.id = current_request.id
RETURNING request.payload, request.state, request.interval_seconds, request.last_polled_at,
    current_request.too_fast;

-- name: DecideApprovalRequest :one
UPDATE oauth2_approval_requests
SET state = @next_state,
    payload = jsonb_set(
        jsonb_set(payload, '{state}', to_jsonb(@next_state::text)),
        '{decided_at}', to_jsonb(@decided_at::timestamptz)
    ),
    updated_at = now()
WHERE id = @id AND tenant_id = @tenant_id AND user_id = @user_id
  AND state = 'pending' AND expires_at > @decided_at
RETURNING payload, state, interval_seconds, last_polled_at;

-- name: ExpireApprovalRequest :one
UPDATE oauth2_approval_requests
SET state = 'expired',
    payload = jsonb_set(payload, '{state}', '"expired"'::jsonb),
    updated_at = now()
WHERE auth_req_id_hash = @auth_req_id_hash AND tenant_id = @tenant_id
  AND state IN ('pending', 'approved') AND expires_at <= @expired_at
RETURNING payload, state, interval_seconds, last_polled_at;

-- name: ConsumeApprovalRequest :one
UPDATE oauth2_approval_requests
SET state = 'consumed',
    payload = jsonb_set(
        jsonb_set(payload, '{state}', '"consumed"'::jsonb),
        '{consumed_at}', to_jsonb(@consumed_at::timestamptz)
    ),
    updated_at = now()
WHERE auth_req_id_hash = @auth_req_id_hash AND tenant_id = @tenant_id
  AND state = 'approved' AND expires_at > @consumed_at
RETURNING payload, state, interval_seconds, last_polled_at;

-- name: DeleteApprovalRequestsForUser :exec
DELETE FROM oauth2_approval_requests WHERE tenant_id = @tenant_id AND user_id = @user_id;

-- name: DeleteExpiredApprovalRequestsBatch :execrows
DELETE FROM oauth2_approval_requests AS o
WHERE o.id IN (
    SELECT s.id FROM oauth2_approval_requests AS s WHERE s.expires_at < @cutoff LIMIT @batch_limit
);
