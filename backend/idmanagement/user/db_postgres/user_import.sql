-- name: InsertImportedPasswordHistory :exec
-- CSV インポートの行の確定と同じトランザクションで、所有 Context の外のテーブルへ書く。
INSERT INTO password_history (id, user_id, encoded, created_at) VALUES ($1, $2, $3, $4);

-- name: InsertUserImportAuditEvent :exec
INSERT INTO audit_events (id, tenant_id, type, user_id, occurred_at, payload)
VALUES (sqlc.arg(id), sqlc.arg(tenant_id), sqlc.arg(type), sqlc.arg(user_id)::uuid, sqlc.arg(occurred_at), sqlc.arg(payload));
