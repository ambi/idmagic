// Package ports declares the Jobs bounded context's abstraction over the durable
// job queue and worker leasing operations backing spec/contexts/jobs.yaml
// interfaces (EnqueueJob / ClaimJobs / HeartbeatJob / CompleteJob / FailJob /
// CancelJob). Implementations live in
// backend/jobs/adapters/persistence/{memory,postgres}.
package ports

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
)

// ErrJobNotFound is returned when a Job ID does not exist.
var ErrJobNotFound = errors.New("jobs: job not found")

// ErrJobLeaseLost is returned by Heartbeat, Complete, and Fail when the caller no
// longer holds jobID's lease (expired and reclaimed by another worker). Unlike the
// usecases-level sentinel errors most other contexts use, lease loss can only be
// detected atomically by the storage layer's conditional update (0 rows affected),
// so it is declared here at the ports level instead.
var ErrJobLeaseLost = errors.New("jobs: lease lost")

// ErrJobAlreadyTerminal is returned by Cancel when the Job already reached a
// terminal JobLifecycle state (JobTerminalStateIsImmutable).
var ErrJobAlreadyTerminal = errors.New("jobs: job already in a terminal state")

// EnqueueInput is the input to JobRepository.Enqueue.
type EnqueueInput struct {
	TenantID    string
	Kind        domain.JobKind
	Params      json.RawMessage
	DedupKey    *string
	MaxAttempts int
	RunAt       time.Time
	Now         time.Time
}

// FailOutcome is the caller's decision for how JobRepository.Fail resolves a
// reported handler failure: retry later (StatusQueued) or dead-letter now
// (StatusFailed). Usecases compute this from the Job's Attempts/MaxAttempts and
// the configured backoff (domain.NextRetryRunAt) before calling Fail, since that
// decision depends on runtime configuration the repository does not own.
type FailOutcome struct {
	NextStatus domain.JobStatus // StatusQueued (retry) or StatusFailed (dead-letter)
	RunAt      time.Time        // next claimable time; only meaningful when retrying
	Error      string
}

// JobRepository is the durable queue and worker leasing port for the Jobs bounded
// context. Implementations must uphold JobLeaseExclusivity, JobTenantIsolation,
// and JobTerminalStateIsImmutable (spec/contexts/jobs.yaml invariants).
type JobRepository interface {
	// Enqueue inserts a new StatusQueued Job. If input.DedupKey is set and a
	// non-terminal Job already exists for the same TenantID and DedupKey, that
	// existing Job is returned instead of creating a duplicate
	// (JobHandlerIdempotency).
	Enqueue(ctx context.Context, input EnqueueInput) (*domain.Job, error)

	// ClaimBatch atomically selects up to batchSize claimable Jobs (StatusQueued,
	// run_at <= now) and transitions them to StatusRunning under a lease held by
	// workerID until now+leaseDuration (JobLeaseExclusivity).
	ClaimBatch(ctx context.Context, workerID string, batchSize int, leaseDuration time.Duration, now time.Time) ([]*domain.Job, error)

	// Heartbeat extends jobID's lease for workerID and returns the new expiry.
	Heartbeat(ctx context.Context, jobID, workerID string, leaseDuration time.Duration, now time.Time) (time.Time, error)

	// Complete marks jobID StatusSucceeded with the given result.
	Complete(ctx context.Context, jobID, workerID string, result json.RawMessage, now time.Time) (*domain.Job, error)

	// Fail applies outcome (retry or dead-letter) to jobID.
	Fail(ctx context.Context, jobID, workerID string, outcome FailOutcome, now time.Time) (*domain.Job, error)

	// Cancel transitions jobID to StatusCanceled if it has not yet reached a
	// terminal state, otherwise returns ErrJobAlreadyTerminal.
	Cancel(ctx context.Context, jobID string, now time.Time) (*domain.Job, error)

	// Get returns jobID, or ErrJobNotFound.
	Get(ctx context.Context, jobID string) (*domain.Job, error)
}
