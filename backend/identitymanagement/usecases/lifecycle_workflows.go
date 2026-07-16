package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

var (
	ErrLifecycleWorkflowNotFound = errors.New("lifecycle workflow not found")
	ErrWorkflowRevisionConflict  = errors.New("workflow revision conflict")
	ErrWorkflowNameConflict      = errors.New("lifecycle workflow name already exists")
)

type LifecycleWorkflowDeps struct {
	Repo    idmports.LifecycleWorkflowRepository
	RunRepo idmports.LifecycleWorkflowRunRepository
	Emit    func(spec.DomainEvent) error
}

// PlanLifecycleWorkflowRuns evaluates enabled definitions against one committed
// mutation. It intentionally stores no before/after attribute values.
func PlanLifecycleWorkflowRuns(ctx context.Context, repo idmports.LifecycleWorkflowRepository, before, after *idmdomain.User, changed []string, occurrenceID, originRunID string, now time.Time) ([]*idmdomain.WorkflowRun, [][]idmdomain.WorkflowStep, error) {
	if repo == nil || after == nil || occurrenceID == "" {
		return nil, nil, nil
	}
	workflows, err := repo.List(ctx, after.TenantID)
	if err != nil {
		return nil, nil, err
	}
	runs := []*idmdomain.WorkflowRun{}
	plans := [][]idmdomain.WorkflowStep{}
	for _, workflow := range workflows {
		if workflow.Status != idmdomain.LifecycleWorkflowEnabled || workflow.EnabledRevision == nil {
			continue
		}
		revision, findErr := repo.FindRevision(ctx, after.TenantID, workflow.ID, *workflow.EnabledRevision)
		if findErr != nil {
			return nil, nil, findErr
		}
		if revision == nil {
			continue
		}
		match, ok := idmdomain.EvaluateWorkflowTrigger(revision.Trigger, before, after, changed, originRunID)
		if !ok {
			continue
		}
		id, idErr := spec.NewUUIDv4()
		if idErr != nil {
			return nil, nil, idErr
		}
		run, steps, planErr := idmdomain.PlanWorkflowRun(id, *revision, after.ID, occurrenceID, match, now)
		if planErr != nil {
			return nil, nil, planErr
		}
		runs, plans = append(runs, run), append(plans, steps)
	}
	return runs, plans, nil
}

type CreateLifecycleWorkflowInput struct {
	Name        string
	Description *string
	Trigger     idmdomain.WorkflowTrigger
	Actions     []idmdomain.WorkflowAction
	ActorUserID string
	Now         time.Time
}

func CreateLifecycleWorkflow(ctx context.Context, deps LifecycleWorkflowDeps, input CreateLifecycleWorkflowInput) (*idmdomain.LifecycleWorkflow, error) {
	if deps.Repo == nil {
		return nil, errors.New("lifecycle workflow repository is required")
	}
	tenantID := tenancy.TenantID(ctx)
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("workflow name is required")
	}
	all, err := deps.Repo.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, workflow := range all {
		if workflow.Status != idmdomain.LifecycleWorkflowArchived && strings.EqualFold(workflow.Name, name) {
			return nil, ErrWorkflowNameConflict
		}
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	now := normalizedNow(input.Now)
	revision := &idmdomain.LifecycleWorkflowRevision{WorkflowID: id, TenantID: tenantID, Revision: 1, Trigger: input.Trigger, Actions: input.Actions, CreatedAt: now}
	if err := revision.Validate(); err != nil {
		return nil, err
	}
	workflow := &idmdomain.LifecycleWorkflow{ID: id, TenantID: tenantID, Name: name, Description: normalizedDescription(input.Description), Status: idmdomain.LifecycleWorkflowDraft, CurrentRevision: 1, CreatedAt: now, UpdatedAt: now}
	if err := workflow.Validate(); err != nil {
		return nil, err
	}
	if err := deps.Repo.Save(ctx, workflow); err != nil {
		return nil, err
	}
	if err := deps.Repo.SaveRevision(ctx, revision); err != nil {
		return nil, err
	}
	if err := adminEmit(deps.Emit, &idmdomain.LifecycleWorkflowCreated{At: now, TenantID: workflow.TenantID, ActorUserID: input.ActorUserID, WorkflowID: workflow.ID}); err != nil {
		return nil, err
	}
	return workflow, nil
}

type UpdateLifecycleWorkflowInput struct {
	WorkflowID       string
	ExpectedRevision int64
	Name             string
	Description      *string
	Trigger          idmdomain.WorkflowTrigger
	Actions          []idmdomain.WorkflowAction
	ActorUserID      string
	Now              time.Time
}

