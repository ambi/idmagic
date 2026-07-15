package memory

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	sharedmem "github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
)

type LifecycleWorkflowRepository struct {
	mu        sync.RWMutex
	workflows map[string]*idmdomain.LifecycleWorkflow
	revisions map[string]*idmdomain.LifecycleWorkflowRevision
}

var _ idmports.LifecycleWorkflowRepository = (*LifecycleWorkflowRepository)(nil)

func NewLifecycleWorkflowRepository() *LifecycleWorkflowRepository {
	return &LifecycleWorkflowRepository{workflows: map[string]*idmdomain.LifecycleWorkflow{}, revisions: map[string]*idmdomain.LifecycleWorkflowRevision{}}
}

func workflowKey(tenantID, workflowID string) string {
	return sharedmem.TenantKey(tenantID, workflowID)
}

func workflowRevisionKey(tenantID, workflowID string, revision int64) string {
	return workflowKey(tenantID, workflowID) + ":" + strconv.FormatInt(revision, 10)
}

func cloneWorkflow(workflow *idmdomain.LifecycleWorkflow) *idmdomain.LifecycleWorkflow {
	if workflow == nil {
		return nil
	}
	cloned := *workflow
	if workflow.EnabledRevision != nil {
		value := *workflow.EnabledRevision
		cloned.EnabledRevision = &value
	}
	if workflow.Description != nil {
		value := *workflow.Description
		cloned.Description = &value
	}
	return &cloned
}

func cloneRevision(revision *idmdomain.LifecycleWorkflowRevision) *idmdomain.LifecycleWorkflowRevision {
	if revision == nil {
		return nil
	}
	cloned := *revision
	cloned.Trigger.WatchedAttributes = slices.Clone(revision.Trigger.WatchedAttributes)
	cloned.Trigger.Filters = slices.Clone(revision.Trigger.Filters)
	cloned.Actions = slices.Clone(revision.Actions)
	return &cloned
}

func (r *LifecycleWorkflowRepository) List(_ context.Context, tenantID string) ([]*idmdomain.LifecycleWorkflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*idmdomain.LifecycleWorkflow{}
	for _, workflow := range r.workflows {
		if workflow.TenantID == tenantID {
			out = append(out, cloneWorkflow(workflow))
		}
	}
	slices.SortFunc(out, func(a, b *idmdomain.LifecycleWorkflow) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

func (r *LifecycleWorkflowRepository) Find(_ context.Context, tenantID, workflowID string) (*idmdomain.LifecycleWorkflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneWorkflow(r.workflows[workflowKey(tenantID, workflowID)]), nil
}

func (r *LifecycleWorkflowRepository) Save(_ context.Context, workflow *idmdomain.LifecycleWorkflow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workflows[workflowKey(workflow.TenantID, workflow.ID)] = cloneWorkflow(workflow)
	return nil
}

func (r *LifecycleWorkflowRepository) FindRevision(_ context.Context, tenantID, workflowID string, revision int64) (*idmdomain.LifecycleWorkflowRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneRevision(r.revisions[workflowRevisionKey(tenantID, workflowID, revision)]), nil
}

func (r *LifecycleWorkflowRepository) SaveRevision(_ context.Context, revision *idmdomain.LifecycleWorkflowRevision) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.revisions[workflowRevisionKey(revision.TenantID, revision.WorkflowID, revision.Revision)] = cloneRevision(revision)
	return nil
}
