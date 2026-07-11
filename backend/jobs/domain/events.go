package domain

import "time"

// The following structs are the Jobs bounded context's domain events
// (spec/contexts/jobs.yaml models, kind: event). Each satisfies
// backend/shared/spec.DomainEvent (EventType() string; OccurredAt() time.Time) by
// structural typing, without importing that package, keeping domain free of
// dependencies on the shared SCL binding layer.

// JobEnqueued is emitted when a Job is added to the queue.
type JobEnqueued struct {
	At       time.Time
	JobID    string
	TenantID string
	Kind     JobKind
}

func (e *JobEnqueued) EventType() string     { return "JobEnqueued" }
func (e *JobEnqueued) OccurredAt() time.Time { return e.At }

// JobStarted is emitted when a worker claims a Job and moves it to Running.
type JobStarted struct {
	At       time.Time
	JobID    string
	TenantID string
	WorkerID string
	Attempt  int
}

func (e *JobStarted) EventType() string     { return "JobStarted" }
func (e *JobStarted) OccurredAt() time.Time { return e.At }

// JobSucceeded is emitted when a handler completes successfully.
type JobSucceeded struct {
	At       time.Time
	JobID    string
	TenantID string
}

func (e *JobSucceeded) EventType() string     { return "JobSucceeded" }
func (e *JobSucceeded) OccurredAt() time.Time { return e.At }

// JobFailed is emitted on handler failure. Terminal is true when attempts reached
// MaxAttempts and the Job is dead-lettered (JobDeadLetterOnMaxAttempts) rather than
// retried.
type JobFailed struct {
	At       time.Time
	JobID    string
	TenantID string
	Attempt  int
	Terminal bool
	Error    string
}

func (e *JobFailed) EventType() string     { return "JobFailed" }
func (e *JobFailed) OccurredAt() time.Time { return e.At }

// JobRetried is emitted when a failed Job is returned to Queued for another
// attempt after a backoff delay.
type JobRetried struct {
	At        time.Time
	JobID     string
	TenantID  string
	Attempt   int
	NextRunAt time.Time
}

func (e *JobRetried) EventType() string     { return "JobRetried" }
func (e *JobRetried) OccurredAt() time.Time { return e.At }

// JobCanceled is emitted when a Job is canceled before reaching a terminal state.
type JobCanceled struct {
	At       time.Time
	JobID    string
	TenantID string
}

func (e *JobCanceled) EventType() string     { return "JobCanceled" }
func (e *JobCanceled) OccurredAt() time.Time { return e.At }
