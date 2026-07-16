package ports

import (
	"context"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
)

// LifecycleWorkflowRunRepository holds the durable handoff record. SaveRun
// reports false for an existing occurrence so callers can safely retry capture.
type LifecycleWorkflowRunRepository interface {
	SaveRun(ctx context.Context, run *idmdomain.WorkflowRun, steps []idmdomain.WorkflowStep) (created bool, err error)
	FindRun(ctx context.Context, tenantID, runID string) (*idmdomain.WorkflowRun, error)
	ListRuns(ctx context.Context, tenantID, workflowID string, limit int) ([]*idmdomain.WorkflowRun, error)
	ListUnenqueuedRuns(ctx context.Context, limit int) ([]*idmdomain.WorkflowRun, error)
	AttachJob(ctx context.Context, tenantID, runID, jobID string) (attached bool, err error)
	ListSteps(ctx context.Context, tenantID, runID string) ([]idmdomain.WorkflowStep, error)
	StartRun(ctx context.Context, tenantID, runID string, now time.Time) (started bool, err error)
	CheckpointStep(ctx context.Context, tenantID, runID string, step idmdomain.WorkflowStep) error
	CompleteRun(ctx context.Context, tenantID, runID string, status idmdomain.WorkflowRunStatus, now time.Time) error
	RetryRun(ctx context.Context, tenantID, runID string) (bool, error)
	// CancelQueuedByWorkflow cancels every queued run for the workflow and
	// returns the canceled runs so callers can emit LifecycleWorkflowRunCanceled
	// per run.
	CancelQueuedByWorkflow(ctx context.Context, tenantID, workflowID string, now time.Time) ([]*idmdomain.WorkflowRun, error)
}

// UserWorkflowCapture is the transaction boundary for a User mutation and its
// derived queued runs. Implementations must commit both or neither.
type UserWorkflowCapture interface {
	SaveUserAndRuns(ctx context.Context, user *idmdomain.User, runs []*idmdomain.WorkflowRun, steps [][]idmdomain.WorkflowStep) error
}
