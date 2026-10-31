package server_http_test

// TokenVault を製品と同じ `Register` の組み立てで確かめる。上流は通信の層で演じるので、
// 認可、失効、外向きの要求の形が実際の経路を通る。

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	authentication "github.com/ambi/idmagic/backend/authentication"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	vaultAgent        = "A1"
	vaultAPIOrigin    = "https://api.upstream.example"
	vaultTokenURL     = "https://idp.upstream.example/token"
	vaultAuthorizeURL = "https://idp.upstream.example/authorize"
	vaultRevokeURL    = "https://idp.upstream.example/revoke"
	vaultRealm        = stack.Issuer + "/realms/default"
	vaultResource     = vaultRealm + "/api/token-vault/v1"
	vaultScope        = "tools:read"
)

// vaultUpstream は上流の認可サーバーと API を演じる。回転した更新トークンの再送を拒否する。
type vaultUpstream struct {
	mu            sync.Mutex
	verifiers     []string
	refreshes     int
	revoked       []string
	apiAuth       []string
	refreshToken  string
	accessToken   string
	expiresIn     int
	apiStatusFunc func(authorization string) int
	// refreshDelay は更新の応答を遅らせ、並行した要求を更新に重ねる。
	refreshDelay time.Duration
}

func (u *vaultUpstream) RoundTrip(request *http.Request) (*http.Response, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	recorder := httptest.NewRecorder()
	switch request.URL.String() {
	case vaultTokenURL:
		u.token(recorder, request)
	case vaultRevokeURL:
		_ = request.ParseForm()
		u.revoked = append(u.revoked, request.PostForm.Get("token"))
	default:
		authorization := request.Header.Get("Authorization")
		u.apiAuth = append(u.apiAuth, authorization)
		status := http.StatusOK
		if u.apiStatusFunc != nil {
			status = u.apiStatusFunc(authorization)
		}
		recorder.Header().Set("Set-Cookie", "upstream=1")
		recorder.Header().Set("Content-Type", "application/json")
		recorder.WriteHeader(status)
		_, _ = recorder.WriteString(`{"ok":true}`)
	}
	return recorder.Result(), nil
}

func (u *vaultUpstream) token(w http.ResponseWriter, request *http.Request) {
	_ = request.ParseForm()
	w.Header().Set("Content-Type", "application/json")
	if request.PostForm.Get("grant_type") == "authorization_code" {
		u.verifiers = append(u.verifiers, request.PostForm.Get("code_verifier"))
		u.accessToken, u.refreshToken = "upstream-access-1", "upstream-refresh-1"
	} else {
		if request.PostForm.Get("refresh_token") != u.refreshToken {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		u.refreshes++
		time.Sleep(u.refreshDelay)
		u.accessToken, u.refreshToken = "upstream-access-refreshed", "upstream-refresh-rotated"
	}
	_, _ = fmt.Fprintf(w, `{"access_token":%q,"token_type":"Bearer","refresh_token":%q,"expires_in":%d,"scope":"read"}`,
		u.accessToken, u.refreshToken, u.expiresIn)
}

type vaultFixture struct {
	*agentRevocationFixture
	upstream    *vaultUpstream
	cookie      *http.Cookie
	connectorID string
}

func newVaultFixture(t *testing.T) *vaultFixture {
	t.Helper()
	upstream := &vaultUpstream{expiresIn: 3600}
	f := &vaultFixture{upstream: upstream, agentRevocationFixture: newAgentRevocationFixtureWith(t, []stack.Option{
		stack.WithAuthorizationCodeFlow(), stack.WithTokenVault(&http.Client{Transport: upstream}),
	}, vaultAgent)}
	f.cookie = f.s.AdminSessionCookie(t, "default")
	now := time.Now().UTC()
	if err := f.s.McpResourceServers.Save(context.Background(), &oauthdomain.McpResourceServer{
		TenantID: tenancydomain.DefaultTenantID, ID: "token-vault", Resource: vaultResource, Name: "token vault",
		Scopes: []string{vaultScope}, State: oauthdomain.McpResourceServerActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	created := f.session(t, f.cookie, http.MethodPost, "/api/admin/v1/token-vault/connectors", map[string]any{
		"name": "upstream", "authorization_endpoint": vaultAuthorizeURL, "token_endpoint": vaultTokenURL, "revocation_endpoint": vaultRevokeURL,
		"client_id": "vault-client", "client_secret": "vault-client-secret", "allowed_scopes": []string{"read"},
		"rules": []map[string]any{{"id": "items", "method": "GET", "origin": vaultAPIOrigin, "path_prefix": "/v1/items", "required_scopes": []string{"read"}}},
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create connector: %d %s", created.Code, created.Body.String())
	}
	if strings.Contains(created.Body.String(), "vault-client-secret") {
		t.Fatal("the client secret appeared in the response")
	}
	var connector struct {
		ID string `json:"id"`
	}
	decodeVault(t, created, &connector)
	f.connectorID = connector.ID
	return f
}

func decodeVault(t *testing.T, recorder *httptest.ResponseRecorder, destination any) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), destination); err != nil {
		t.Fatalf("decode %s: %v", recorder.Body.String(), err)
	}
}

func (f *vaultFixture) session(t *testing.T, cookie *http.Cookie, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == nil {
		reader = strings.NewReader("")
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = strings.NewReader(string(raw))
	}
	request := httptest.NewRequest(method, vaultRealm+path, reader)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", stack.Issuer)
	request.Header.Set(support.CSRFHeader, "vault-csrf")
	request.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "vault-csrf"})
	if cookie != nil {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	return recorder
}

