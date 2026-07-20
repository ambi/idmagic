package db_postgres

import (
	"context"
	"errors"

	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	igports "github.com/ambi/idmagic/backend/idgovernance/ports"
	impostgres "github.com/ambi/idmagic/backend/idmanagement/user/db_postgres"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

var ErrInvalidWorkflowCapture = errors.New("workflow runs and steps length mismatch")

// UserWorkflowCapture keeps a User mutation and all derived queued runs in one
// PostgreSQL transaction, so a committed user event cannot lose its handoff.
type UserWorkflowCapture struct{ Pool sharedpg.DB }

var _ igports.UserWorkflowCapture = (*UserWorkflowCapture)(nil)

func (c *UserWorkflowCapture) SaveUserAndRuns(ctx context.Context, user *userdomain.User, runs []*igdomain.WorkflowRun, steps [][]igdomain.WorkflowStep) error {
	if len(runs) != len(steps) {
		return ErrInvalidWorkflowCapture
	}
	tx, err := c.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := impostgres.SaveUserTx(ctx, tx, user); err != nil {
		return err
	}
	for i, run := range runs {
		if _, err := saveWorkflowRun(ctx, tx, run, steps[i]); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
