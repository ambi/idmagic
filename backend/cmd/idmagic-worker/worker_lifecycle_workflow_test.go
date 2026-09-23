package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	"github.com/ambi/idmagic/backend/idgovernance"
	igmemory "github.com/ambi/idmagic/backend/idgovernance/db_memory"
	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	igusecases "github.com/ambi/idmagic/backend/idgovernance/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/shared/events/sinks_console"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// worker が組み立てる実行ハンドラーは、無効化されたワークフローの WorkflowRun を始めずに打ち切る。
// ワークフローの保存先を渡し忘れると、この打ち切りは黙って効かなくなる。
func TestWorkerLifecycleWorkflowHandlerCancelsRunsOfADisabledWorkflow(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	workflows := igmemory.NewLifecycleWorkflowRepository()
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "unused", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now})
	if err := workflows.Save(ctx, &igdomain.LifecycleWorkflow{ID: "workflow-1", TenantID: "tenant-a", Name: "Leaver", Status: igdomain.LifecycleWorkflowDisabled, CurrentRevision: 1, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	action := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionDisableUser}
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "occurrence-1", TargetUserID: "user-1", TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{action}, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
	if _, err := runs.SaveRun(ctx, run, []igdomain.WorkflowStep{{RunID: run.ID, Action: action, Outcome: igdomain.WorkflowStepPending}}); err != nil {
		t.Fatal(err)
	}
	deps := &bootstrap.Dependencies{
		IdManagement: idmanagement.Module{UserRepo: users, GroupRepo: groupmemory.NewGroupRepository()},
		IdGovernance: idgovernance.Module{LifecycleWorkflowRepo: workflows, LifecycleWorkflowRunRepo: runs},
		OAuth2:       oauth2.Module{EventSink: sinks_console.NewConsoleSink()},
	}
	logger := logging.New(os.Stderr, logging.ParseLevel("error"), "idmagic-worker-test", "test")
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}

	handler := igusecases.LifecycleWorkflowRunHandler(lifecycleWorkflowExecutorDeps(deps, logger))
	if _, err := handler(ctx, &jobsdomain.Job{TenantID: "tenant-a", Params: params, Attempts: 1, MaxAttempts: 5}); err != nil {
		t.Fatal(err)
	}

	if stored, err := runs.FindRun(ctx, "tenant-a", run.ID); err != nil || stored.Status != igdomain.WorkflowRunCanceled {
		t.Fatalf("run = %+v, %v; want canceled", stored, err)
	}
	if user, err := users.FindBySub(ctx, "user-1"); err != nil || user.Lifecycle.Status != idmdomain.UserStatusActive {
		t.Fatalf("user = %+v, %v; want the disable_user step never to run", user, err)
	}
}

// recordingSink は worker の Emit が監査へ流すイベントの種類を集める。
type recordingSink struct{ types []string }

func (s *recordingSink) Emit(_ context.Context, event spec.DomainEvent) error {
	s.types = append(s.types, event.EventType())
	return nil
}

func (s *recordingSink) count(eventType string) int {
	n := 0
	for _, t := range s.types {
		if t == eventType {
			n++
		}
	}
	return n
}

