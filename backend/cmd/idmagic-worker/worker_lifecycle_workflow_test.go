package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	"github.com/ambi/idmagic/backend/idgovernance"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/events/sinks_console"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/sharedsignals"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	sharedsignalsports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	"github.com/ambi/idmagic/backend/signingkeys"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// worker が組み立てる実行ハンドラーは、無効化されたワークフローの WorkflowRun を始めずに打ち切る。
// ワークフローの保存先を渡し忘れると、この打ち切りは黙って効かなくなる。
func TestWorkerLifecycleWorkflowHandlerCancelsRunsOfADisabledWorkflow(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	users := idmanagement.NewMemoryModule().UserRepo
	governance := idgovernance.NewMemoryModule(users)
	workflows, runs := governance.LifecycleWorkflowRepo, governance.LifecycleWorkflowRunRepo
	_ = users.Save(context.Background(), &userdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "unused", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now})
	if err := saveWorkflowFixture(ctx, t, governance, &workflowFixture{ID: "workflow-1", TenantID: "tenant-a", Name: "Leaver", Status: "disabled", CurrentRevision: 1, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	action := workflowActionFixture{Kind: "disable_user"}
	run := &workflowRunFixture{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "occurrence-1", TargetUserID: "user-1", TriggerKind: "user_created", Actions: []workflowActionFixture{action}, Status: "queued", TriggeredAt: now}
	if err := saveWorkflowRunFixture(ctx, t, governance, run, []workflowStepFixture{{RunID: run.ID, Action: action, Outcome: "pending"}}); err != nil {
		t.Fatal(err)
	}
	deps := &bootstrap.Dependencies{
		IdManagement: idmanagement.Module{UserRepo: users, GroupRepo: idmanagement.NewMemoryModule().GroupRepo},
		IdGovernance: idgovernance.Module{LifecycleWorkflowRepo: workflows, LifecycleWorkflowRunRepo: runs},
		OAuth2:       oauth2.Module{EventSink: sinks_console.NewConsoleSink()},
	}
	logger := logging.New(os.Stderr, logging.ParseLevel("error"), "idmagic-worker-test", "test")
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}

	handler := deps.IdGovernance.Executor(lifecycleWorkflowExecutorDeps(deps, logger))
	if _, err := handler(ctx, &jobsdomain.Job{TenantID: "tenant-a", Params: params, Attempts: 1, MaxAttempts: 5}); err != nil {
		t.Fatal(err)
	}

	if stored, err := runs.FindRun(ctx, "tenant-a", run.ID); err != nil || stored.Status != "canceled" {
		t.Fatalf("run = %+v, %v; want canceled", stored, err)
	}
	if user, err := users.FindBySub(ctx, "user-1"); err != nil || user.Lifecycle.Status != idmdomain.UserStatusActive {
		t.Fatalf("user = %+v, %v; want the disable_user step never to run", user, err)
	}
}

// recordingSink は worker の Emit が監査へ流すイベントの種類を集める。
type recordingSink struct{ types []string }

type workflowGroupNotifier struct{ calls int }

func (n *workflowGroupNotifier) NotifyGroupMutation(_ context.Context, _, _ string, _ groupports.ProvisioningTrigger, _ time.Time) error {
	n.calls++
	return nil
}

