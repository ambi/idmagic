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
	Repo idmports.LifecycleWorkflowRepository
}

type CreateLifecycleWorkflowInput struct {
	Name    string
	Trigger idmdomain.WorkflowTrigger
	Actions []idmdomain.WorkflowAction
	Now     time.Time
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
		if strings.EqualFold(workflow.Name, name) {
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
	workflow := &idmdomain.LifecycleWorkflow{ID: id, TenantID: tenantID, Name: name, Status: idmdomain.LifecycleWorkflowDraft, CurrentRevision: 1, CreatedAt: now, UpdatedAt: now}
	if err := workflow.Validate(); err != nil {
		return nil, err
	}
	if err := deps.Repo.SaveRevision(ctx, revision); err != nil {
		return nil, err
	}
	if err := deps.Repo.Save(ctx, workflow); err != nil {
		return nil, err
	}
	return workflow, nil
}

type UpdateLifecycleWorkflowInput struct {
	WorkflowID       string
	ExpectedRevision int64
	Name             string
	Trigger          idmdomain.WorkflowTrigger
	Actions          []idmdomain.WorkflowAction
	Now              time.Time
}

func UpdateLifecycleWorkflow(ctx context.Context, deps LifecycleWorkflowDeps, input UpdateLifecycleWorkflowInput) (*idmdomain.LifecycleWorkflow, error) {
	workflow, err := tenantWorkflow(ctx, deps, input.WorkflowID)
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
		if other.ID != workflow.ID && strings.EqualFold(other.Name, name) {
			return nil, ErrWorkflowNameConflict
		}
	}
	now := normalizedNow(input.Now)
	next := workflow.CurrentRevision + 1
	revision := &idmdomain.LifecycleWorkflowRevision{WorkflowID: workflow.ID, TenantID: workflow.TenantID, Revision: next, Trigger: input.Trigger, Actions: input.Actions, CreatedAt: now}
	if err := revision.Validate(); err != nil {
		return nil, err
	}
	workflow.Name, workflow.CurrentRevision, workflow.UpdatedAt = name, next, now
	if err := deps.Repo.SaveRevision(ctx, revision); err != nil {
		return nil, err
	}
	if err := deps.Repo.Save(ctx, workflow); err != nil {
		return nil, err
	}
	return workflow, nil
}

func EnableLifecycleWorkflow(ctx context.Context, deps LifecycleWorkflowDeps, workflowID string, expectedRevision int64, now time.Time) (*idmdomain.LifecycleWorkflow, error) {
	workflow, err := tenantWorkflow(ctx, deps, workflowID)
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
	if err := workflow.Enable(expectedRevision, normalizedNow(now)); err != nil {
		return nil, err
	}
	return workflow, deps.Repo.Save(ctx, workflow)
}

func DisableLifecycleWorkflow(ctx context.Context, deps LifecycleWorkflowDeps, workflowID string, now time.Time) (*idmdomain.LifecycleWorkflow, error) {
	workflow, err := tenantWorkflow(ctx, deps, workflowID)
	if err != nil {
		return nil, err
	}
	if err := workflow.Disable(normalizedNow(now)); err != nil {
		return nil, err
	}
	return workflow, deps.Repo.Save(ctx, workflow)
}

func ArchiveLifecycleWorkflow(ctx context.Context, deps LifecycleWorkflowDeps, workflowID string, now time.Time) (*idmdomain.LifecycleWorkflow, error) {
	workflow, err := tenantWorkflow(ctx, deps, workflowID)
	if err != nil {
		return nil, err
	}
	if err := workflow.Archive(normalizedNow(now)); err != nil {
		return nil, err
	}
	return workflow, deps.Repo.Save(ctx, workflow)
}

func tenantWorkflow(ctx context.Context, deps LifecycleWorkflowDeps, workflowID string) (*idmdomain.LifecycleWorkflow, error) {
	if deps.Repo == nil {
		return nil, errors.New("lifecycle workflow repository is required")
	}
	workflow, err := deps.Repo.Find(ctx, tenancy.TenantID(ctx), workflowID)
	if err != nil {
		return nil, err
	}
	if workflow == nil {
		return nil, ErrLifecycleWorkflowNotFound
	}
	return workflow, nil
}