//spec:covers REQ-APPLICATION-014, EX-APPLICATION-014-01, EX-APPLICATION-014-02: 動的グループを介したグループ割り当てを持つ User に、worker が組み立てた実行ハンドラーで assign_application を実行すると直接割り当てが作られ、同じ visibility での再実行は no_op で ApplicationAssigned を増やさず、unassign_application で直接割り当てだけが消え、グループ割り当ての行は変わらず、フェデレーションの関門が許可を返し続けることを固定する。
func TestWorkerLifecycleWorkflowAppliesTheDirectAssignmentBesideAGroupAssignment(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	workflows := igmemory.NewLifecycleWorkflowRepository()
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{ID: "alice", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "unused", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now})
	groups := groupmemory.NewGroupRepository()
	if err := groups.Save(ctx, &groupdomain.Group{ID: "engineering", TenantID: "tenant-a", Name: "Engineering", MembershipType: groupdomain.GroupMembershipDynamic, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := groups.SaveDynamicRule(ctx, &groupdomain.DynamicGroupRule{GroupID: "engineering", TenantID: "tenant-a", Expression: `department == "Engineering"`, Enabled: true, Version: 1, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := groups.AddMember(ctx, &groupdomain.GroupMember{GroupID: "engineering", UserID: "alice", Source: groupdomain.MembershipSourceDynamicRule, RuleVersion: new(int64(1)), CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	applications := appmemory.NewApplicationRepository()
	if err := applications.Save(ctx, &appdomain.Application{
		TenantID: "tenant-a", ID: "portal", Name: "Portal", Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
		Protocol: &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: "portal-client"}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	assignments := appmemory.NewApplicationAssignmentRepository()
	groupAssignment := appdomain.ApplicationAssignment{TenantID: "tenant-a", ApplicationID: "portal", SubjectType: appdomain.AssignmentSubjectGroup, SubjectID: "engineering", Visibility: appdomain.AssignmentVisible, CreatedAt: now, UpdatedAt: now}
	if err := assignments.Save(ctx, &groupAssignment); err != nil {
		t.Fatal(err)
	}
	if err := workflows.Save(ctx, &igdomain.LifecycleWorkflow{ID: "workflow-1", TenantID: "tenant-a", Name: "Joiner", Status: igdomain.LifecycleWorkflowEnabled, CurrentRevision: 1, EnabledRevision: new(int64(1)), CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	sink := &recordingSink{}
	deps := &bootstrap.Dependencies{
		IdManagement: idmanagement.Module{UserRepo: users, GroupRepo: groups},
		IdGovernance: idgovernance.Module{LifecycleWorkflowRepo: workflows, LifecycleWorkflowRunRepo: runs},
		Application:  application.Module{Repo: applications, AssignmentRepo: assignments},
		OAuth2:       oauth2.Module{EventSink: sink},
	}
	logger := logging.New(os.Stderr, logging.ParseLevel("error"), "idmagic-worker-test", "test")
	handler := igusecases.LifecycleWorkflowRunHandler(lifecycleWorkflowExecutorDeps(deps, logger))
	execute := func(runID string, action igdomain.WorkflowAction) igdomain.WorkflowStepOutcome {
		t.Helper()
		run := &igdomain.WorkflowRun{ID: runID, TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: runID, TargetUserID: "alice", TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{action}, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
		if _, err := runs.SaveRun(ctx, run, []igdomain.WorkflowStep{{RunID: runID, Action: action, Outcome: igdomain.WorkflowStepPending}}); err != nil {
			t.Fatal(err)
		}
		params, err := json.Marshal(map[string]string{"run_id": runID})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := handler(ctx, &jobsdomain.Job{TenantID: "tenant-a", Params: params, Attempts: 1, MaxAttempts: 1}); err != nil {
			t.Fatal(err)
		}
		steps, err := runs.ListSteps(ctx, "tenant-a", runID)
		if err != nil || len(steps) != 1 {
			t.Fatalf("steps = %+v, %v", steps, err)
		}
		return steps[0].Outcome
	}
	rowOf := func(subjectType appdomain.AssignmentSubjectType, subjectID string) *appdomain.ApplicationAssignment {
		t.Helper()
		rows, err := assignments.ListBySubjects(ctx, "tenant-a", []appports.SubjectRef{{Type: subjectType, ID: subjectID}})
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range rows {
			if row.ApplicationID == "portal" {
				return row
			}
		}
		return nil
	}
	assign := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAssignApplication, ApplicationID: "portal"}

	if got := execute("run-assign", assign); got != igdomain.WorkflowStepChanged {
		t.Fatalf("assign step = %s, want changed", got)
	}
	if direct := rowOf(appdomain.AssignmentSubjectUser, "alice"); direct == nil || direct.Visibility != appdomain.AssignmentVisible {
		t.Fatalf("direct assignment = %+v, want a visible user assignment", direct)
	}
	if got := execute("run-assign-again", assign); got != igdomain.WorkflowStepNoop {
		t.Fatalf("repeated assign step = %s, want no_op", got)
	}
	if n := sink.count("ApplicationAssigned"); n != 1 {
		t.Fatalf("ApplicationAssigned emitted %d times, want 1 for two identical runs", n)
	}
	if got := execute("run-unassign", igdomain.WorkflowAction{Kind: igdomain.WorkflowActionUnassignApplication, ApplicationID: "portal"}); got != igdomain.WorkflowStepChanged {
		t.Fatalf("unassign step = %s, want changed", got)
	}
	if direct := rowOf(appdomain.AssignmentSubjectUser, "alice"); direct != nil {
		t.Fatalf("direct assignment after unassign = %+v, want none", direct)
	}
	if n := sink.count("ApplicationUnassigned"); n != 1 {
		t.Fatalf("ApplicationUnassigned emitted %d times, want 1", n)
	}
	if row := rowOf(appdomain.AssignmentSubjectGroup, "engineering"); row == nil || *row != groupAssignment {
		t.Fatalf("group assignment = %+v, want unchanged %+v", row, groupAssignment)
	}
	decision, err := deps.Application.Gate(groups, 0).EvaluateApplicationAccess(ctx, "tenant-a", appdomain.ApplicationProtocolOIDC, "portal-client", "alice", nil, "")
	if err != nil || !decision.Allowed {
		t.Fatalf("federation decision = %+v, %v; want allowed through the group assignment", decision, err)
	}
}
