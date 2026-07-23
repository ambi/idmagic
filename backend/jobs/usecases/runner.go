package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
)

// RunnerConfig holds the ADR-099 tunables for a worker's poll loop, plus the
// ADR-129 Lane it claims. Zero values fall back to the ADR-099 defaults (see
// withDefaults); Lane has no default and must be a valid domain.ExecutionLane
// (NewRunner panics otherwise, a startup-time programmer error).
type RunnerConfig struct {
	WorkerID      string
	Lane          domain.ExecutionLane
	PollInterval  time.Duration
	Concurrency   int
	LeaseDuration time.Duration
	BackoffBase   time.Duration
	BackoffCap    time.Duration
}

func (c RunnerConfig) withDefaults() RunnerConfig {
	if c.PollInterval <= 0 {
		c.PollInterval = 2 * time.Second
	}
	if c.Concurrency <= 0 {
		c.Concurrency = 4
	}
	if c.LeaseDuration <= 0 {
		c.LeaseDuration = 5 * time.Minute
	}
	if c.BackoffBase <= 0 {
		c.BackoffBase = domain.DefaultBackoffBase
	}
	if c.BackoffCap <= 0 {
		c.BackoffCap = domain.DefaultBackoffCap
	}
	return c
}

// RunnerDeps are the dependencies for Runner.
type RunnerDeps struct {
	Repo     ports.JobRepository
	Handlers *HandlerRegistry
	Emit     func(spec.DomainEvent)
	// Now returns the current time; defaults to time.Now().UTC() when nil.
	// Tests inject a fixed/controllable clock.
	Now func() time.Time
	// Metrics records lane-scoped observability signals (wi-261 T006). Nil is
	// valid and records nothing.
	Metrics JobsMetrics
	// QuotaRepo frees the tenant's active_jobs Hard Quota slot when a Job
	// reaches a terminal state (wi-160, ADR-134). The increment side lives in
	// Enqueue. nil skips enforcement (wiring gaps in tests/tools); production
	// bootstrap always sets it.
	QuotaRepo tenantports.QuotaRepository
}

// decrementActiveJobsQuota frees one active_jobs quota slot for tenantID.
// nil QuotaRepo is a no-op; a decrement failure is logged, not propagated,
// since the Job's own terminal state transition has already committed by the
// time this runs (mirrors this package's existing "log, don't fail an
// already-committed step" treatment of side-channel errors).
func decrementActiveJobsQuota(ctx context.Context, quotaRepo tenantports.QuotaRepository, tenantID string) {
	if quotaRepo == nil {
		return
	}
	if err := tenancyusecases.DecrementQuota(ctx, quotaRepo, tenantID, tenancydomain.ResourceActiveJobs, 1); err != nil {
		logging.Error(ctx, "quota: failed to decrement active_jobs on job completion", "error", err, "tenant_id", tenantID)
	}
}

// Runner is the worker pool (ADR-099): it polls JobRepository for claimable
// Jobs, executes up to Concurrency of them at once, heartbeats in-flight
// leases, and applies retry-with-backoff or dead-letter on failure.
type Runner struct {
	cfg  RunnerConfig
	deps RunnerDeps
	sem  chan struct{}
	wg   sync.WaitGroup
}

// NewRunner panics if cfg.Lane is not a valid domain.ExecutionLane: a Runner
// with no lane, or an unrecognized one, can never claim anything and is a
// worker misconfiguration caught at startup (ADR-129), matching
// HandlerRegistry.Register's panic-on-invalid-JobKind precedent.
func NewRunner(cfg RunnerConfig, deps RunnerDeps) *Runner {
	if !cfg.Lane.Valid() {
		panic(fmt.Sprintf("jobs: cannot start Runner with invalid ExecutionLane %q", cfg.Lane))
	}
	cfg = cfg.withDefaults()
	if deps.Now == nil {
		deps.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Runner{cfg: cfg, deps: deps, sem: make(chan struct{}, cfg.Concurrency)}
}

// Run polls until ctx is canceled, then stops claiming new Jobs and blocks
// until all in-flight executions finish. This is the drain half of ADR-099:
// the caller (cmd/idmagic-worker) decides how long to wait for Run to return
// before giving up and letting the process exit, at which point any
// still-unfinished Job's lease expires naturally and another worker reclaims
// it (JobLeaseExclusivity survives a hard kill).
func (rn *Runner) Run(ctx context.Context) error {
	ticker := time.NewTicker(rn.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			rn.wg.Wait()
			return ctx.Err()
		case <-ticker.C:
			rn.poll(ctx)
		}
	}
}

