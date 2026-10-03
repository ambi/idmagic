package usecases_test

// Agent の登録、束縛、更新、無効化、停止、削除が約束する細部を、保存層とイベントから
// 読んで固定する。

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	agentusecases "github.com/ambi/idmagic/backend/idmanagement/agent/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

var agentRulesNow = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

func registerAgent(t *testing.T, deps agentusecases.AdminAgentDeps, name string) string {
	t.Helper()
	agent, err := agentusecases.RegisterAgent(defaultTenantCtx(), deps, agentusecases.RegisterAgentInput{
		ActorUserID: "operator", Name: name, Kind: idmdomain.AgentKindAutonomous, Now: agentRulesNow,
	})
	if err != nil {
		t.Fatal(err)
	}
	return agent.ID
}

func agentsUsage(t *testing.T, deps agentusecases.AdminAgentDeps) int {
	t.Helper()
	usage, err := deps.QuotaRepo.GetUsage(context.Background(), tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	return usage.Agents
}

func boundClientIDs(t *testing.T, deps agentusecases.AdminAgentDeps, agentID string) []string {
	t.Helper()
	bindings, err := deps.AgentRepo.ListBindings(context.Background(), tenancydomain.DefaultTenantID, agentID)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(bindings))
	for i, binding := range bindings {
		out[i] = binding.ClientID
	}
	return out
}

//spec:covers EX-IDMANAGEMENT-073-01, EX-IDMANAGEMENT-073-03: 所有者を省いた登録が登録した管理者を所有者とし、大文字と小文字だけが異なる名前を agent_name_conflict で拒否すること。
func TestRegisterAgentDefaultsTheOwnerAndComparesNamesCaseInsensitively(t *testing.T) {
	deps, _ := newAgentDeps(t)
	ctx := defaultTenantCtx()
	agent, err := agentusecases.RegisterAgent(ctx, deps, agentusecases.RegisterAgentInput{
		ActorUserID: "operator", Name: " deploy-bot ", Kind: idmdomain.AgentKindAutonomous, Now: agentRulesNow,
	})
	if err != nil || agent.OwnerUserID != "operator" || agent.Name != "deploy-bot" {
		t.Fatalf("agent=%+v err=%v, want owner operator and trimmed name", agent, err)
	}
	if _, err := agentusecases.RegisterAgent(ctx, deps, agentusecases.RegisterAgentInput{
		ActorUserID: "operator", Name: "Deploy-Bot", Kind: idmdomain.AgentKindAutonomous, Now: agentRulesNow,
	}); !errors.Is(err, agentusecases.ErrAgentNameConflict) {
		t.Fatalf("err=%v, want ErrAgentNameConflict", err)
	}
}

//spec:covers EX-IDMANAGEMENT-073-02: Active でない所有者と別のテナントの所有者の登録を agent_owner_not_found で拒否し、Agent も使用量も増やさないこと。
func TestRegisterAgentRequiresAnActiveOwnerInTheTenant(t *testing.T) {
	deps, _ := newAgentDeps(t)
	users := deps.UserRepo.(*usermemory.UserRepository)
	users.Seed(&userdomain.User{
		ID: "disabled_owner", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "disabled-owner", PasswordHash: "hash",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusDisabled}, CreatedAt: agentRulesNow, UpdatedAt: agentRulesNow,
	})
	for _, owner := range []string{"disabled_owner", "acme_owner", "nobody"} {
		if _, err := agentusecases.RegisterAgent(defaultTenantCtx(), deps, agentusecases.RegisterAgentInput{
			ActorUserID: "operator", Name: "bot-" + owner, Kind: idmdomain.AgentKindAutonomous, OwnerUserID: owner, Now: agentRulesNow,
		}); !errors.Is(err, agentusecases.ErrAgentOwnerNotFound) {
			t.Fatalf("%s: err=%v, want ErrAgentOwnerNotFound", owner, err)
		}
	}
	if count, _ := deps.AgentRepo.Count(context.Background(), tenancydomain.DefaultTenantID); count != 0 {
		t.Fatalf("agents=%d, want 0", count)
	}
	if usage := agentsUsage(t, deps); usage != 0 {
		t.Fatalf("agents usage=%d, want 0", usage)
	}
}

