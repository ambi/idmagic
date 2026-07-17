package usecases

import (
	"context"

	igports "github.com/ambi/idmagic/backend/idgovernance/ports"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// UserMutationCommitter implements the IdManagement boundary port
// idmports.UserMutationCommitter. It owns the transactional capture that keeps a
// User mutation and its derived lifecycle workflow runs consistent, so
// IdManagement stays free of lifecycle workflow types (wi-237, ADR-117).
type UserMutationCommitter struct {
	WorkflowRepo igports.LifecycleWorkflowRepository
	// Capture commits the user and runs in one transaction (production). When
	// nil the committer falls back to UserRepo + RunRepo, mirroring the previous
	// non-transactional wiring for lightweight setups.
	Capture  igports.UserWorkflowCapture
	UserRepo idmports.UserRepository
	RunRepo  igports.LifecycleWorkflowRunRepository
}

var _ idmports.UserMutationCommitter = UserMutationCommitter{}

func (c UserMutationCommitter) CommitUserMutation(ctx context.Context, m idmports.UserMutation) error {
	if c.WorkflowRepo == nil {
		return c.UserRepo.Save(ctx, m.After)
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
		return c.Capture.SaveUserAndRuns(ctx, m.After, runs, steps)
	}
	if err := c.UserRepo.Save(ctx, m.After); err != nil {
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
