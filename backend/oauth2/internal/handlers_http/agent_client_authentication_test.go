package handlers_http_test

// REQ-OAUTH2-055 を /token の経路で確かめる。判定そのものは usecase 層の
// TestRejectAgentClientWithoutKeyProof が 5 つの認証方式で持ち、ここは /token が
// グラントの分岐より前でその判定を通していることを見る。

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/testing_clientauth"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	agentClientID     = "agent-app"
	agentClientSecret = "agent-secret"
	agentTokenHTU     = "http://test/realms/default/token"
)

// agentClientGrants は Agent のクライアントに宣言するグラントである。Agent を解決する
// グラント (client_credentials、トークン交換) と解決しないグラントの両方を含める。
var agentClientGrants = []spec.GrantType{
	spec.GrantClientCredentials, spec.GrantTokenExchange,
	spec.GrantAuthorizationCode, spec.GrantRefreshToken,
}

// seedActiveAgentBoundTo は Active な所有者と Autonomous な Agent を作り、clientID に束縛する。
// 所有者と区分の拒否 (REQ-OAUTH2-046、REQ-OAUTH2-050) に当たらない Agent にしておき、
// 拒否が認証方式だけから来ることを保つ。
func seedActiveAgentBoundTo(t *testing.T, f tokenFixture, clientID string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	if err := f.userRepo.Save(ctx, &userdomain.User{
		ID: "owner-1", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "owner-1",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive, StatusChangedAt: &now},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	if err := f.agentRepo.Save(ctx, &agentdomain.Agent{
		ID: "agent-1", TenantID: tenancydomain.DefaultTenantID, Name: "agent-1",
		Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "owner-1", Status: idmdomain.AgentStatusActive,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed agent: %v", err)
	}
	if _, err := f.agentRepo.AddBinding(ctx, &agentdomain.AgentCredentialBinding{
		AgentID: "agent-1", ClientID: clientID, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed binding: %v", err)
	}
}

func seedSharedSecretAgentClient(f tokenFixture, method domain.TokenEndpointAuthMethod) {
	hash := domain.HashClientSecret(agentClientSecret)
	f.clientRepo.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: agentClientID, ClientSecretHash: &hash,
		ClientType: spec.ClientConfidential, RedirectURIs: []string{"https://app.example/cb"},
		GrantTypes: agentClientGrants, ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: method, Scope: "openid profile",
		FapiProfile: domain.FapiNone, CreatedAt: time.Now().UTC(),
	})
}

// seedPrivateKeyJWTAgentClient は private_key_jwt のクライアントを作り、その署名鍵を返す。
func seedPrivateKeyJWTAgentClient(t *testing.T, f tokenFixture) *testing_clientauth.PrivateKeyJWT {
	t.Helper()
	credential := testing_clientauth.NewPrivateKeyJWT(t, agentClientID)
	f.clientRepo.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: agentClientID,
		ClientType: spec.ClientConfidential, RedirectURIs: []string{"https://app.example/cb"},
		GrantTypes: agentClientGrants, ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: domain.AuthMethodPrivateKeyJwt,
		JWKS:                    credential.JWKS(), Scope: "openid profile",
		FapiProfile: domain.FapiNone, CreatedAt: time.Now().UTC(),
	})
	return credential
}

// agentClientCredentialsForm は seedPrivateKeyJWTAgentClient のクライアントとして認証する
// client_credentials の要求である。
func agentClientCredentialsForm(t *testing.T, credential *testing_clientauth.PrivateKeyJWT) url.Values {
	t.Helper()
	return credential.AuthenticateForm(t, url.Values{"grant_type": {"client_credentials"}, "scope": {"openid"}}, agentTokenHTU)
}

// grantForms はグラントごとに、クライアント認証を除いた要求である。Agent を解決しない
// グラントの資格情報 (code、refresh_token、subject_token) はわざと無効にしてある。
// 認証方式の判定がグラントの処理より前にあれば invalid_client、後ろにあれば
// invalid_grant などグラント側のエラーになり、判定の位置を応答から読める。
func grantForms() map[string]url.Values {
	return map[string]url.Values{
		"client_credentials": {"grant_type": {"client_credentials"}, "scope": {"openid"}},
		"token_exchange": {
			"grant_type":         {string(spec.GrantTokenExchange)},
			"subject_token":      {"not-a-token"},
			"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		},
		"authorization_code": {
			"grant_type": {"authorization_code"}, "code": {"not-a-code"},
			"redirect_uri": {"https://app.example/cb"}, "code_verifier": {strings.Repeat("v", 43)},
		},
		"refresh_token": {"grant_type": {"refresh_token"}, "refresh_token": {"not-a-refresh-token"}},
	}
}

func postAgentToken(t *testing.T, f tokenFixture, form url.Values, basic bool) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/realms/default/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if basic {
		req.SetBasicAuth(agentClientID, agentClientSecret)
	}
	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, req)
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec, body
}

