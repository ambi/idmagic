-- name: ListRelationTupleSubjects :many
SELECT subject_type,subject_id,subject_relation
FROM authorization_relation_tuples
WHERE tenant_id=$1 AND resource_type=$2 AND resource_id=$3 AND relation=$4
ORDER BY subject_type,subject_id,subject_relation;

-- name: ListRelationTuples :many
SELECT resource_type,resource_id,relation,subject_type,subject_id,subject_relation
FROM authorization_relation_tuples
WHERE tenant_id=$1
  AND ($2 = '' OR resource_type=$2)
  AND ($3 = '' OR resource_id=$3)
  AND ($4 = '' OR relation=$4)
  AND ($5 = '' OR subject_type=$5)
  AND ($6 = '' OR subject_id=$6)
ORDER BY resource_type,resource_id,relation,subject_type,subject_id,subject_relation
LIMIT $7;

-- name: ListRelationTupleResourceIDs :many
SELECT DISTINCT resource_id
FROM authorization_relation_tuples
WHERE tenant_id=$1 AND resource_type=$2
ORDER BY resource_id
LIMIT $3;

-- name: InsertRelationTuple :execrows
INSERT INTO authorization_relation_tuples
 (tenant_id,resource_type,resource_id,relation,subject_type,subject_id,subject_relation)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT DO NOTHING;

-- name: DeleteRelationTuple :execrows
DELETE FROM authorization_relation_tuples
WHERE tenant_id=$1 AND resource_type=$2 AND resource_id=$3 AND relation=$4
  AND subject_type=$5 AND subject_id=$6 AND subject_relation=$7;

-- name: DeleteRelationTuplesForObject :execrows
DELETE FROM authorization_relation_tuples
WHERE tenant_id=$1
  AND ((resource_type=$2 AND resource_id=$3) OR (subject_type=$2 AND subject_id=$3));

-- name: GetAuthorizationWriteVersion :one
SELECT version FROM authorization_write_versions WHERE tenant_id=$1;

-- name: BumpAuthorizationWriteVersion :one
INSERT INTO authorization_write_versions (tenant_id,version,updated_at)
VALUES ($1,1,now())
ON CONFLICT (tenant_id) DO UPDATE
 SET version=authorization_write_versions.version+1,updated_at=now()
RETURNING version;

-- name: InsertAuthorizationModel :one
INSERT INTO authorization_models (id,tenant_id,version,definition,created_at)
VALUES (
 $1,$2,
 (SELECT COALESCE(MAX(version),0)+1 FROM authorization_models WHERE tenant_id=$2),
 $3,$4
)
RETURNING id,tenant_id,version,definition,created_at;

-- name: GetLatestAuthorizationModel :one
SELECT id,tenant_id,version,definition,created_at
FROM authorization_models WHERE tenant_id=$1 ORDER BY version DESC LIMIT 1;

-- name: GetAuthorizationModelByVersion :one
SELECT id,tenant_id,version,definition,created_at
FROM authorization_models WHERE tenant_id=$1 AND version=$2;
