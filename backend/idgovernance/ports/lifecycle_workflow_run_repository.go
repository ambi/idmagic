package ports

import (
	"context"
	"time"

	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// LifecycleWorkflowRunRepository holds the durable handoff record. SaveRun
// reports false for an existing occurrence so callers can safely retry capture.
type LifecycleWorkflowRunRepository interface {
	SaveRun(ctx context.Context, run *igdomain.WorkflowRun, steps []igdomain.WorkflowStep) (created bool, err error)
	FindRun(ctx context.Context, tenantID, runID string) (*igdomain.WorkflowRun, error)
	ListRuns(ctx context.Context, tenantID, workflowID string, limit int) ([]*igdomain.WorkflowRun, error)
	ListUnenqueuedRuns(ctx context.Context, limit int) ([]*igdomain.WorkflowRun, error)
	AttachJob(ctx context.Context, tenantID, runID, jobID string) (attached bool, err error)
	ListSteps(ctx context.Context, tenantID, runID string) ([]igdomain.WorkflowStep, error)
	StartRun(ctx context.Context, tenantID, runID string, now time.Time) (started bool, err error)
	CheckpointStep(ctx context.Context, tenantID, runID string, step igdomain.WorkflowStep) error
	CompleteRun(ctx context.Context, tenantID, runID string, status igdomain.WorkflowRunStatus, now time.Time) error
	RetryRun(ctx context.Context, tenantID, runID string) (bool, error)
	// CancelQueuedByWorkflow cancels every queued run for the workflow and
	// returns the canceled runs so callers can emit LifecycleWorkflowRunCanceled
	// per run.
	CancelQueuedByWorkflow(ctx context.Context, tenantID, workflowID string, now time.Time) ([]*igdomain.WorkflowRun, error)
}

// UserWorkflowCapture is the transaction boundary for a User mutation and its
// derived queued runs. Implementations must commit both or neither.
type UserWorkflowCapture interface {
	SaveUserAndRuns(ctx context.Context, user *userdomain.User, runs []*igdomain.WorkflowRun, steps [][]igdomain.WorkflowStep) error
}
