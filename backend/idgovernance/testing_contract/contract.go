// Package testing_contract defines the shared identity-governance persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idgovernance/domain"
	"github.com/ambi/idmagic/backend/idgovernance/ports"
)

type Fixture struct {
	Workflows  ports.LifecycleWorkflowRepository
	TenantA    string
	TenantB    string
	WorkflowID string
	OtherID    string
	Now        time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	description := "offboarding"
	enabledRevision := int64(1)
	workflow := &domain.LifecycleWorkflow{
		ID: f.WorkflowID, TenantID: f.TenantA, Name: "Leaver", Description: &description,
		Status: domain.LifecycleWorkflowEnabled, CurrentRevision: 1, EnabledRevision: &enabledRevision,
		CreatedAt: f.Now, UpdatedAt: f.Now,
	}
	if err := f.Workflows.Save(ctx, workflow); err != nil {
		t.Fatalf("Save: %v", err)
	}
	otherWorkflow := &domain.LifecycleWorkflow{
		ID: f.OtherID, TenantID: f.TenantA, Name: "Joiner", Status: domain.LifecycleWorkflowDraft,
		CurrentRevision: 1, CreatedAt: f.Now, UpdatedAt: f.Now,
	}
	if err := f.Workflows.Save(ctx, otherWorkflow); err != nil {
		t.Fatalf("Save second workflow: %v", err)
	}
	found, err := f.Workflows.Find(ctx, f.TenantA, f.WorkflowID)
	if err != nil || found == nil || found.Name != workflow.Name || found.EnabledRevision == nil || *found.EnabledRevision != 1 {
		t.Fatalf("Find = (%+v, %v)", found, err)
	}
	if leaked, err := f.Workflows.Find(ctx, f.TenantB, f.WorkflowID); err != nil || leaked != nil {
		t.Fatalf("other tenant Find = (%+v, %v)", leaked, err)
	}
	listed, err := f.Workflows.List(ctx, f.TenantA)
	if err != nil || len(listed) != 2 || listed[0].ID != f.OtherID || listed[1].ID != f.WorkflowID {
		t.Fatalf("List = (%+v, %v)", listed, err)
	}
	revision := &domain.LifecycleWorkflowRevision{
		WorkflowID: f.WorkflowID, TenantID: f.TenantA, Revision: 1,
		Trigger: domain.WorkflowTrigger{Kind: domain.WorkflowTriggerUserCreated},
		Actions: []domain.WorkflowAction{{Kind: domain.WorkflowActionDisableUser}}, CreatedAt: f.Now,
	}
	if err := f.Workflows.SaveRevision(ctx, revision); err != nil {
		t.Fatalf("SaveRevision: %v", err)
	}
	foundRevision, err := f.Workflows.FindRevision(ctx, f.TenantA, f.WorkflowID, 1)
	if err != nil || foundRevision == nil || len(foundRevision.Actions) != 1 || foundRevision.Actions[0].Kind != domain.WorkflowActionDisableUser {
		t.Fatalf("FindRevision = (%+v, %v)", foundRevision, err)
	}
	if missing, err := f.Workflows.FindRevision(ctx, f.TenantA, f.WorkflowID, 2); err != nil || missing != nil {
		t.Fatalf("missing FindRevision = (%+v, %v)", missing, err)
	}
}