// userCookie は管理者とは別の本人のセッションを作る。
func (f *vaultFixture) userCookie(t *testing.T) *http.Cookie {
	t.Helper()
	authn, err := f.s.Sessions.Create(f.s.RealmContext(t, "default"), stack.UserID, []string{"pwd"}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: authentication.SessionCookie, Value: authn.SessionID}
}

func (f *vaultFixture) authorize(t *testing.T) url.Values {
	t.Helper()
	response := f.session(t, f.cookie, http.MethodPost, "/api/account/v1/token-vault/connections/"+f.connectorID+"/authorize", map[string]any{})
	if response.Code != http.StatusOK {
		t.Fatalf("authorize: %d %s", response.Code, response.Body.String())
	}
	var body struct {
		AuthorizationURL string `json:"authorization_url"`
	}
	decodeVault(t, response, &body)
	parsed, err := url.Parse(body.AuthorizationURL)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.Query()
}

func (f *vaultFixture) callback(t *testing.T, cookie *http.Cookie, query url.Values) string {
	t.Helper()
	response := f.session(t, cookie, http.MethodGet, "/api/token-vault/v1/callback/"+f.connectorID+"?"+query.Encode(), nil)
	if response.Code != http.StatusFound {
		t.Fatalf("callback: %d %s", response.Code, response.Body.String())
	}
	location, err := url.Parse(response.Header().Get("Location"))
	if err != nil || location.Path != "/realms/default/account/connections" {
		t.Fatalf("location: %q, %v", response.Header().Get("Location"), err)
	}
	return location.Query().Get("result")
}

func (f *vaultFixture) link(t *testing.T) {
	t.Helper()
	query := f.authorize(t)
	if result := f.callback(t, f.cookie, url.Values{"state": {query.Get("state")}, "code": {"code-1"}}); result != "linked" {
		t.Fatalf("link result = %q", result)
	}
}

