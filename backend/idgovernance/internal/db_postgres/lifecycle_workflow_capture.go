package db_postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	igdomain "github.com/ambi/idmagic/backend/idgovernance/internal/domain"
	igports "github.com/ambi/idmagic/backend/idgovernance/internal/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

var ErrInvalidWorkflowCapture = errors.New("workflow runs and steps length mismatch")

// UserWorkflowCapture keeps a User mutation and all derived queued runs in one
// PostgreSQL transaction, so a committed user event cannot lose its handoff.
type UserWorkflowCapture struct {
	pool      sharedpg.DB
	usersInTx func(pgx.Tx) userports.UserWriter
}

// NewUserWorkflowCapture は、WorkflowRun と同じトランザクションで User を保存するため、
// IdManagement の UserWriter を作る usersInTx を受け取る。
func NewUserWorkflowCapture(pool sharedpg.DB, usersInTx func(pgx.Tx) userports.UserWriter) *UserWorkflowCapture {
	return &UserWorkflowCapture{pool: pool, usersInTx: usersInTx}
}

var _ igports.UserWorkflowCapture = (*UserWorkflowCapture)(nil)

func (c *UserWorkflowCapture) SaveUserAndRuns(ctx context.Context, user *userdomain.User, runs []*igdomain.WorkflowRun, steps [][]igdomain.WorkflowStep) error {
	if len(runs) != len(steps) {
		return ErrInvalidWorkflowCapture
	}
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := c.usersInTx(tx).Save(ctx, user); err != nil {
		return err
	}
	for i, run := range runs {
		if _, err := saveWorkflowRun(ctx, tx, run, steps[i]); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
