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
	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/token/usecases"
	"github.com/ambi/idmagic/backend/shared/events/sinks_console"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/sharedsignals"
	ssmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingmemory "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
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

// REQ-IDMANAGEMENT-081 の主要な使い方のうち、worker の組み立ての経路。UserLifecycle を渡し忘れると
// disable_user が失敗し、IdManagement の依存を渡し忘れると所有する Agent の無効化が黙って止まる。
//
//spec:covers EX-IDMANAGEMENT-081-02: worker が組み立てた disable_user の手順が User を Disabled にして UserDisabled を発行し、所有する Agent を Disabled にすること。
func TestWorkerDisableUserStepStopsTheAgentsTheUserOwns(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	workflows := igmemory.NewLifecycleWorkflowRepository()
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{ID: "alice", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "unused", Roles: []string{}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now})
	agents := agentmemory.NewAgentRepository()
	if err := agents.Save(ctx, &agentdomain.Agent{ID: "deploy-bot", TenantID: "tenant-a", Name: "deploy-bot", Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "alice", Status: idmdomain.AgentStatusActive, Roles: []string{}, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := workflows.Save(ctx, &igdomain.LifecycleWorkflow{ID: "workflow-1", TenantID: "tenant-a", Name: "Leaver", Status: igdomain.LifecycleWorkflowEnabled, CurrentRevision: 1, EnabledRevision: new(int64(1)), CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	action := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionDisableUser}
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "occurrence-1", TargetUserID: "alice", TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{action}, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
	if _, err := runs.SaveRun(ctx, run, []igdomain.WorkflowStep{{RunID: run.ID, Action: action, Outcome: igdomain.WorkflowStepPending}}); err != nil {
		t.Fatal(err)
	}
	sink := &recordingSink{}
	deps := &bootstrap.Dependencies{
		IdManagement: idmanagement.Module{UserRepo: users, GroupRepo: groupmemory.NewGroupRepository(), AgentRepo: agents},
		IdGovernance: idgovernance.Module{LifecycleWorkflowRepo: workflows, LifecycleWorkflowRunRepo: runs},
		OAuth2:       oauth2.Module{EventSink: sink},
	}
	logger := logging.New(os.Stderr, logging.ParseLevel("error"), "idmagic-worker-test", "test")
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := igusecases.LifecycleWorkflowRunHandler(lifecycleWorkflowExecutorDeps(deps, logger))(ctx, &jobsdomain.Job{TenantID: "tenant-a", Params: params, Attempts: 1, MaxAttempts: 1}); err != nil {
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
	epochs   *ssmemory.AgentRevocationEpochRepository
	streams  *ssmemory.SsfStreamRepository
	configs  *ssmemory.SsfTransmitterConfigRepository
	delivery *ssmemory.SecurityEventDeliveryRepository
	agentIDs []string
}

func newRevocationFixture(t *testing.T) *revocationFixture {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{ID: "alice", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "unused", Roles: []string{}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now})
	agents := agentmemory.NewAgentRepository()
	f := &revocationFixture{
		sink:   &recordingSink{},
		epochs: ssmemory.NewAgentRevocationEpochRepository(), streams: ssmemory.NewSsfStreamRepository(),
		configs: ssmemory.NewSsfTransmitterConfigRepository(), delivery: ssmemory.NewSecurityEventDeliveryRepository(),
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
	workflows := igmemory.NewLifecycleWorkflowRepository()
	if err := workflows.Save(ctx, &igdomain.LifecycleWorkflow{ID: "workflow-1", TenantID: "tenant-a", Name: "Leaver", Status: igdomain.LifecycleWorkflowEnabled, CurrentRevision: 1, EnabledRevision: new(int64(1)), CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	keyStore, err := signingmemory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	f.deps = &bootstrap.Dependencies{
		IdManagement: idmanagement.Module{UserRepo: users, GroupRepo: groupmemory.NewGroupRepository(), AgentRepo: agents},
		IdGovernance: idgovernance.Module{LifecycleWorkflowRepo: workflows, LifecycleWorkflowRunRepo: igmemory.NewLifecycleWorkflowRunRepository()},
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
	action := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionDisableUser}
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "occurrence-1", TargetUserID: "alice", TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{action}, Status: igdomain.WorkflowRunQueued, TriggeredAt: time.Now().UTC()}
	if _, err := f.deps.IdGovernance.LifecycleWorkflowRunRepo.SaveRun(ctx, run, []igdomain.WorkflowStep{{RunID: run.ID, Action: action, Outcome: igdomain.WorkflowStepPending}}); err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}
	logger := logging.New(os.Stderr, logging.ParseLevel("error"), "idmagic-worker-test", "test")
	if _, err := igusecases.LifecycleWorkflowRunHandler(lifecycleWorkflowExecutorDeps(f.deps, logger))(ctx, &jobsdomain.Job{TenantID: "tenant-a", Params: params, Attempts: 1, MaxAttempts: 1}); err != nil {
		t.Fatal(err)
	}
	steps, err := f.deps.IdGovernance.LifecycleWorkflowRunRepo.ListSteps(ctx, "tenant-a", run.ID)
	if err != nil || len(steps) != 1 || steps[0].Outcome != igdomain.WorkflowStepChanged {
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

	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-a"}, "", "")
	introspect := tokenusecases.IntrospectDeps{AgentRepo: f.deps.IdManagement.AgentRepo, RevocationEpochRepo: f.epochs}
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
		revoked, err := tokenusecases.AccessTokenIsRevoked(ctx, introspect, &oauthports.IntrospectionResult{Active: true, ClientID: id + "-client", Iat: issuedAt.Unix()})
		if err != nil || !revoked {
			t.Fatalf("token of %s issued before the disable: revoked = %v, %v; want true", id, revoked, err)
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
