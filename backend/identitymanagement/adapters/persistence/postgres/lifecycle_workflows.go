package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
)

type LifecycleWorkflowRepository struct{ Pool sharedpg.DB }

var _ idmports.LifecycleWorkflowRepository = (*LifecycleWorkflowRepository)(nil)

const lifecycleWorkflowColumns = `id,tenant_id,name,description,status,current_revision,enabled_revision,created_at,updated_at`

func scanLifecycleWorkflow(row sharedpg.RowScanner) (*idmdomain.LifecycleWorkflow, error) {
	workflow := &idmdomain.LifecycleWorkflow{}
	var enabled pgtype.Int8
	var description pgtype.Text
	if err := row.Scan(&workflow.ID, &workflow.TenantID, &workflow.Name, &description, &workflow.Status, &workflow.CurrentRevision, &enabled, &workflow.CreatedAt, &workflow.UpdatedAt); err != nil {
		return nil, err
	}
	if description.Valid {
		value := description.String
		workflow.Description = &value
	}
	if enabled.Valid {
		value := enabled.Int64
		workflow.EnabledRevision = &value
	}
	return workflow, workflow.Validate()
}

func (r *LifecycleWorkflowRepository) List(ctx context.Context, tenantID string) ([]*idmdomain.LifecycleWorkflow, error) {
	rows, err := r.Pool.Query(ctx, `SELECT `+lifecycleWorkflowColumns+` FROM lifecycle_workflows WHERE tenant_id=$1 ORDER BY name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*idmdomain.LifecycleWorkflow{}
	for rows.Next() {
		workflow, scanErr := scanLifecycleWorkflow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, workflow)
	}
	return out, rows.Err()
}

func (r *LifecycleWorkflowRepository) Find(ctx context.Context, tenantID, workflowID string) (*idmdomain.LifecycleWorkflow, error) {
	workflow, err := scanLifecycleWorkflow(r.Pool.QueryRow(ctx, `SELECT `+lifecycleWorkflowColumns+` FROM lifecycle_workflows WHERE tenant_id=$1 AND id=$2`, tenantID, workflowID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return workflow, err
}

func (r *LifecycleWorkflowRepository) Save(ctx context.Context, workflow *idmdomain.LifecycleWorkflow) error {
	if err := workflow.Validate(); err != nil {
		return err
	}
	var enabled pgtype.Int8
	if workflow.EnabledRevision != nil {
		enabled = pgtype.Int8{Int64: *workflow.EnabledRevision, Valid: true}
	}
	_, err := r.Pool.Exec(ctx, `INSERT INTO lifecycle_workflows (id,tenant_id,name,description,status,current_revision,enabled_revision,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,description=EXCLUDED.description,status=EXCLUDED.status,current_revision=EXCLUDED.current_revision,enabled_revision=EXCLUDED.enabled_revision,updated_at=EXCLUDED.updated_at WHERE lifecycle_workflows.tenant_id=EXCLUDED.tenant_id`, workflow.ID, workflow.TenantID, workflow.Name, workflow.Description, workflow.Status, workflow.CurrentRevision, enabled, workflow.CreatedAt, workflow.UpdatedAt)
	return err
}

func (r *LifecycleWorkflowRepository) FindRevision(ctx context.Context, tenantID, workflowID string, number int64) (*idmdomain.LifecycleWorkflowRevision, error) {
	revision := &idmdomain.LifecycleWorkflowRevision{}
	var trigger, actions []byte
	err := r.Pool.QueryRow(ctx, `SELECT workflow_id,tenant_id,revision,trigger,actions,created_at FROM lifecycle_workflow_revisions WHERE tenant_id=$1 AND workflow_id=$2 AND revision=$3`, tenantID, workflowID, number).Scan(&revision.WorkflowID, &revision.TenantID, &revision.Revision, &trigger, &actions, &revision.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(trigger, &revision.Trigger); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(actions, &revision.Actions); err != nil {
		return nil, err
	}
	return revision, revision.Validate()
}

func (r *LifecycleWorkflowRepository) SaveRevision(ctx context.Context, revision *idmdomain.LifecycleWorkflowRevision) error {
	if err := revision.Validate(); err != nil {
		return err
	}
	trigger, err := json.Marshal(revision.Trigger)
	if err != nil {
		return err
	}
	actions, err := json.Marshal(revision.Actions)
	if err != nil {
		return err
	}
	_, err = r.Pool.Exec(ctx, `INSERT INTO lifecycle_workflow_revisions (workflow_id,tenant_id,revision,trigger,actions,created_at) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (workflow_id,revision) DO NOTHING`, revision.WorkflowID, revision.TenantID, revision.Revision, trigger, actions, revision.CreatedAt)
	return err
}
