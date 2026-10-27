package usecases_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedusecases "github.com/ambi/idmagic/backend/oauth2/internal/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// 認証方式の 5 値すべてを、束縛がある側とない側の両方で踏む。片側だけでは、
// 許可集合を取り違えた実装と、束縛を見ずに一律で拒否する実装を見分けられない。
//
//spec:covers REQ-OAUTH2-055: Agent に束縛したクライアントは private_key_jwt と tls_client_auth だけが通り、共有シークレットの 2 方式と none は invalid_client で拒否されて拒否理由のイベントが残り、束縛のないクライアントは方式によらず通ること。
func TestRejectAgentClientWithoutKeyProof(t *testing.T) {
	ctx := tenantContext(tenancydomain.DefaultTenantID)
	now := time.Now().UTC()

	agentRepo := idmanagement.NewMemoryModule().AgentRepo
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

	cases := []struct {
		method   domain.TokenEndpointAuthMethod
		accepted bool
	}{
		{domain.AuthMethodPrivateKeyJwt, true},
		{domain.AuthMethodTlsClientAuth, true},
		{domain.AuthMethodClientSecretBasic, false},
		{domain.AuthMethodClientSecretPost, false},
		{domain.AuthMethodNone, false},
	}
	for _, tc := range cases {
		t.Run("bound/"+string(tc.method), func(t *testing.T) {
			var events []spec.DomainEvent
			deps := sharedusecases.AgentIssuanceDeps{
				AgentRepo: agentRepo, Emit: func(event spec.DomainEvent) { events = append(events, event) },
			}
			client := &domain.OAuth2Client{ClientID: "agent_client", TokenEndpointAuthMethod: tc.method}

			err := sharedusecases.RejectAgentClientWithoutKeyProof(ctx, deps, client, "client_credentials", now)

			if tc.accepted {
				if err != nil || len(events) != 0 {
					t.Fatalf("err=%v events=%v, want acceptance without events", err, events)
				}
				return
			}
			assertInvalidClient(t, err)
			if len(events) != 1 {
				t.Fatalf("events=%v, want one AgentClientAuthenticationRejected", events)
			}
			rejected, ok := events[0].(*domain.AgentClientAuthenticationRejected)
			if !ok {
				t.Fatalf("event=%T, want *AgentClientAuthenticationRejected", events[0])
			}
			want := domain.AgentClientAuthenticationRejected{
				At: now, TenantID: tenancydomain.DefaultTenantID, AgentID: "agent_1", ClientID: "agent_client",
				TokenEndpointAuthMethod: string(tc.method), GrantType: "client_credentials",
			}
			if *rejected != want {
				t.Fatalf("event=%+v, want %+v", *rejected, want)
			}
			// 監査は TypeSpec の項目名で payload を読む。Go のフィールド値が正しくても、
			// JSON 名が違えば拒否の理由は検索できない。
			raw, err := json.Marshal(rejected)
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]any
			if err := json.Unmarshal(raw, &payload); err != nil {
				t.Fatal(err)
			}
			wantPayload := map[string]any{
				"tenantId": tenancydomain.DefaultTenantID, "agentId": "agent_1", "clientId": "agent_client",
				"tokenEndpointAuthMethod": string(tc.method), "grantType": "client_credentials",
			}
			if !reflect.DeepEqual(payload, wantPayload) {
				t.Fatalf("payload=%v, want %v", payload, wantPayload)
			}
		})
		t.Run("unbound/"+string(tc.method), func(t *testing.T) {
			deps := sharedusecases.AgentIssuanceDeps{AgentRepo: agentRepo}
			client := &domain.OAuth2Client{ClientID: "plain_client", TokenEndpointAuthMethod: tc.method}
			if err := sharedusecases.RejectAgentClientWithoutKeyProof(ctx, deps, client, "client_credentials", now); err != nil {
				t.Fatalf("unbound client with %s refused: %v", tc.method, err)
			}
		})
	}
}
