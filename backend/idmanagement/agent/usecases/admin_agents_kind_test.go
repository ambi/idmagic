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
	})

	t.Run("UnknownValueIsRejectedNotRounded", func(t *testing.T) {
		deps, _ := newAgentDeps(t)
		_, err := agentusecases.RegisterAgent(defaultTenantCtx(), deps, agentusecases.RegisterAgentInput{
			ActorUserID: "operator", Name: "deploy-bot", Kind: idmdomain.AgentKind("mystery"), Now: now,
		})
		if !errors.Is(err, agentusecases.ErrAgentKindInvalid) {
			t.Fatalf("expected ErrAgentKindInvalid, got %v", err)
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
