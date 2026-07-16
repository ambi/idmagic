package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/postgres/sqlcgen"
	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type LifecycleWorkflowRunRepository struct{ Pool sharedpg.DB }

var _ idmports.LifecycleWorkflowRunRepository = (*LifecycleWorkflowRunRepository)(nil)

func (r *LifecycleWorkflowRunRepository) SaveRun(ctx context.Context, run *idmdomain.WorkflowRun, steps []idmdomain.WorkflowStep) (bool, error) {
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

func saveWorkflowRun(ctx context.Context, tx pgx.Tx, run *idmdomain.WorkflowRun, steps []idmdomain.WorkflowStep) (bool, error) {
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
	row := tx.QueryRow(ctx, `INSERT INTO lifecycle_workflow_runs (id,tenant_id,workflow_id,revision,source_occurrence_id,target_user_id,trigger_kind,changed_fields,actions,status,triggered_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (tenant_id,workflow_id,revision,source_occurrence_id,target_user_id) DO NOTHING RETURNING id`, run.ID, run.TenantID, run.WorkflowID, run.Revision, run.SourceOccurrenceID, run.TargetUserID, run.TriggerKind, changed, actions, run.Status, run.TriggeredAt)
	var id string
	if err := row.Scan(&id); errors.Is(err, pgx.ErrNoRows) {
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
		_, err = tx.Exec(ctx, `INSERT INTO lifecycle_workflow_steps (run_id,step_index,action,outcome) VALUES ($1,$2,$3,$4)`, run.ID, step.Index, action, step.Outcome)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func scanWorkflowRun(row sharedpg.RowScanner) (*idmdomain.WorkflowRun, error) {
	run := &idmdomain.WorkflowRun{}
	var changed, actions []byte
	var job pgtype.Text
	err := row.Scan(&run.ID, &run.TenantID, &run.WorkflowID, &run.Revision, &run.SourceOccurrenceID, &run.TargetUserID, &run.TriggerKind, &changed, &actions, &run.Status, &job, &run.TriggeredAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(changed, &run.ChangedFields); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(actions, &run.Actions); err != nil {
		return nil, err
	}
	if job.Valid {
		v := job.String
		run.JobID = &v
	}
	return run, run.Validate()
}

func (r *LifecycleWorkflowRunRepository) FindRun(ctx context.Context, tenantID, runID string) (*idmdomain.WorkflowRun, error) {
	run, err := scanWorkflowRun(r.Pool.QueryRow(ctx, `SELECT id,tenant_id,workflow_id,revision,source_occurrence_id,target_user_id,trigger_kind,changed_fields,actions,status,job_id,triggered_at FROM lifecycle_workflow_runs WHERE tenant_id=$1 AND id=$2`, tenantID, runID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return run, err
}

func (r *LifecycleWorkflowRunRepository) ListRuns(ctx context.Context, tenantID, workflowID string, limit int) ([]*idmdomain.WorkflowRun, error) {
	if limit < 1 || limit > math.MaxInt32 {
		return nil, errors.New("invalid workflow run limit")
	}
	rows, err := sqlcgen.New(r.Pool).ListLifecycleWorkflowRuns(ctx, sqlcgen.ListLifecycleWorkflowRunsParams{TenantID: tenantID, WorkflowID: workflowID, Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]*idmdomain.WorkflowRun, 0, len(rows))
	for _, row := range rows {
		run := &idmdomain.WorkflowRun{ID: row.ID, TenantID: row.TenantID, WorkflowID: row.WorkflowID, Revision: row.Revision, SourceOccurrenceID: row.SourceOccurrenceID, TargetUserID: row.TargetUserID, TriggerKind: idmdomain.WorkflowTriggerKind(row.TriggerKind), Status: idmdomain.WorkflowRunStatus(row.Status), TriggeredAt: row.TriggeredAt}
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
	q := sqlcgen.New(tx)
	affected, err := q.RetryLifecycleWorkflowRun(ctx, sqlcgen.RetryLifecycleWorkflowRunParams{TenantID: tenantID, ID: runID})
	if err != nil || affected != 1 {
		return false, err
	}
	if err := q.ResetFailedLifecycleWorkflowSteps(ctx, runID); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func (r *LifecycleWorkflowRunRepository) CancelQueuedByWorkflow(ctx context.Context, tenantID, workflowID string, _ time.Time) ([]*idmdomain.WorkflowRun, error) {
	rows, err := sqlcgen.New(r.Pool).CancelQueuedLifecycleWorkflowRuns(ctx, sqlcgen.CancelQueuedLifecycleWorkflowRunsParams{TenantID: tenantID, WorkflowID: workflowID})
	if err != nil {
		return nil, err
	}
	canceled := make([]*idmdomain.WorkflowRun, 0, len(rows))
	for _, row := range rows {
		canceled = append(canceled, &idmdomain.WorkflowRun{ID: row.ID, TenantID: tenantID, WorkflowID: workflowID, TargetUserID: row.TargetUserID})
	}
	return canceled, nil
}

func (r *LifecycleWorkflowRunRepository) ListUnenqueuedRuns(ctx context.Context, limit int) ([]*idmdomain.WorkflowRun, error) {
	rows, err := r.Pool.Query(ctx, `SELECT id,tenant_id,workflow_id,revision,source_occurrence_id,target_user_id,trigger_kind,changed_fields,actions,status,job_id,triggered_at FROM lifecycle_workflow_runs WHERE status='queued' AND job_id IS NULL ORDER BY triggered_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*idmdomain.WorkflowRun{}
	for rows.Next() {
		run, scanErr := scanWorkflowRun(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

func (r *LifecycleWorkflowRunRepository) AttachJob(ctx context.Context, tenantID, runID, jobID string) (bool, error) {
	tag, err := r.Pool.Exec(ctx, `UPDATE lifecycle_workflow_runs SET job_id=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2 AND status='queued' AND job_id IS NULL`, tenantID, runID, jobID)
	return tag.RowsAffected() == 1, err
}

func (r *LifecycleWorkflowRunRepository) ListSteps(ctx context.Context, tenantID, runID string) ([]idmdomain.WorkflowStep, error) {
	rows, err := r.Pool.Query(ctx, `SELECT s.step_index,s.action,s.outcome,COALESCE(s.error_code,''),s.completed_at FROM lifecycle_workflow_steps s JOIN lifecycle_workflow_runs r ON r.id=s.run_id WHERE r.tenant_id=$1 AND s.run_id=$2 ORDER BY s.step_index`, tenantID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []idmdomain.WorkflowStep{}
	for rows.Next() {
		var step idmdomain.WorkflowStep
		var action []byte
		var completed pgtype.Timestamptz
		step.RunID = runID
		if err := rows.Scan(&step.Index, &action, &step.Outcome, &step.ErrorCode, &completed); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(action, &step.Action); err != nil {
			return nil, err
		}
		if completed.Valid {
			v := completed.Time
			step.CompletedAt = &v
		}
		out = append(out, step)
	}
	return out, rows.Err()
}

func (r *LifecycleWorkflowRunRepository) StartRun(ctx context.Context, tenantID, runID string, _ time.Time) (bool, error) {
	tag, err := r.Pool.Exec(ctx, `UPDATE lifecycle_workflow_runs candidate SET status='running',updated_at=now() WHERE candidate.tenant_id=$1 AND candidate.id=$2 AND candidate.status='queued' AND NOT EXISTS (SELECT 1 FROM lifecycle_workflow_runs prior WHERE prior.tenant_id=candidate.tenant_id AND prior.target_user_id=candidate.target_user_id AND prior.id<>candidate.id AND prior.status IN ('queued','running') AND prior.triggered_at<candidate.triggered_at)`, tenantID, runID)
	return tag.RowsAffected() == 1, err
}

func (r *LifecycleWorkflowRunRepository) CheckpointStep(ctx context.Context, tenantID, runID string, step idmdomain.WorkflowStep) error {
	_, err := r.Pool.Exec(ctx, `UPDATE lifecycle_workflow_steps SET outcome=$3,error_code=NULLIF($4,''),completed_at=$5 WHERE run_id=$1 AND step_index=$2 AND EXISTS (SELECT 1 FROM lifecycle_workflow_runs WHERE id=$1 AND tenant_id=$6)`, runID, step.Index, step.Outcome, step.ErrorCode, step.CompletedAt, tenantID)
	return err
}

func (r *LifecycleWorkflowRunRepository) CompleteRun(ctx context.Context, tenantID, runID string, status idmdomain.WorkflowRunStatus, _ time.Time) error {
	_, err := r.Pool.Exec(ctx, `UPDATE lifecycle_workflow_runs SET status=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2`, tenantID, runID, status)
	return err
}