// subjectToken は本人の認可を経たトークンを演じる。
func (f *vaultFixture) subjectToken(t *testing.T, details []spec.AuthorizationDetail) string {
	t.Helper()
	ctx := f.s.RealmContext(t, "default")
	client, err := f.s.Clients.FindByID(ctx, tenancydomain.DefaultTenantID, stack.ResourceServerClientID)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := f.s.Signer.SignAccessToken(ctx, oauthports.AccessTokenInput{
		Client: client, Sub: stack.AdminUserID, Scopes: []string{vaultScope}, AuthorizationDetails: details, Audiences: []string{vaultResource},
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// agentBoundToken は Agent に束縛したクライアントへの、Token Exchange を経ないトークンを演じる。
// 代行付与の撤回の判定が働かない形である。
func (f *vaultFixture) agentBoundToken(t *testing.T, details []spec.AuthorizationDetail) string {
	t.Helper()
	ctx := f.s.RealmContext(t, "default")
	client, err := f.s.Clients.FindByID(ctx, tenancydomain.DefaultTenantID, agentClientID(vaultAgent))
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := f.s.Signer.SignAccessToken(ctx, oauthports.AccessTokenInput{
		Client: client, Sub: stack.AdminUserID, Scopes: []string{vaultScope}, AgentChain: []string{vaultAgent},
		AuthorizationDetails: details, Audiences: []string{vaultResource},
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// delegatedToken は本人の付与を経て、Agent が実際の Token Exchange で得る代行トークンを返す。
func (f *vaultFixture) delegatedToken(t *testing.T, details []spec.AuthorizationDetail) string {
	t.Helper()
	return f.exchangeFor(t, details, vaultResource)
}

func (f *vaultFixture) exchangeFor(t *testing.T, details []spec.AuthorizationDetail, resource string) string {
	t.Helper()
	granted := f.session(t, f.cookie, http.MethodPost, "/api/account/v1/agent-delegations", map[string]any{"agent_id": vaultAgent, "scopes": []string{vaultScope}})
	if granted.Code != http.StatusCreated {
		t.Fatalf("grant: %d %s", granted.Code, granted.Body.String())
	}
	// 付与より後の発行であることは秒の単位で比べる。
	time.Sleep(1100 * time.Millisecond)
	return f.exchangeAgain(t, details, resource)
}

// delegatedTokenAgain は、既にある付与のまま別のトークンを交換する。
func (f *vaultFixture) delegatedTokenAgain(t *testing.T, details []spec.AuthorizationDetail) string {
	t.Helper()
	return f.exchangeAgain(t, details, vaultResource)
}

func (f *vaultFixture) exchangeAgain(t *testing.T, details []spec.AuthorizationDetail, resource string) string {
	t.Helper()
	subject := f.subjectToken(t, details)
	status, body := f.post(t, "token", agentClientID(vaultAgent), url.Values{
		"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token":      {subject},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		"resource":           {resource},
		"scope":              {vaultScope},
	})
	token, _ := body["access_token"].(string)
	if status != http.StatusOK || token == "" {
		t.Fatalf("exchange: %d %v", status, body)
	}
	return token
}

func (f *vaultFixture) detail(actions ...string) []spec.AuthorizationDetail {
	return []spec.AuthorizationDetail{{
		Type: "token_vault_connection", Identifier: f.connectorID, Actions: actions, Locations: []string{vaultAPIOrigin},
	}}
}

func (f *vaultFixture) execute(t *testing.T, token, method, target string, extra map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]any{"method": method, "url": target}
	maps.Copy(body, extra)
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, vaultRealm+"/api/token-vault/v1/connectors/"+f.connectorID+"/execute", strings.NewReader(string(raw)))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	return recorder
}

//spec:covers REQ-TOKENVAULT-005: PKCE と state を持つ連携の開始、REQ-TOKENVAULT-004: state の本人への束縛と一回性、REQ-TOKENVAULT-006: 一覧にトークンが現れないこと
func TestCallbackBindsStateToUserAndConsumesItOnce(t *testing.T) {
	f := newVaultFixture(t)
	query := f.authorize(t)
	if query.Get("code_challenge_method") != "S256" || query.Get("response_type") != "code" || query.Get("client_id") != "vault-client" {
		t.Fatalf("authorization request: %v", query)
	}
	callback := url.Values{"state": {query.Get("state")}, "code": {"code-1"}}

	if result := f.callback(t, nil, callback); result != "invalid_state" {
		t.Fatalf("without a session: %q", result)
	}
	if result := f.callback(t, f.userCookie(t), callback); result != "invalid_state" {
		t.Fatalf("another user: %q", result)
	}
	if result := f.callback(t, f.cookie, url.Values{"state": {"forged"}, "code": {"code-1"}}); result != "invalid_state" {
		t.Fatalf("forged state: %q", result)
	}
	if len(f.upstream.verifiers) != 0 {
		t.Fatal("a refused callback exchanged the code")
	}
	f.s.Events.AssertNotEmitted(t, "TokenVaultConnectionLinked")

	if result := f.callback(t, f.cookie, callback); result != "linked" {
		t.Fatalf("owner: %q", result)
	}
	sum := sha256.Sum256([]byte(f.upstream.verifiers[0]))
	if base64.RawURLEncoding.EncodeToString(sum[:]) != query.Get("code_challenge") {
		t.Fatal("the verifier does not match the challenge")
	}
	f.s.Events.AssertEmitted(t, "TokenVaultConnectionLinked")
	if result := f.callback(t, f.cookie, callback); result != "invalid_state" {
		t.Fatalf("replay: %q", result)
	}
	if len(f.upstream.verifiers) != 1 {
		t.Fatal("a replayed state exchanged the code again")
	}

	list := f.session(t, f.cookie, http.MethodGet, "/api/account/v1/token-vault/connections", nil)
	if list.Code != http.StatusOK || strings.Contains(list.Body.String(), "upstream-access") || strings.Contains(list.Body.String(), "upstream-refresh") {
		t.Fatalf("list: %d %s", list.Code, list.Body.String())
	}
	if !strings.Contains(list.Body.String(), `"status":"active"`) {
		t.Fatalf("list: %s", list.Body.String())
	}

	denied := f.authorize(t)
	if result := f.callback(t, f.cookie, url.Values{"state": {denied.Get("state")}, "error": {"access_denied"}}); result != "denied" {
		t.Fatalf("upstream error: %q", result)
	}
}

//spec:covers REQ-TOKENVAULT-005: トークンでは連携を開始、解除できないこと
func TestLinkingRequiresBrowserSession(t *testing.T) {
	f := newVaultFixture(t)
	f.link(t)
	token := f.delegatedToken(t, f.detail("GET"))
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/api/account/v1/token-vault/connections/" + f.connectorID + "/authorize"},
		{http.MethodDelete, "/api/account/v1/token-vault/connections/" + f.connectorID},
	} {
		request := httptest.NewRequest(tc.method, vaultRealm+tc.path, strings.NewReader("{}"))
		request.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()
		f.s.Echo.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden || !strings.Contains(recorder.Body.String(), "access_denied") {
			t.Fatalf("%s %s: %d %s", tc.method, tc.path, recorder.Code, recorder.Body.String())
		}
	}
}

//spec:covers REQ-TOKENVAULT-008: 保管したトークンで送り、応答に現れないこと、REQ-TOKENVAULT-011: 返す項目、REQ-TOKENVAULT-015: イベントに秘密を載せないこと
func TestExecuteForwardsWithStoredTokenAndNeverReturnsIt(t *testing.T) {
	f := newVaultFixture(t)
	f.link(t)
	token := f.delegatedToken(t, f.detail("GET"))

	response := f.execute(t, token, http.MethodGet, vaultAPIOrigin+"/v1/items?secret-query=1", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("execute: %d %s", response.Code, response.Body.String())
	}
	if got := f.upstream.apiAuth; len(got) != 1 || got[0] != "Bearer upstream-access-1" {
		t.Fatalf("upstream saw: %v", got)
	}
	var body map[string]any
	decodeVault(t, response, &body)
	if body["status"] != float64(200) || body["body_base64"] != base64.StdEncoding.EncodeToString([]byte(`{"ok":true}`)) {
		t.Fatalf("body: %v", body)
	}
	if strings.Contains(response.Body.String(), "upstream-access") || response.Header().Get("Set-Cookie") != "" {
		t.Fatalf("leak: %s %v", response.Body.String(), response.Header())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache-control: %q", response.Header().Get("Cache-Control"))
	}
	f.s.Events.AssertEmitted(t, "TokenVaultExecutionPerformed")
	for _, event := range f.s.Events.All() {
		raw, err := spec.MarshalDomainEvent(event)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"upstream-access", "upstream-refresh", "vault-client-secret", "secret-query", "/v1/items"} {
			if strings.Contains(string(raw), secret) {
				t.Fatalf("event %s contains %q: %s", event.EventType(), secret, raw)
			}
		}
	}
}

//spec:covers REQ-TOKENVAULT-008: 論理積の各項の拒否で上流へ送らないこと、REQ-TOKENVAULT-009: 不正な要求の拒否
func TestExecuteRefusalsDoNotReachUpstream(t *testing.T) {
	f := newVaultFixture(t)
	f.link(t)
	allowed := f.delegatedToken(t, f.detail("GET", "POST"))
	noDetail := f.delegatedTokenAgain(t, nil)
	notDelegated := f.subjectToken(t, f.detail("GET"))
	notExchanged := f.agentBoundToken(t, f.detail("GET"))
	otherAudience := f.exchangeAgain(t, f.detail("GET"), agentRevocationResource)
	cases := []struct {
		name, token, method, target string
		extra                       map[string]any
		status                      int
		code                        string
	}{
		{"no token", "", "GET", vaultAPIOrigin + "/v1/items", nil, 401, "invalid_token"},
		{"forged token", "forged", "GET", vaultAPIOrigin + "/v1/items", nil, 401, "invalid_token"},
		{"agent without exchange", notExchanged, "GET", vaultAPIOrigin + "/v1/items", nil, 403, "connector_access_denied"},
		{"not delegated", notDelegated, "GET", vaultAPIOrigin + "/v1/items", nil, 403, "connector_access_denied"},
		{"other audience", otherAudience, "GET", vaultAPIOrigin + "/v1/items", nil, 401, "invalid_token"},
		{"no detail", noDetail, "GET", vaultAPIOrigin + "/v1/items", nil, 403, "connector_access_denied"},
		{"method outside detail", allowed, "DELETE", vaultAPIOrigin + "/v1/items", nil, 403, "connector_access_denied"},
		{"origin outside detail", allowed, "GET", "https://evil.example/v1/items", nil, 403, "connector_access_denied"},
		{"path outside rule", allowed, "GET", vaultAPIOrigin + "/v1/admin", nil, 403, "connector_access_denied"},
		{"method outside rule", allowed, "POST", vaultAPIOrigin + "/v1/items", nil, 403, "connector_access_denied"},
		{"dot segments", allowed, "GET", vaultAPIOrigin + "/v1/items/../admin", nil, 400, "invalid_request"},
		{"encoded slash", allowed, "GET", vaultAPIOrigin + "/v1/items%2F..%2Fadmin", nil, 400, "invalid_request"},
		{"internal literal", allowed, "GET", "https://169.254.169.254/v1/items", nil, 400, "invalid_request"},
		{"authorization header", allowed, "GET", vaultAPIOrigin + "/v1/items", map[string]any{"headers": map[string]string{"Authorization": "x"}}, 400, "invalid_request"},
		{"cookie header", allowed, "GET", vaultAPIOrigin + "/v1/items", map[string]any{"headers": map[string]string{"Cookie": "x"}}, 400, "invalid_request"},
		{"bad base64", allowed, "GET", vaultAPIOrigin + "/v1/items", map[string]any{"body_base64": "***"}, 400, "invalid_request"},
		{"oversized body", allowed, "GET", vaultAPIOrigin + "/v1/items", map[string]any{"body_base64": base64.StdEncoding.EncodeToString(make([]byte, 1<<20+1))}, 413, "request_too_large"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := f.execute(t, tc.token, tc.method, tc.target, tc.extra)
			if response.Code != tc.status || !strings.Contains(response.Body.String(), tc.code) {
				t.Fatalf("got %d %s", response.Code, response.Body.String())
			}
			if len(f.upstream.apiAuth) != 0 {
				t.Fatal("a refused request reached the upstream")
			}
		})
	}
	f.s.Events.AssertEmitted(t, "TokenVaultExecutionDenied")
}

//spec:covers REQ-TOKENVAULT-008: Agent の停止と付与の撤回が次の呼び出しから効くこと
func TestExecuteReevaluatesAgentAndDelegation(t *testing.T) {
	t.Run("killed agent", func(t *testing.T) {
		f := newVaultFixture(t)
		f.link(t)
		token := f.delegatedToken(t, f.detail("GET"))
		if response := f.execute(t, token, "GET", vaultAPIOrigin+"/v1/items", nil); response.Code != http.StatusOK {
			t.Fatalf("before: %d %s", response.Code, response.Body.String())
		}
		f.admin(t, http.MethodPost, "/api/admin/v1/agents/"+vaultAgent+"/kill", map[string]any{})
		if response := f.execute(t, token, "GET", vaultAPIOrigin+"/v1/items", nil); response.Code != http.StatusUnauthorized {
			t.Fatalf("after: %d %s", response.Code, response.Body.String())
		}
		if len(f.upstream.apiAuth) != 1 {
			t.Fatal("a killed agent reached the upstream")
		}
	})
	t.Run("withdrawn delegation", func(t *testing.T) {
		f := newVaultFixture(t)
		f.link(t)
		token := f.delegatedToken(t, f.detail("GET"))
		revoked := f.session(t, f.cookie, http.MethodPost, "/api/account/v1/agent-delegations/"+vaultAgent+"/revoke", nil)
		if revoked.Code != http.StatusNoContent {
			t.Fatalf("revoke: %d %s", revoked.Code, revoked.Body.String())
		}
		if response := f.execute(t, token, "GET", vaultAPIOrigin+"/v1/items", nil); response.Code != http.StatusUnauthorized {
			t.Fatalf("after: %d %s", response.Code, response.Body.String())
		}
	})
}

//spec:covers REQ-TOKENVAULT-007: 解除の後の実行の拒否と上流への失効、REQ-TOKENVAULT-002: 無効化と削除が実行を止めること
func TestDisconnectRefusesLaterExecution(t *testing.T) {
	f := newVaultFixture(t)
	f.link(t)
	token := f.delegatedToken(t, f.detail("GET"))

	disabled := f.session(t, f.cookie, http.MethodPatch, "/api/admin/v1/token-vault/connectors/"+f.connectorID, map[string]any{"status": "disabled"})
	if disabled.Code != http.StatusOK {
		t.Fatalf("disable: %d %s", disabled.Code, disabled.Body.String())
	}
	if response := f.execute(t, token, "GET", vaultAPIOrigin+"/v1/items", nil); response.Code != http.StatusNotFound {
		t.Fatalf("disabled: %d", response.Code)
	}
	enabled := f.session(t, f.cookie, http.MethodPatch, "/api/admin/v1/token-vault/connectors/"+f.connectorID, map[string]any{"status": "active"})
	if enabled.Code != http.StatusOK {
		t.Fatalf("enable: %d %s", enabled.Code, enabled.Body.String())
	}
	if response := f.execute(t, token, "GET", vaultAPIOrigin+"/v1/items", nil); response.Code != http.StatusOK {
		t.Fatalf("enabled: %d %s", response.Code, response.Body.String())
	}

	unlinked := f.session(t, f.cookie, http.MethodDelete, "/api/account/v1/token-vault/connections/"+f.connectorID, nil)
	if unlinked.Code != http.StatusNoContent {
		t.Fatalf("unlink: %d %s", unlinked.Code, unlinked.Body.String())
	}
	if got := f.upstream.revoked; len(got) != 1 || got[0] != "upstream-refresh-1" {
		t.Fatalf("revoked: %v", got)
	}
	sent := len(f.upstream.apiAuth)
	response := f.execute(t, token, "GET", vaultAPIOrigin+"/v1/items", nil)
	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "connection_not_found") {
		t.Fatalf("after unlink: %d %s", response.Code, response.Body.String())
	}
	if len(f.upstream.apiAuth) != sent {
		t.Fatal("an unlinked connection reached the upstream")
	}
	if again := f.session(t, f.cookie, http.MethodDelete, "/api/account/v1/token-vault/connections/"+f.connectorID, nil); again.Code != http.StatusNotFound {
		t.Fatalf("second unlink: %d", again.Code)
	}

	f.link(t)
	if deleted := f.session(t, f.cookie, http.MethodDelete, "/api/admin/v1/token-vault/connectors/"+f.connectorID, nil); deleted.Code != http.StatusNoContent {
		t.Fatalf("delete connector: %d", deleted.Code)
	}
	if response := f.execute(t, token, "GET", vaultAPIOrigin+"/v1/items", nil); response.Code != http.StatusNotFound {
		t.Fatalf("after delete: %d", response.Code)
	}
}

