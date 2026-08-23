// Package postgres is the PostgreSQL JobRepository implementation:
// claim uses `FOR UPDATE SKIP LOCKED` inside a single atomic
// `UPDATE ... FROM (SELECT ... FOR UPDATE SKIP LOCKED) RETURNING` statement, so
// no explicit transaction is needed.
package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// JobRepository persists Jobs to PostgreSQL. Pool is DBTX-compatible
// ; a fresh Queries is created per call, matching the
// convention in backend/sourcing/scim/db_postgres.
type JobRepository struct{ Pool sharedpg.DB }

var _ ports.JobRepository = (*JobRepository)(nil)

func textOrNil(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func nilIfEmpty(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func timePtrOrNil(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	tt := t.Time
	return &tt
}

func jobFromRow(row *Job) *domain.Job {
	return &domain.Job{
		ID:             row.ID,
		TenantID:       row.TenantID,
		Kind:           domain.JobKind(row.Kind),
		Lane:           domain.ExecutionLane(row.Lane),
		Status:         domain.JobStatus(row.Status),
		Params:         json.RawMessage(row.Params),
		Result:         json.RawMessage(row.Result),
		Error:          nilIfEmpty(row.Error),
		Attempts:       int(row.Attempts),
		MaxAttempts:    int(row.MaxAttempts),
		DedupKey:       nilIfEmpty(row.DedupKey),
		LeaseOwner:     nilIfEmpty(row.LeaseOwner),
		LeaseExpiresAt: timePtrOrNil(row.LeaseExpiresAt),
		RunAt:          row.RunAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func (r *JobRepository) Enqueue(ctx context.Context, input ports.EnqueueInput) (*domain.Job, bool, error) {
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, false, err
	}
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	runAt := input.RunAt
	if runAt.IsZero() {
		runAt = now
	}
	dedup := textOrNil(input.DedupKey)

	row, err := New(r.Pool).InsertJob(ctx, InsertJobParams{
		ID:          id,
		TenantID:    input.TenantID,
		Kind:        string(input.Kind),
		Lane:        string(input.Lane),
		Params:      []byte(input.Params),
		MaxAttempts: int32(input.MaxAttempts), //nolint:gosec // G115: MaxAttempts is a small retry budget, well under int32 max
		DedupKey:    dedup,
		RunAt:       runAt,
		CreatedAt:   now,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// ON CONFLICT DO NOTHING: an active Job already holds this dedup key.
		existing, findErr := New(r.Pool).FindActiveJobByDedupKey(ctx, FindActiveJobByDedupKeyParams{
			TenantID: input.TenantID, DedupKey: dedup,
		})
		if findErr != nil {
			return nil, false, findErr
		}
		return jobFromRow(existing), false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return jobFromRow(row), true, nil
}

func (r *JobRepository) ClaimBatch(ctx context.Context, workerID string, lane domain.ExecutionLane, batchSize int, leaseDuration time.Duration, now time.Time) ([]*domain.Job, error) {
	if batchSize <= 0 {
		return nil, nil
	}
	rows, err := New(r.Pool).ClaimJobs(ctx, ClaimJobsParams{
		UpdatedAt:      now,
		Lane:           string(lane),
		Limit:          int32(batchSize), //nolint:gosec // G115: batchSize is a worker's concurrency slot count, well under int32 max
		LeaseOwner:     pgtype.Text{String: workerID, Valid: true},
		LeaseExpiresAt: pgtype.Timestamptz{Time: now.Add(leaseDuration), Valid: true},
	})
	if err != nil {
		return nil, err
	}
	claimed := make([]*domain.Job, 0, len(rows))
	for _, row := range rows {
		claimed = append(claimed, jobFromRow(row))
	}
	return claimed, nil
}

func (r *JobRepository) Heartbeat(ctx context.Context, jobID, workerID string, leaseDuration time.Duration, now time.Time) (time.Time, error) {
	expiresAt, err := New(r.Pool).HeartbeatJob(ctx, HeartbeatJobParams{
		ID:             jobID,
		LeaseOwner:     pgtype.Text{String: workerID, Valid: true},
		UpdatedAt:      now,
		LeaseExpiresAt: pgtype.Timestamptz{Time: now.Add(leaseDuration), Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, r.leaseLostOrNotFound(ctx, jobID)
	}
	if err != nil {
		return time.Time{}, err
	}
	return expiresAt.Time, nil
}

func (r *JobRepository) Complete(ctx context.Context, jobID, workerID string, result json.RawMessage, now time.Time) (*domain.Job, error) {
	row, err := New(r.Pool).CompleteJob(ctx, CompleteJobParams{
		ID:         jobID,
		LeaseOwner: pgtype.Text{String: workerID, Valid: true},
		UpdatedAt:  now,
		Result:     []byte(result),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, r.leaseLostOrNotFound(ctx, jobID)
	}
	if err != nil {
		return nil, err
	}
	return jobFromRow(row), nil
}

func (r *JobRepository) Fail(ctx context.Context, jobID, workerID string, outcome ports.FailOutcome, now time.Time) (*domain.Job, error) {
	row, err := New(r.Pool).FailJob(ctx, FailJobParams{
		ID:         jobID,
		LeaseOwner: pgtype.Text{String: workerID, Valid: true},
		UpdatedAt:  now,
		Status:     string(outcome.NextStatus),
		Error:      pgtype.Text{String: outcome.Error, Valid: true},
		RunAt:      outcome.RunAt,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, r.leaseLostOrNotFound(ctx, jobID)
	}
	if err != nil {
		return nil, err
	}
	return jobFromRow(row), nil
}

func (r *JobRepository) Cancel(ctx context.Context, jobID string, now time.Time) (*domain.Job, error) {
	row, err := New(r.Pool).CancelJob(ctx, CancelJobParams{ID: jobID, UpdatedAt: now})
	if errors.Is(err, pgx.ErrNoRows) {
		if _, getErr := New(r.Pool).GetJob(ctx, jobID); errors.Is(getErr, pgx.ErrNoRows) {
			return nil, ports.ErrJobNotFound
		}
		return nil, ports.ErrJobAlreadyTerminal
	}
	if err != nil {
		return nil, err
	}
	return jobFromRow(row), nil
}

func (r *JobRepository) LaneDepths(ctx context.Context) ([]ports.LaneDepth, error) {
	rows, err := New(r.Pool).LaneDepths(ctx)
	if err != nil {
		return nil, err
	}
	depths := make([]ports.LaneDepth, 0, len(rows))
	for _, row := range rows {
		depths = append(depths, ports.LaneDepth{Lane: domain.ExecutionLane(row.Lane), Queued: int(row.Queued), Running: int(row.Running)})
	}
	return depths, nil
}

func (r *JobRepository) ListByTenantAndKinds(ctx context.Context, tenantID string, kinds []domain.JobKind, limit int) ([]*domain.Job, error) {
	if len(kinds) == 0 {
		return nil, nil
	}
	kindStrs := make([]string, len(kinds))
	for i, k := range kinds {
		kindStrs[i] = string(k)
	}
	// NULLIF($3, 0) treats 0 as "no cap"; clamp out-of-range limits to 0 so an
	// overflowing int32 conversion can't silently flip the cap.
	limit32 := int32(0)
	if limit > 0 && limit <= math.MaxInt32 {
		limit32 = int32(limit)
	}
	rows, err := New(r.Pool).ListJobsByTenantAndKinds(ctx, ListJobsByTenantAndKindsParams{
		TenantID: tenantID,
		Column2:  kindStrs,
		Column3:  limit32,
	})
	if err != nil {
		return nil, err
	}
	jobs := make([]*domain.Job, len(rows))
	for i, row := range rows {
		jobs[i] = jobFromRow(row)
	}
	return jobs, nil
}

// adminJobColumns は ListForAdmin が読む列。jobFromRow が期待する並びと一致させる。
const adminJobColumns = `id, tenant_id, kind, lane, status, params, result, error, attempts, ` +
	`max_attempts, dedup_key, lease_owner, lease_expires_at, run_at, created_at, updated_at`

// ListForAdmin は管理コンソールの一覧を 1 ページ返す (wi-157)。
//
// 絞り込みが要求ごとに変わるため、この文だけは sqlc の静的な文にできない
// (docs/persistence.md の「問い合わせの構造が実行時まで決まらない場合」)。値はすべて
// プレースホルダーで渡し、SQL の文字列へは列名と演算子しか組み立てない。
func (r *JobRepository) ListForAdmin(ctx context.Context, filter ports.AdminJobFilter) ([]*domain.Job, error) {
	if filter.TenantID == "" && !filter.AllTenants {
		return nil, ports.ErrAdminJobFilterUnscoped
	}
	var conds []string
	var args []any
	add := func(expr string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(expr, len(args)))
	}
	if !filter.AllTenants {
		add("tenant_id = $%d", filter.TenantID)
	}
	if len(filter.Statuses) > 0 {
		statuses := make([]string, len(filter.Statuses))
		for i, s := range filter.Statuses {
			statuses[i] = string(s)
		}
		add("status = ANY($%d::text[])", statuses)
	}
	if len(filter.Kinds) > 0 {
		kinds := make([]string, len(filter.Kinds))
		for i, k := range filter.Kinds {
			kinds[i] = string(k)
		}
		add("kind = ANY($%d::text[])", kinds)
	}
	if filter.Lane != "" {
		add("lane = $%d", string(filter.Lane))
	}
	// キーセットの継続。id を組に含めるのは、同じ瞬間に投入された 2 件がページの
	// 境目で落ちたり重複したりしないようにするためである。
	if !filter.BeforeCreatedAt.IsZero() || filter.BeforeID != "" {
		args = append(args, filter.BeforeCreatedAt)
		createdIdx := len(args)
		args = append(args, filter.BeforeID)
		idIdx := len(args)
		conds = append(conds, fmt.Sprintf("(created_at, id) < ($%d, $%d)", createdIdx, idIdx))
	}

	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	limit := filter.Limit
	if limit <= 0 || limit > math.MaxInt32 {
		limit = defaultAdminJobPageSize
	}
	args = append(args, limit)

	query := "SELECT " + adminJobColumns + " FROM jobs" + where +
		" ORDER BY created_at DESC, id DESC" + fmt.Sprintf(" LIMIT $%d", len(args))
	rows, err := r.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*domain.Job
	for rows.Next() {
		var row Job
		if err := rows.Scan(&row.ID, &row.TenantID, &row.Kind, &row.Lane, &row.Status, &row.Params,
			&row.Result, &row.Error, &row.Attempts, &row.MaxAttempts, &row.DedupKey,
			&row.LeaseOwner, &row.LeaseExpiresAt, &row.RunAt, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, jobFromRow(&row))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return jobs, nil
}

// defaultAdminJobPageSize は Limit を与えられなかった場合の上限。usecases が既定を
// 決めるので通常は使われないが、無制限に読むより閉じた値へ倒す。
const defaultAdminJobPageSize = 50

func (r *JobRepository) Get(ctx context.Context, jobID string) (*domain.Job, error) {
	row, err := New(r.Pool).GetJob(ctx, jobID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ports.ErrJobNotFound
	}
	if err != nil {
		return nil, err
	}
	return jobFromRow(row), nil
}

// leaseLostOrNotFound distinguishes ErrJobNotFound from ErrJobLeaseLost after a
// conditional Heartbeat/Complete/Fail UPDATE affects zero rows: both look the
// same to the UPDATE (0 rows RETURNING), so a follow-up GetJob tells them apart.
func (r *JobRepository) leaseLostOrNotFound(ctx context.Context, jobID string) error {
	_, err := New(r.Pool).GetJob(ctx, jobID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.ErrJobNotFound
	}
	if err != nil {
		return err
	}
	return ports.ErrJobLeaseLost
}
