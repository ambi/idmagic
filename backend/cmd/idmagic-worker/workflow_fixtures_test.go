package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idgovernance"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

// fixture は内部型を公開せず、保存契約へ同じ JSON 値を渡す。
type workflowFixture struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	CurrentRevision int64     `json:"current_revision"`
	EnabledRevision *int64    `json:"enabled_revision,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type workflowActionFixture struct {
	Kind           string                   `json:"kind"`
	GroupID        string                   `json:"group_id,omitempty"`
	ApplicationID  string                   `json:"application_id,omitempty"`
	RequiredAction idmdomain.RequiredAction `json:"required_action,omitempty"`
}

type workflowRunFixture struct {
	ID                 string                  `json:"id"`
	TenantID           string                  `json:"tenant_id"`
	WorkflowID         string                  `json:"workflow_id"`
	Revision           int64                   `json:"revision"`
	SourceOccurrenceID string                  `json:"source_occurrence_id"`
	TargetUserID       string                  `json:"target_user_id"`
	TriggerKind        string                  `json:"trigger_kind"`
	Actions            []workflowActionFixture `json:"actions"`
	Status             string                  `json:"status"`
	TriggeredAt        time.Time               `json:"triggered_at"`
}

type workflowStepFixture struct {
	RunID   string                `json:"run_id"`
	Action  workflowActionFixture `json:"action"`
	Outcome string                `json:"outcome"`
}

func decodeWorkflowFixture(t *testing.T, fixture, target any) {
	t.Helper()
	data, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func saveWorkflowFixture(ctx context.Context, t *testing.T, m idgovernance.Module, fixture *workflowFixture) error {
	t.Helper()
	value, err := m.LifecycleWorkflowRepo.Find(ctx, fixture.TenantID, fixture.ID)
	if err != nil {
		return err
	}
	decodeWorkflowFixture(t, fixture, &value)
	return m.LifecycleWorkflowRepo.Save(ctx, value)
}

func saveWorkflowRunFixture(ctx context.Context, t *testing.T, m idgovernance.Module, fixture *workflowRunFixture, steps []workflowStepFixture) error {
	t.Helper()
	value, err := m.LifecycleWorkflowRunRepo.FindRun(ctx, fixture.TenantID, fixture.ID)
	if err != nil {
		return err
	}
	storedSteps, err := m.LifecycleWorkflowRunRepo.ListSteps(ctx, fixture.TenantID, fixture.ID)
	if err != nil {
		return err
	}
	decodeWorkflowFixture(t, fixture, &value)
	decodeWorkflowFixture(t, steps, &storedSteps)
	_, err = m.LifecycleWorkflowRunRepo.SaveRun(ctx, value, storedSteps)
	return err
}
