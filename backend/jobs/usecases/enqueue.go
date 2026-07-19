package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// EnqueueDeps are the dependencies for Enqueue.
type EnqueueDeps struct {
	Repo ports.JobRepository
	Emit func(spec.DomainEvent)
}

// Enqueue validates input and inserts a new Job (EnqueueJob,
// spec/contexts/jobs.yaml). It rejects unregistered JobKinds, applies
// domain.DefaultMaxAttempts when input.MaxAttempts is unset, defaults RunAt to
// now when unset, and emits JobEnqueued only when a new Job was actually
// created (not on a JobHandlerIdempotency dedup hit).
func Enqueue(ctx context.Context, deps EnqueueDeps, input ports.EnqueueInput, now time.Time) (*domain.Job, error) {
	lane, ok := domain.LaneFor(input.Kind)
	if !ok {
		return nil, fmt.Errorf("jobs: invalid job kind %q", input.Kind)
	}
	input.Lane = lane
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if input.MaxAttempts <= 0 {
		input.MaxAttempts = domain.DefaultMaxAttempts
	}
	if input.RunAt.IsZero() {
		input.RunAt = now
	}
	input.Now = now

	job, created, err := deps.Repo.Enqueue(ctx, input)
	if err != nil {
		return nil, err
	}
	if created {
		emit(deps.Emit, &domain.JobEnqueued{At: now, JobID: job.ID, TenantID: job.TenantID, Kind: job.Kind})
	}
	return job, nil
}