//spec:covers REQ-TOKENVAULT-012: 同時の期限切れで上流の更新が一度だけで、回転した更新トークンを使うこと
func TestConcurrentExecutionsRefreshOnceAndKeepRotatedToken(t *testing.T) {
	f := newVaultFixture(t)
	f.upstream.expiresIn = 1
	f.link(t)
	f.upstream.expiresIn = 3600
	f.upstream.refreshDelay = 300 * time.Millisecond
	token := f.delegatedToken(t, f.detail("GET"))

	const callers = 8
	codes := make([]int, callers)
	var group sync.WaitGroup
	for i := range callers {
		group.Go(func() { codes[i] = f.execute(t, token, "GET", vaultAPIOrigin+"/v1/items", nil).Code })
	}
	group.Wait()
	for i, code := range codes {
		if code != http.StatusOK {
			t.Fatalf("caller %d: %d", i, code)
		}
	}
	if f.upstream.refreshes != 1 {
		t.Fatalf("refreshes = %d", f.upstream.refreshes)
	}
	for _, authorization := range f.upstream.apiAuth {
		if authorization != "Bearer upstream-access-refreshed" {
			t.Fatalf("upstream saw %q", authorization)
		}
	}
}

//spec:covers REQ-TOKENVAULT-001: 管理 API の検証、シークレットの維持、スコープとテナントの境界
func TestConnectorManagementValidatesAndIsolates(t *testing.T) {
	f := newVaultFixture(t)
	bad := f.session(t, f.cookie, http.MethodPost, "/api/admin/v1/token-vault/connectors", map[string]any{
		"name": "bad", "authorization_endpoint": "http://idp.example/a", "token_endpoint": vaultTokenURL, "client_id": "c", "client_secret": "s",
	})
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("invalid: %d", bad.Code)
	}
	duplicate := f.session(t, f.cookie, http.MethodPost, "/api/admin/v1/token-vault/connectors", map[string]any{
		"name": "upstream", "authorization_endpoint": vaultAuthorizeURL, "token_endpoint": vaultTokenURL, "client_id": "c", "client_secret": "s",
	})
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate: %d", duplicate.Code)
	}
	if missing := f.session(t, f.cookie, http.MethodGet, "/api/admin/v1/token-vault/connectors/00000000-0000-4000-8000-000000000000", nil); missing.Code != http.StatusNotFound {
		t.Fatalf("missing: %d", missing.Code)
	}
	if forbidden := f.session(t, f.userCookie(t), http.MethodGet, "/api/admin/v1/token-vault/connectors", nil); forbidden.Code != http.StatusForbidden {
		t.Fatalf("non-admin: %d", forbidden.Code)
	}
	renamed := f.session(t, f.cookie, http.MethodPatch, "/api/admin/v1/token-vault/connectors/"+f.connectorID, map[string]any{"name": "renamed"})
	if renamed.Code != http.StatusOK || !strings.Contains(renamed.Body.String(), `"client_secret_configured":true`) {
		t.Fatalf("rename: %d %s", renamed.Code, renamed.Body.String())
	}
}

