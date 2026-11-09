package handlers_http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/testing_clientauth"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type agentIssuanceFixture struct {
	*refusalFixture
	agents agentports.AgentRepository
	key    *testing_clientauth.PrivateKeyJWT
}

func newAgentIssuanceFixture(t *testing.T) agentIssuanceFixture {
	t.Helper()
	agents := idmanagement.NewMemoryModule().AgentRepo
	f := newRefusalServer(t, func(d *httpadapter.Deps) {
		d.IdManagement.AgentRepo = agents
		d.OAuth2.ClientAssertionReplayStore = oauth2memory.NewClientAssertionReplayStore()
		d.OAuth2.ApprovalRequestStore = oauth2memory.NewApprovalRequestStore()
	})
	key := testing_clientauth.NewPrivateKeyJWT(t, refusalClientID)
	client, err := f.clients.FindByID(context.Background(), tenancydomain.DefaultTenantID, refusalClientID)
	if err != nil || client == nil {
		t.Fatalf("client=%v err=%v", client, err)
	}
	client.TokenEndpointAuthMethod = domain.AuthMethodPrivateKeyJwt
	client.JWKS = key.JWKS()
	client.GrantTypes = append(client.GrantTypes, spec.GrantTokenExchange, spec.GrantCiba, spec.GrantJWTBearer, spec.GrantJWTDPoP)
	f.clients.Seed(client)
	fixture := agentIssuanceFixture{refusalFixture: f, agents: agents, key: key}
	fixture.setState(t, idmdomain.AgentStatusActive, "active")
	if _, err := agents.AddBinding(context.Background(), &agentdomain.AgentCredentialBinding{
		AgentID: "agent-issuance", ClientID: refusalClientID, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	return fixture
}

// 所有者はトークンの利用者と分け、利用者の無効化による拒否で隠れないようにする。
func (f agentIssuanceFixture) setState(t *testing.T, status idmdomain.AgentStatus, ownerState string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	ownerID := "issuance-owner"
	if ownerState == "missing" {
		ownerID = "missing-owner"
	} else {
		ownerStatus := idmdomain.UserStatusActive
		if ownerState == "disabled" {
			ownerStatus = idmdomain.UserStatusDisabled
		}
		if err := f.users.Save(ctx, &userdomain.User{
			ID: ownerID, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: ownerID,
			Lifecycle: userdomain.UserLifecycle{Status: ownerStatus}, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.agents.Save(ctx, &agentdomain.Agent{
		ID: "agent-issuance", TenantID: tenancydomain.DefaultTenantID, Name: "issuance",
		Kind: idmdomain.AgentKindAutonomous, OwnerUserID: ownerID, Status: status,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
}

func (f agentIssuanceFixture) post(t *testing.T, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	return postTokenForm(t, f.refusalFixture, tenancydomain.DefaultRealm,
		f.key.AuthenticateForm(t, form, refusalTokenHTU))
}

func (f agentIssuanceFixture) grant(t *testing.T, grant spec.GrantType, credential string) url.Values {
	t.Helper()
	switch grant {
	case spec.GrantAuthorizationCode:
		seedAuthorizationCode(t, f.refusalFixture, tenancydomain.DefaultTenantID, credential, []string{"openid", "offline_access"})
		return url.Values{
			"grant_type": {string(grant)}, "code": {credential},
			"redirect_uri": {refusalRedirectURI}, "code_verifier": {refusalVerifier},
		}
	case spec.GrantRefreshToken:
		return url.Values{"grant_type": {string(grant)}, "refresh_token": {credential}}
	case spec.GrantDeviceCode:
		seedApprovedDeviceCode(t, f.refusalFixture, tenancydomain.DefaultTenantID, credential, "user-"+credential)
		return url.Values{"grant_type": {string(grant)}, "device_code": {credential}}
	default:
		t.Fatalf("unexpected grant %s", grant)
		return nil
	}
}

//spec:covers REQ-OAUTH2-046, EX-OAUTH2-046-05, EX-OAUTH2-046-06, EX-OAUTH2-046-07: 有効な各グラントでも Agent と所有者の停止後は 401 invalid_client となり、資格情報と Agent を変えず、発行イベントを出さない。状態を戻すと同じ資格情報で発行できる。
//verify:operation Token: 認可コード、リフレッシュ、デバイスコードの成功 JSON と停止後の 401 invalid_client の JSON 応答
func TestStoppedAgentCannotExchangeValidGrants(t *testing.T) {
	for _, grant := range []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken, spec.GrantDeviceCode} {
		for _, state := range []struct {
			name  string
			agent idmdomain.AgentStatus
			owner string
		}{
			{"disabled-agent", idmdomain.AgentStatusDisabled, "active"},
			{"killed-agent", idmdomain.AgentStatusKilled, "active"},
			{"disabled-owner", idmdomain.AgentStatusActive, "disabled"},
			{"missing-owner", idmdomain.AgentStatusActive, "missing"},
		} {
			t.Run(string(grant)+"/"+state.name, func(t *testing.T) {
				f := newAgentIssuanceFixture(t)
				// リフレッシュは保存値の捏造を避け、実際の認可コード交換から取得する。
				initial := oauthContractJSON(t, f.post(t, f.grant(t, spec.GrantAuthorizationCode, "initial")), http.StatusOK)
				refresh, ok := initial["refresh_token"].(string)
				if !ok || refresh == "" {
					t.Fatalf("initial response has no refresh token: %v", initial)
				}
				credential := "before"
				if grant == spec.GrantRefreshToken {
					credential = refresh
				}
				accepted := oauthContractJSON(t, f.post(t, f.grant(t, grant, credential)), http.StatusOK)
				if token, ok := accepted["access_token"].(string); !ok || token == "" || accepted["token_type"] != "Bearer" {
					t.Fatalf("accepted=%v", accepted)
				}
				credential = "after"
				if grant == spec.GrantRefreshToken {
					credential, ok = accepted["refresh_token"].(string)
					if !ok || credential == "" {
						t.Fatalf("rotated refresh token missing: %v", accepted)
					}
				}
				form := f.grant(t, grant, credential)
				f.setState(t, state.agent, state.owner)
				ctx := context.Background()
				before, err := f.agents.FindByID(ctx, tenancydomain.DefaultTenantID, "agent-issuance")
				if err != nil {
					t.Fatal(err)
				}
				eventsBefore := len(*f.events)
				refused := f.post(t, form)
				oauthContractError(t, refused, http.StatusUnauthorized, "invalid_client")
				assertNoTokenIssued(t, refused.Body.Bytes())
				if len(*f.events) != eventsBefore {
					t.Fatalf("refusal emitted events: %v", (*f.events)[eventsBefore:])
				}
				after, err := f.agents.FindByID(ctx, tenancydomain.DefaultTenantID, "agent-issuance")
				if err != nil || !reflect.DeepEqual(before, after) {
					t.Fatalf("agent changed: before=%+v after=%+v err=%v", before, after, err)
				}
				switch grant {
				case spec.GrantAuthorizationCode:
					record, err := f.codes.Find(ctx, credential)
					if err != nil || record == nil || record.State != spec.AuthCodeRecordIssued {
						t.Fatalf("code consumed: %+v err=%v", record, err)
					}
				case spec.GrantRefreshToken:
					record, err := f.refreshes.FindByHash(ctx, domain.HashRefreshToken(credential))
					if err != nil || record == nil || record.Rotated || record.Revoked {
						t.Fatalf("refresh token consumed: %+v err=%v", record, err)
					}
				case spec.GrantDeviceCode:
					record, err := f.devices.FindByDeviceCodeHash(ctx, domain.HashDeviceCode(credential))
					if err != nil || record == nil || record.State != spec.DeviceFlowApproved {
						t.Fatalf("device code consumed: %+v err=%v", record, err)
					}
				}
				f.setState(t, idmdomain.AgentStatusActive, "active")
				restored := oauthContractJSON(t, f.post(t, form), http.StatusOK)
				if token, ok := restored["access_token"].(string); !ok || token == "" {
					t.Fatalf("restored=%v", restored)
				}
			})
		}
	}
}

//spec:covers REQ-OAUTH2-046: 全グラントで Agent と所有者の停止を資格情報の検証より先に拒否し、トークンを返さない。
func TestStoppedAgentIsRefusedBeforeEveryGrant(t *testing.T) {
	for _, owner := range []string{"active", "disabled"} {
		f := newAgentIssuanceFixture(t)
		status := idmdomain.AgentStatusActive
		if owner == "active" {
			status = idmdomain.AgentStatusDisabled
		}
		f.setState(t, status, owner)
		for _, grant := range []spec.GrantType{
			spec.GrantAuthorizationCode, spec.GrantRefreshToken, spec.GrantClientCredentials,
			spec.GrantDeviceCode, spec.GrantTokenExchange, spec.GrantCiba, spec.GrantJWTBearer, spec.GrantJWTDPoP,
		} {
			t.Run(owner+"/"+string(grant), func(t *testing.T) {
				refused := f.post(t, url.Values{
					"grant_type": {string(grant)}, "code": {"invalid"}, "refresh_token": {"invalid"},
					"device_code": {"invalid"}, "auth_req_id": {"invalid"}, "assertion": {"invalid"},
					"subject_token": {"invalid"}, "subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
				})
				oauthContractError(t, refused, http.StatusUnauthorized, "invalid_client")
				assertNoTokenIssued(t, refused.Body.Bytes())
			})
		}
	}
}
