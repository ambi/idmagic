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

// Agent 主体の access token は issued_at と SharedSignals の revocation epoch を比較する。
//
// 境界の両側を踏むのは、比較演算子の向きを取り違えた実装を片側だけでは見分けられないためである。
//
//spec:covers EX-OAUTH2-012-01, EX-OAUTH2-012-02: epoch 以前に発行された token は fail-closed で active=false だけを返して claim を 1 つも運ばず、epoch より後に発行された token は active=true と claim を返す。
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
			Active: true, JTI: "jti-before", ClientID: "agent_client", Sub: "agent_client",
			Scope: "openid", Iat: epoch.Add(-time.Minute).Unix(), Exp: epoch.Add(time.Hour).Unix(),
		}
		resp, err := IntrospectToken(ctx, deps, IntrospectInput{Token: "t", TokenTypeHint: "access_token"}, now)
		if err != nil {
			t.Fatal(err)
		}
		assertOnlyInactive(t, resp)
	})

	t.Run("IssuedAfterEpochStaysActive", func(t *testing.T) {
		introspector.result = &ports.IntrospectionResult{
			Active: true, JTI: "jti-after", ClientID: "agent_client", Sub: "agent_client",
			Scope: "openid", Iat: epoch.Add(time.Minute).Unix(), Exp: epoch.Add(time.Hour).Unix(),
		}
		resp, err := IntrospectToken(ctx, deps, IntrospectInput{Token: "t", TokenTypeHint: "access_token"}, now)
		if err != nil {
			t.Fatal(err)
		}
		if !resp.Active {
			t.Fatal("expected a token issued after the revocation epoch to stay active")
		}
		if resp.Sub != "agent_client" || resp.Scope != "openid" {
			t.Fatalf("kill 後に再発行された token の claim が失われている: %+v", resp)
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
