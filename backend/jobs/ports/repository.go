// Package ports declares the Jobs bounded context's abstraction over the durable
// job queue and worker leasing operations backing spec/contexts/jobs.yaml
// interfaces (EnqueueJob / ClaimJobs / HeartbeatJob / CompleteJob / FailJob /
// CancelJob). Implementations live in
// backend/jobs/{db_memory,db_postgres}.
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
	TenantID string
	Kind     domain.JobKind
	// Lane is derived by usecases.Enqueue from Kind's registered
	// domain.ExecutionLane; callers of the usecase cannot set it
	// directly. The repository just persists what it is given.
	Lane        domain.ExecutionLane
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
	// Enqueue inserts a new StatusQueued Job and reports created=true. If
	// input.DedupKey is set and a non-terminal Job already exists for the same
	// TenantID and DedupKey, that existing Job is returned instead with
	// created=false rather than creating a duplicate (JobHandlerIdempotency).
	Enqueue(ctx context.Context, input EnqueueInput) (job *domain.Job, created bool, err error)

	// ClaimBatch atomically selects up to batchSize claimable Jobs within lane
	// and places them under a lease held by workerID until now+leaseDuration
	// (JobLeaseExclusivity). Claimable means either StatusQueued with
	// run_at <= now (transitions to StatusRunning), or already StatusRunning
	// with an expired lease from a crashed/drained worker (status unchanged,
	// re-leased to workerID). Jobs outside lane are never selected (
	// lane isolation). Each claimed Job's Attempts is incremented by 1 as part
	// of the claim, so the returned Job's Attempts is the attempt number now
	// starting (JobStarted's Attempt payload).
	ClaimBatch(ctx context.Context, workerID string, lane domain.ExecutionLane, batchSize int, leaseDuration time.Duration, now time.Time) ([]*domain.Job, error)

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

	// LaneDepths returns the current StatusQueued and StatusRunning row count
	// for every lane that has at least one such row (a lane with zero of both
	// is simply absent, not a zero-valued entry). It is a point-in-time
	// snapshot for the wi-261 T006 queue-depth/active gauges, not part of the
	// claim/lease contract above.
	LaneDepths(ctx context.Context) ([]LaneDepth, error)

	// ListByTenantAndKinds returns the tenant's Jobs whose Kind is in kinds,
	// newest-created first, capped at limit (limit <= 0 means no cap). It is a
	// scoped read for feature consumers that surface their own kind's Jobs to
	// admins (wi-148 DataExport listing). Passing no kinds returns no rows.
	ListByTenantAndKinds(ctx context.Context, tenantID string, kinds []domain.JobKind, limit int) ([]*domain.Job, error)

	// ListForAdmin returns one keyset page of the cross-kind administration
	// listing (wi-157), newest-created first. It is the console's read: the
	// filter is assembled at request time from optional status/kind/lane
	// narrowing, so unlike every other query here it cannot be a static
	// statement. Tenant scoping is part of the filter and is decided by the
	// caller's authorization, never by a request parameter.
	ListForAdmin(ctx context.Context, filter AdminJobFilter) ([]*domain.Job, error)
}

// AdminJobFilter narrows ListForAdmin. The zero value would list every tenant's
// Jobs, so a caller must set either TenantID or AllTenants deliberately;
// implementations reject a filter that sets neither rather than reading across
// tenants by omission (JobTenantIsolation).
type AdminJobFilter struct {
	// TenantID confines the listing to one tenant. Ignored when AllTenants.
	TenantID string
	// AllTenants lifts the tenant confinement. Only a system_admin calling
	// through the control-plane tenant may set it (REQ-JOBS-012); the
	// repository takes it as already-authorized.
	AllTenants bool
	// Statuses, when non-empty, keeps only Jobs in one of these states.
	Statuses []domain.JobStatus
	// Kinds, when non-empty, keeps only Jobs of one of these kinds.
	Kinds []domain.JobKind
	// Lane, when non-empty, keeps only Jobs on that lane.
	Lane domain.ExecutionLane
	// Limit caps the page (<= 0 means the implementation's default page size).
	Limit int
	// BeforeCreatedAt and BeforeID continue a previous page: return only rows
	// ordering strictly after (created_at DESC, id DESC). Zero values start
	// from the newest row.
	BeforeCreatedAt time.Time
	BeforeID        string
}

// ErrAdminJobFilterUnscoped is returned by ListForAdmin when the filter names
// neither a tenant nor an authorized cross-tenant read. Listing every tenant
// because a field was left empty is exactly the failure JobTenantIsolation
// exists to prevent, so it is an error rather than a default.
var ErrAdminJobFilterUnscoped = errors.New("jobs: admin job filter must set TenantID or AllTenants")

// LaneDepth is one ExecutionLane's current queue depth (wi-261 T006).
type LaneDepth struct {
	Lane    domain.ExecutionLane
	Queued  int
	Running int
}
