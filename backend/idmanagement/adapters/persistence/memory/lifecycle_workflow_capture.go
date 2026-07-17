package memory

import (
	"context"
	"errors"
	"sync"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
)

var errInvalidWorkflowCapture = errors.New("workflow runs and steps length mismatch")

// UserWorkflowCapture prevents memory-mode observers from seeing a User
// mutation before the queued workflow runs derived from it are stored.
type UserWorkflowCapture struct {
	mu    sync.Mutex
	Users *UserRepository
	Runs  *LifecycleWorkflowRunRepository
}

var _ idmports.UserWorkflowCapture = (*UserWorkflowCapture)(nil)

func (c *UserWorkflowCapture) SaveUserAndRuns(ctx context.Context, user *idmdomain.User, runs []*idmdomain.WorkflowRun, steps [][]idmdomain.WorkflowStep) error {
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
	for i, run := range runs {
		if _, err := c.Runs.SaveRun(ctx, run, steps[i]); err != nil {
			return err
		}
	}
	return c.Users.Save(ctx, user)
}
