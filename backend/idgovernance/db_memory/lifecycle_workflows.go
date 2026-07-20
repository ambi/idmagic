package db_memory

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"

	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	igports "github.com/ambi/idmagic/backend/idgovernance/ports"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
)

type LifecycleWorkflowRepository struct {
	mu        sync.RWMutex
	workflows map[string]*igdomain.LifecycleWorkflow
	revisions map[string]*igdomain.LifecycleWorkflowRevision
}

var _ igports.LifecycleWorkflowRepository = (*LifecycleWorkflowRepository)(nil)

func NewLifecycleWorkflowRepository() *LifecycleWorkflowRepository {
	return &LifecycleWorkflowRepository{workflows: map[string]*igdomain.LifecycleWorkflow{}, revisions: map[string]*igdomain.LifecycleWorkflowRevision{}}
}

func workflowKey(tenantID, workflowID string) string {
	return sharedmem.TenantKey(tenantID, workflowID)
}

func workflowRevisionKey(tenantID, workflowID string, revision int64) string {
	return workflowKey(tenantID, workflowID) + ":" + strconv.FormatInt(revision, 10)
}

func cloneWorkflow(workflow *igdomain.LifecycleWorkflow) *igdomain.LifecycleWorkflow {
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

func cloneRevision(revision *igdomain.LifecycleWorkflowRevision) *igdomain.LifecycleWorkflowRevision {
	if revision == nil {
		return nil
	}
	cloned := *revision
	cloned.Trigger.WatchedAttributes = slices.Clone(revision.Trigger.WatchedAttributes)
	cloned.Trigger.Filters = slices.Clone(revision.Trigger.Filters)
	cloned.Actions = slices.Clone(revision.Actions)
	return &cloned
}

func (r *LifecycleWorkflowRepository) List(_ context.Context, tenantID string) ([]*igdomain.LifecycleWorkflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*igdomain.LifecycleWorkflow{}
	for _, workflow := range r.workflows {
		if workflow.TenantID == tenantID {
			out = append(out, cloneWorkflow(workflow))
		}
	}
	slices.SortFunc(out, func(a, b *igdomain.LifecycleWorkflow) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

func (r *LifecycleWorkflowRepository) Find(_ context.Context, tenantID, workflowID string) (*igdomain.LifecycleWorkflow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneWorkflow(r.workflows[workflowKey(tenantID, workflowID)]), nil
}

func (r *LifecycleWorkflowRepository) Save(_ context.Context, workflow *igdomain.LifecycleWorkflow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workflows[workflowKey(workflow.TenantID, workflow.ID)] = cloneWorkflow(workflow)
	return nil
}

func (r *LifecycleWorkflowRepository) FindRevision(_ context.Context, tenantID, workflowID string, revision int64) (*igdomain.LifecycleWorkflowRevision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneRevision(r.revisions[workflowRevisionKey(tenantID, workflowID, revision)]), nil
}

func (r *LifecycleWorkflowRepository) SaveRevision(_ context.Context, revision *igdomain.LifecycleWorkflowRevision) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.revisions[workflowRevisionKey(revision.TenantID, revision.WorkflowID, revision.Revision)] = cloneRevision(revision)
	return nil
}
