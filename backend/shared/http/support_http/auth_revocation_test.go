package support_http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/token/usecases"
	ssmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

// TestResolveAuthnContextAppliesRevocation — RED: REQ-OAUTH2-047
// (spec/contexts/oauth2/SPECIFICATION.md)。admin / account portal の Bearer 認証は
// /introspect と同じ失効判定 (AgentRevocationEpoch と AccessTokenDenylist) を通る。
// wi-58 T006 の時点では、この経路だけが両判定を迂回していた。
func TestResolveAuthnContextAppliesRevocation(t *testing.T) {
	now := time.Now().UTC()
	const clientID = "agent_client"

	agentRepo := agentmemory.NewAgentRepository()
	if err := agentRepo.Save(context.Background(), &agentdomain.Agent{
		ID: "agent_1", TenantID: tenancydomain.DefaultTenantID, Name: "agent_1",
		Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "owner_1", Status: idmdomain.AgentStatusKilled,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed agent: %v", err)
	}
	if _, err := agentRepo.AddBinding(context.Background(), &agentdomain.AgentCredentialBinding{
		AgentID: "agent_1", ClientID: clientID, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	epochRepo := ssmemory.NewAgentRevocationEpochRepository()
	if err := epochRepo.Advance(context.Background(), ssdomain.AgentRevocationEpoch{
		AgentID: "agent_1", TenantID: tenancydomain.DefaultTenantID, Epoch: now,
		Reason: ssdomain.RevocationReasonAgentKilled, AdvancedAt: now,
	}); err != nil {
		t.Fatalf("seed epoch: %v", err)
	}

	denylist := oauth2memory.NewAccessTokenDenylist()
	if err := denylist.Add(context.Background(), "jti-denied", now.Add(time.Hour)); err != nil {
		t.Fatalf("seed denylist: %v", err)
	}

	for _, tc := range []struct {
		name          string
		result        *oauthports.IntrospectionResult
		authenticated bool
	}{
		{
			name: "issued before the agent revocation epoch",
			result: &oauthports.IntrospectionResult{
				Active: true, Sub: "user-1", Scope: "idmagic.admin", JTI: "jti-1",
				ClientID: clientID, Iat: now.Add(-time.Minute).Unix(),
			},
		},
		{
			name: "jti on the access token denylist",
			result: &oauthports.IntrospectionResult{
				Active: true, Sub: "user-1", Scope: "idmagic.admin", JTI: "jti-denied",
				ClientID: clientID, Iat: now.Add(time.Minute).Unix(),
			},
		},
		{
			name: "issued after the agent revocation epoch",
			result: &oauthports.IntrospectionResult{
				Active: true, Sub: "user-1", Scope: "idmagic.admin", JTI: "jti-2",
				ClientID: clientID, Iat: now.Add(time.Minute).Unix(),
			},
			authenticated: true,
		},
		{
			name: "client bound to no agent",
			result: &oauthports.IntrospectionResult{
				Active: true, Sub: "user-1", Scope: "idmagic.admin", JTI: "jti-3",
				ClientID: "plain_client", Iat: now.Add(-time.Hour).Unix(),
			},
			authenticated: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/users", http.NoBody)
			req.Header.Set("Authorization", "Bearer jwt")
			c := e.NewContext(req, httptest.NewRecorder())
			a := Authenticator{
				TokenIntrospector: authTestIntrospector{result: tc.result},
				Revocation: tokenusecases.IntrospectDeps{
					AgentRepo: agentRepo, RevocationEpochRepo: epochRepo, AccessTokenDenylist: denylist,
				},
			}

			got, err := a.resolveAuthnContext(c)
			if tc.authenticated {
				if err != nil {
					t.Fatal(err)
				}
				if got == nil || got.UserID != "user-1" {
					t.Fatalf("authn=%+v", got)
				}
				return
			}
			var tokenErr *InvalidTokenError
			if !errors.As(err, &tokenErr) {
				t.Fatalf("err=%v authn=%+v; want InvalidTokenError", err, got)
			}
		})
	}
}
