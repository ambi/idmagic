package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/sharedsignals"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// 失効判定は、現在の client_id の束縛ではなく、トークンが発行時に記録した Agent を読む。
// 束縛の解除、再束縛、削除、委譲の連鎖のそれぞれで、現在の束縛からは発行時の Agent に
// たどり着けない。対照として、発行時の Agent がすべて Active でエポックより後のトークンは
// 有効のままであることを置く。
//
//spec:covers REQ-OAUTH2-012: 発行時の Agent (agent_chain と agent_id) のエポック、存在、状態で判定し、現在の束縛に依存しない
func TestIntrospectToken_UsesIssuanceTimeAgents(t *testing.T) {
	ctx := tenantContext()
	epoch := time.Now().UTC().Truncate(time.Second)
	before := epoch.Add(-time.Minute).Unix()
	after := epoch.Add(time.Minute).Unix()

	agents := idmanagement.NewMemoryModule().AgentRepo
	epochs := sharedsignals.NewAgentRevocationEpochRepository()
	seed := func(id string, status idmdomain.AgentStatus) {
		t.Helper()
		if err := agents.Save(context.Background(), &agentdomain.Agent{
			ID: id, TenantID: tenancydomain.DefaultTenantID, Name: id, Kind: idmdomain.AgentKindAutonomous,
			OwnerUserID: "user_1", Status: status, CreatedAt: epoch, UpdatedAt: epoch,
		}); err != nil {
			t.Fatal(err)
		}
	}
	advance := func(id string) {
		t.Helper()
		if err := epochs.Advance(context.Background(), ssdomain.AgentRevocationEpoch{
			AgentID: id, TenantID: tenancydomain.DefaultTenantID, Epoch: epoch,
			Reason: ssdomain.RevocationReasonAgentCredentialUnbound, AdvancedAt: epoch,
		}); err != nil {
			t.Fatal(err)
		}
	}
	// unbound は束縛を解除してエポックを進めた Agent、rebound はその後でクライアントを
	// 束縛し直した先の Agent である。stopped はエポックより先に状態だけが止まった Agent。
	seed("unbound", idmdomain.AgentStatusActive)
	advance("unbound")
	seed("rebound", idmdomain.AgentStatusActive)
	if _, err := agents.AddBinding(context.Background(), &agentdomain.AgentCredentialBinding{
		AgentID: "rebound", ClientID: "shared_client", CreatedAt: epoch,
	}); err != nil {
		t.Fatal(err)
	}
	seed("stopped", idmdomain.AgentStatusDisabled)
	seed("subject", idmdomain.AgentStatusActive)

	introspector := &fakeIntrospector{}
	deps := IntrospectDeps{Introspector: introspector, AgentRepo: agents, RevocationEpochRepo: epochs}
	cases := []struct {
		name   string
		token  ports.IntrospectionResult
		active bool
	}{
		{"束縛を解除した Agent の agent_id", ports.IntrospectionResult{ClientID: "released_client", AgentID: "unbound", Iat: before}, false},
		{"別の Agent へ束縛し直したクライアントの解除前のトークン", ports.IntrospectionResult{ClientID: "shared_client", AgentChain: []string{"unbound"}, Iat: before}, false},
		{"束縛し直した先の Agent のもとで発行したトークン", ports.IntrospectionResult{ClientID: "shared_client", AgentChain: []string{"rebound"}, Iat: after}, true},
		{"削除した Agent", ports.IntrospectionResult{ClientID: "deleted_client", AgentChain: []string{"deleted"}, Iat: after}, false},
		{"エポックの前に状態が止まった Agent", ports.IntrospectionResult{ClientID: "stopped_client", AgentChain: []string{"stopped"}, Iat: after}, false},
		{"連鎖の中間の Agent のエポックより前", ports.IntrospectionResult{ClientID: "c_client", AgentChain: []string{"subject", "unbound"}, Iat: before}, false},
		{"連鎖の全員が Active でエポックより後", ports.IntrospectionResult{ClientID: "c_client", AgentChain: []string{"subject", "unbound"}, Iat: after}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token := tc.token
			token.Active, token.Sub, token.Exp = true, "sub", epoch.Add(time.Hour).Unix()
			introspector.result = &token
			resp, err := IntrospectToken(ctx, deps, IntrospectInput{Token: "t", TokenTypeHint: "access_token"}, epoch)
			if err != nil {
				t.Fatal(err)
			}
			if resp.Active != tc.active {
				t.Fatalf("active=%v、want %v", resp.Active, tc.active)
			}
			if !tc.active {
				assertOnlyInactive(t, resp)
			}
		})
	}
}

type failingAgentRepository struct {
	agentports.AgentRepository
}

func (failingAgentRepository) FindByID(context.Context, string, string) (*agentdomain.Agent, error) {
	return nil, errors.New("agent store unavailable")
}

// 発行時の Agent を読めないときは、有効として返さない。
//
//spec:covers REQ-OAUTH2-012: 発行時の Agent の状態を読めない場合はトークンを有効として返さない
func TestIntrospectToken_IssuanceAgentLookupFailureIsNotActive(t *testing.T) {
	introspector := &fakeIntrospector{result: &ports.IntrospectionResult{
		Active: true, Sub: "sub", ClientID: "client", AgentChain: []string{"agent_1"},
		Iat: time.Now().Unix(), Exp: time.Now().Add(time.Hour).Unix(),
	}}
	resp, err := IntrospectToken(tenantContext(), IntrospectDeps{
		Introspector: introspector, AgentRepo: failingAgentRepository{AgentRepository: idmanagement.NewMemoryModule().AgentRepo},
		RevocationEpochRepo: sharedsignals.NewAgentRevocationEpochRepository(),
	}, IntrospectInput{Token: "t", TokenTypeHint: "access_token"}, time.Now())
	if err == nil && resp.Active {
		t.Fatal("発行時の Agent を読めないトークンを active=true で返した")
	}
}