func (rn *Runner) poll(ctx context.Context) {
	available := cap(rn.sem) - len(rn.sem)
	if available <= 0 {
		return
	}
	now := rn.deps.Now()
	claimed, err := rn.deps.Repo.ClaimBatch(ctx, rn.cfg.WorkerID, rn.cfg.Lane, available, rn.cfg.LeaseDuration, now)
	if err != nil {
		logging.Warn(ctx, "jobs: claim batch failed", "error", err)
		return
	}
	for _, job := range claimed {
		emit(rn.deps.Emit, &domain.JobStarted{
			At: now, JobID: job.ID, TenantID: job.TenantID,
			WorkerID: rn.cfg.WorkerID, Attempt: job.Attempts,
		})
		if latency := now.Sub(job.RunAt); latency > 0 {
			rn.jobsMetrics().RecordJobClaimLatency(rn.cfg.Lane, latency)
		}
		rn.sem <- struct{}{}
		rn.wg.Add(1)
		// Deliberately context.Background() inside execute, not ctx:
		// in-flight work must not be aborted the moment drain begins. See
		// Run's doc comment for how the process eventually gives up.
		go func(job *domain.Job) { //nolint:gosec,contextcheck // G118: detached on purpose so drain doesn't abort in-flight jobs
			defer rn.wg.Done()
			defer func() { <-rn.sem }()
			rn.execute(context.Background(), job)
		}(job)
	}
}

func (rn *Runner) execute(ctx context.Context, job *domain.Job) {
	execCtx, stopHeartbeat := context.WithCancel(ctx)
	defer stopHeartbeat()
	go rn.heartbeatLoop(execCtx, job)

	handler, err := rn.deps.Handlers.Lookup(job.Kind)
	if err != nil {
		rn.fail(execCtx, job, err)
		return
	}
	result, err := handler(execCtx, job)
	if err != nil {
		rn.fail(execCtx, job, err)
		return
	}
	rn.complete(execCtx, job, result)
}

func (rn *Runner) heartbeatLoop(ctx context.Context, job *domain.Job) {
	interval := rn.cfg.LeaseDuration / 3
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := rn.deps.Repo.Heartbeat(ctx, job.ID, rn.cfg.WorkerID, rn.cfg.LeaseDuration, rn.deps.Now()); err != nil {
				logging.Warn(ctx, "jobs: heartbeat failed, lease likely lost", "job_id", job.ID, "error", err)
				return
			}
		}
	}
}

func (rn *Runner) complete(ctx context.Context, job *domain.Job, result json.RawMessage) {
	now := rn.deps.Now()
	updated, err := rn.deps.Repo.Complete(ctx, job.ID, rn.cfg.WorkerID, result, now)
	if err != nil {
		if !errors.Is(err, ports.ErrJobLeaseLost) {
			logging.Warn(ctx, "jobs: complete failed", "job_id", job.ID, "error", err)
		}
		return
	}
	decrementActiveJobsQuota(ctx, rn.deps.QuotaRepo, updated.TenantID)
	emit(rn.deps.Emit, &domain.JobSucceeded{At: now, JobID: updated.ID, TenantID: updated.TenantID})
	rn.jobsMetrics().RecordJobOutcome(rn.cfg.Lane, "succeeded")
}

// fail decides retry vs. dead-letter (JobDeadLetterOnMaxAttempts) from the
// Job's Attempts as of the claim that is failing, computes backoff via
// domain.NextRetryRunAt when retrying, and emits JobFailed (always) plus
// JobRetried (only when a retry was scheduled).
func (rn *Runner) fail(ctx context.Context, job *domain.Job, handlerErr error) {
	now := rn.deps.Now()
	terminal := job.Attempts >= job.MaxAttempts
	outcome := ports.FailOutcome{Error: handlerErr.Error()}
	if terminal {
		outcome.NextStatus = domain.StatusFailed
	} else {
		outcome.NextStatus = domain.StatusQueued
		outcome.RunAt = domain.NextRetryRunAt(now, job.Attempts, rn.cfg.BackoffBase, rn.cfg.BackoffCap)
	}

	updated, err := rn.deps.Repo.Fail(ctx, job.ID, rn.cfg.WorkerID, outcome, now)
	if err != nil {
		if !errors.Is(err, ports.ErrJobLeaseLost) {
			logging.Warn(ctx, "jobs: fail failed", "job_id", job.ID, "error", err)
		}
		return
	}

	if terminal {
		decrementActiveJobsQuota(ctx, rn.deps.QuotaRepo, updated.TenantID)
	}
	emit(rn.deps.Emit, &domain.JobFailed{
		At: now, JobID: updated.ID, TenantID: updated.TenantID,
		Attempt: job.Attempts, Terminal: terminal, Error: handlerErr.Error(),
	})
	if !terminal {
		emit(rn.deps.Emit, &domain.JobRetried{
			At: now, JobID: updated.ID, TenantID: updated.TenantID,
			Attempt: job.Attempts, NextRunAt: outcome.RunAt,
		})
		rn.jobsMetrics().RecordJobRetry(rn.cfg.Lane)
	} else {
		rn.jobsMetrics().RecordJobOutcome(rn.cfg.Lane, "failed")
	}
}
