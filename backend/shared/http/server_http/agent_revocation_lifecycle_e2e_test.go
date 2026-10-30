package server_http_test

// Agent の停止が発行済みのトークンへ届くことを、束縛を変える操作と委譲の連鎖を含めて
// 製品と同じ `Register` の組み立てで確かめる。
//
// 引き金は管理 API と `/revoke` への要求にし、結果は `/introspect` と `/token` の
// トークン交換で読む。保存先へ直接書くと、IdManagement の確定から SharedSignals の
// エポックへ届く経路と、OAuth2 が発行時の Agent を識別する経路のどちらも通らない。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/testing_clientauth"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const agentRevocationResource = "https://mcp.agent-revocation.example/"

// agentRevocationFixture は、Agent ごとに private_key_jwt のクライアントを 1 つ束縛し、
// client_credentials とトークン交換の両方を使えるようにした組み立てである。
type agentRevocationFixture struct {
	s           *stack.Stack
	adminToken  string
	credentials map[string]*testing_clientauth.PrivateKeyJWT
}

func newAgentRevocationFixture(t *testing.T, agentIDs ...string) *agentRevocationFixture {
	t.Helper()
	return newAgentRevocationFixtureWith(t, nil, agentIDs...)
}

func newAgentRevocationFixtureWith(t *testing.T, extra []stack.Option, agentIDs ...string) *agentRevocationFixture {
	t.Helper()
	options := append([]stack.Option{stack.WithOAuth2Clients(), stack.WithTokenIssuance(), stack.WithApiTokens(), stack.WithAgentRevocation()}, extra...)
	s := stack.New(t, options...)
	now := time.Now().UTC()
	if err := s.McpResourceServers.Save(context.Background(), &oauthdomain.McpResourceServer{
		TenantID: tenancydomain.DefaultTenantID, ID: "mcp-agent-revocation", Resource: agentRevocationResource,
		Name: "agent revocation", Scopes: []string{"tools:read"}, State: oauthdomain.McpResourceServerActive,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	adminToken, _ := s.IssueApiToken(t, "default", apitokendomain.ScopeAgentsWrite, apitokendomain.ScopeAgentsRead)
	f := &agentRevocationFixture{s: s, adminToken: adminToken, credentials: map[string]*testing_clientauth.PrivateKeyJWT{}}
	for _, agentID := range agentIDs {
		f.seedAgent(t, agentID)
		f.bind(t, agentID, agentClientID(agentID))
	}
	return f
}

func agentClientID(agentID string) string {
	return "agent-client-" + strings.ToLower(agentID)
}

// seedAgent は Agent と、その Agent が使うクライアントを置く。束縛は管理 API で作る。
func (f *agentRevocationFixture) seedAgent(t *testing.T, agentID string) {
	t.Helper()
	f.seedClient(t, agentID)
	now := time.Now().UTC()
	if err := f.s.Agents.Save(context.Background(), &agentdomain.Agent{
		ID: agentID, TenantID: tenancydomain.DefaultTenantID, Name: agentID, Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: stack.UserID, Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed agent %s: %v", agentID, err)
	}
}

// seedClient は Agent が使う private_key_jwt のクライアントだけを置く。
func (f *agentRevocationFixture) seedClient(t *testing.T, agentID string) {
	t.Helper()
	now := time.Now().UTC()
	clientID := agentClientID(agentID)
	credential := testing_clientauth.NewPrivateKeyJWT(t, clientID)
	f.credentials[clientID] = credential
	if err := f.s.Clients.Save(context.Background(), &oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: clientID,
		ClientType:              spec.ClientConfidential,
		GrantTypes:              []spec.GrantType{spec.GrantClientCredentials, spec.GrantTokenExchange},
		TokenEndpointAuthMethod: oauthdomain.AuthMethodPrivateKeyJwt,
		JWKS:                    credential.JWKS(),
		Scope:                   "tools:read",
		FapiProfile:             oauthdomain.FapiNone,
		CreatedAt:               now,
	}); err != nil {
		t.Fatal(err)
	}
}

func (f *agentRevocationFixture) admin(t *testing.T, method, path string, body any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	request := httptest.NewRequest(method, stack.Issuer+"/realms/default"+path, reader)
	request.Header.Set("Authorization", "Bearer "+f.adminToken)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	if recorder.Code/100 != 2 {
		t.Fatalf("%s %s status=%d body=%s", method, path, recorder.Code, recorder.Body.String())
	}
}

func (f *agentRevocationFixture) bind(t *testing.T, agentID, clientID string) {
	t.Helper()
	f.admin(t, http.MethodPost, "/api/admin/v1/agents/"+agentID+"/credentials", map[string]any{"client_id": clientID})
}

func (f *agentRevocationFixture) unbind(t *testing.T, agentID, clientID string) {
	t.Helper()
	f.admin(t, http.MethodDelete, "/api/admin/v1/agents/"+agentID+"/credentials/"+clientID, nil)
}

func (f *agentRevocationFixture) post(t *testing.T, path, clientID string, values url.Values) (int, map[string]any) {
	t.Helper()
	endpoint := stack.Issuer + "/realms/default/" + path
	form := f.credentials[clientID].AuthenticateForm(t, values, stack.Issuer+"/realms/default/token")
	request := httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	body := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &body)
	return recorder.Code, body
}

func (f *agentRevocationFixture) clientCredentials(t *testing.T, clientID string) string {
	t.Helper()
	status, body := f.post(t, "token", clientID, url.Values{"grant_type": {"client_credentials"}, "scope": {"tools:read"}})
	token, _ := body["access_token"].(string)
	if status != http.StatusOK || token == "" {
		t.Fatalf("client_credentials for %s status=%d body=%v", clientID, status, body)
	}
	return token
}

func (f *agentRevocationFixture) exchange(t *testing.T, clientID, subjectToken string) (int, map[string]any) {
	t.Helper()
	return f.post(t, "token", clientID, url.Values{
		"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token":      {subjectToken},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		"resource":           {agentRevocationResource},
		"scope":              {"tools:read"},
	})
}

func (f *agentRevocationFixture) mustExchange(t *testing.T, clientID, subjectToken string) string {
	t.Helper()
	status, body := f.exchange(t, clientID, subjectToken)
	token, _ := body["access_token"].(string)
	if status != http.StatusOK || token == "" {
		t.Fatalf("exchange by %s status=%d body=%v", clientID, status, body)
	}
	return token
}

func (f *agentRevocationFixture) active(t *testing.T, token string) bool {
	t.Helper()
	return f.s.Introspect(t, "default", token)["active"] == true
}

// requireActive は停止の前に同じ入口が通ることを観測する。停止の後の拒否が停止の結果だと
// 読めるのは、直前に同じトークンが有効だったときだけである。
func (f *agentRevocationFixture) requireActive(t *testing.T, what, token string) {
	t.Helper()
	if !f.active(t, token) {
		t.Fatalf("%s: 停止の前に active=false", what)
	}
}

func (f *agentRevocationFixture) requireInactive(t *testing.T, what, token string) {
	t.Helper()
	if f.active(t, token) {
		t.Fatalf("%s: 停止の後も active=true", what)
	}
}

// 対照：束縛を残したまま強制終了すれば、発行済みのトークンは無効になる。
//
//spec:covers REQ-SHAREDSIGNALS-001, REQ-OAUTH2-012: 束縛を残した強制終了の対照
func TestAgentKillWithBindingKeptRevokesIssuedToken(t *testing.T) {
	f := newAgentRevocationFixture(t, "A1")
	token := f.clientCredentials(t, agentClientID("A1"))
	f.requireActive(t, "A1 のトークン", token)

	f.admin(t, http.MethodPost, "/api/admin/v1/agents/A1/kill", nil)

	f.requireInactive(t, "強制終了した A1 のトークン", token)
}

// 束縛の解除は Agent のエポックを進める。解除の後は client_id から Agent を引けないので、
// 発行時の Agent をトークン自身から識別しない限り、進めたエポックを読めない。
//
//spec:covers REQ-SHAREDSIGNALS-001, REQ-OAUTH2-012: 束縛の解除
func TestUnbindingAgentCredentialRevokesIssuedToken(t *testing.T) {
	f := newAgentRevocationFixture(t, "A1")
	token := f.clientCredentials(t, agentClientID("A1"))
	f.requireActive(t, "A1 のトークン", token)

	f.unbind(t, "A1", agentClientID("A1"))

	f.requireInactive(t, "束縛を解除した A1 のトークン", token)
}

// 別の Agent へ束縛し直しても、前の Agent の停止は前の Agent のトークンへ効き続ける。
// 新しい Agent のもとで発行したトークンは、前の Agent の停止で失効しない。
//
//spec:covers REQ-SHAREDSIGNALS-001, REQ-OAUTH2-012: 別の Agent への再束縛
func TestRebindingCredentialKeepsPreviousAgentRevocation(t *testing.T) {
	f := newAgentRevocationFixture(t, "A1")
	f.seedAgent(t, "B1")
	clientID := agentClientID("A1")
	before := f.clientCredentials(t, clientID)
	f.requireActive(t, "A1 のもとで発行したトークン", before)

	f.unbind(t, "A1", clientID)
	f.bind(t, "B1", clientID)
	after := f.clientCredentials(t, clientID)

	f.requireInactive(t, "A1 から外したクライアントの、解除前のトークン", before)
	f.requireActive(t, "B1 のもとで発行したトークン", after)
}

// 削除は束縛を cascade で消す。削除した Agent のトークンを有効と判定しない。
//
//spec:covers REQ-SHAREDSIGNALS-001, REQ-OAUTH2-012: Agent の削除
func TestDeletingAgentRevokesIssuedToken(t *testing.T) {
	f := newAgentRevocationFixture(t, "A1")
	token := f.clientCredentials(t, agentClientID("A1"))
	f.requireActive(t, "A1 のトークン", token)

	f.admin(t, http.MethodDelete, "/api/admin/v1/agents/A1", nil)

	f.requireInactive(t, "削除した A1 のトークン", token)
}

// A のトークンを B が交換すると、交換後のトークンの client_id は B になる。A を止めたとき、
// 現在の client_id の束縛だけを見ると B しか読まれない。
//
//spec:covers REQ-SHAREDSIGNALS-001, REQ-OAUTH2-012: 委譲の subject の停止
func TestStoppingSubjectAgentRevokesExchangedToken(t *testing.T) {
	f := newAgentRevocationFixture(t, "A1", "B1")
	subject := f.clientCredentials(t, agentClientID("A1"))
	exchanged := f.mustExchange(t, agentClientID("B1"), subject)
	f.requireActive(t, "B1 が交換したトークン", exchanged)

	f.admin(t, http.MethodPost, "/api/admin/v1/agents/A1/disable", nil)

	f.requireInactive(t, "subject の A1 を止めた後の交換済みトークン", exchanged)
}

// A → B → C の連鎖で中間の B を止めると、C が持つトークンも無効になる。
//
//spec:covers REQ-SHAREDSIGNALS-001, REQ-OAUTH2-012: 委譲の中間の停止
func TestStoppingIntermediateAgentRevokesDelegatedToken(t *testing.T) {
	f := newAgentRevocationFixture(t, "A1", "B1", "C1")
	subject := f.clientCredentials(t, agentClientID("A1"))
	viaB := f.mustExchange(t, agentClientID("B1"), subject)
	viaC := f.mustExchange(t, agentClientID("C1"), viaB)
	f.requireActive(t, "C1 が交換したトークン", viaC)

	f.admin(t, http.MethodPost, "/api/admin/v1/agents/B1/disable", nil)

	f.requireInactive(t, "中間の B1 を止めた後の C1 のトークン", viaC)
	f.requireActive(t, "連鎖に B1 を含まない A1 のトークン", subject)
}

// 停止した Agent のトークンと、明示的に失効させたトークンは、交換の入力として受け付けない。
//
//spec:covers RFC8693-SUBJECT-TOKEN: イントロスペクションの失効判定を通らない subject_token
func TestTokenExchangeRejectsRevokedSubjectToken(t *testing.T) {
	t.Run("強制終了した Agent のトークン", func(t *testing.T) {
		f := newAgentRevocationFixture(t, "A1", "B1")
		subject := f.clientCredentials(t, agentClientID("A1"))
		f.admin(t, http.MethodPost, "/api/admin/v1/agents/A1/kill", nil)

		status, body := f.exchange(t, agentClientID("B1"), subject)

		if status != http.StatusBadRequest || body["error"] != "invalid_grant" {
			t.Fatalf("status=%d body=%v、want 400 invalid_grant", status, body)
		}
	})
	t.Run("RFC 7009 で失効させたトークン", func(t *testing.T) {
		f := newAgentRevocationFixture(t, "A1", "B1")
		subject := f.clientCredentials(t, agentClientID("A1"))
		if status, body := f.post(t, "revoke", agentClientID("A1"), url.Values{"token": {subject}, "token_type_hint": {"access_token"}}); status != http.StatusOK {
			t.Fatalf("revoke status=%d body=%v", status, body)
		}
		f.requireInactive(t, "失効させたトークン", subject)

		status, body := f.exchange(t, agentClientID("B1"), subject)

		if status != http.StatusBadRequest || body["error"] != "invalid_grant" {
			t.Fatalf("status=%d body=%v、want 400 invalid_grant", status, body)
		}
	})
}
