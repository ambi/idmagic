package domain

import (
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

func TestEvaluateWorkflowTrigger(t *testing.T) {
	department := "engineering"
	before := &idmdomain.User{Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, Attributes: map[string]idmdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &department}}}
	afterDepartment := "security"
	after := &idmdomain.User{Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, Attributes: map[string]idmdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &afterDepartment}}}
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

func TestEvaluateWorkflowAction(t *testing.T) {
	activeUser := &idmdomain.User{Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive, RequiredActions: []idmdomain.RequiredAction{idmdomain.RequiredActionVerifyEmail}}}
	disabledUser := &idmdomain.User{Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusDisabled}}
	tests := []struct {
		name        string
		action      WorkflowAction
		user        *idmdomain.User
		state       WorkflowActionState
		wantOutcome WorkflowActionOutcome
		wantReason  string
	}{
		{"add group member blocked when group missing", WorkflowAction{Kind: WorkflowActionAddGroupMember, GroupID: "g"}, activeUser, WorkflowActionState{}, WorkflowActionBlocked, "resource_not_found"},
		{"add group member no-op when already a member", WorkflowAction{Kind: WorkflowActionAddGroupMember, GroupID: "g"}, activeUser, WorkflowActionState{GroupExists: true, UserIsGroupMember: true}, WorkflowActionNoOp, ""},
		{"add group member would change when not a member", WorkflowAction{Kind: WorkflowActionAddGroupMember, GroupID: "g"}, activeUser, WorkflowActionState{GroupExists: true}, WorkflowActionWouldChange, ""},
		{"remove group member no-op when not a member", WorkflowAction{Kind: WorkflowActionRemoveGroupMember, GroupID: "g"}, activeUser, WorkflowActionState{GroupExists: true}, WorkflowActionNoOp, ""},
		{"remove group member would change when a member", WorkflowAction{Kind: WorkflowActionRemoveGroupMember, GroupID: "g"}, activeUser, WorkflowActionState{GroupExists: true, UserIsGroupMember: true}, WorkflowActionWouldChange, ""},
		{"assign application blocked when application missing", WorkflowAction{Kind: WorkflowActionAssignApplication, ApplicationID: "a"}, activeUser, WorkflowActionState{}, WorkflowActionBlocked, "resource_not_found"},
		{"assign application no-op when already assigned", WorkflowAction{Kind: WorkflowActionAssignApplication, ApplicationID: "a"}, activeUser, WorkflowActionState{ApplicationExists: true, UserIsAssigned: true}, WorkflowActionNoOp, ""},
		{"assign application would change when not assigned", WorkflowAction{Kind: WorkflowActionAssignApplication, ApplicationID: "a"}, activeUser, WorkflowActionState{ApplicationExists: true}, WorkflowActionWouldChange, ""},
		{"unassign application no-op when not assigned", WorkflowAction{Kind: WorkflowActionUnassignApplication, ApplicationID: "a"}, activeUser, WorkflowActionState{ApplicationExists: true}, WorkflowActionNoOp, ""},
		{"unassign application would change when assigned", WorkflowAction{Kind: WorkflowActionUnassignApplication, ApplicationID: "a"}, activeUser, WorkflowActionState{ApplicationExists: true, UserIsAssigned: true}, WorkflowActionWouldChange, ""},
		{"set required action no-op when already set", WorkflowAction{Kind: WorkflowActionSetRequiredAction, RequiredAction: idmdomain.RequiredActionVerifyEmail}, activeUser, WorkflowActionState{}, WorkflowActionNoOp, ""},
		{"set required action would change when unset", WorkflowAction{Kind: WorkflowActionSetRequiredAction, RequiredAction: idmdomain.RequiredActionUpdatePassword}, activeUser, WorkflowActionState{}, WorkflowActionWouldChange, ""},
		{"clear required action no-op when already unset", WorkflowAction{Kind: WorkflowActionClearRequiredAction, RequiredAction: idmdomain.RequiredActionUpdatePassword}, activeUser, WorkflowActionState{}, WorkflowActionNoOp, ""},
		{"clear required action would change when set", WorkflowAction{Kind: WorkflowActionClearRequiredAction, RequiredAction: idmdomain.RequiredActionVerifyEmail}, activeUser, WorkflowActionState{}, WorkflowActionWouldChange, ""},
		{"enable user no-op when already active", WorkflowAction{Kind: WorkflowActionEnableUser}, activeUser, WorkflowActionState{}, WorkflowActionNoOp, ""},
		{"enable user would change when disabled", WorkflowAction{Kind: WorkflowActionEnableUser}, disabledUser, WorkflowActionState{}, WorkflowActionWouldChange, ""},
		{"disable user no-op when already disabled", WorkflowAction{Kind: WorkflowActionDisableUser}, disabledUser, WorkflowActionState{}, WorkflowActionNoOp, ""},
		{"disable user would change when active", WorkflowAction{Kind: WorkflowActionDisableUser}, activeUser, WorkflowActionState{}, WorkflowActionWouldChange, ""},
		{"send email blocked when not sendable", WorkflowAction{Kind: WorkflowActionSendEmail, TemplateKey: "welcome"}, activeUser, WorkflowActionState{}, WorkflowActionBlocked, "notification_unavailable"},
		{"send email would change when sendable", WorkflowAction{Kind: WorkflowActionSendEmail, TemplateKey: "welcome"}, activeUser, WorkflowActionState{EmailSendable: true}, WorkflowActionWouldChange, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outcome, reason := EvaluateWorkflowAction(tc.action, tc.user, tc.state)
			if outcome != tc.wantOutcome || reason != tc.wantReason {
				t.Fatalf("EvaluateWorkflowAction() = (%q, %q), want (%q, %q)", outcome, reason, tc.wantOutcome, tc.wantReason)
			}
		})
	}
}

func TestEvaluateWorkflowFilters(t *testing.T) {
	department := "engineering"
	user := &idmdomain.User{Attributes: map[string]idmdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &department}}}
	matching := []WorkflowFilter{{Field: "department", Operator: WorkflowFilterEqual, Value: "engineering"}}
	if !EvaluateWorkflowFilters(matching, user) {
		t.Fatal("filters on the User's current attributes must match")
	}
	mismatching := []WorkflowFilter{{Field: "department", Operator: WorkflowFilterEqual, Value: "security"}}
	if EvaluateWorkflowFilters(mismatching, user) {
		t.Fatal("filters that do not match the User's current attributes must not match")
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
