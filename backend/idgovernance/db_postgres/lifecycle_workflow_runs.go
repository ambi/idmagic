package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	igports "github.com/ambi/idmagic/backend/idgovernance/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type LifecycleWorkflowRunRepository struct{ Pool sharedpg.DB }

var _ igports.LifecycleWorkflowRunRepository = (*LifecycleWorkflowRunRepository)(nil)

func (r *LifecycleWorkflowRunRepository) SaveRun(ctx context.Context, run *igdomain.WorkflowRun, steps []igdomain.WorkflowStep) (bool, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	created, err := saveWorkflowRun(ctx, tx, run, steps)
	if err != nil {
		return false, err
	}
	if !created {
		return false, tx.Commit(ctx)
	}
	return true, tx.Commit(ctx)
}

func saveWorkflowRun(ctx context.Context, tx pgx.Tx, run *igdomain.WorkflowRun, steps []igdomain.WorkflowStep) (bool, error) {
	if err := run.Validate(); err != nil {
		return false, err
	}
	if len(steps) != len(run.Actions) {
		return false, errors.New("workflow steps must match actions")
	}
	actions, err := json.Marshal(run.Actions)
	if err != nil {
		return false, err
	}
	changed, err := json.Marshal(run.ChangedFields)
	if err != nil {
		return false, err
	}

	q := New(tx)
	_, err = q.InsertLifecycleWorkflowRun(ctx, InsertLifecycleWorkflowRunParams{
		ID:                 run.ID,
		TenantID:           run.TenantID,
		WorkflowID:         run.WorkflowID,
		Revision:           run.Revision,
		SourceOccurrenceID: run.SourceOccurrenceID,
		TargetUserID:       run.TargetUserID,
		TriggerKind:        string(run.TriggerKind),
		ChangedFields:      changed,
		Actions:            actions,
		Status:             string(run.Status),
		TriggeredAt:        run.TriggeredAt,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	for _, step := range steps {
		if err := step.Validate(); err != nil {
			return false, err
		}
		action, marshalErr := json.Marshal(step.Action)
		if marshalErr != nil {
			return false, marshalErr
		}
		err = q.InsertLifecycleWorkflowStep(ctx, InsertLifecycleWorkflowStepParams{
			RunID:     run.ID,
			StepIndex: int32(step.Index), //nolint:gosec // safe downcast
			Action:    action,
			Outcome:   string(step.Outcome),
		})
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func (r *LifecycleWorkflowRunRepository) FindRun(ctx context.Context, tenantID, runID string) (*igdomain.WorkflowRun, error) {
	row, err := New(r.Pool).FindLifecycleWorkflowRun(ctx, FindLifecycleWorkflowRunParams{
		TenantID: tenantID,
		ID:       runID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	run := &igdomain.WorkflowRun{
		ID:                 row.ID,
		TenantID:           row.TenantID,
		WorkflowID:         row.WorkflowID,
		Revision:           row.Revision,
		SourceOccurrenceID: row.SourceOccurrenceID,
		TargetUserID:       row.TargetUserID,
		TriggerKind:        igdomain.WorkflowTriggerKind(row.TriggerKind),
		Status:             igdomain.WorkflowRunStatus(row.Status),
		TriggeredAt:        row.TriggeredAt,
	}
	if err := json.Unmarshal(row.ChangedFields, &run.ChangedFields); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.Actions, &run.Actions); err != nil {
		return nil, err
	}
	if row.JobID.Valid {
		value := row.JobID.String()
		run.JobID = &value
	}
	return run, run.Validate()
}

func (r *LifecycleWorkflowRunRepository) ListRuns(ctx context.Context, tenantID, workflowID string, limit int) ([]*igdomain.WorkflowRun, error) {
	if limit < 1 || limit > math.MaxInt32 {
		return nil, errors.New("invalid workflow run limit")
	}
	rows, err := New(r.Pool).ListLifecycleWorkflowRuns(ctx, ListLifecycleWorkflowRunsParams{TenantID: tenantID, WorkflowID: workflowID, Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]*igdomain.WorkflowRun, 0, len(rows))
	for _, row := range rows {
		run := &igdomain.WorkflowRun{ID: row.ID, TenantID: row.TenantID, WorkflowID: row.WorkflowID, Revision: row.Revision, SourceOccurrenceID: row.SourceOccurrenceID, TargetUserID: row.TargetUserID, TriggerKind: igdomain.WorkflowTriggerKind(row.TriggerKind), Status: igdomain.WorkflowRunStatus(row.Status), TriggeredAt: row.TriggeredAt}
		if err := json.Unmarshal(row.ChangedFields, &run.ChangedFields); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(row.Actions, &run.Actions); err != nil {
			return nil, err
		}
		if row.JobID.Valid {
			value := row.JobID.String()
			run.JobID = &value
		}
		out = append(out, run)
	}
	return out, nil
}

func (r *LifecycleWorkflowRunRepository) RetryRun(ctx context.Context, tenantID, runID string) (bool, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	affected, err := q.RetryLifecycleWorkflowRun(ctx, RetryLifecycleWorkflowRunParams{TenantID: tenantID, ID: runID})
	if err != nil || affected != 1 {
		return false, err
	}
	if err := q.ResetFailedLifecycleWorkflowSteps(ctx, runID); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func (r *LifecycleWorkflowRunRepository) CancelQueuedByWorkflow(ctx context.Context, tenantID, workflowID string, _ time.Time) ([]*igdomain.WorkflowRun, error) {
	rows, err := New(r.Pool).CancelQueuedLifecycleWorkflowRuns(ctx, CancelQueuedLifecycleWorkflowRunsParams{TenantID: tenantID, WorkflowID: workflowID})
	if err != nil {
		return nil, err
	}
	canceled := make([]*igdomain.WorkflowRun, 0, len(rows))
	for _, row := range rows {
		canceled = append(canceled, &igdomain.WorkflowRun{ID: row.ID, TenantID: tenantID, WorkflowID: workflowID, TargetUserID: row.TargetUserID})
	}
	return canceled, nil
}

func (r *LifecycleWorkflowRunRepository) ListUnenqueuedRuns(ctx context.Context, limit int) ([]*igdomain.WorkflowRun, error) {
	rows, err := New(r.Pool).ListUnenqueuedLifecycleWorkflowRuns(ctx, int32(limit)) //nolint:gosec // safe downcast
	if err != nil {
		return nil, err
	}
	out := make([]*igdomain.WorkflowRun, 0, len(rows))
	for _, row := range rows {
		run := &igdomain.WorkflowRun{
			ID:                 row.ID,
			TenantID:           row.TenantID,
			WorkflowID:         row.WorkflowID,
			Revision:           row.Revision,
			SourceOccurrenceID: row.SourceOccurrenceID,
			TargetUserID:       row.TargetUserID,
			TriggerKind:        igdomain.WorkflowTriggerKind(row.TriggerKind),
			Status:             igdomain.WorkflowRunStatus(row.Status),
			TriggeredAt:        row.TriggeredAt,
		}
		if err := json.Unmarshal(row.ChangedFields, &run.ChangedFields); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(row.Actions, &run.Actions); err != nil {
			return nil, err
		}
		if row.JobID.Valid {
			value := row.JobID.String()
			run.JobID = &value
		}
		if err := run.Validate(); err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, nil
}

func (r *LifecycleWorkflowRunRepository) AttachJob(ctx context.Context, tenantID, runID, jobID string) (bool, error) {
	var parsedJobID pgtype.UUID
	if err := parsedJobID.Scan(jobID); err != nil {
		return false, err
	}
	affected, err := New(r.Pool).AttachLifecycleWorkflowRunJob(ctx, AttachLifecycleWorkflowRunJobParams{
		TenantID: tenantID,
		ID:       runID,
		JobID:    parsedJobID,
	})
	return affected == 1, err
}

func (r *LifecycleWorkflowRunRepository) ListSteps(ctx context.Context, tenantID, runID string) ([]igdomain.WorkflowStep, error) {
	rows, err := New(r.Pool).ListLifecycleWorkflowSteps(ctx, ListLifecycleWorkflowStepsParams{
		TenantID: tenantID,
		RunID:    runID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]igdomain.WorkflowStep, 0, len(rows))
	for _, row := range rows {
		step := igdomain.WorkflowStep{
			RunID:     runID,
			Index:     int(row.StepIndex),
			Outcome:   igdomain.WorkflowStepOutcome(row.Outcome),
			ErrorCode: row.ErrorCode,
		}
		if err := json.Unmarshal(row.Action, &step.Action); err != nil {
			return nil, err
		}
		if row.CompletedAt.Valid {
			v := row.CompletedAt.Time
			step.CompletedAt = &v
		}
		out = append(out, step)
	}
	return out, nil
}

func (r *LifecycleWorkflowRunRepository) StartRun(ctx context.Context, tenantID, runID string, _ time.Time) (bool, error) {
	affected, err := New(r.Pool).StartLifecycleWorkflowRun(ctx, StartLifecycleWorkflowRunParams{
		TenantID: tenantID,
		ID:       runID,
	})
	return affected == 1, err
}

func (r *LifecycleWorkflowRunRepository) CheckpointStep(ctx context.Context, tenantID, runID string, step igdomain.WorkflowStep) error {
	completedAt := pgtype.Timestamptz{}
	if step.CompletedAt != nil {
		completedAt = pgtype.Timestamptz{Time: *step.CompletedAt, Valid: true}
	}
	return New(r.Pool).CheckpointLifecycleWorkflowStep(ctx, CheckpointLifecycleWorkflowStepParams{
		RunID:       runID,
		StepIndex:   int32(step.Index), //nolint:gosec // safe downcast
		Outcome:     string(step.Outcome),
		Column4:     step.ErrorCode,
		CompletedAt: completedAt,
		TenantID:    tenantID,
	})
}

func (r *LifecycleWorkflowRunRepository) CompleteRun(ctx context.Context, tenantID, runID string, status igdomain.WorkflowRunStatus, _ time.Time) error {
	return New(r.Pool).CompleteLifecycleWorkflowRun(ctx, CompleteLifecycleWorkflowRunParams{
		TenantID: tenantID,
		ID:       runID,
		Status:   string(status),
	})
}