func UpdateLifecycleWorkflow(ctx context.Context, deps LifecycleWorkflowDeps, input UpdateLifecycleWorkflowInput) (*idmdomain.LifecycleWorkflow, error) {
	workflow, err := tenantWorkflow(ctx, deps.Repo, input.WorkflowID)
	if err != nil {
		return nil, err
	}
	if input.ExpectedRevision != workflow.CurrentRevision {
		return nil, ErrWorkflowRevisionConflict
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("workflow name is required")
	}
	all, err := deps.Repo.List(ctx, workflow.TenantID)
	if err != nil {
		return nil, err
	}
	for _, other := range all {
		if other.Status != idmdomain.LifecycleWorkflowArchived && other.ID != workflow.ID && strings.EqualFold(other.Name, name) {
			return nil, ErrWorkflowNameConflict
		}
	}
	now := normalizedNow(input.Now)
	next := workflow.CurrentRevision + 1
	revision := &idmdomain.LifecycleWorkflowRevision{WorkflowID: workflow.ID, TenantID: workflow.TenantID, Revision: next, Trigger: input.Trigger, Actions: input.Actions, CreatedAt: now}
	if err := revision.Validate(); err != nil {
		return nil, err
	}
	workflow.Name, workflow.Description, workflow.CurrentRevision, workflow.UpdatedAt = name, normalizedDescription(input.Description), next, now
	if err := deps.Repo.SaveRevision(ctx, revision); err != nil {
		return nil, err
	}
	if err := deps.Repo.Save(ctx, workflow); err != nil {
		return nil, err
	}
	if err := adminEmit(deps.Emit, &idmdomain.LifecycleWorkflowUpdated{At: now, TenantID: workflow.TenantID, ActorUserID: input.ActorUserID, WorkflowID: workflow.ID, NewRevision: &next}); err != nil {
		return nil, err
	}
	return workflow, nil
}

func EnableLifecycleWorkflow(ctx context.Context, deps LifecycleWorkflowDeps, workflowID string, expectedRevision int64, actorUserID string, now time.Time) (*idmdomain.LifecycleWorkflow, error) {
	workflow, err := tenantWorkflow(ctx, deps.Repo, workflowID)
	if err != nil {
		return nil, err
	}
	if expectedRevision != workflow.CurrentRevision {
		return nil, ErrWorkflowRevisionConflict
	}
	revision, err := deps.Repo.FindRevision(ctx, workflow.TenantID, workflow.ID, expectedRevision)
	if err != nil {
		return nil, err
	}
	if revision == nil {
		return nil, errors.New("workflow revision not found")
	}
	now = normalizedNow(now)
	if err := workflow.Enable(expectedRevision, now); err != nil {
		return nil, err
	}
	if err := deps.Repo.Save(ctx, workflow); err != nil {
		return nil, err
	}
	if err := adminEmit(deps.Emit, &idmdomain.LifecycleWorkflowEnabledEvent{At: now, TenantID: workflow.TenantID, ActorUserID: actorUserID, WorkflowID: workflow.ID, Revision: expectedRevision}); err != nil {
		return nil, err
	}
	return workflow, nil
}

func DisableLifecycleWorkflow(ctx context.Context, deps LifecycleWorkflowDeps, workflowID string, expectedRevision int64, actorUserID string, now time.Time) (*idmdomain.LifecycleWorkflow, error) {
	workflow, err := tenantWorkflow(ctx, deps.Repo, workflowID)
	if err != nil {
		return nil, err
	}
	if expectedRevision != workflow.CurrentRevision {
		return nil, ErrWorkflowRevisionConflict
	}
	now = normalizedNow(now)
	if err := workflow.Disable(now); err != nil {
		return nil, err
	}
	if err := deps.Repo.Save(ctx, workflow); err != nil {
		return nil, err
	}
	if err := adminEmit(deps.Emit, &idmdomain.LifecycleWorkflowDisabledEvent{At: now, TenantID: workflow.TenantID, ActorUserID: actorUserID, WorkflowID: workflow.ID}); err != nil {
		return nil, err
	}
	if deps.RunRepo != nil {
		canceled, err := deps.RunRepo.CancelQueuedByWorkflow(ctx, workflow.TenantID, workflow.ID, now)
		if err != nil {
			return nil, err
		}
		for _, run := range canceled {
			if err := adminEmit(deps.Emit, &idmdomain.LifecycleWorkflowRunCanceled{At: now, TenantID: run.TenantID, WorkflowID: run.WorkflowID, RunID: run.ID, TargetUserID: run.TargetUserID}); err != nil {
				return nil, err
			}
		}
	}
	return workflow, nil
}

