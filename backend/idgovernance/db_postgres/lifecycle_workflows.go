package db_postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	igports "github.com/ambi/idmagic/backend/idgovernance/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

type LifecycleWorkflowRepository struct{ Pool sharedpg.DB }

var _ igports.LifecycleWorkflowRepository = (*LifecycleWorkflowRepository)(nil)

func lifecycleWorkflowFromRow(row *LifecycleWorkflow) (*igdomain.LifecycleWorkflow, error) {
	workflow := &igdomain.LifecycleWorkflow{
		ID:              row.ID,
		TenantID:        row.TenantID,
		Name:            row.Name,
		Status:          igdomain.LifecycleWorkflowStatus(row.Status),
		CurrentRevision: row.CurrentRevision,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
	if row.Description.Valid {
		value := row.Description.String
		workflow.Description = &value
	}
	if row.EnabledRevision.Valid {
		value := row.EnabledRevision.Int64
		workflow.EnabledRevision = &value
	}
	return workflow, workflow.Validate()
}

func (r *LifecycleWorkflowRepository) List(ctx context.Context, tenantID string) ([]*igdomain.LifecycleWorkflow, error) {
	rows, err := New(r.Pool).ListLifecycleWorkflowsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*igdomain.LifecycleWorkflow, 0, len(rows))
	for _, row := range rows {
		workflow, err := lifecycleWorkflowFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, workflow)
	}
	return out, nil
}

func (r *LifecycleWorkflowRepository) Find(ctx context.Context, tenantID, workflowID string) (*igdomain.LifecycleWorkflow, error) {
	row, err := New(r.Pool).FindLifecycleWorkflow(ctx, FindLifecycleWorkflowParams{TenantID: tenantID, ID: workflowID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return lifecycleWorkflowFromRow(row)
}

func (r *LifecycleWorkflowRepository) Save(ctx context.Context, workflow *igdomain.LifecycleWorkflow) error {
	if err := workflow.Validate(); err != nil {
		return err
	}
	var description pgtype.Text
	if workflow.Description != nil {
		description = pgtype.Text{String: *workflow.Description, Valid: true}
	}
	var enabled pgtype.Int8
	if workflow.EnabledRevision != nil {
		enabled = pgtype.Int8{Int64: *workflow.EnabledRevision, Valid: true}
	}
	return New(r.Pool).SaveLifecycleWorkflow(ctx, SaveLifecycleWorkflowParams{
		ID:              workflow.ID,
		TenantID:        workflow.TenantID,
		Name:            workflow.Name,
		Description:     description,
		Status:          string(workflow.Status),
		CurrentRevision: workflow.CurrentRevision,
		EnabledRevision: enabled,
		CreatedAt:       workflow.CreatedAt,
		UpdatedAt:       workflow.UpdatedAt,
	})
}

func (r *LifecycleWorkflowRepository) FindRevision(ctx context.Context, tenantID, workflowID string, number int64) (*igdomain.LifecycleWorkflowRevision, error) {
	row, err := New(r.Pool).FindLifecycleWorkflowRevision(ctx, FindLifecycleWorkflowRevisionParams{
		TenantID: tenantID, WorkflowID: workflowID, Revision: number,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	revision := &igdomain.LifecycleWorkflowRevision{
		WorkflowID: row.WorkflowID,
		TenantID:   row.TenantID,
		Revision:   row.Revision,
		CreatedAt:  row.CreatedAt,
	}
	if err := json.Unmarshal(row.Trigger, &revision.Trigger); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(row.Actions, &revision.Actions); err != nil {
		return nil, err
	}
	return revision, revision.Validate()
}

func (r *LifecycleWorkflowRepository) SaveRevision(ctx context.Context, revision *igdomain.LifecycleWorkflowRevision) error {
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
	return New(r.Pool).SaveLifecycleWorkflowRevision(ctx, SaveLifecycleWorkflowRevisionParams{
		WorkflowID: revision.WorkflowID,
		TenantID:   revision.TenantID,
		Revision:   revision.Revision,
		Trigger:    trigger,
		Actions:    actions,
		CreatedAt:  revision.CreatedAt,
	})
}