func vaultProblem(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if recorder.Code != status || recorder.Header().Get("Content-Type") != support.ProblemContentType {
		t.Fatalf("got %d %q: %s", recorder.Code, recorder.Header().Get("Content-Type"), recorder.Body.String())
	}
	var problem struct {
		Type   string `json:"type"`
		Status int    `json:"status"`
	}
	decodeVault(t, recorder, &problem)
	if problem.Type != "urn:idmagic:error:"+code || problem.Status != status {
		t.Fatalf("problem: %+v", problem)
	}
}

func vaultObject(t *testing.T, recorder *httptest.ResponseRecorder, status int) map[string]any {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("got %d: %s", recorder.Code, recorder.Body.String())
	}
	body := map[string]any{}
	decodeVault(t, recorder, &body)
	return body
}

//spec:covers REQ-TOKENVAULT-001, REQ-TOKENVAULT-006: 管理と本人の操作の応答の形、シークレットとトークンを含めないこと
//verify:operation CreateTokenVaultConnector: 201 JSON は既定の認証方式と client_secret_configured を返しシークレットを含めず、重複は 409 connector_conflict、不正な定義は 400 invalid_request の Problem Details となる。
//verify:operation ListTokenVaultConnectors: 200 JSON の items は作成したコネクタを返す。
//verify:operation GetTokenVaultConnector: 200 JSON は同じコネクタを返し、未知の ID は 404 connector_not_found の Problem Details となる。
//verify:operation UpdateTokenVaultConnector: 200 JSON は更新した status を返し、未知の ID は 404 の Problem Details となる。
//verify:operation DeleteTokenVaultConnector: 空の 204 を返し、再度の削除は 404 の Problem Details となる。
//verify:operation ListMyTokenVaultConnections: 200 JSON の items はコネクタと接続の状態を返し、トークンを含めない。
//verify:operation AuthorizeMyTokenVaultConnection: 200 JSON は authorization_url を返し、許可されないスコープは 400 invalid_scope、未知のコネクタは 404、トークンの提示は 403 access_denied の Problem Details となる。
//verify:operation UnlinkMyTokenVaultConnection: 空の 204 を返し、接続がなければ 404 connection_not_found の Problem Details となる。
//verify:operation CompleteTokenVaultConnection: 302 の Location は結果だけを持ち、上流の値を含めない。
//verify:operation ExecuteTokenVaultRequest: 200 JSON は status、content_type、body_base64 だけを no-store で返し、拒否は 401、403、404 の Problem Details となる。
func TestTokenVaultHTTPContracts(t *testing.T) {
	f := newVaultFixture(t)
	base := "/api/admin/v1/token-vault/connectors"
	definition := map[string]any{
		"name": "contract", "authorization_endpoint": vaultAuthorizeURL, "token_endpoint": vaultTokenURL,
		"client_id": "c", "client_secret": "contract-secret", "allowed_scopes": []string{"read"}, "rules": []map[string]any{},
	}
	createdRecorder := f.session(t, f.cookie, http.MethodPost, base, definition)
	created := vaultObject(t, createdRecorder, http.StatusCreated)
	if created["token_endpoint_auth_method"] != "client_secret_basic" || created["client_secret_configured"] != true || created["status"] != "active" {
		t.Fatalf("created: %v", created)
	}
	if strings.Contains(createdRecorder.Body.String(), "contract-secret") {
		t.Fatal("the secret appeared in the response")
	}
	id, _ := created["id"].(string)
	vaultProblem(t, f.session(t, f.cookie, http.MethodPost, base, definition), http.StatusConflict, "connector_conflict")
	vaultProblem(t, f.session(t, f.cookie, http.MethodPost, base, map[string]any{"name": "x"}), http.StatusBadRequest, "invalid_request")

	listed := vaultObject(t, f.session(t, f.cookie, http.MethodGet, base, nil), http.StatusOK)
	if items, _ := listed["items"].([]any); len(items) != 2 {
		t.Fatalf("listed: %v", listed)
	}
	if got := vaultObject(t, f.session(t, f.cookie, http.MethodGet, base+"/"+id, nil), http.StatusOK); got["id"] != id {
		t.Fatalf("got: %v", got)
	}
	unknown := base + "/00000000-0000-4000-8000-000000000000"
	vaultProblem(t, f.session(t, f.cookie, http.MethodGet, unknown, nil), http.StatusNotFound, "connector_not_found")
	vaultProblem(t, f.session(t, f.cookie, http.MethodPatch, unknown, map[string]any{"name": "y"}), http.StatusNotFound, "connector_not_found")
	if got := vaultObject(t, f.session(t, f.cookie, http.MethodPatch, base+"/"+id, map[string]any{"status": "disabled"}), http.StatusOK); got["status"] != "disabled" {
		t.Fatalf("updated: %v", got)
	}
	deleted := f.session(t, f.cookie, http.MethodDelete, base+"/"+id, nil)
	if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 {
		t.Fatalf("deleted: %d %s", deleted.Code, deleted.Body.String())
	}
	vaultProblem(t, f.session(t, f.cookie, http.MethodDelete, base+"/"+id, nil), http.StatusNotFound, "connector_not_found")

	account := "/api/account/v1/token-vault/connections/"
	vaultProblem(t, f.session(t, f.cookie, http.MethodPost, account+f.connectorID+"/authorize", map[string]any{"scopes": []string{"admin"}}), http.StatusBadRequest, "invalid_scope")
	vaultProblem(t, f.session(t, f.cookie, http.MethodPost, account+"00000000-0000-4000-8000-000000000000/authorize", map[string]any{}), http.StatusNotFound, "connector_not_found")
	vaultProblem(t, f.session(t, f.cookie, http.MethodDelete, account+f.connectorID, nil), http.StatusNotFound, "connection_not_found")
	if query := f.authorize(t); query.Get("state") == "" {
		t.Fatalf("authorize: %v", query)
	}
	f.link(t)
	listing := f.session(t, f.cookie, http.MethodGet, "/api/account/v1/token-vault/connections", nil)
	items, _ := vaultObject(t, listing, http.StatusOK)["items"].([]any)
	if len(items) != 1 || strings.Contains(listing.Body.String(), "upstream-") {
		t.Fatalf("listing: %s", listing.Body.String())
	}
	entry, _ := items[0].(map[string]any)
	if entry["connector_id"] != f.connectorID || entry["connector_name"] != "upstream" || entry["status"] != "active" {
		t.Fatalf("entry: %v", entry)
	}

	token := f.delegatedToken(t, f.detail("GET"))
	executed := f.execute(t, token, http.MethodGet, vaultAPIOrigin+"/v1/items", nil)
	body := vaultObject(t, executed, http.StatusOK)
	if len(body) != 3 || body["status"] != float64(200) || body["content_type"] != "application/json" || body["body_base64"] == "" {
		t.Fatalf("executed: %v", body)
	}
	vaultProblem(t, f.execute(t, "", http.MethodGet, vaultAPIOrigin+"/v1/items", nil), http.StatusUnauthorized, "invalid_token")
	vaultProblem(t, f.execute(t, token, http.MethodDelete, vaultAPIOrigin+"/v1/items", nil), http.StatusForbidden, "connector_access_denied")

	unlinked := f.session(t, f.cookie, http.MethodDelete, account+f.connectorID, nil)
	if unlinked.Code != http.StatusNoContent || unlinked.Body.Len() != 0 {
		t.Fatalf("unlinked: %d", unlinked.Code)
	}
	vaultProblem(t, f.execute(t, token, http.MethodGet, vaultAPIOrigin+"/v1/items", nil), http.StatusNotFound, "connection_not_found")
}