// 正しいシークレットで認証は成立している。拒否は資格情報の誤りではなく、Agent の
// クライアントとして認証方式が足りないことから来る。
//
//spec:covers AIMS00-AGENT-CLIENT-AUTH, EX-OAUTH2-055-01: Agent に束縛した client_secret_basic と client_secret_post のクライアントは、正しいシークレットでも client_credentials、トークン交換、認可コード、リフレッシュトークンのすべてで 401 invalid_client を受け、トークンを得られないこと。
func TestAgentBoundSharedSecretClientIsRefusedOnEveryGrant(t *testing.T) {
	for _, method := range []domain.TokenEndpointAuthMethod{
		domain.AuthMethodClientSecretBasic, domain.AuthMethodClientSecretPost,
	} {
		f := newTokenServer(t)
		seedSharedSecretAgentClient(f, method)
		seedActiveAgentBoundTo(t, f, agentClientID)

		for grant, form := range grantForms() {
			t.Run(string(method)+"/"+grant, func(t *testing.T) {
				basic := method == domain.AuthMethodClientSecretBasic
				if !basic {
					form.Set("client_id", agentClientID)
					form.Set("client_secret", agentClientSecret)
				}
				rec, body := postAgentToken(t, f, form, basic)
				if rec.Code != http.StatusUnauthorized || body["error"] != "invalid_client" {
					t.Fatalf("status=%d body=%s, want 401 invalid_client", rec.Code, rec.Body.String())
				}
				assertNoAccessToken(t, rec)
			})
		}
	}
}

// 束縛は作成時にしか検査されない、という実装を見分ける。束縛した時点では鍵の所有を
// 証明するクライアントだったものを、後から共有シークレットへ変える。
//
//spec:covers EX-OAUTH2-055-02: 束縛の後で client_secret_post へ変えたクライアントのトークン交換は 401 invalid_client で拒否され、変える前の同じクライアントは client_credentials でトークンを得られること。
func TestAgentClientSwitchedToSharedSecretAfterBindingIsRefused(t *testing.T) {
	f := newTokenServer(t)
	key := seedPrivateKeyJWTAgentClient(t, f)
	seedActiveAgentBoundTo(t, f, agentClientID)

	if rec, body := postAgentToken(t, f, agentClientCredentialsForm(t, key), false); rec.Code != http.StatusOK || body["access_token"] == nil {
		t.Fatalf("前提が壊れている: 切り替え前の発行が status=%d body=%s", rec.Code, rec.Body.String())
	}

	seedSharedSecretAgentClient(f, domain.AuthMethodClientSecretPost)
	after := grantForms()["token_exchange"]
	after.Set("client_id", agentClientID)
	after.Set("client_secret", agentClientSecret)
	rec, body := postAgentToken(t, f, after, false)
	if rec.Code != http.StatusUnauthorized || body["error"] != "invalid_client" {
		t.Fatalf("status=%d body=%s, want 401 invalid_client", rec.Code, rec.Body.String())
	}
	assertNoAccessToken(t, rec)
}

//spec:covers EX-OAUTH2-055-03: Agent に束縛した private_key_jwt のクライアントは client_credentials で agent_id に Agent を載せたアクセストークンを得ること。
func TestAgentBoundPrivateKeyJWTClientObtainsAnAgentToken(t *testing.T) {
	f := newTokenServer(t)
	key := seedPrivateKeyJWTAgentClient(t, f)
	seedActiveAgentBoundTo(t, f, agentClientID)

	rec, body := postAgentToken(t, f, agentClientCredentialsForm(t, key), false)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200", rec.Code, rec.Body.String())
	}
	token, _ := body["access_token"].(string)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("access_token=%q is not a JWS", token)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatal(err)
	}
	if claims["agent_id"] != "agent-1" {
		t.Fatalf("agent_id=%v, want agent-1", claims["agent_id"])
	}
}
