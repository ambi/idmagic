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
ORDER BY window_start DESC, kind DESC, key_hash DESC
LIMIT $2;

-- name: ListAuthEventBucketsAfter :many
-- Continuation page for ListAuthenticationEventBuckets keyset pagination
-- (wi-159, ADR-158): resumes strictly after the (window_start, kind,
-- key_hash) keyset of the last row the caller saw. kind/key_hash (not
-- count, which isn't unique) is the tie-break matching the table's own
-- PRIMARY KEY (tenant_id, kind, key_hash, window_start); all three columns
-- sort DESC so the row-value comparison below matches ORDER BY exactly.
SELECT tenant_id, kind, key_hash, window_start, count, first_seen, last_seen
FROM authentication_event_buckets
WHERE ($1::text = '' OR tenant_id = $1::text)
  AND (window_start, kind, key_hash) < (sqlc.arg(after_window_start)::timestamptz, sqlc.arg(after_kind)::text, sqlc.arg(after_key_hash)::text)
ORDER BY window_start DESC, kind DESC, key_hash DESC
LIMIT sqlc.arg(page_limit);

-- name: ListAuthEventBucketsBefore :many
SELECT tenant_id, kind, key_hash, window_start, count, first_seen, last_seen
FROM authentication_event_buckets
WHERE ($1::text = '' OR tenant_id = $1::text)
  AND (window_start, kind, key_hash) > (sqlc.arg(before_window_start)::timestamptz, sqlc.arg(before_kind)::text, sqlc.arg(before_key_hash)::text)
ORDER BY window_start ASC, kind ASC, key_hash ASC
LIMIT sqlc.arg(page_limit);
