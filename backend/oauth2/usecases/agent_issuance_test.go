package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	sharedusecases "github.com/ambi/idmagic/backend/oauth2/usecases"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// TestResolveIssuableAgent_OwnerOffboarding — RED: REQ-OAUTH2-046
// (docs/contexts/oauth2/scenarios.md)。所有者がオフボードされた Agent は
// client_credentials で新しいトークンを取得できない。所有者の状態は Agent の
// status を書き換えず、発行のたびに解決する。
func TestResolveIssuableAgent_OwnerOffboarding(t *testing.T) {
	ctx := tenantContext(tenancydomain.DefaultTenantID)
	now := time.Now().UTC()

	seed := func(t *testing.T, ownerStatus idmdomain.UserStatus, ownerExists bool) sharedusecases.AgentIssuanceDeps {
		t.Helper()
		agentRepo := agentmemory.NewAgentRepository()
		if err := agentRepo.Save(context.Background(), &agentdomain.Agent{
			ID: "agent_1", TenantID: tenancydomain.DefaultTenantID, Name: "agent_1",
			Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "owner_1", Status: idmdomain.AgentStatusActive,
			CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("seed agent: %v", err)
		}
		if _, err := agentRepo.AddBinding(context.Background(), &agentdomain.AgentCredentialBinding{
			AgentID: "agent_1", ClientID: "agent_client", CreatedAt: now,
		}); err != nil {
			t.Fatalf("seed binding: %v", err)
		}
		userRepo := usermemory.NewUserRepository()
		if ownerExists {
			if err := userRepo.Save(context.Background(), &userdomain.User{
				ID: "owner_1", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "owner_1",
				Lifecycle: userdomain.UserLifecycle{Status: ownerStatus, StatusChangedAt: &now},
				CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				t.Fatalf("seed owner: %v", err)
			}
		}
		return sharedusecases.AgentIssuanceDeps{AgentRepo: agentRepo, UserRepo: userRepo}
	}

	t.Run("ActiveOwnerIssues", func(t *testing.T) {
		agent, err := sharedusecases.ResolveIssuableAgent(ctx, seed(t, idmdomain.UserStatusActive, true), "agent_client")
		if err != nil {
			t.Fatalf("expected an active owner to allow issuance: %v", err)
		}
		if agent == nil || agent.ID != "agent_1" {
			t.Fatalf("expected the bound agent, got %v", agent)
		}
	})

	t.Run("DisabledOwnerRejects", func(t *testing.T) {
		deps := seed(t, idmdomain.UserStatusDisabled, true)
		_, err := sharedusecases.ResolveIssuableAgent(ctx, deps, "agent_client")
		assertInvalidClient(t, err)
		// 所有者の状態は Agent の status を書き換えない。
		agent, findErr := deps.AgentRepo.FindByID(ctx, tenancydomain.DefaultTenantID, "agent_1")
		if findErr != nil {
			t.Fatal(findErr)
		}
		if agent.Status != idmdomain.AgentStatusActive {
			t.Fatalf("expected the agent to stay Active, got %q", agent.Status)
		}
	})

	t.Run("DeletedOwnerRejects", func(t *testing.T) {
		_, err := sharedusecases.ResolveIssuableAgent(ctx, seed(t, idmdomain.UserStatusActive, false), "agent_client")
		assertInvalidClient(t, err)
	})

	t.Run("UnboundClientIsUnaffected", func(t *testing.T) {
		agent, err := sharedusecases.ResolveIssuableAgent(ctx, seed(t, idmdomain.UserStatusActive, false), "plain_client")
		if err != nil {
			t.Fatalf("expected a client with no bound agent to be unaffected: %v", err)
		}
		if agent != nil {
			t.Fatalf("expected no agent, got %v", agent)
		}
	})

	t.Run("DisabledAgentRejects", func(t *testing.T) {
		deps := seed(t, idmdomain.UserStatusActive, true)
		disabled := now
		if err := deps.AgentRepo.Save(ctx, &agentdomain.Agent{
			ID: "agent_1", TenantID: tenancydomain.DefaultTenantID, Name: "agent_1",
			Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "owner_1", Status: idmdomain.AgentStatusDisabled,
			CreatedAt: now, UpdatedAt: now, DisabledAt: &disabled,
		}); err != nil {
			t.Fatal(err)
		}
		_, err := sharedusecases.ResolveIssuableAgent(ctx, deps, "agent_client")
		assertInvalidClient(t, err)
	})
}

func assertInvalidClient(t *testing.T, err error) {
	t.Helper()
	var oauthErr *sharedusecases.OAuthError
	if !errors.As(err, &oauthErr) {
		t.Fatalf("expected an OAuthError, got %v", err)
	}
	if oauthErr.Code != "invalid_client" {
		t.Fatalf("expected invalid_client, got %q", oauthErr.Code)
	}
}
