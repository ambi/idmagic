// Package postgres is the PostgreSQL JobRepository implementation (ADR-098):
// claim uses `FOR UPDATE SKIP LOCKED` inside a single atomic
// `UPDATE ... FROM (SELECT ... FOR UPDATE SKIP LOCKED) RETURNING` statement, so
// no explicit transaction is needed (there is no external side effect between
// claim and marking the Job Running, unlike
// backend/shared/adapters/eventsink/kafka_relay.go's Kafka-publish-in-the-middle
// case).
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ambi/idmagic/backend/jobs/adapters/persistence/postgres/sqlcgen"
	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// JobRepository persists Jobs to PostgreSQL. Pool is sqlcgen.DBTX-compatible
// (ADR-090); a fresh sqlcgen.Queries is created per call, matching the
// convention in backend/scim/adapters/persistence/postgres.
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

func jobFromRow(row *sqlcgen.Job) *domain.Job {
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

	row, err := sqlcgen.New(r.Pool).InsertJob(ctx, sqlcgen.InsertJobParams{
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
		existing, findErr := sqlcgen.New(r.Pool).FindActiveJobByDedupKey(ctx, sqlcgen.FindActiveJobByDedupKeyParams{
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
	rows, err := sqlcgen.New(r.Pool).ClaimJobs(ctx, sqlcgen.ClaimJobsParams{
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
	expiresAt, err := sqlcgen.New(r.Pool).HeartbeatJob(ctx, sqlcgen.HeartbeatJobParams{
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
	row, err := sqlcgen.New(r.Pool).CompleteJob(ctx, sqlcgen.CompleteJobParams{
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
	row, err := sqlcgen.New(r.Pool).FailJob(ctx, sqlcgen.FailJobParams{
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
	row, err := sqlcgen.New(r.Pool).CancelJob(ctx, sqlcgen.CancelJobParams{ID: jobID, UpdatedAt: now})
	if errors.Is(err, pgx.ErrNoRows) {
		if _, getErr := sqlcgen.New(r.Pool).GetJob(ctx, jobID); errors.Is(getErr, pgx.ErrNoRows) {
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
	rows, err := sqlcgen.New(r.Pool).LaneDepths(ctx)
	if err != nil {
		return nil, err
	}
	depths := make([]ports.LaneDepth, 0, len(rows))
	for _, row := range rows {
		depths = append(depths, ports.LaneDepth{Lane: domain.ExecutionLane(row.Lane), Queued: int(row.Queued), Running: int(row.Running)})
	}
	return depths, nil
}

func (r *JobRepository) Get(ctx context.Context, jobID string) (*domain.Job, error) {
	row, err := sqlcgen.New(r.Pool).GetJob(ctx, jobID)
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
	_, err := sqlcgen.New(r.Pool).GetJob(ctx, jobID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.ErrJobNotFound
	}
	if err != nil {
		return err
	}
	return ports.ErrJobLeaseLost
}