//spec:covers EX-IDMANAGEMENT-074-01, EX-IDMANAGEMENT-074-02: ほかの Agent に束縛済みの OAuth2Client の束縛を agent_client_already_bound で拒否し、同じ束縛の繰り返しはイベントを発行しないこと。
func TestBindCredentialIsExclusiveAndIdempotent(t *testing.T) {
	deps, events := newAgentDeps(t)
	ctx := defaultTenantCtx()
	deploy := registerAgent(t, deps, "deploy-bot")
	report := registerAgent(t, deps, "report-bot")
	if err := agentusecases.BindCredential(ctx, deps, "operator", deploy, " svc_client ", agentRulesNow); err != nil {
		t.Fatal(err)
	}
	*events = nil
	if err := agentusecases.BindCredential(ctx, deps, "operator", report, "svc_client", agentRulesNow); !errors.Is(err, agentusecases.ErrAgentClientBound) {
		t.Fatalf("err=%v, want ErrAgentClientBound", err)
	}
	if got := boundClientIDs(t, deps, report); len(got) != 0 {
		t.Fatalf("report-bot bindings=%v, want none", got)
	}
	if err := agentusecases.BindCredential(ctx, deps, "operator", deploy, "svc_client", agentRulesNow); err != nil {
		t.Fatal(err)
	}
	if err := agentusecases.UnbindCredential(ctx, deps, "operator", report, "svc_client", agentRulesNow); err != nil {
		t.Fatal(err)
	}
	if len(*events) != 0 {
		t.Fatalf("events=%v, want none", agentEventTypes(*events))
	}
	if err := agentusecases.BindCredential(ctx, deps, "operator", report, "  ", agentRulesNow); !errors.Is(err, agentusecases.ErrAgentClientNotFound) {
		t.Fatalf("空の client_id: err=%v, want ErrAgentClientNotFound", err)
	}
}

//spec:covers EX-IDMANAGEMENT-074-03: Killed の Agent からも束縛を解除でき、AgentCredentialUnbound を発行すること。
func TestUnbindCredentialWorksOnAKilledAgent(t *testing.T) {
	deps, events := newAgentDeps(t)
	ctx := defaultTenantCtx()
	deploy := registerAgent(t, deps, "deploy-bot")
	if err := agentusecases.BindCredential(ctx, deps, "operator", deploy, "svc_client", agentRulesNow); err != nil {
		t.Fatal(err)
	}
	if _, err := agentusecases.KillAgent(ctx, deps, "operator", deploy, agentRulesNow); err != nil {
		t.Fatal(err)
	}
	*events = nil
	if err := agentusecases.UnbindCredential(ctx, deps, "operator", deploy, "svc_client", agentRulesNow); err != nil {
		t.Fatal(err)
	}
	if got := boundClientIDs(t, deps, deploy); len(got) != 0 {
		t.Fatalf("bindings=%v, want none", got)
	}
	if got := agentEventTypes(*events); !slices.Equal(got, []string{"AgentCredentialUnbound"}) {
		t.Fatalf("events=%v", got)
	}
}