type LifecycleWorkflowRunView struct {
	Run   *idmdomain.WorkflowRun
	Steps []idmdomain.WorkflowStep
}

func ListLifecycleWorkflowRuns(ctx context.Context, deps LifecycleWorkflowDeps, workflowID string, limit int) ([]LifecycleWorkflowRunView, error) {
	if _, err := tenantWorkflow(ctx, deps.Repo, workflowID); err != nil {
		return nil, err
	}
	if deps.RunRepo == nil {
		return nil, errors.New("lifecycle workflow run repository is required")
	}
	runs, err := deps.RunRepo.ListRuns(ctx, tenancy.TenantID(ctx), workflowID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]LifecycleWorkflowRunView, 0, len(runs))
	for _, run := range runs {
		steps, err := deps.RunRepo.ListSteps(ctx, run.TenantID, run.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, LifecycleWorkflowRunView{Run: run, Steps: steps})
	}
	return out, nil
}

func GetLifecycleWorkflowRun(ctx context.Context, deps LifecycleWorkflowDeps, runID string) (*LifecycleWorkflowRunView, error) {
	if deps.RunRepo == nil {
		return nil, errors.New("lifecycle workflow run repository is required")
	}
	run, err := deps.RunRepo.FindRun(ctx, tenancy.TenantID(ctx), runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrLifecycleWorkflowNotFound
	}
	steps, err := deps.RunRepo.ListSteps(ctx, run.TenantID, run.ID)
	if err != nil {
		return nil, err
	}
	return &LifecycleWorkflowRunView{Run: run, Steps: steps}, nil
}

func RetryLifecycleWorkflowRun(ctx context.Context, deps LifecycleWorkflowDeps, runID string) (*LifecycleWorkflowRunView, error) {
	view, err := GetLifecycleWorkflowRun(ctx, deps, runID)
	if err != nil {
		return nil, err
	}
	workflow, err := tenantWorkflow(ctx, deps.Repo, view.Run.WorkflowID)
	if err != nil {
		return nil, err
	}
	if workflow.Status != idmdomain.LifecycleWorkflowEnabled {
		return nil, errors.New("workflow is disabled")
	}
	ok, err := deps.RunRepo.RetryRun(ctx, view.Run.TenantID, view.Run.ID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("workflow run is not retryable")
	}
	return GetLifecycleWorkflowRun(ctx, deps, runID)
}

func DeleteLifecycleWorkflow(ctx context.Context, deps LifecycleWorkflowDeps, workflowID string, expectedRevision int64, actorUserID string, now time.Time) error {
	workflow, err := tenantWorkflow(ctx, deps.Repo, workflowID)
	if err != nil {
		return err
	}
	if expectedRevision != workflow.CurrentRevision {
		return ErrWorkflowRevisionConflict
	}
	now = normalizedNow(now)
	if err := workflow.Delete(now); err != nil {
		return err
	}
	if err := deps.Repo.Save(ctx, workflow); err != nil {
		return err
	}
	if deps.RunRepo != nil {
		canceled, err := deps.RunRepo.CancelQueuedByWorkflow(ctx, workflow.TenantID, workflow.ID, now)
		if err != nil {
			return err
		}
		for _, run := range canceled {
			if err := adminEmit(deps.Emit, &idmdomain.LifecycleWorkflowRunCanceled{At: now, TenantID: run.TenantID, WorkflowID: run.WorkflowID, RunID: run.ID, TargetUserID: run.TargetUserID}); err != nil {
				return err
			}
		}
	}
	return adminEmit(deps.Emit, &idmdomain.LifecycleWorkflowDeleted{At: now, TenantID: workflow.TenantID, ActorUserID: actorUserID, WorkflowID: workflow.ID})
}

func normalizedDescription(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func tenantWorkflow(ctx context.Context, repo idmports.LifecycleWorkflowRepository, workflowID string) (*idmdomain.LifecycleWorkflow, error) {
	if repo == nil {
		return nil, errors.New("lifecycle workflow repository is required")
	}
	workflow, err := repo.Find(ctx, tenancy.TenantID(ctx), workflowID)
	if err != nil {
		return nil, err
	}
	if workflow == nil || workflow.Status == idmdomain.LifecycleWorkflowArchived {
		return nil, ErrLifecycleWorkflowNotFound
	}
	return workflow, nil
}
