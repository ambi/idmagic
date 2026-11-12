-- name: LockDirectoryPasswordQueue :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(queue_key)::text, 0));

-- name: CountDirectoryPasswordChallenges :one
SELECT count(*) FROM directory_password_challenges
WHERE tenant_id=sqlc.arg(tenant_id) AND connector_id=sqlc.arg(connector_id)
AND expires_at>sqlc.arg(now) AND result='';

-- name: EnqueueDirectoryPasswordChallenge :exec
INSERT INTO directory_password_challenges(id,tenant_id,connector_id,connector_revision,key_id,expires_at,encrypted_request)
VALUES($1,$2,$3,$4,$5,$6,$7);

-- name: ClaimDirectoryPasswordChallenge :one
WITH next AS MATERIALIZED (
    SELECT pending.id,pending.encrypted_request FROM directory_password_challenges AS pending
    WHERE pending.tenant_id=sqlc.arg(tenant_id) AND pending.connector_id=sqlc.arg(connector_id)
    AND pending.connector_revision=sqlc.arg(connector_revision) AND pending.expires_at>sqlc.arg(now) AND NOT pending.delivered
    ORDER BY pending.expires_at,pending.id FOR UPDATE SKIP LOCKED LIMIT 1
)
UPDATE directory_password_challenges AS q SET delivered=TRUE,encrypted_request=''
FROM next WHERE q.id=next.id
RETURNING q.id,q.tenant_id,q.connector_id,q.connector_revision,q.key_id,q.expires_at,next.encrypted_request;

-- name: CompleteDirectoryPasswordChallenge :execrows
UPDATE directory_password_challenges SET result=sqlc.arg(result)
WHERE tenant_id=sqlc.arg(tenant_id) AND connector_id=sqlc.arg(connector_id)
AND id=sqlc.arg(id) AND connector_revision=sqlc.arg(connector_revision)
AND delivered AND expires_at>sqlc.arg(now) AND (result='' OR result=sqlc.arg(result));

-- name: FindDirectoryPasswordChallenge :one
SELECT * FROM directory_password_challenges WHERE tenant_id=$1 AND id=$2;

-- name: DeleteDirectoryPasswordChallenge :exec
DELETE FROM directory_password_challenges WHERE tenant_id=$1 AND id=$2;

-- name: SweepDirectoryPasswordChallenges :exec
DELETE FROM directory_password_challenges WHERE expires_at<=$1;

-- name: FindDirectoryUserCorrelationByUser :one
SELECT * FROM directory_user_correlations WHERE tenant_id=$1 AND user_id=$2;
