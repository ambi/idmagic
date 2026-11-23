package usecases

import (
	"context"

	igports "github.com/ambi/idmagic/backend/idgovernance/internal/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/kernel"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// UserMutationCommitter implements the IdManagement boundary port
// userports.UserMutationCommitter. It owns the transactional capture that keeps a
// User mutation and its derived lifecycle workflow runs consistent, so
// IdManagement stays free of lifecycle workflow types (wi-237).
type UserMutationCommitter struct {
	WorkflowRepo igports.LifecycleWorkflowRepository
	// Capture commits the user and runs in one transaction (production). When
	// nil the committer falls back to UserRepo + RunRepo, mirroring the previous
	// non-transactional wiring for lightweight setups.
	Capture  igports.UserWorkflowCapture
	UserRepo userports.UserRepository
	RunRepo  igports.LifecycleWorkflowRunRepository
}

var _ userports.UserMutationCommitter = UserMutationCommitter{}

func (c UserMutationCommitter) CommitUserMutation(ctx context.Context, m userports.UserMutation) error {
	save := func() error {
		if m.Apply == nil {
			return c.UserRepo.Save(ctx, m.After)
		}
		saved, err := c.UserRepo.Update(ctx, m.After.ID, m.Apply)
		if err == nil && saved == nil {
			return kernel.ErrPreconditionFailed
		}
		if err == nil {
			*m.After = *saved
		}
		return err
	}
	if c.WorkflowRepo == nil {
		return save()
	}
	occurrenceID, err := spec.NewUUIDv4()
	if err != nil {
		return err
	}
	runs, steps, err := PlanLifecycleWorkflowRuns(ctx, c.WorkflowRepo, m.Before, m.After, m.Changed, occurrenceID, "", m.Now)
	if err != nil {
		return err
	}
	if c.Capture != nil {
		if m.Apply != nil {
			return c.Capture.SaveUserAndRuns(ctx, m.After, runs, steps, m.Apply)
		}
		return c.Capture.SaveUserAndRuns(ctx, m.After, runs, steps)
	}
	if err := save(); err != nil {
		return err
	}
	if c.RunRepo == nil {
		return nil
	}
	for i, run := range runs {
		if _, err := c.RunRepo.SaveRun(ctx, run, steps[i]); err != nil {
			return err
		}
	}
	return nil
}
