package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
)

type (
	LifecycleWorkflowDispatcherDeps struct {
		RunRepo idmports.LifecycleWorkflowRunRepository
		JobRepo jobsports.JobRepository
	}
	lifecycleWorkflowJobParams struct {
		RunID string `json:"run_id"`
	}
)

// DispatchQueuedLifecycleWorkflowRuns is safe to invoke after every mutation and
// periodically from the worker. A failed enqueue leaves job_id null, allowing a
// later invocation to recover it; Jobs' dedup key collapses racing dispatchers.
func DispatchQueuedLifecycleWorkflowRuns(ctx context.Context, deps LifecycleWorkflowDispatcherDeps, limit int, now time.Time) error {
	runs, err := deps.RunRepo.ListUnenqueuedRuns(ctx, limit)
	if err != nil {
		return err
	}
	for _, run := range runs {
		params, marshalErr := json.Marshal(lifecycleWorkflowJobParams{RunID: run.ID})
		if marshalErr != nil {
			return marshalErr
		}
		dedup := "lifecycle-workflow-run:" + run.ID
		job, enqueueErr := jobsusecases.Enqueue(ctx, jobsusecases.EnqueueDeps{Repo: deps.JobRepo}, jobsports.EnqueueInput{TenantID: run.TenantID, Kind: jobsdomain.KindLifecycleWorkflowRun, Params: params, DedupKey: &dedup}, now)
		if enqueueErr != nil {
			return enqueueErr
		}
		if _, attachErr := deps.RunRepo.AttachJob(ctx, run.TenantID, run.ID, job.ID); attachErr != nil {
			return attachErr
		}
	}
	return nil
}

// LifecycleWorkflowRunHandler is intentionally side-effect free in WI-217: it
// fail-closes tenant mismatches and confirms the durable handoff. WI-218 adds the
// step executor behind the same handler registration.
func LifecycleWorkflowRunHandler(runRepo idmports.LifecycleWorkflowRunRepository) func(context.Context, *jobsdomain.Job) (json.RawMessage, error) {
	return func(ctx context.Context, job *jobsdomain.Job) (json.RawMessage, error) {
		var params lifecycleWorkflowJobParams
		if err := json.Unmarshal(job.Params, &params); err != nil || params.RunID == "" {
			return nil, fmt.Errorf("invalid lifecycle workflow job params")
		}
		run, err := runRepo.FindRun(ctx, job.TenantID, params.RunID)
		if err != nil {
			return nil, err
		}
		if run == nil || run.TenantID != job.TenantID || run.Status != idmdomain.WorkflowRunQueued {
			return nil, fmt.Errorf("lifecycle workflow run not runnable")
		}
		return json.Marshal(map[string]string{"run_id": run.ID, "status": string(run.Status)})
	}
}
