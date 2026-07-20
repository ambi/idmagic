-- name: ListApplicationAssignmentsByTenant :many
SELECT a.tenant_id, aa.application_id, aa.subject_type, aa.subject_id, aa.visibility, aa.created_at, aa.updated_at
FROM application_assignments aa JOIN applications a ON a.application_id = aa.application_id
WHERE a.tenant_id = $1;

-- name: ListApplicationAssignmentsByApplication :many
SELECT a.tenant_id, aa.application_id, aa.subject_type, aa.subject_id, aa.visibility, aa.created_at, aa.updated_at
FROM application_assignments aa JOIN applications a ON a.application_id = aa.application_id
WHERE a.tenant_id = $1 AND aa.application_id = $2
ORDER BY aa.subject_type, aa.subject_id;

-- ListApplicationAssignmentsBySubjects は (subject_type, subject_id) ペア配列との
-- UNNEST 突き合わせが必要で、sqlc の静的解析が UNNEST の引数型を解決できない
-- (動的クエリのエスケープハッチ、ADR-090)。手書き pgx として applications.go に残す。

-- name: UpsertApplicationAssignment :exec
INSERT INTO application_assignments (application_id, subject_type, subject_id, visibility, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (application_id, subject_type, subject_id) DO UPDATE SET
  visibility = EXCLUDED.visibility,
  updated_at = EXCLUDED.updated_at;

-- name: DeleteApplicationAssignment :exec
DELETE FROM application_assignments aa
WHERE aa.application_id = $2 AND aa.subject_type = $3 AND aa.subject_id = $4
  AND EXISTS (SELECT 1 FROM applications a WHERE a.tenant_id = $1 AND a.application_id = $2);

-- name: DeleteApplicationAssignmentsByApplication :exec
DELETE FROM application_assignments aa WHERE aa.application_id = $2
  AND EXISTS (SELECT 1 FROM applications a WHERE a.tenant_id = $1 AND a.application_id = $2);
