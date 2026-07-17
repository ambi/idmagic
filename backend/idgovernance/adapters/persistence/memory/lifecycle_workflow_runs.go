package memory

import (
	"context"
	"slices"
	"sort"
	"sync"
	"time"

	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	igports "github.com/ambi/idmagic/backend/idgovernance/ports"
	sharedmem "github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
)

type LifecycleWorkflowRunRepository struct {
	mu          sync.RWMutex
	runs        map[string]*igdomain.WorkflowRun
	steps       map[string][]igdomain.WorkflowStep
	occurrences map[string]string
}

var _ igports.LifecycleWorkflowRunRepository = (*LifecycleWorkflowRunRepository)(nil)

func NewLifecycleWorkflowRunRepository() *LifecycleWorkflowRunRepository {
	return &LifecycleWorkflowRunRepository{runs: map[string]*igdomain.WorkflowRun{}, steps: map[string][]igdomain.WorkflowStep{}, occurrences: map[string]string{}}
}
func runKey(tenantID, id string) string { return sharedmem.TenantKey(tenantID, id) }
func occurrenceKey(r *igdomain.WorkflowRun) string {
	return runKey(r.TenantID, r.WorkflowID) + ":" + r.SourceOccurrenceID + ":" + r.TargetUserID
}

func cloneRun(r *igdomain.WorkflowRun) *igdomain.WorkflowRun {
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

func (r *LifecycleWorkflowRunRepository) SaveRun(_ context.Context, run *igdomain.WorkflowRun, steps []igdomain.WorkflowStep) (bool, error) {
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

func (r *LifecycleWorkflowRunRepository) ListSteps(_ context.Context, tenantID, runID string) ([]igdomain.WorkflowStep, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Clone(r.steps[runKey(tenantID, runID)]), nil
}

func (r *LifecycleWorkflowRunRepository) StartRun(_ context.Context, tenantID, runID string, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run := r.runs[runKey(tenantID, runID)]
	if run == nil || run.Status != igdomain.WorkflowRunQueued {
		return false, nil
	}
	for _, other := range r.runs {
		if other.TenantID == tenantID && other.TargetUserID == run.TargetUserID && other.ID != run.ID && !other.Status.Terminal() && other.TriggeredAt.Before(run.TriggeredAt) {
			return false, nil
		}
	}
	run.Status = igdomain.WorkflowRunRunning
	return true, nil
}

func (r *LifecycleWorkflowRunRepository) CheckpointStep(_ context.Context, tenantID, runID string, step igdomain.WorkflowStep) error {
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

func (r *LifecycleWorkflowRunRepository) CompleteRun(_ context.Context, tenantID, runID string, status igdomain.WorkflowRunStatus, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	run := r.runs[runKey(tenantID, runID)]
	if run != nil {
		run.Status = status
	}
	return nil
}

func (r *LifecycleWorkflowRunRepository) FindRun(_ context.Context, tenantID, runID string) (*igdomain.WorkflowRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneRun(r.runs[runKey(tenantID, runID)]), nil
}

func (r *LifecycleWorkflowRunRepository) ListRuns(_ context.Context, tenantID, workflowID string, limit int) ([]*igdomain.WorkflowRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*igdomain.WorkflowRun{}
	for _, run := range r.runs {
		if run.TenantID == tenantID && run.WorkflowID == workflowID {
			out = append(out, cloneRun(run))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TriggeredAt.After(out[j].TriggeredAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *LifecycleWorkflowRunRepository) RetryRun(_ context.Context, tenantID, runID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := runKey(tenantID, runID)
	run := r.runs[key]
	if run == nil || (run.Status != igdomain.WorkflowRunFailed && run.Status != igdomain.WorkflowRunPartiallyFailed) {
		return false, nil
	}
	for i := range r.steps[key] {
		if r.steps[key][i].Outcome == igdomain.WorkflowStepFailed {
			r.steps[key][i].Outcome, r.steps[key][i].ErrorCode, r.steps[key][i].CompletedAt = igdomain.WorkflowStepPending, "", nil
		}
	}
	run.Status, run.JobID = igdomain.WorkflowRunQueued, nil
	return true, nil
}

func (r *LifecycleWorkflowRunRepository) CancelQueuedByWorkflow(_ context.Context, tenantID, workflowID string, _ time.Time) ([]*igdomain.WorkflowRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	canceled := []*igdomain.WorkflowRun{}
	for _, run := range r.runs {
		if run.TenantID == tenantID && run.WorkflowID == workflowID && run.Status == igdomain.WorkflowRunQueued {
			run.Status = igdomain.WorkflowRunCanceled
			canceled = append(canceled, cloneRun(run))
		}
	}
	return canceled, nil
}

func (r *LifecycleWorkflowRunRepository) ListUnenqueuedRuns(_ context.Context, limit int) ([]*igdomain.WorkflowRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*igdomain.WorkflowRun{}
	for _, run := range r.runs {
		if run.Status == igdomain.WorkflowRunQueued && run.JobID == nil {
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
