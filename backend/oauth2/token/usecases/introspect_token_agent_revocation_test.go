package usecases

import (
	"context"
	"testing"
	"time"

	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	ssmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// TestIntrospectToken_AgentRevocationEpoch — RED: scenario
// `kill-switchは既発行トークンをintrospectionで即時無効化する` (spec/contexts/oauth2.yaml
// Introspect). Agent 主体の access token は issued_at と SharedSignals の
// revocation epoch を比較し、epoch 以前に発行された token は fail-closed で
// active=false になる。epoch より後に発行された token は通常どおり active=true。
func TestIntrospectToken_AgentRevocationEpoch(t *testing.T) {
	ctx := tenantContext()
	now := time.Now().UTC()

	agentRepo := agentmemory.NewAgentRepository()
	if err := agentRepo.Save(context.Background(), &agentdomain.Agent{
		ID: "agent_1", TenantID: tenancydomain.DefaultTenantID, Name: "agent_1",
		Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "user_1", Status: idmdomain.AgentStatusKilled,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed agent: %v", err)
	}
	if _, err := agentRepo.AddBinding(context.Background(), &agentdomain.AgentCredentialBinding{
		AgentID: "agent_1", ClientID: "agent_client", CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	epochRepo := ssmemory.NewAgentRevocationEpochRepository()
	epoch := now
	if err := epochRepo.Advance(context.Background(), ssdomain.AgentRevocationEpoch{
		AgentID: "agent_1", TenantID: tenancydomain.DefaultTenantID,
		Epoch: epoch, Reason: ssdomain.RevocationReasonAgentKilled, AdvancedAt: epoch,
	}); err != nil {
		t.Fatalf("seed epoch: %v", err)
	}

	introspector := &fakeIntrospector{}
	deps := IntrospectDeps{
		Introspector: introspector, AgentRepo: agentRepo, RevocationEpochRepo: epochRepo,
	}

	t.Run("IssuedBeforeEpochIsInactive", func(t *testing.T) {
		introspector.result = &ports.IntrospectionResult{
			Active: true, JTI: "jti-before", ClientID: "agent_client",
			Iat: epoch.Add(-time.Minute).Unix(), Exp: epoch.Add(time.Hour).Unix(),
		}
		resp, err := IntrospectToken(ctx, deps, IntrospectInput{Token: "t", TokenTypeHint: "access_token"}, now)
		if err != nil {
			t.Fatal(err)
		}
		if resp.Active {
			t.Fatal("expected inactive for a token issued before the revocation epoch")
		}
	})

	t.Run("IssuedAfterEpochStaysActive", func(t *testing.T) {
		introspector.result = &ports.IntrospectionResult{
			Active: true, JTI: "jti-after", ClientID: "agent_client",
			Iat: epoch.Add(time.Minute).Unix(), Exp: epoch.Add(time.Hour).Unix(),
		}
		resp, err := IntrospectToken(ctx, deps, IntrospectInput{Token: "t", TokenTypeHint: "access_token"}, now)
		if err != nil {
			t.Fatal(err)
		}
		if !resp.Active {
			t.Fatal("expected a token issued after the revocation epoch to stay active")
		}
	})

	t.Run("NonAgentClientIsUnaffected", func(t *testing.T) {
		introspector.result = &ports.IntrospectionResult{
			Active: true, JTI: "jti-non-agent", ClientID: "not_bound_to_any_agent",
			Iat: epoch.Add(-time.Hour).Unix(), Exp: epoch.Add(time.Hour).Unix(),
		}
		resp, err := IntrospectToken(ctx, deps, IntrospectInput{Token: "t", TokenTypeHint: "access_token"}, now)
		if err != nil {
			t.Fatal(err)
		}
		if !resp.Active {
			t.Fatal("expected a non-agent client's token to be unaffected by revocation epoch")
		}
	})
}
