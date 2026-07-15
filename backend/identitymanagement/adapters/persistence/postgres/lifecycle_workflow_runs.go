package postgres

import (
	"context"
	"encoding/json"
	"errors"

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
