// Package domain implements the Jobs bounded context business types: the Job
// entity, its JobLifecycle state machine, and retry backoff (spec/contexts/jobs.yaml).
package domain

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// JobStatus is a JobLifecycle state (spec/contexts/jobs.yaml states.JobLifecycle).
type JobStatus string

const (
	StatusQueued    JobStatus = "queued"
	StatusRunning   JobStatus = "running"
	StatusSucceeded JobStatus = "succeeded"
	StatusFailed    JobStatus = "failed"
	StatusCanceled  JobStatus = "canceled"
)

func (s JobStatus) Valid() bool {
	switch s {
	case StatusQueued, StatusRunning, StatusSucceeded, StatusFailed, StatusCanceled:
		return true
	}
	return false
}

// JobKind identifies which worker handler processes a Job (spec/contexts/jobs.yaml
// models.JobKind). Adding a new kind requires registering it in
// spec/contexts/jobs.yaml first (SCL-first) before a consumer WI implements the
// handler.
type JobKind string

const (
	// KindNoopEcho is the wi-126 core-runtime smoke-test job kind.
	KindNoopEcho              JobKind = "noop_echo"
	KindUserImportPreview     JobKind = "user_import_preview"
	KindUserImportApply       JobKind = "user_import_apply"
	KindDynamicGroupReconcile JobKind = "dynamic_group_reconcile"
)

// ExecutionLane is the ADR-129 execution-lane vocabulary
// (spec/contexts/jobs.yaml models.ExecutionLane). A JobKind is assigned
// exactly one lane at registration; enqueue callers cannot choose it.
type ExecutionLane string

const (
	LaneLatencySensitive ExecutionLane = "latency_sensitive"
	LaneDefault          ExecutionLane = "default"
	LaneBulk             ExecutionLane = "bulk"
)

func (l ExecutionLane) Valid() bool {
	switch l {
	case LaneLatencySensitive, LaneDefault, LaneBulk:
		return true
	}
	return false
}

// kindLanes holds every registered JobKind's ExecutionLane, both the
// built-in kinds (registered by this package's init below) and
// extension kinds (registered by the owning bounded context via
// RegisterKind). A JobKind is Valid() only once it has a registered lane
// (ADR-129): "未割当の JobKind は拒否する" is enforced by construction rather
// than as a separate check.
var kindLanes sync.Map // map[JobKind]ExecutionLane

func init() {
	RegisterKind(KindNoopEcho, LaneDefault)
	RegisterKind(KindUserImportPreview, LaneBulk)
	RegisterKind(KindUserImportApply, LaneBulk)
	RegisterKind(KindDynamicGroupReconcile, LaneBulk)
}

// RegisterKind declares a context-owned job kind and the ExecutionLane it
// runs in (ADR-129). The Jobs context owns the durable queue and lane
// vocabulary, while the caller owns the handler vocabulary; this prevents
// feature-specific kinds from accumulating in Jobs. RegisterKind panics if
// lane is not a valid ExecutionLane, or if kind was already registered with a
// different lane (a programmer error caught at startup); re-registering the
// same kind with the same lane is idempotent.
func RegisterKind(kind JobKind, lane ExecutionLane) {
	if kind == "" {
		return
	}
	if !lane.Valid() {
		panic(fmt.Sprintf("jobs: cannot register JobKind %q with invalid ExecutionLane %q", kind, lane))
	}
	if existing, loaded := kindLanes.LoadOrStore(kind, lane); loaded && existing != lane {
		panic(fmt.Sprintf("jobs: JobKind %q already registered with ExecutionLane %q, cannot re-register with %q", kind, existing, lane))
	}
}

// LaneFor returns the ExecutionLane registered for kind, or
// (\"\", false) if kind has no registration.
func LaneFor(kind JobKind) (ExecutionLane, bool) {
	v, ok := kindLanes.Load(kind)
	if !ok {
		return "", false
	}
	lane, ok := v.(ExecutionLane)
	return lane, ok
}