//spec:covers REQ-IDGOVERNANCE-003, REQ-IDGOVERNANCE-005, REQ-IDMANAGEMENT-015, REQ-IDMANAGEMENT-047: Worker の正式な組み立てからグループ所属と必須操作を変更するとイベントとグループ通知を伴い、同じ操作は no_op となり、別テナントの対象は変更されないことを固定する。
func TestWorkerLifecycleWorkflowIdentityActions(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	users := idmanagement.NewMemoryModule().UserRepo
	_ = users.Save(context.Background(), &userdomain.User{ID: "alice", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "unused", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now})
	_ = users.Save(context.Background(), &userdomain.User{ID: "foreign", TenantID: "tenant-b", PreferredUsername: "foreign", PasswordHash: "unused", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now})
	groups := idmanagement.NewMemoryModule().GroupRepo
	for _, group := range []*groupdomain.Group{
		{ID: "engineering", TenantID: "tenant-a", Name: "Engineering", MembershipType: groupdomain.GroupMembershipManual, CreatedAt: now, UpdatedAt: now},
		{ID: "foreign-group", TenantID: "tenant-b", Name: "Foreign", MembershipType: groupdomain.GroupMembershipManual, CreatedAt: now, UpdatedAt: now},
	} {
		if err := groups.Save(ctx, group); err != nil {
			t.Fatal(err)
		}
	}
	governance := idgovernance.NewMemoryModule(users)
	workflows, runs := governance.LifecycleWorkflowRepo, governance.LifecycleWorkflowRunRepo
	if err := saveWorkflowFixture(ctx, t, governance, &workflowFixture{ID: "workflow-1", TenantID: "tenant-a", Name: "Identity actions", Status: "enabled", CurrentRevision: 1, EnabledRevision: new(int64(1)), CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	sink := &recordingSink{}
	notifier := &workflowGroupNotifier{}
	deps := &bootstrap.Dependencies{
		IdManagement: idmanagement.Module{UserRepo: users, GroupRepo: groups, GroupProvisioningNotifier: notifier},
		IdGovernance: idgovernance.Module{LifecycleWorkflowRepo: workflows, LifecycleWorkflowRunRepo: runs},
		OAuth2:       oauth2.Module{EventSink: sink},
	}
	logger := logging.New(os.Stderr, logging.ParseLevel("error"), "idmagic-worker-test", "test")
	handler := deps.IdGovernance.Executor(lifecycleWorkflowExecutorDeps(deps, logger))
	execute := func(runID, userID string, action workflowActionFixture) string {
		t.Helper()
		run := &workflowRunFixture{ID: runID, TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: runID, TargetUserID: userID, TriggerKind: "user_created", Actions: []workflowActionFixture{action}, Status: "queued", TriggeredAt: now}
		if err := saveWorkflowRunFixture(ctx, t, governance, run, []workflowStepFixture{{RunID: runID, Action: action, Outcome: "pending"}}); err != nil {
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
		return string(steps[0].Outcome)
	}
	for i, tc := range []struct {
		action workflowActionFixture
		event  string
	}{
		{workflowActionFixture{Kind: "add_group_member", GroupID: "engineering"}, "GroupMemberAdded"},
		{workflowActionFixture{Kind: "set_required_action", RequiredAction: idmdomain.RequiredActionUpdatePassword}, "UserRequiredActionSet"},
		{workflowActionFixture{Kind: "clear_required_action", RequiredAction: idmdomain.RequiredActionUpdatePassword}, "UserRequiredActionCleared"},
		{workflowActionFixture{Kind: "remove_group_member", GroupID: "engineering"}, "GroupMemberRemoved"},
	} {
		runID := fmt.Sprintf("run-%d", i)
		if got := execute(runID, "alice", tc.action); got != "changed" {
			t.Fatalf("%s outcome = %s", tc.event, got)
		}
		if got := sink.count(tc.event); got != 1 {
			t.Fatalf("%s count = %d, want 1", tc.event, got)
		}
		if got := execute(runID+"-repeat", "alice", tc.action); got != "no_op" {
			t.Fatalf("repeat %s outcome = %s", tc.event, got)
		}
		if got := sink.count(tc.event); got != 1 {
			t.Fatalf("repeat %s count = %d", tc.event, got)
		}
	}
	if notifier.calls != 2 {
		t.Fatalf("group notifications = %d, want 2", notifier.calls)
	}
	before := len(sink.types)
	for i, tc := range []struct {
		user   string
		action workflowActionFixture
	}{
		{"foreign", workflowActionFixture{Kind: "set_required_action", RequiredAction: idmdomain.RequiredActionUpdatePassword}},
		{"alice", workflowActionFixture{Kind: "add_group_member", GroupID: "foreign-group"}},
	} {
		if got := execute(fmt.Sprintf("foreign-%d", i), tc.user, tc.action); got != "failed" {
			t.Fatalf("foreign outcome = %s", got)
		}
	}
	if sink.count("GroupMemberAdded") != 1 || sink.count("UserRequiredActionSet") != 1 || notifier.calls != 2 {
		t.Fatalf("foreign mutation emitted events or notifications (events before=%d, after=%d)", before, len(sink.types))
	}
}

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
	users := idmanagement.NewMemoryModule().UserRepo
	governance := idgovernance.NewMemoryModule(users)
	workflows, runs := governance.LifecycleWorkflowRepo, governance.LifecycleWorkflowRunRepo
	_ = users.Save(context.Background(), &userdomain.User{ID: "alice", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "unused", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now})
	groups := idmanagement.NewMemoryModule().GroupRepo
	if err := groups.Save(ctx, &groupdomain.Group{ID: "engineering", TenantID: "tenant-a", Name: "Engineering", MembershipType: groupdomain.GroupMembershipDynamic, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := groups.SaveDynamicRule(ctx, &groupdomain.DynamicGroupRule{GroupID: "engineering", TenantID: "tenant-a", Expression: `department == "Engineering"`, Enabled: true, Version: 1, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := groups.AddMember(ctx, &groupdomain.GroupMember{GroupID: "engineering", UserID: "alice", Source: groupdomain.MembershipSourceDynamicRule, RuleVersion: new(int64(1)), CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	applications := application.NewMemoryModule().Repo
	if err := applications.Save(ctx, &appdomain.Application{
		TenantID: "tenant-a", ID: "portal", Name: "Portal", Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
		Protocol: &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: "portal-client"}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	assignments := application.NewMemoryModule().AssignmentRepo
	groupAssignment := appdomain.ApplicationAssignment{TenantID: "tenant-a", ApplicationID: "portal", SubjectType: appdomain.AssignmentSubjectGroup, SubjectID: "engineering", Visibility: appdomain.AssignmentVisible, CreatedAt: now, UpdatedAt: now}
	if err := assignments.Save(ctx, &groupAssignment); err != nil {
		t.Fatal(err)
	}
	if err := saveWorkflowFixture(ctx, t, governance, &workflowFixture{ID: "workflow-1", TenantID: "tenant-a", Name: "Joiner", Status: "enabled", CurrentRevision: 1, EnabledRevision: new(int64(1)), CreatedAt: now, UpdatedAt: now}); err != nil {
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
	handler := deps.IdGovernance.Executor(lifecycleWorkflowExecutorDeps(deps, logger))
	execute := func(runID string, action workflowActionFixture) string {
		t.Helper()
		run := &workflowRunFixture{ID: runID, TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: runID, TargetUserID: "alice", TriggerKind: "user_created", Actions: []workflowActionFixture{action}, Status: "queued", TriggeredAt: now}
		if err := saveWorkflowRunFixture(ctx, t, governance, run, []workflowStepFixture{{RunID: runID, Action: action, Outcome: "pending"}}); err != nil {
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
		return string(steps[0].Outcome)
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
	assign := workflowActionFixture{Kind: "assign_application", ApplicationID: "portal"}

	if got := execute("run-assign", assign); got != "changed" {
		t.Fatalf("assign step = %s, want changed", got)
	}
	if direct := rowOf(appdomain.AssignmentSubjectUser, "alice"); direct == nil || direct.Visibility != appdomain.AssignmentVisible {
		t.Fatalf("direct assignment = %+v, want a visible user assignment", direct)
	}
	if got := execute("run-assign-again", assign); got != "no_op" {
		t.Fatalf("repeated assign step = %s, want no_op", got)
	}
	if n := sink.count("ApplicationAssigned"); n != 1 {
		t.Fatalf("ApplicationAssigned emitted %d times, want 1 for two identical runs", n)
	}
	if got := execute("run-unassign", workflowActionFixture{Kind: "unassign_application", ApplicationID: "portal"}); got != "changed" {
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
	decision, err := deps.Application.Gate(groups, 0, deps.SignInPolicy.ApplicationRepo, deps.SignInPolicy.DefaultRepo).EvaluateApplicationAccess(ctx, "tenant-a", appdomain.ApplicationProtocolOIDC, "portal-client", "alice", nil, "")
	if err != nil || !decision.Allowed {
		t.Fatalf("federation decision = %+v, %v; want allowed through the group assignment", decision, err)
	}
}

// REQ-IDMANAGEMENT-081 の主要な使い方のうち、worker の組み立ての経路。UserLifecycle を渡し忘れると
// disable_user が失敗し、IdManagement の依存を渡し忘れると所有する Agent の無効化が黙って止まる。
//
//spec:covers EX-IDMANAGEMENT-081-02: worker が組み立てた disable_user の手順が User を Disabled にして UserDisabled を発行し、所有する Agent を Disabled にすること。
func TestWorkerDisableUserStepStopsTheAgentsTheUserOwns(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	users := idmanagement.NewMemoryModule().UserRepo
	governance := idgovernance.NewMemoryModule(users)
	workflows, runs := governance.LifecycleWorkflowRepo, governance.LifecycleWorkflowRunRepo
	_ = users.Save(context.Background(), &userdomain.User{ID: "alice", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "unused", Roles: []string{}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now})
	agents := idmanagement.NewMemoryModule().AgentRepo
	if err := agents.Save(ctx, &agentdomain.Agent{ID: "deploy-bot", TenantID: "tenant-a", Name: "deploy-bot", Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "alice", Status: idmdomain.AgentStatusActive, Roles: []string{}, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := saveWorkflowFixture(ctx, t, governance, &workflowFixture{ID: "workflow-1", TenantID: "tenant-a", Name: "Leaver", Status: "enabled", CurrentRevision: 1, EnabledRevision: new(int64(1)), CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	action := workflowActionFixture{Kind: "disable_user"}
	run := &workflowRunFixture{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "occurrence-1", TargetUserID: "alice", TriggerKind: "user_created", Actions: []workflowActionFixture{action}, Status: "queued", TriggeredAt: now}
	if err := saveWorkflowRunFixture(ctx, t, governance, run, []workflowStepFixture{{RunID: run.ID, Action: action, Outcome: "pending"}}); err != nil {
		t.Fatal(err)
	}
	sink := &recordingSink{}
	deps := &bootstrap.Dependencies{
		IdManagement: idmanagement.Module{UserRepo: users, GroupRepo: idmanagement.NewMemoryModule().GroupRepo, AgentRepo: agents},
		IdGovernance: idgovernance.Module{LifecycleWorkflowRepo: workflows, LifecycleWorkflowRunRepo: runs},
		OAuth2:       oauth2.Module{EventSink: sink},
	}
	logger := logging.New(os.Stderr, logging.ParseLevel("error"), "idmagic-worker-test", "test")
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := deps.IdGovernance.Executor(lifecycleWorkflowExecutorDeps(deps, logger))(ctx, &jobsdomain.Job{TenantID: "tenant-a", Params: params, Attempts: 1, MaxAttempts: 1}); err != nil {
		t.Fatal(err)
	}

	if user, err := users.FindBySub(ctx, "alice"); err != nil || user.Lifecycle.Status != idmdomain.UserStatusDisabled {
		t.Fatalf("user = %+v, %v; want disabled", user, err)
	}
	if agent, err := agents.FindByID(ctx, "tenant-a", "deploy-bot"); err != nil || agent.Status != idmdomain.AgentStatusDisabled {
		t.Fatalf("agent = %+v, %v; want disabled", agent, err)
	}
	if sink.count("UserDisabled") != 1 || sink.count("AgentDisabled") != 1 {
		t.Fatalf("UserDisabled=%d AgentDisabled=%d, want 1 each", sink.count("UserDisabled"), sink.count("AgentDisabled"))
	}
}

// revocationFixture は、Agent を二つ所有する alice を disable_user で止める、worker の組み立ての経路である。
// 二つの Agent はそれぞれ OAuth2 クライアントに紐づき、SharedSignals の Repository と署名鍵は実物のメモリ実装を使う。
type revocationFixture struct {
	deps     *bootstrap.Dependencies
	sink     *recordingSink
	epochs   sharedsignalsports.AgentRevocationEpochRepository
	streams  sharedsignalsports.SsfStreamRepository
	configs  sharedsignalsports.SsfTransmitterConfigRepository
	delivery sharedsignalsports.SecurityEventDeliveryRepository
	agentIDs []string
}

func newRevocationFixture(t *testing.T) *revocationFixture {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	users := idmanagement.NewMemoryModule().UserRepo
	_ = users.Save(context.Background(), &userdomain.User{ID: "alice", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "unused", Roles: []string{}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now})
	agents := idmanagement.NewMemoryModule().AgentRepo
	f := &revocationFixture{
		sink:   &recordingSink{},
		epochs: sharedsignals.NewAgentRevocationEpochRepository(), streams: sharedsignals.NewSsfStreamRepository(),
		configs: sharedsignals.NewSsfTransmitterConfigRepository(), delivery: sharedsignals.NewSecurityEventDeliveryRepository(),
		agentIDs: []string{"deploy-bot", "report-bot"},
	}
	for _, id := range f.agentIDs {
		if err := agents.Save(ctx, &agentdomain.Agent{ID: id, TenantID: "tenant-a", Name: id, Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "alice", Status: idmdomain.AgentStatusActive, Roles: []string{}, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
		if _, err := agents.AddBinding(ctx, &agentdomain.AgentCredentialBinding{AgentID: id, ClientID: id + "-client", CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	governance := idgovernance.NewMemoryModule(users)
	if err := saveWorkflowFixture(ctx, t, governance, &workflowFixture{ID: "workflow-1", TenantID: "tenant-a", Name: "Leaver", Status: "enabled", CurrentRevision: 1, EnabledRevision: new(int64(1)), CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	keyStore, err := signingkeys.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	f.deps = &bootstrap.Dependencies{
		IdManagement: idmanagement.Module{UserRepo: users, GroupRepo: idmanagement.NewMemoryModule().GroupRepo, AgentRepo: agents},
		IdGovernance: governance,
		OAuth2:       oauth2.Module{EventSink: f.sink},
		SigningKeys:  signingkeys.Module{KeyStore: keyStore},
		SharedSignals: sharedsignals.Module{
			RevocationEpochRepo: f.epochs, StreamRepo: f.streams, TransmitterConfigRepo: f.configs, DeliveryRepo: f.delivery,
		},
		Issuer: "https://idp.example",
	}
	return f
}

// disableAlice は disable_user だけを持つ WorkflowRun を、worker が組み立てた実行ハンドラーで実行する。
func (f *revocationFixture) disableAlice(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	action := workflowActionFixture{Kind: "disable_user"}
	run := &workflowRunFixture{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "occurrence-1", TargetUserID: "alice", TriggerKind: "user_created", Actions: []workflowActionFixture{action}, Status: "queued", TriggeredAt: time.Now().UTC()}
	if err := saveWorkflowRunFixture(ctx, t, f.deps.IdGovernance, run, []workflowStepFixture{{RunID: run.ID, Action: action, Outcome: "pending"}}); err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}
	logger := logging.New(os.Stderr, logging.ParseLevel("error"), "idmagic-worker-test", "test")
	if _, err := f.deps.IdGovernance.Executor(lifecycleWorkflowExecutorDeps(f.deps, logger))(ctx, &jobsdomain.Job{TenantID: "tenant-a", Params: params, Attempts: 1, MaxAttempts: 1}); err != nil {
		t.Fatal(err)
	}
	steps, err := f.deps.IdGovernance.LifecycleWorkflowRunRepo.ListSteps(ctx, "tenant-a", run.ID)
	if err != nil || len(steps) != 1 || steps[0].Outcome != "changed" {
		t.Fatalf("steps = %+v, %v; want one changed disable_user step", steps, err)
	}
}

// 管理 API では閉じていた経路が、worker で実行するワークフローでも閉じることを確かめる。
// 失効エポックの判定はイントロスペクションと Bearer の検証が共有する AccessTokenIsRevoked で読む。
//
//spec:covers REQ-PLATFORM-001: worker が組み立てた disable_user の手順で User を止めると、所有する二つの Agent の失効エポックが同じ時刻へ進み、それより前に発行されたトークンが失効と判定され、AgentAccessRevoked が Agent ごとに 1 件発行されることを固定する。
func TestWorkerDisableUserStepRevokesTheTokensOfTheAgentsTheUserOwns(t *testing.T) {
	f := newRevocationFixture(t)
	issuedAt := time.Now().UTC().Add(-time.Minute)

	f.disableAlice(t)

	ctx := tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-a"}, "", "")
	var epochs []time.Time
	for _, id := range f.agentIDs {
		epoch, err := f.epochs.FindByAgent(ctx, "tenant-a", id)
		if err != nil || epoch == nil {
			t.Fatalf("revocation epoch of %s = %+v, %v; want advanced", id, epoch, err)
		}
		if epoch.Reason != ssdomain.RevocationReasonOwnerDisabled {
			t.Fatalf("revocation reason of %s = %s, want %s", id, epoch.Reason, ssdomain.RevocationReasonOwnerDisabled)
		}
		epochs = append(epochs, epoch.Epoch)
		accessTokens := oauth2.Module{TokenIntrospector: staticIntrospection{result: &oauthports.IntrospectionResult{
			Active: true, Sub: id, ClientID: id + "-client", Iat: issuedAt.Unix(),
		}}}.ResourceAccessTokens(f.deps.IdManagement.AgentRepo, f.epochs)
		verified, err := accessTokens.IntrospectAccessToken(ctx, id+"-token")
		if err != nil || verified != nil {
			t.Fatalf("token of %s issued before the disable: verified = %+v, %v; want revoked", id, verified, err)
		}
	}
	if !epochs[0].Equal(epochs[1]) {
		t.Fatalf("revocation epochs = %v, want the same instant for both agents", epochs)
	}
	if n := f.sink.count("AgentAccessRevoked"); n != len(f.agentIDs) {
		t.Fatalf("AgentAccessRevoked emitted %d times, want %d", n, len(f.agentIDs))
	}
}

// 外部への伝播は REQ-PLATFORM-001 の保証の外だが、worker で進めた失効も api と同じく受信側へ届ける。
// 署名に使う鍵と issuer が worker まで届かないと、配送は作られない。
//
//spec:covers REQ-SHAREDSIGNALS-007: worker が組み立てた disable_user の手順で User を止めると、session-revoked を購読する有効な Transmit ストリームへ、worker の issuer で署名した SET を載せた pending の配送が Agent ごとに作られることを固定する。
func TestWorkerDisableUserStepQueuesTheRevocationForTransmitStreams(t *testing.T) {
	f := newRevocationFixture(t)
	ctx := context.Background()
	if err := f.streams.Save(ctx, &ssdomain.SsfStream{
		ID: "stream-1", TenantID: "tenant-a", Direction: ssdomain.SsfStreamDirectionTransmit,
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}, Status: ssdomain.SsfStreamStatusEnabled, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.configs.Save(ctx, "tenant-a", &ssdomain.SsfTransmitterConfig{
		StreamID: "stream-1", DeliveryEndpoint: "https://receiver.example/events", Audience: "https://receiver.example",
		MaxDeliveryAttempts: ssdomain.DefaultMaxDeliveryAttempts,
	}); err != nil {
		t.Fatal(err)
	}

	f.disableAlice(t)

	deliveries, err := f.delivery.ListByStream(ctx, "tenant-a", "stream-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(deliveries) != len(f.agentIDs) {
		t.Fatalf("deliveries = %d, want one per agent (%d)", len(deliveries), len(f.agentIDs))
	}
	for _, d := range deliveries {
		if d.Status != ssdomain.SecurityEventDeliveryStatusPending || d.Set.Issuer != "https://idp.example" {
			t.Fatalf("delivery = status %s, iss %q; want pending and signed under the worker's issuer", d.Status, d.Set.Issuer)
		}
	}
}

// staticIntrospection は、イントロスペクションの結果を固定したアクセストークンの検証器である。
type staticIntrospection struct {
	result *oauthports.IntrospectionResult
}

func (i staticIntrospection) IntrospectAccessToken(context.Context, string) (*oauthports.IntrospectionResult, error) {
	return i.result, nil
}
