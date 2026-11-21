-- name: CreateAccessReviewPlan :exec
INSERT INTO access_review_plans (id,tenant_id,enabled,next_start_at,data) VALUES ($1,$2,$3,$4,$5);

-- name: FindAccessReviewPlan :one
SELECT tenant_id,data FROM access_review_plans WHERE tenant_id=$1 AND id=$2;

-- name: LockAccessReviewPlan :one
SELECT tenant_id,data FROM access_review_plans WHERE tenant_id=$1 AND id=$2 FOR UPDATE;

-- name: SaveAccessReviewPlan :exec
UPDATE access_review_plans SET enabled=$3,next_start_at=$4,data=$5 WHERE tenant_id=$1 AND id=$2;

-- name: ListAccessReviewPlans :many
SELECT tenant_id,data FROM access_review_plans WHERE tenant_id=$1 ORDER BY id;

-- name: ReadyAccessReviewPlans :many
SELECT tenant_id,data FROM access_review_plans WHERE enabled AND next_start_at<=$1 ORDER BY next_start_at,id;

-- name: InsertAccessReview :exec
INSERT INTO access_reviews (id,tenant_id,plan_id,status,started_at,data) VALUES ($1,$2,$3,$4,$5,$6);

-- name: LockAccessReview :one
SELECT tenant_id,data FROM access_reviews WHERE tenant_id=$1 AND id=$2 FOR UPDATE;

-- name: SaveAccessReview :exec
UPDATE access_reviews SET status=$3,data=$4 WHERE tenant_id=$1 AND id=$2;

-- name: ListAccessReviews :many
SELECT tenant_id,data FROM access_reviews WHERE tenant_id=$1 AND plan_id=$2 ORDER BY started_at DESC,id;

-- name: ListReviewerAccessReviews :many
SELECT tenant_id,data FROM access_reviews WHERE tenant_id=$1 AND EXISTS (
    SELECT 1 FROM jsonb_array_elements(data->'review'->'items') AS item WHERE item->>'reviewer_id'=$2::text
) ORDER BY id;

-- name: ListActiveAccessReviews :many
SELECT tenant_id,data FROM access_reviews WHERE status<>'completed' ORDER BY started_at,id;
