-- name: InsertCSVArtifact :exec
INSERT INTO csv_artifacts (id, tenant_id, sha256, byte_size) VALUES ($1, $2, $3, $4);

-- name: InsertCSVArtifactChunk :exec
INSERT INTO csv_artifact_chunks (artifact_id, chunk_number, payload) VALUES ($1, $2, $3);

-- name: UpdateCSVArtifactDigest :exec
UPDATE csv_artifacts SET sha256 = $2, byte_size = $3 WHERE id = $1;

-- name: FindCSVArtifact :one
SELECT sha256, byte_size FROM csv_artifacts WHERE tenant_id = $1 AND id = $2;

-- name: ListCSVArtifactChunkPayloadsFrom :many
SELECT payload FROM csv_artifact_chunks
WHERE artifact_id = sqlc.arg(artifact_id) AND chunk_number >= sqlc.arg(from_chunk_number)
ORDER BY chunk_number
LIMIT sqlc.arg(page_limit);

-- name: FindCSVArtifactPage :one
SELECT a.sha256, a.byte_size, c.payload FROM csv_artifacts a
JOIN csv_artifact_chunks c ON c.artifact_id = a.id
WHERE a.tenant_id = $1 AND a.id = $2 AND c.chunk_number = $3;
