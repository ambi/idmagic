package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	idmmemory "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/memory"
	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	"github.com/ambi/idmagic/backend/identitymanagement/usecases"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func workflowContext() context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-a"}, "", "")
}

func workflowInput() usecases.CreateLifecycleWorkflowInput {
	return usecases.CreateLifecycleWorkflowInput{Name: "Joiner", Trigger: idmdomain.WorkflowTrigger{Kind: idmdomain.WorkflowTriggerUserCreated}, Actions: []idmdomain.WorkflowAction{{Kind: idmdomain.WorkflowActionDisableUser}}, Now: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)}
}

func TestLifecycleWorkflowCreateUpdateAndTransitions(t *testing.T) {
	deps := usecases.LifecycleWorkflowDeps{Repo: idmmemory.NewLifecycleWorkflowRepository()}
	workflow, err := usecases.CreateLifecycleWorkflow(workflowContext(), deps, workflowInput())
	if err != nil {
		t.Fatalf("CreateLifecycleWorkflow: %v", err)
	}
	updated, err := usecases.UpdateLifecycleWorkflow(workflowContext(), deps, usecases.UpdateLifecycleWorkflowInput{WorkflowID: workflow.ID, ExpectedRevision: 1, Name: "Joiner v2", Trigger: idmdomain.WorkflowTrigger{Kind: idmdomain.WorkflowTriggerUserCreated}, Actions: []idmdomain.WorkflowAction{{Kind: idmdomain.WorkflowActionEnableUser}}})
	if err != nil || updated.CurrentRevision != 2 {
		t.Fatalf("UpdateLifecycleWorkflow = %#v, %v", updated, err)
	}
	if _, err := usecases.EnableLifecycleWorkflow(workflowContext(), deps, workflow.ID, 1, time.Time{}); !errors.Is(err, usecases.ErrWorkflowRevisionConflict) {
		t.Fatalf("stale enable error = %v", err)
	}
	if _, err := usecases.EnableLifecycleWorkflow(workflowContext(), deps, workflow.ID, 2, time.Time{}); err != nil {
		t.Fatalf("EnableLifecycleWorkflow: %v", err)
	}
	if _, err := usecases.DisableLifecycleWorkflow(workflowContext(), deps, workflow.ID, time.Time{}); err != nil {
		t.Fatalf("DisableLifecycleWorkflow: %v", err)
	}
	if _, err := usecases.ArchiveLifecycleWorkflow(workflowContext(), deps, workflow.ID, time.Time{}); err != nil {
		t.Fatalf("ArchiveLifecycleWorkflow: %v", err)
	}
}

func TestLifecycleWorkflowTenantIsolation(t *testing.T) {
	deps := usecases.LifecycleWorkflowDeps{Repo: idmmemory.NewLifecycleWorkflowRepository()}
	workflow, err := usecases.CreateLifecycleWorkflow(workflowContext(), deps, workflowInput())
	if err != nil {
		t.Fatal(err)
	}
	other := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-b"}, "", "")
	if _, err := usecases.DisableLifecycleWorkflow(other, deps, workflow.ID, time.Time{}); !errors.Is(err, usecases.ErrLifecycleWorkflowNotFound) {
		t.Fatalf("cross-tenant access error = %v", err)
	}
}
