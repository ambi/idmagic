package db_memory

import (
	"context"
	"errors"
	"sync"

	igdomain "github.com/ambi/idmagic/backend/idgovernance/internal/domain"
	igports "github.com/ambi/idmagic/backend/idgovernance/internal/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/kernel"
)

var errInvalidWorkflowCapture = errors.New("workflow runs and steps length mismatch")

// UserWorkflowCapture prevents memory-mode observers from seeing a User
// mutation before the queued workflow runs derived from it are stored.
type UserWorkflowCapture struct {
	mu    sync.Mutex
	Users userports.UserRepository
	Runs  *LifecycleWorkflowRunRepository
}

var _ igports.UserWorkflowCapture = (*UserWorkflowCapture)(nil)

func (c *UserWorkflowCapture) SaveUserAndRuns(ctx context.Context, user *userdomain.User, runs []*igdomain.WorkflowRun, steps [][]igdomain.WorkflowStep, changes ...userports.UserChange) error {
	if len(runs) != len(steps) {
		return errInvalidWorkflowCapture
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, run := range runs {
		if err := run.Validate(); err != nil {
			return err
		}
		if len(steps[i]) != len(run.Actions) {
			return errInvalidWorkflowCapture
		}
		for _, step := range steps[i] {
			if err := step.Validate(); err != nil {
				return err
			}
		}
	}
	saveRuns := func() error {
		for i, run := range runs {
			if _, err := c.Runs.SaveRun(ctx, run, steps[i]); err != nil {
				return err
			}
		}
		return nil
	}
	if len(changes) > 0 {
		saved, err := c.Users.Update(ctx, user.ID, func(current *userdomain.User) (bool, error) {
			changed, err := changes[0](current)
			if err != nil {
				return false, err
			}
			return changed, saveRuns()
		})
		if err == nil && saved != nil {
			*user = *saved
		}
		if err == nil && saved == nil {
			return kernel.ErrPreconditionFailed
		}
		return err
	}
	if err := saveRuns(); err != nil {
		return err
	}
	return c.Users.Save(ctx, user)
}