//spec:covers EX-IDMANAGEMENT-075-01, EX-IDMANAGEMENT-075-02: 所有者の変更が owner_sub を載せた AgentUpdated に続けて変更前後の所有者の AgentOwnerChanged を発行し、何も変わらない更新は updated_at を進めずイベントを発行しないこと。
func TestUpdateAgentRecordsOwnerChangesSeparatelyAndSkipsNoOps(t *testing.T) {
	deps, events := newAgentDeps(t)
	ctx := defaultTenantCtx()
	deploy := registerAgent(t, deps, "deploy-bot")
	*events = nil
	newOwner := "user_new"
	if _, err := agentusecases.UpdateAgent(ctx, deps, agentusecases.UpdateAgentInput{ActorUserID: "operator", ID: deploy, OwnerUserID: &newOwner, Now: agentRulesNow.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if got := agentEventTypes(*events); !slices.Equal(got, []string{"AgentUpdated", "AgentOwnerChanged"}) {
		t.Fatalf("events=%v", got)
	}
	updated := (*events)[0].(*idmdomain.AgentUpdated)
	changed := (*events)[1].(*idmdomain.AgentOwnerChanged)
	if !slices.Equal(updated.ChangedFields, []string{"owner_sub"}) || changed.PreviousOwnerUserID != "operator" || changed.NewOwnerUserID != "user_new" {
		t.Fatalf("updated=%+v changed=%+v", updated, changed)
	}

	*events = nil
	name := "deploy-bot"
	if _, err := agentusecases.UpdateAgent(ctx, deps, agentusecases.UpdateAgentInput{ActorUserID: "operator", ID: deploy, Name: &name, Now: agentRulesNow.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	stored, _ := deps.AgentRepo.FindByID(ctx, tenancydomain.DefaultTenantID, deploy)
	if !stored.UpdatedAt.Equal(agentRulesNow.Add(time.Minute)) || len(*events) != 0 {
		t.Fatalf("updated_at=%v events=%v, want unchanged and none", stored.UpdatedAt, agentEventTypes(*events))
	}
}

//spec:covers EX-IDMANAGEMENT-076-01: Disabled の Agent の無効化が disabled_at を操作の時刻へ進め、AgentDisabled をもう一度発行すること。
func TestDisablingADisabledAgentRecordsItAgain(t *testing.T) {
	deps, events := newAgentDeps(t)
	ctx := defaultTenantCtx()
	deploy := registerAgent(t, deps, "deploy-bot")
	if _, err := agentusecases.SetAgentDisabled(ctx, deps, "operator", deploy, true, agentRulesNow); err != nil {
		t.Fatal(err)
	}
	later := agentRulesNow.Add(time.Hour)
	if _, err := agentusecases.SetAgentDisabled(ctx, deps, "operator", deploy, true, later); err != nil {
		t.Fatal(err)
	}
	stored, _ := deps.AgentRepo.FindByID(ctx, tenancydomain.DefaultTenantID, deploy)
	if stored.DisabledAt == nil || !stored.DisabledAt.Equal(later) {
		t.Fatalf("disabled_at=%v, want %v", stored.DisabledAt, later)
	}
	disabled := 0
	for _, eventType := range agentEventTypes(*events) {
		if eventType == "AgentDisabled" {
			disabled++
		}
	}
	if disabled != 2 {
		t.Fatalf("AgentDisabled=%d, want 2", disabled)
	}
}

//spec:covers EX-IDMANAGEMENT-077-01, EX-IDMANAGEMENT-077-02: Killed の Agent の更新、無効化、再有効化、停止、束縛を ErrAgentKilled で拒否し、Agent を変えずイベントを発行しないこと。
func TestKilledAgentCannotBeChanged(t *testing.T) {
	deps, events := newAgentDeps(t)
	ctx := defaultTenantCtx()
	deploy := registerAgent(t, deps, "deploy-bot")
	if _, err := agentusecases.KillAgent(ctx, deps, "operator", deploy, agentRulesNow); err != nil {
		t.Fatal(err)
	}
	*events = nil
	name := "renamed"
	attempts := map[string]error{}
	_, attempts["update"] = agentusecases.UpdateAgent(ctx, deps, agentusecases.UpdateAgentInput{ActorUserID: "operator", ID: deploy, Name: &name, Now: agentRulesNow})
	_, attempts["disable"] = agentusecases.SetAgentDisabled(ctx, deps, "operator", deploy, true, agentRulesNow)
	_, attempts["enable"] = agentusecases.SetAgentDisabled(ctx, deps, "operator", deploy, false, agentRulesNow)
	_, attempts["kill"] = agentusecases.KillAgent(ctx, deps, "operator", deploy, agentRulesNow)
	attempts["bind"] = agentusecases.BindCredential(ctx, deps, "operator", deploy, "svc_client", agentRulesNow)
	for operation, err := range attempts {
		if !errors.Is(err, agentusecases.ErrAgentKilled) {
			t.Fatalf("%s: err=%v, want ErrAgentKilled", operation, err)
		}
	}
	stored, _ := deps.AgentRepo.FindByID(ctx, tenancydomain.DefaultTenantID, deploy)
	if stored.Status != idmdomain.AgentStatusKilled || stored.Name != "deploy-bot" || len(*events) != 0 {
		t.Fatalf("agent=%+v events=%v, want unchanged and none", stored, agentEventTypes(*events))
	}
}

//spec:covers EX-IDMANAGEMENT-078-01: 束縛を持つ Agent の削除が記録と束縛を消し、使用量を一つ減らして AgentDeleted を発行すること。
func TestDeleteAgentRemovesBindingsAndReleasesQuota(t *testing.T) {
	deps, events := newAgentDeps(t)
	ctx := defaultTenantCtx()
	deploy := registerAgent(t, deps, "deploy-bot")
	if err := agentusecases.BindCredential(ctx, deps, "operator", deploy, "svc_client", agentRulesNow); err != nil {
		t.Fatal(err)
	}
	if err := agentusecases.DeleteAgent(ctx, deps, "operator", deploy, agentRulesNow); err != nil {
		t.Fatal(err)
	}
	if stored, _ := deps.AgentRepo.FindByID(ctx, tenancydomain.DefaultTenantID, deploy); stored != nil {
		t.Fatalf("削除した Agent が残る: %+v", stored)
	}
	if owner, _ := deps.AgentRepo.FindByClientID(ctx, tenancydomain.DefaultTenantID, "svc_client"); owner != nil {
		t.Fatalf("svc_client がまだ %s に束縛されている", owner.ID)
	}
	if usage := agentsUsage(t, deps); usage != 0 {
		t.Fatalf("agents usage=%d, want 0", usage)
	}
	if got := agentEventTypes(*events); got[len(got)-1] != "AgentDeleted" {
		t.Fatalf("events=%v, want AgentDeleted last", got)
	}
}

//spec:covers EX-IDMANAGEMENT-078-02: Killed の Agent の削除を ErrAgentKilled で拒否し、記録と使用量を残すこと。
func TestDeleteAgentRefusesAKilledAgent(t *testing.T) {
	deps, _ := newAgentDeps(t)
	ctx := defaultTenantCtx()
	deploy := registerAgent(t, deps, "deploy-bot")
	if _, err := agentusecases.KillAgent(ctx, deps, "operator", deploy, agentRulesNow); err != nil {
		t.Fatal(err)
	}
	if err := agentusecases.DeleteAgent(ctx, deps, "operator", deploy, agentRulesNow); !errors.Is(err, agentusecases.ErrAgentKilled) {
		t.Fatalf("err=%v, want ErrAgentKilled", err)
	}
	if stored, _ := deps.AgentRepo.FindByID(ctx, tenancydomain.DefaultTenantID, deploy); stored == nil {
		t.Fatalf("拒否した削除が記録を消した")
	}
	if usage := agentsUsage(t, deps); usage != 1 {
		t.Fatalf("agents usage=%d, want 1", usage)
	}
}

//spec:covers REQ-IDMANAGEMENT-075: 区分だけを変える更新が changed_fields に kind だけを載せ、同じ区分の指定は記録しないこと。
func TestUpdateAgentRecordsAKindChange(t *testing.T) {
	deps, events := newAgentDeps(t)
	ctx := defaultTenantCtx()
	deploy := registerAgent(t, deps, "deploy-bot")
	*events = nil
	same := idmdomain.AgentKindAutonomous
	if _, err := agentusecases.UpdateAgent(ctx, deps, agentusecases.UpdateAgentInput{ActorUserID: "operator", ID: deploy, Kind: &same, Now: agentRulesNow}); err != nil {
		t.Fatal(err)
	}
	if len(*events) != 0 {
		t.Fatalf("同じ区分の指定が記録された: %v", agentEventTypes(*events))
	}
	supervised := idmdomain.AgentKindSupervised
	if _, err := agentusecases.UpdateAgent(ctx, deps, agentusecases.UpdateAgentInput{ActorUserID: "operator", ID: deploy, Kind: &supervised, Now: agentRulesNow}); err != nil {
		t.Fatal(err)
	}
	if len(*events) != 1 || !slices.Equal((*events)[0].(*idmdomain.AgentUpdated).ChangedFields, []string{"kind"}) {
		t.Fatalf("events=%+v, want AgentUpdated with changed_fields [kind]", *events)
	}
}

// REQ-IDMANAGEMENT-082 の主要な使い方：所有者が止まっている Agent は再有効化できない。
//
//spec:covers EX-IDMANAGEMENT-082-01: 所有者の User が Disabled の Agent の再有効化を ErrAgentOwnerInactive で拒否し、Agent を Disabled のまま残しイベントを発行しないこと。
func TestEnablingAnAgentWhoseOwnerIsInactiveIsRefused(t *testing.T) {
	deps, events := newAgentDeps(t)
	ctx := defaultTenantCtx()
	agent, err := agentusecases.RegisterAgent(ctx, deps, agentusecases.RegisterAgentInput{
		ActorUserID: "operator", Name: "deploy-bot", Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "user_new", Now: agentRulesNow,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := agentusecases.SetAgentDisabled(ctx, deps, "operator", agent.ID, true, agentRulesNow); err != nil {
		t.Fatal(err)
	}
	owner, err := deps.UserRepo.FindBySub(ctx, "user_new")
	if err != nil || owner == nil {
		t.Fatalf("owner=%v err=%v", owner, err)
	}
	owner.Lifecycle.Status = idmdomain.UserStatusDisabled
	if err := deps.UserRepo.Save(ctx, owner); err != nil {
		t.Fatal(err)
	}
	*events = nil

	if _, err := agentusecases.SetAgentDisabled(ctx, deps, "operator", agent.ID, false, agentRulesNow); !errors.Is(err, agentusecases.ErrAgentOwnerInactive) {
		t.Fatalf("err=%v, want ErrAgentOwnerInactive", err)
	}
	stored, _ := deps.AgentRepo.FindByID(ctx, tenancydomain.DefaultTenantID, agent.ID)
	if stored.Status != idmdomain.AgentStatusDisabled || len(*events) != 0 {
		t.Fatalf("status=%s events=%v, want disabled and none", stored.Status, agentEventTypes(*events))
	}
}
