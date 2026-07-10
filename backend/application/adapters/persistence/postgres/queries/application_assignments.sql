-- name: ListApplicationAssignmentsByTenant :many
SELECT tenant_id, application_id, subject_type, subject_id, visibility, created_at, updated_at
FROM application_assignments
WHERE tenant_id = $1;

-- name: ListApplicationAssignmentsByApplication :many
SELECT tenant_id, application_id, subject_type, subject_id, visibility, created_at, updated_at
FROM application_assignments
WHERE tenant_id = $1 AND application_id = $2
ORDER BY subject_type, subject_id;

-- ListApplicationAssignmentsBySubjects は (subject_type, subject_id) ペア配列との
-- UNNEST 突き合わせが必要で、sqlc の静的解析が UNNEST の引数型を解決できない
-- (動的クエリのエスケープハッチ、ADR-090)。手書き pgx として applications.go に残す。

-- name: UpsertApplicationAssignment :exec
INSERT INTO application_assignments (tenant_id, application_id, subject_type, subject_id, visibility, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (tenant_id, application_id, subject_type, subject_id) DO UPDATE SET
  visibility = EXCLUDED.visibility,
  updated_at = EXCLUDED.updated_at;

-- name: DeleteApplicationAssignment :exec
DELETE FROM application_assignments
WHERE tenant_id = $1 AND application_id = $2 AND subject_type = $3 AND subject_id = $4;

-- name: DeleteApplicationAssignmentsByApplication :exec
DELETE FROM application_assignments WHERE tenant_id = $1 AND application_id = $2;
