package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// RunnerConfig holds the ADR-099 tunables for a worker's poll loop. Zero
// values fall back to the ADR-099 defaults (see withDefaults).
type RunnerConfig struct {
	WorkerID      string
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

func NewRunner(cfg RunnerConfig, deps RunnerDeps) *Runner {
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
	claimed, err := rn.deps.Repo.ClaimBatch(ctx, rn.cfg.WorkerID, available, rn.cfg.LeaseDuration, now)
	if err != nil {
		logging.Warn(ctx, "jobs: claim batch failed", "error", err)
		return
	}
	for _, job := range claimed {
		emit(rn.deps.Emit, &domain.JobStarted{
			At: now, JobID: job.ID, TenantID: job.TenantID,
			WorkerID: rn.cfg.WorkerID, Attempt: job.Attempts,
		})
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
	emit(rn.deps.Emit, &domain.JobSucceeded{At: now, JobID: updated.ID, TenantID: updated.TenantID})
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

	emit(rn.deps.Emit, &domain.JobFailed{
		At: now, JobID: updated.ID, TenantID: updated.TenantID,
		Attempt: job.Attempts, Terminal: terminal, Error: handlerErr.Error(),
	})
	if !terminal {
		emit(rn.deps.Emit, &domain.JobRetried{
			At: now, JobID: updated.ID, TenantID: updated.TenantID,
			Attempt: job.Attempts, NextRunAt: outcome.RunAt,
		})
	}
}
