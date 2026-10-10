package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// EnqueueDeps are the dependencies for Enqueue.
type EnqueueDeps struct {
	Repo ports.JobRepository
	Emit func(spec.DomainEvent)
	// QuotaRepo enforces the tenant's Hard Quota on active_jobs (wi-160). nil skips enforcement (wiring gaps in tests/tools);
	// production bootstrap always sets it.
	QuotaRepo tenantports.QuotaRepository
}

// enqueuer は ports.Enqueuer を Enqueue で実装する。
type enqueuer struct{ deps EnqueueDeps }

// NewEnqueuer は、repo へジョブを作り、quotas で active_jobs を確認し、emit で JobEnqueued を発行する投入器を返す。
// 三つとも REQ-JOBS-002 と REQ-TENANCY-013 が投入のたびに求める作用なので、どれかが nil なら
// 組み立ての誤りとして panic する。
func NewEnqueuer(repo ports.JobRepository, quotas tenantports.QuotaRepository, emit func(spec.DomainEvent)) ports.Enqueuer {
	if repo == nil || quotas == nil || emit == nil {
		panic("jobs: NewEnqueuer requires a job repository, a quota repository, and an event sink")
	}
	return enqueuer{deps: EnqueueDeps{Repo: repo, QuotaRepo: quotas, Emit: emit}}
}

func (e enqueuer) Enqueue(ctx context.Context, input ports.EnqueueInput, now time.Time) (*domain.Job, error) {
	return Enqueue(ctx, e.deps, input, now)
}

// Enqueue validates input and inserts a new Job (EnqueueJob,
// docs/modules/jobs/). It rejects unregistered JobKinds, applies
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

	// Enqueue runs before the quota check (not after) because whether this
	// call actually creates a new Job is a JobHandlerIdempotency dedup
	// decision only the repository can make: a dedup hit must never consume
	// quota. A genuinely new Job that then fails the check is immediately
	// Canceled below so it doesn't linger as a counted-but-rejected row
	// (wi-160).
	job, created, err := deps.Repo.Enqueue(ctx, input)
	if err != nil {
		return nil, err
	}
	if !created {
		return job, nil
	}
	if deps.QuotaRepo != nil {
		quotaErr := deps.QuotaRepo.CheckAndIncrement(ctx, input.TenantID, tenancydomain.ResourceActiveJobs, 1)
		if qErr, ok := errors.AsType[*tenancydomain.QuotaExceededError](quotaErr); ok {
			emit(deps.Emit, &tenancydomain.QuotaExceeded{At: now, TenantID: input.TenantID, Resource: qErr.Resource, HardLimit: true})
			if _, cancelErr := deps.Repo.Cancel(ctx, job.ID, now); cancelErr != nil {
				logging.Error(ctx, "quota: failed to cancel job rejected by active_jobs quota", "error", cancelErr, "job_id", job.ID)
			}
		}
		if quotaErr != nil {
			return nil, quotaErr
		}
	}
	emit(deps.Emit, &domain.JobEnqueued{At: now, JobID: job.ID, TenantID: job.TenantID, Kind: job.Kind})
	return job, nil
}
