package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	idmmemory "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/memory"
	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	"github.com/ambi/idmagic/backend/identitymanagement/usecases"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/adapters/persistence/memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
)

type failOnceJobRepository struct {
	jobsports.JobRepository
	fail bool
}

func (r *failOnceJobRepository) Enqueue(ctx context.Context, in jobsports.EnqueueInput) (*jobsdomain.Job, bool, error) {
	if r.fail {
		r.fail = false
		return nil, false, errors.New("transient enqueue failure")
	}
	return r.JobRepository.Enqueue(ctx, in)
}

func TestDispatchQueuedLifecycleWorkflowRunsAttachesDeduplicatedJob(t *testing.T) {
	runs := idmmemory.NewLifecycleWorkflowRunRepository()
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	run := &idmdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "source-1", TargetUserID: "user-1", TriggerKind: idmdomain.WorkflowTriggerUserCreated, Actions: []idmdomain.WorkflowAction{{Kind: idmdomain.WorkflowActionDisableUser}}, Status: idmdomain.WorkflowRunQueued, TriggeredAt: now}
	steps := []idmdomain.WorkflowStep{{RunID: run.ID, Index: 0, Action: run.Actions[0], Outcome: idmdomain.WorkflowStepPending}}
	if created, err := runs.SaveRun(context.Background(), run, steps); err != nil || !created {
		t.Fatalf("SaveRun = %v, %v", created, err)
	}
	jobs := &failOnceJobRepository{JobRepository: jobsmemory.NewJobRepository(), fail: true}
	deps := usecases.LifecycleWorkflowDispatcherDeps{RunRepo: runs, JobRepo: jobs}
	if err := usecases.DispatchQueuedLifecycleWorkflowRuns(context.Background(), deps, 10, now); err == nil {
		t.Fatal("first dispatch must expose enqueue failure")
	}
	if err := usecases.DispatchQueuedLifecycleWorkflowRuns(context.Background(), deps, 10, now); err != nil {
		t.Fatal(err)
	}
	stored, err := runs.FindRun(context.Background(), "tenant-a", run.ID)
	if err != nil || stored.JobID == nil {
		t.Fatalf("run job attachment = %#v, %v", stored, err)
	}
}
