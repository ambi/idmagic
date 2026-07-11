-- name: RecordAuthEventBucket :one
INSERT INTO authentication_event_buckets (tenant_id, kind, key_hash, window_start, count, first_seen, last_seen)
VALUES ($1, $2, $3, $4, 1, $5, $5)
ON CONFLICT (tenant_id, kind, key_hash, window_start)
DO UPDATE SET count = authentication_event_buckets.count + 1, last_seen = EXCLUDED.last_seen, updated_at = now()
RETURNING count, first_seen, last_seen, (xmax = 0) AS inserted;

-- name: DeleteAuthEventBucketsOlderThan :execrows
DELETE FROM authentication_event_buckets WHERE window_start < $1;

-- name: ListAuthEventBuckets :many
SELECT tenant_id, kind, key_hash, window_start, count, first_seen, last_seen
FROM authentication_event_buckets
WHERE ($1::text = '' OR tenant_id = $1::text)
ORDER BY window_start DESC, count DESC
LIMIT $2;
