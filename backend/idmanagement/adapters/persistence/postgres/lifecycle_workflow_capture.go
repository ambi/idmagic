package postgres

import (
	"context"
	"errors"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
)

var ErrInvalidWorkflowCapture = errors.New("workflow runs and steps length mismatch")

// UserWorkflowCapture keeps a User mutation and all derived queued runs in one
// PostgreSQL transaction, so a committed user event cannot lose its handoff.
type UserWorkflowCapture struct{ Pool sharedpg.DB }

var _ idmports.UserWorkflowCapture = (*UserWorkflowCapture)(nil)

func (c *UserWorkflowCapture) SaveUserAndRuns(ctx context.Context, user *idmdomain.User, runs []*idmdomain.WorkflowRun, steps [][]idmdomain.WorkflowStep) error {
	if len(runs) != len(steps) {
		return ErrInvalidWorkflowCapture
	}
	tx, err := c.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := saveUser(ctx, tx, user); err != nil {
		return err
	}
	for i, run := range runs {
		if _, err := saveWorkflowRun(ctx, tx, run, steps[i]); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