// Valid reports whether kind has been registered with an ExecutionLane
// (ADR-129: every valid JobKind has exactly one lane).
func (k JobKind) Valid() bool {
	_, ok := LaneFor(k)
	return ok
}

// JobLifecycleEvent is a JobLifecycle state machine event.
type JobLifecycleEvent string

const (
	EventClaim    JobLifecycleEvent = "Claim"
	EventComplete JobLifecycleEvent = "Complete"
	EventFail     JobLifecycleEvent = "Fail"
	EventRetry    JobLifecycleEvent = "Retry"
	EventCancel   JobLifecycleEvent = "Cancel"
)

type jobTransition struct {
	From  JobStatus
	Event JobLifecycleEvent
	To    JobStatus
}

// jobTransitions は SCL の states.JobLifecycle.transitions と一致させる。
var jobTransitions = []jobTransition{
	{StatusQueued, EventClaim, StatusRunning},
	{StatusRunning, EventComplete, StatusSucceeded},
	{StatusRunning, EventFail, StatusFailed},
	{StatusRunning, EventRetry, StatusQueued},
	{StatusQueued, EventCancel, StatusCanceled},
	{StatusRunning, EventCancel, StatusCanceled},
}

// TransitionJobLifecycle applies event to from and returns the resulting status,
// or an error if the transition is not declared in spec/contexts/jobs.yaml
// states.JobLifecycle.
func TransitionJobLifecycle(from JobStatus, event JobLifecycleEvent) (JobStatus, error) {
	for _, t := range jobTransitions {
		if t.From == from && t.Event == event {
			return t.To, nil
		}
	}
	return "", fmt.Errorf("jobs: no transition from %q on event %q", from, event)
}

// IsJobLifecycleTerminal reports whether s is one of JobLifecycle's terminal
// states (JobTerminalStateIsImmutable).
func IsJobLifecycleTerminal(s JobStatus) bool {
	switch s {
	case StatusSucceeded, StatusFailed, StatusCanceled:
		return true
	}
	return false
}

// DefaultBackoffBase and DefaultBackoffCap are the ADR-099 default retry backoff
// parameters: exponential starting at 30s, capped at 30 minutes.
const (
	DefaultBackoffBase = 30 * time.Second
	DefaultBackoffCap  = 30 * time.Minute
)

// DefaultMaxAttempts is the ADR-099 default attempt budget applied when
// EnqueueJob does not specify one; a JobKind's handler registration may
// override it per kind.
const DefaultMaxAttempts = 5

// NextRetryRunAt computes the run_at for a Job returned to Queued via EventRetry,
// using exponential backoff (ADR-099): base * 2^(attempts-1), capped at maxBackoff.
// attempts is the Job's attempt count after the failure being retried (>= 1).
func NextRetryRunAt(now time.Time, attempts int, base, maxBackoff time.Duration) time.Time {
	if attempts < 1 {
		attempts = 1
	}
	delay := base
	for i := 1; i < attempts; i++ {
		if delay >= maxBackoff {
			delay = maxBackoff
			break
		}
		delay *= 2
	}
	if delay > maxBackoff {
		delay = maxBackoff
	}
	return now.Add(delay)
}

// JobProgress is an optional progress snapshot a handler may report while Running
// (spec/contexts/jobs.yaml models.JobProgress). No interface writes it yet in this
// WI's core runtime; a future consumer WI adds one via SCL when needed.
type JobProgress struct {
	Percent   *int
	Message   *string
	UpdatedAt time.Time
}

// Job is the Jobs bounded context entity (spec/contexts/jobs.yaml models.Job).
type Job struct {
	ID          string
	TenantID    string
	Kind        JobKind
	Lane        ExecutionLane
	Status      JobStatus
	Params      json.RawMessage
	Result      json.RawMessage
	Error       *string
	Attempts    int
	MaxAttempts int
	// DedupKey, when set, is unique per TenantID among non-terminal Jobs
	// (JobHandlerIdempotency).
	DedupKey       *string
	Progress       *JobProgress
	LeaseOwner     *string
	LeaseExpiresAt *time.Time
	RunAt          time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
