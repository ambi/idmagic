package usecases_test

// REQ-IDMANAGEMENT-009 の kind 節 (wi-376 T004)。区分は実行時のトークン発行可否を
// 決めるため (REQ-OAUTH2-050)、登録では必須とし、既知でない値も既定値へ丸めない。

import (
	"errors"
	"testing"
	"time"

	agentusecases "github.com/ambi/idmagic/backend/idmanagement/agent/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

// registeredAgentKinds は保存されている Agent の区分を名前つきで返す。
// 「既定値で補わない」「既知の値へ丸めない」はどちらも保存された区分の話であり、
// エラーの型だけを見るテストでは、丸めてから別の理由で失敗する実装と区別できない。
func registeredAgentKinds(
	t *testing.T, deps agentusecases.AdminAgentDeps,
) map[string]idmdomain.AgentKind {
	t.Helper()
	views, err := agentusecases.ListAgents(defaultTenantCtx(), deps, "", "", 100)
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	kinds := map[string]idmdomain.AgentKind{}
	for _, view := range views {
		kinds[view.Agent.Name] = view.Agent.Kind
	}
	return kinds
}

//spec:covers EX-IDMANAGEMENT-009-02, EX-IDMANAGEMENT-009-03: kind を省いた登録が AgentKindRequiredError で、既知でない kind が InvalidAgentKindError で拒否され、どちらも Agent を残さないこと。
func TestRegisterAgentRequiresAKind(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)

	t.Run("OmittedIsRejected", func(t *testing.T) {
		deps, _ := newAgentDeps(t)
		_, err := agentusecases.RegisterAgent(defaultTenantCtx(), deps, agentusecases.RegisterAgentInput{
			ActorUserID: "operator", Name: "deploy-bot", Now: now,
		})
		if !errors.Is(err, agentusecases.ErrAgentKindRequired) {
			t.Fatalf("expected ErrAgentKindRequired, got %v", err)
		}
		// 「既定値で補わない」の観測。補ってから登録する実装は、エラーの型だけでは通る。
		if kinds := registeredAgentKinds(t, deps); len(kinds) != 0 {
			t.Fatalf("kind を省いた登録が Agent を残した: %v", kinds)
		}
	})

	t.Run("UnknownValueIsRejectedNotRounded", func(t *testing.T) {
		deps, _ := newAgentDeps(t)
		_, err := agentusecases.RegisterAgent(defaultTenantCtx(), deps, agentusecases.RegisterAgentInput{
			ActorUserID: "operator", Name: "deploy-bot", Kind: idmdomain.AgentKind("mystery"), Now: now,
		})
		if !errors.Is(err, agentusecases.ErrAgentKindInvalid) {
			t.Fatalf("expected ErrAgentKindInvalid, got %v", err)
		}
		// 「既知の値へ丸めない」の観測。丸めた区分で登録してしまう実装を落とす。
		if kinds := registeredAgentKinds(t, deps); len(kinds) != 0 {
			t.Fatalf("既知でない kind の登録が Agent を残した: %v", kinds)
		}
	})

	t.Run("DeclaredKindIsKept", func(t *testing.T) {
		deps, _ := newAgentDeps(t)
		agent, err := agentusecases.RegisterAgent(defaultTenantCtx(), deps, agentusecases.RegisterAgentInput{
			ActorUserID: "operator", Name: "deploy-bot", Kind: idmdomain.AgentKindSupervised, Now: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		if agent.Kind != idmdomain.AgentKindSupervised {
			t.Fatalf("kind = %q, want supervised", agent.Kind)
		}
	})
}

func TestUpdateAgentRejectsAnUnknownKind(t *testing.T) {
	ctx := defaultTenantCtx()
	deps, _ := newAgentDeps(t)
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)

	agent, err := agentusecases.RegisterAgent(ctx, deps, agentusecases.RegisterAgentInput{
		ActorUserID: "operator", Name: "deploy-bot", Kind: idmdomain.AgentKindAutonomous, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	unknown := idmdomain.AgentKind("mystery")
	if _, err := agentusecases.UpdateAgent(ctx, deps, agentusecases.UpdateAgentInput{
		ActorUserID: "operator", ID: agent.ID, Kind: &unknown, Now: now.Add(time.Hour),
	}); !errors.Is(err, agentusecases.ErrAgentKindInvalid) {
		t.Fatalf("expected ErrAgentKindInvalid, got %v", err)
	}
	reread, err := agentusecases.GetAgent(ctx, deps, agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reread.Agent.Kind != idmdomain.AgentKindAutonomous {
		t.Fatalf("kind = %q, want the rejected update to leave it alone", reread.Agent.Kind)
	}
}
