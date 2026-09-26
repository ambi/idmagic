-- name: InsertGroupImportAuditEvent :exec
-- CSV インポートの行の確定と同じトランザクションで、所有 Context の外のテーブルへ書く。
INSERT INTO audit_events (id, tenant_id, type, user_id, occurred_at, payload)
VALUES (sqlc.arg(id), sqlc.arg(tenant_id), sqlc.arg(type), sqlc.arg(user_id)::uuid, sqlc.arg(occurred_at), sqlc.arg(payload));

-- name: EnqueueGroupReconcileJob :exec
-- dedup_key は Jobs の部分一意索引がそのまま効く。
INSERT INTO jobs (id, tenant_id, kind, lane, status, params, attempts, max_attempts, dedup_key, run_at, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(tenant_id), sqlc.arg(kind), sqlc.arg(lane), 'queued', sqlc.arg(params), 0, sqlc.arg(max_attempts),
  sqlc.arg(dedup_key)::text, sqlc.arg(now), sqlc.arg(now), sqlc.arg(now))
ON CONFLICT (tenant_id, dedup_key) WHERE dedup_key IS NOT NULL AND status IN ('queued', 'running')
DO NOTHING;
