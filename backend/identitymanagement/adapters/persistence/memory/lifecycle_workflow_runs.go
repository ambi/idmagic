package memory

import (
	"context"
	"slices"
	"sync"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	sharedmem "github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
)

type LifecycleWorkflowRunRepository struct {
	mu          sync.RWMutex
	runs        map[string]*idmdomain.WorkflowRun
	steps       map[string][]idmdomain.WorkflowStep
	occurrences map[string]string
}

var _ idmports.LifecycleWorkflowRunRepository = (*LifecycleWorkflowRunRepository)(nil)

func NewLifecycleWorkflowRunRepository() *LifecycleWorkflowRunRepository {
	return &LifecycleWorkflowRunRepository{runs: map[string]*idmdomain.WorkflowRun{}, steps: map[string][]idmdomain.WorkflowStep{}, occurrences: map[string]string{}}
}
func runKey(tenantID, id string) string { return sharedmem.TenantKey(tenantID, id) }
func occurrenceKey(r *idmdomain.WorkflowRun) string {
	return runKey(r.TenantID, r.WorkflowID) + ":" + r.SourceOccurrenceID + ":" + r.TargetUserID
}

func cloneRun(r *idmdomain.WorkflowRun) *idmdomain.WorkflowRun {
	if r == nil {
		return nil
	}
	c := *r
	c.Actions = slices.Clone(r.Actions)
	c.ChangedFields = slices.Clone(r.ChangedFields)
	if r.JobID != nil {
		v := *r.JobID
		c.JobID = &v
	}
	return &c
}

func (r *LifecycleWorkflowRunRepository) SaveRun(_ context.Context, run *idmdomain.WorkflowRun, steps []idmdomain.WorkflowStep) (bool, error) {
	if err := run.Validate(); err != nil {
		return false, err
	}
	for _, s := range steps {
		if err := s.Validate(); err != nil {
			return false, err
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := occurrenceKey(run)
	if _, ok := r.occurrences[key]; ok {
		return false, nil
	}
	r.runs[runKey(run.TenantID, run.ID)] = cloneRun(run)
	r.steps[runKey(run.TenantID, run.ID)] = slices.Clone(steps)
	r.occurrences[key] = run.ID
	return true, nil
}

func (r *LifecycleWorkflowRunRepository) ListSteps(_ context.Context, tenantID, runID string) ([]idmdomain.WorkflowStep, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Clone(r.steps[runKey(tenantID, runID)]), nil
}

func (r *LifecycleWorkflowRunRepository) StartRun(_ context.Context, tenantID, runID string, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run := r.runs[runKey(tenantID, runID)]
	if run == nil || run.Status != idmdomain.WorkflowRunQueued {
		return false, nil
	}
	for _, other := range r.runs {
		if other.TenantID == tenantID && other.TargetUserID == run.TargetUserID && other.ID != run.ID && !other.Status.Terminal() && other.TriggeredAt.Before(run.TriggeredAt) {
			return false, nil
		}
	}
	run.Status = idmdomain.WorkflowRunRunning
	return true, nil
}

func (r *LifecycleWorkflowRunRepository) CheckpointStep(_ context.Context, tenantID, runID string, step idmdomain.WorkflowStep) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := runKey(tenantID, runID)
	steps := r.steps[key]
	if step.Index < 0 || step.Index >= len(steps) {
		return nil
	}
	steps[step.Index] = step
	r.steps[key] = steps
	return nil
}

func (r *LifecycleWorkflowRunRepository) CompleteRun(_ context.Context, tenantID, runID string, status idmdomain.WorkflowRunStatus, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	run := r.runs[runKey(tenantID, runID)]
	if run != nil {
		run.Status = status
	}
	return nil
}

func (r *LifecycleWorkflowRunRepository) FindRun(_ context.Context, tenantID, runID string) (*idmdomain.WorkflowRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneRun(r.runs[runKey(tenantID, runID)]), nil
}

func (r *LifecycleWorkflowRunRepository) ListUnenqueuedRuns(_ context.Context, limit int) ([]*idmdomain.WorkflowRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*idmdomain.WorkflowRun{}
	for _, run := range r.runs {
		if run.Status == idmdomain.WorkflowRunQueued && run.JobID == nil {
			out = append(out, cloneRun(run))
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *LifecycleWorkflowRunRepository) AttachJob(_ context.Context, tenantID, runID, jobID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run := r.runs[runKey(tenantID, runID)]
	if run == nil || run.JobID != nil {
		return false, nil
	}
	run.JobID = &jobID
	return true, nil
}
