-- name: UpsertClientSession :exec
INSERT INTO oauth2_client_sessions (
  tenant_id, sid, client_id, first_issued_at, last_issued_at
) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (tenant_id, sid, client_id) DO UPDATE SET
  last_issued_at = EXCLUDED.last_issued_at;

-- name: ListClientSessionsBySid :many
SELECT tenant_id, sid, client_id, first_issued_at, last_issued_at
FROM oauth2_client_sessions
WHERE tenant_id = $1 AND sid = $2
ORDER BY client_id;

-- name: SaveLogoutNotification :exec
INSERT INTO oauth2_logout_notifications (
    id, tenant_id, sid, client_id, logout_token_jti, target_uri, state,
    attempts, last_error, job_id, created_at, delivered_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (id) DO UPDATE SET
    state = EXCLUDED.state,
    attempts = EXCLUDED.attempts,
    last_error = EXCLUDED.last_error,
    job_id = EXCLUDED.job_id,
    delivered_at = EXCLUDED.delivered_at;

-- name: FindLogoutNotificationByID :one
SELECT id, tenant_id, sid, client_id, logout_token_jti, target_uri, state,
       attempts, last_error, job_id, created_at, delivered_at
FROM oauth2_logout_notifications
WHERE tenant_id = $1 AND id = $2;
