package domain

import (
	"testing"
	"time"
)

func TestEvaluateWorkflowTrigger(t *testing.T) {
	department := "engineering"
	before := &User{Lifecycle: UserLifecycle{Status: UserStatusActive}, Attributes: map[string]AttributeValue{"department": {Type: AttributeTypeString, String: &department}}}
	afterDepartment := "security"
	after := &User{Lifecycle: UserLifecycle{Status: UserStatusActive}, Attributes: map[string]AttributeValue{"department": {Type: AttributeTypeString, String: &afterDepartment}}}
	trigger := WorkflowTrigger{Kind: WorkflowTriggerUserAttributesChanged, WatchedAttributes: []string{"department"}, Filters: []WorkflowFilter{{Field: "department", Operator: WorkflowFilterEqual, Value: "security"}}}
	if _, ok := EvaluateWorkflowTrigger(trigger, before, after, []string{"department"}, ""); !ok {
		t.Fatal("trigger did not match changed post-state")
	}
	if _, ok := EvaluateWorkflowTrigger(trigger, before, after, []string{"department"}, "run-origin"); ok {
		t.Fatal("workflow-origin mutation must be suppressed")
	}
}

func TestLifecycleWorkflowTransitions(t *testing.T) {
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	w := LifecycleWorkflow{ID: "wf", TenantID: "tenant", Name: "Joiner", Status: LifecycleWorkflowDraft, CurrentRevision: 2, CreatedAt: now, UpdatedAt: now}
	if err := w.Enable(2, now); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if err := w.Delete(now); err != nil {
		t.Fatalf("Delete enabled workflow: %v", err)
	}
	if w.Status != LifecycleWorkflowArchived || w.EnabledRevision != nil {
		t.Fatalf("deleted workflow = %#v", w)
	}
	if err := w.Enable(2, now); err == nil {
		t.Fatal("enable deleted workflow = nil, want error")
	}
}

func TestLifecycleWorkflowDeleteDraft(t *testing.T) {
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	w := LifecycleWorkflow{ID: "wf", TenantID: "tenant", Name: "Draft", Status: LifecycleWorkflowDraft, CurrentRevision: 1, CreatedAt: now, UpdatedAt: now}
	if err := w.Delete(now); err != nil {
		t.Fatalf("Delete draft workflow: %v", err)
	}
}

func TestPlanWorkflowRunFreezesRevision(t *testing.T) {
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	revision := LifecycleWorkflowRevision{WorkflowID: "wf", TenantID: "tenant", Revision: 1, Trigger: WorkflowTrigger{Kind: WorkflowTriggerUserCreated}, Actions: []WorkflowAction{{Kind: WorkflowActionDisableUser}}, CreatedAt: now}
	run, steps, err := PlanWorkflowRun("run", revision, "user", "source", TriggerMatch{Kind: WorkflowTriggerUserCreated}, now)
	if err != nil {
		t.Fatalf("PlanWorkflowRun: %v", err)
	}
	revision.Actions[0].Kind = WorkflowActionEnableUser
	if run.Actions[0].Kind != WorkflowActionDisableUser || steps[0].Action.Kind != WorkflowActionDisableUser {
		t.Fatal("run plan must retain its original action snapshot")
	}
}
