-- name: FindAuditExportArtifact :one
SELECT * FROM audit_export_artifacts WHERE tenant_id=$1 AND job_id=$2 AND published;

-- name: CreateAuditExportArtifact :exec
INSERT INTO audit_export_artifacts(id,job_id,tenant_id,events,byte_size,sha256,expires_at)
VALUES($1,$2,$3,$4,$5,$6,$7);

-- name: DeleteUnpublishedAuditExport :exec
DELETE FROM audit_export_artifacts WHERE id=$1 AND NOT published;

-- name: InsertAuditExportChunk :exec
INSERT INTO audit_export_chunks(artifact_id,chunk_number,payload) VALUES($1,$2,$3);

-- name: PublishAuditExportArtifact :exec
UPDATE audit_export_artifacts SET published=TRUE WHERE id=$1;

-- name: ExpireAuditExportArtifacts :many
DELETE FROM audit_export_artifacts WHERE id IN (
 SELECT expired.id FROM audit_export_artifacts AS expired WHERE expired.expires_at <= $1
 ORDER BY expired.expires_at,expired.id LIMIT 1000 FOR UPDATE SKIP LOCKED
) RETURNING *;

-- name: AuditExportArtifactUsage :one
SELECT COALESCE(SUM(byte_size),0)::bigint FROM audit_export_artifacts WHERE tenant_id=$1 AND published;

-- name: AuditExportArtifactTotalBytes :one
SELECT COALESCE(SUM(byte_size),0)::bigint FROM audit_export_artifacts WHERE published;

-- name: ReadAuditExportChunks :many
SELECT payload FROM audit_export_chunks WHERE artifact_id=$1 AND chunk_number >= $2 ORDER BY chunk_number LIMIT 16;
