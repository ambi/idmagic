package ports

import (
	"context"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
)

// LifecycleWorkflowRunRepository holds the durable handoff record. SaveRun
// reports false for an existing occurrence so callers can safely retry capture.
type LifecycleWorkflowRunRepository interface {
	SaveRun(ctx context.Context, run *idmdomain.WorkflowRun, steps []idmdomain.WorkflowStep) (created bool, err error)
	FindRun(ctx context.Context, tenantID, runID string) (*idmdomain.WorkflowRun, error)
	ListUnenqueuedRuns(ctx context.Context, limit int) ([]*idmdomain.WorkflowRun, error)
	AttachJob(ctx context.Context, tenantID, runID, jobID string) (attached bool, err error)
}

// UserWorkflowCapture is the transaction boundary for a User mutation and its
// derived queued runs. Implementations must commit both or neither.
type UserWorkflowCapture interface {
	SaveUserAndRuns(ctx context.Context, user *idmdomain.User, runs []*idmdomain.WorkflowRun, steps [][]idmdomain.WorkflowStep) error
}
