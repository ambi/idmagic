package server_http_test

// 外部のリソースサーバーの評価を、製品と同じ `Register` の組み立てと、別配布の参照 PEP を
// 通して確かめる。業務作用は PEP が包むハンドラーの呼び出し回数で読む。PEP の判断の表示では
// なく、作用の有無が保証そのものだからである。

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	pep "github.com/ambi/idmagic/examples/resource-server-pep"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	evalResourceServerClient = "rs-evaluator"
	evalResourceServerSecret = "rs-evaluator-secret"
	evalOtherResource        = "https://mcp.other.example/"
)

// roundTripToEcho は参照 PEP の HTTP 要求を、ソケットを開かずに製品の経路へ渡す。
type roundTripToEcho struct{ handler http.Handler }

func (r roundTripToEcho) RoundTrip(request *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	r.handler.ServeHTTP(recorder, request)
	return recorder.Result(), nil
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network down")
}

type resourceAccessFixture struct {
	*agentRevocationFixture
	cookie *http.Cookie
}

func newResourceAccessFixture(t *testing.T, agentIDs ...string) *resourceAccessFixture {
	t.Helper()
	f := &resourceAccessFixture{agentRevocationFixture: newAgentRevocationFixtureWith(t,
		[]stack.Option{stack.WithAuthorizationCodeFlow(), stack.WithAccessControl()}, agentIDs...)}
	f.cookie = f.s.AdminSessionCookie(t, "default")
	f.adminSession(t, "/api/admin/v1/authorization/model", map[string]any{"resource_types": []map[string]any{
		{"name": "user"},
		{"name": "agent"},
		{"name": "document", "relations": []map[string]any{{
			"name": "viewer", "rewrites": []map[string]any{{"kind": "direct", "direct_subject_types": []string{"user", "agent"}}},
		}}},
	}})
	f.seedEvaluator(t)
	return f
}

func (f *resourceAccessFixture) adminSession(t *testing.T, path string, body any) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, stack.Issuer+"/realms/default"+path, strings.NewReader(string(raw)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", stack.Issuer)
	request.Header.Set(support.CSRFHeader, "resource-access-csrf")
	request.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "resource-access-csrf"})
	request.AddCookie(f.cookie)
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	if recorder.Code/100 != 2 {
		t.Fatalf("POST %s status=%d body=%s", path, recorder.Code, recorder.Body.String())
	}
}

func d1ViewerTuple() map[string]any {
	return map[string]any{
		"resource_type": "document", "resource_id": "d1", "relation": "viewer",
		"subject_type": "agent", "subject_id": "A1",
	}
}

// grantD1 は文書 d1 の viewer の関係を Agent A1 へ与える。
func (f *resourceAccessFixture) grantD1(t *testing.T) {
	t.Helper()
	f.adminSession(t, "/api/admin/v1/authorization/relation-tuples", map[string]any{"writes": []map[string]any{d1ViewerTuple()}})
}

// seedEvaluator は評価用のクライアントを置き、登録の束縛と規則を設定する。
func (f *resourceAccessFixture) seedEvaluator(t *testing.T) {
	t.Helper()
	now := time.Now().UTC()
	secretHash := oauthdomain.HashClientSecret(evalResourceServerSecret)
	if err := f.s.Clients.Save(context.Background(), &oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: evalResourceServerClient,
		ClientType: spec.ClientConfidential, TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic,
		ClientSecretHash: &secretHash, GrantTypes: []spec.GrantType{spec.GrantClientCredentials},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	server, err := f.s.McpResourceServers.FindByResource(context.Background(), tenancydomain.DefaultTenantID, agentRevocationResource)
	if err != nil || server == nil {
		t.Fatalf("resource server: %v %v", server, err)
	}
	server.ClientID = evalResourceServerClient
	server.AccessRules = []oauthdomain.ResourceAccessRule{{ResourceType: "document", Relation: "viewer", Scopes: []string{"tools:read"}}}
	if err := f.s.McpResourceServers.Save(context.Background(), server); err != nil {
		t.Fatal(err)
	}
}

func (f *resourceAccessFixture) resourceToken(t *testing.T, clientID, resource string, mutate func(*http.Request)) string {
	t.Helper()
	values := url.Values{"grant_type": {"client_credentials"}, "scope": {"tools:read"}, "resource": {resource}}
	endpoint := stack.Issuer + "/realms/default/token"
	form := f.credentials[clientID].AuthenticateForm(t, values, endpoint)
	request := httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if mutate != nil {
		mutate(request)
	}
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	var body struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(recorder.Body.Bytes(), &body)
	if recorder.Code != http.StatusOK || body.AccessToken == "" {
		t.Fatalf("token status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	return body.AccessToken
}

func (f *resourceAccessFixture) evaluate(t *testing.T, clientID, secret string, values url.Values) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, stack.Issuer+"/realms/default/resource-access/evaluate", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(clientID, secret)
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	body := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &body)
	return recorder, body
}

func evalForm(token string) url.Values {
	return url.Values{"token": {token}, "resource_type": {"document"}, "resource_id": {"d1"}, "relation": {"viewer"}}
}

func requireReason(t *testing.T, body map[string]any, reason string) {
	t.Helper()
	reasons, _ := body["reasons"].([]any)
	if body["decision"] != false || len(reasons) != 1 || reasons[0] != reason {
		t.Fatalf("body=%v, want decision=false reason=%s", body, reason)
	}
}

// businessServer は参照 PEP で包んだ、独立した MCP サーバーの業務処理である。
type businessServer struct {
	effects int
	handler http.Handler
}

func (f *resourceAccessFixture) businessServer(transport http.RoundTripper) *businessServer {
	server := &businessServer{}
	server.handler = pep.Enforcer{
		EvaluateURL: stack.Issuer + "/realms/default/resource-access/evaluate",
		ClientID:    evalResourceServerClient, ClientSecret: evalResourceServerSecret,
		PublicBaseURL: agentRevocationResource,
		HTTPClient:    &http.Client{Transport: transport},
	}.Protect(func(r *http.Request) pep.Operation {
		return pep.Operation{ResourceType: "document", ResourceID: strings.TrimPrefix(r.URL.Path, "/documents/"), Relation: "viewer"}
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		server.effects++
		w.WriteHeader(http.StatusOK)
	}))
	return server
}

func (s *businessServer) call(t *testing.T, token, documentID string) int {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, agentRevocationResource+"documents/"+documentID, http.NoBody)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	s.handler.ServeHTTP(recorder, request)
	return recorder.Code
}

// 許可された操作は業務作用を一度起こし、停止の後、権限のない資源、評価不能では起こさない。
//
//spec:covers REQ-OAUTH2-062, REQ-AUTHORIZATION-004, REQ-AUTHORIZATION-005: 参照 PEP を通した業務作用の有無
func TestExternalPEPStopsBusinessEffectAfterAgentKill(t *testing.T) {
	f := newResourceAccessFixture(t, "A1")
	f.grantD1(t)
	token := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, nil)
	business := f.businessServer(roundTripToEcho{f.s.Echo})

	if status := business.call(t, token, "d1"); status != http.StatusOK || business.effects != 1 {
		t.Fatalf("許可の対照: status=%d effects=%d", status, business.effects)
	}
	if status := business.call(t, token, "d2"); status != http.StatusForbidden || business.effects != 1 {
		t.Fatalf("権限のない資源: status=%d effects=%d", status, business.effects)
	}
	unavailable := f.businessServer(failingTransport{})
	if status := unavailable.call(t, token, "d1"); status != http.StatusServiceUnavailable || unavailable.effects != 0 {
		t.Fatalf("評価不能: status=%d effects=%d", status, unavailable.effects)
	}

	f.admin(t, http.MethodPost, "/api/admin/v1/agents/A1/kill", nil)

	if status := business.call(t, token, "d1"); status != http.StatusForbidden || business.effects != 1 {
		t.Fatalf("停止の後: status=%d effects=%d", status, business.effects)
	}
}

// 別のテナントで署名したトークンは、audience と Agent が同じでも評価を通らない。
//
//spec:covers REQ-OAUTH2-062: 別のテナントのトークンの拒否
func TestEvaluateRejectsTokenFromAnotherTenant(t *testing.T) {
	f := newResourceAccessFixture(t, "A1")
	f.grantD1(t)
	token, _, err := f.s.Signer.SignAccessToken(f.s.RealmContext(t, stack.OtherRealm), oauthports.AccessTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: agentClientID("A1")}, Sub: agentClientID("A1"),
		Scopes: []string{"tools:read"}, Audiences: []string{agentRevocationResource}, AgentID: "A1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token))
	requireReason(t, body, oauthdomain.ReasonTokenRejected)
}

// 関係を取り消すと、同じトークンの次の評価から業務作用が起きない。
//
//spec:covers REQ-OAUTH2-062: 権限の変更が次の評価へ届くこと
func TestExternalPEPStopsBusinessEffectAfterRelationRemoved(t *testing.T) {
	f := newResourceAccessFixture(t, "A1")
	f.grantD1(t)
	token := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, nil)
	business := f.businessServer(roundTripToEcho{f.s.Echo})
	if status := business.call(t, token, "d1"); status != http.StatusOK || business.effects != 1 {
		t.Fatalf("許可の対照: status=%d effects=%d", status, business.effects)
	}

	f.adminSession(t, "/api/admin/v1/authorization/relation-tuples", map[string]any{"deletes": []map[string]any{d1ViewerTuple()}})

	if status := business.call(t, token, "d1"); status != http.StatusForbidden || business.effects != 1 {
		t.Fatalf("取り消しの後: status=%d effects=%d", status, business.effects)
	}
}

//spec:covers REQ-OAUTH2-062: 応答の再利用の禁止、許可の内容、申告した主体を使わないこと
//verify:operation EvaluateResourceAccess: 許可と不許可の本文、no-store、拒否の状態コード
func TestEvaluateResourceAccessContract(t *testing.T) {
	f := newResourceAccessFixture(t, "A1", "A2")
	f.grantD1(t)
	token := f.resourceToken(t, agentClientID("A2"), agentRevocationResource, nil)

	// A2 は関係を持たない。本文で A1 を名乗っても判定は変わらない。
	claimed := evalForm(token)
	claimed.Set("subject", "agent:A1")
	claimed.Set("actor", "agent:A1")
	recorder, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, claimed)
	if recorder.Code != http.StatusOK || body["decision"] != false {
		t.Fatalf("申告した主体: status=%d body=%v", recorder.Code, body)
	}
	if body["scope"] != nil {
		t.Fatalf("不許可でクレームを返した: %v", body)
	}

	permitted := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, nil)
	recorder, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(permitted))
	if recorder.Code != http.StatusOK || body["decision"] != true || body["scope"] != "tools:read" {
		t.Fatalf("許可: status=%d body=%v", recorder.Code, body)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control=%q", got)
	}

	if recorder, _ := f.evaluate(t, evalResourceServerClient, "wrong", evalForm(permitted)); recorder.Code != http.StatusUnauthorized {
		t.Fatalf("認証の失敗: status=%d", recorder.Code)
	}
	if recorder, _ := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, url.Values{"token": {permitted}}); recorder.Code != http.StatusBadRequest {
		t.Fatalf("欠落: status=%d", recorder.Code)
	}
	unknownRule := evalForm(permitted)
	unknownRule.Set("relation", "owner")
	_, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, unknownRule)
	requireReason(t, body, oauthdomain.ReasonOperationNotRegistered)
}

// 別の資源宛てのトークン、未束縛のクライアント、無効にした登録を拒否する。
//
//spec:covers REQ-OAUTH2-062: audience の不一致、未束縛のクライアント、Disabled の登録
func TestEvaluateRejectsOtherAudienceAndUnboundClient(t *testing.T) {
	f := newResourceAccessFixture(t, "A1")
	f.grantD1(t)
	now := time.Now().UTC()
	if err := f.s.McpResourceServers.Save(context.Background(), &oauthdomain.McpResourceServer{
		TenantID: tenancydomain.DefaultTenantID, ID: "mcp-other", Resource: evalOtherResource,
		Name: "other", Scopes: []string{"tools:read"}, State: oauthdomain.McpResourceServerActive,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	other := f.resourceToken(t, agentClientID("A1"), evalOtherResource, nil)
	_, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(other))
	requireReason(t, body, oauthdomain.ReasonTokenRejected)

	token := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, nil)
	secretHash := oauthdomain.HashClientSecret("unbound-secret")
	if err := f.s.Clients.Save(context.Background(), &oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: "rs-unbound",
		ClientType: spec.ClientConfidential, TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic,
		ClientSecretHash: &secretHash, GrantTypes: []spec.GrantType{spec.GrantClientCredentials},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if recorder, _ := f.evaluate(t, "rs-unbound", "unbound-secret", evalForm(token)); recorder.Code != http.StatusForbidden {
		t.Fatalf("未束縛: status=%d", recorder.Code)
	}

	server, _ := f.s.McpResourceServers.FindByResource(context.Background(), tenancydomain.DefaultTenantID, agentRevocationResource)
	server.State = oauthdomain.McpResourceServerDisabled
	if err := f.s.McpResourceServers.Save(context.Background(), server); err != nil {
		t.Fatal(err)
	}
	if recorder, _ := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token)); recorder.Code != http.StatusForbidden {
		t.Fatalf("Disabled: status=%d", recorder.Code)
	}
}

// DPoP に束縛したトークンは、証明なし、別の鍵、再利用、resource の外の htu で不許可になる。
//
//spec:covers REQ-OAUTH2-063: 証明の欠落、鍵の不一致、再利用、htu の範囲、正しい証明の許可
func TestEvaluateRequiresSenderProof(t *testing.T) {
	f := newResourceAccessFixture(t, "A1")
	f.grantD1(t)
	key, jwk, _ := tpDPoPKey(t)
	otherKey, otherJWK, _ := tpDPoPKey(t)
	now := time.Now().UTC()
	endpoint := stack.Issuer + "/realms/default/token"
	token := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, func(request *http.Request) {
		request.Header.Set("DPoP", tpDPoPProof(t, key, jwk, tpDPoPProofInput{htm: http.MethodPost, htu: endpoint, jti: "issue", iat: now}))
	})
	sum := sha256.Sum256([]byte(token))
	ath := base64.RawURLEncoding.EncodeToString(sum[:])
	htu := agentRevocationResource + "documents/d1"
	withProof := func(proof, target string) url.Values {
		form := evalForm(token)
		form.Set("dpop_proof", proof)
		form.Set("htm", http.MethodPost)
		form.Set("htu", target)
		return form
	}

	_, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token))
	requireReason(t, body, oauthdomain.ReasonSenderProofInvalid)

	wrongKey := tpDPoPProof(t, otherKey, otherJWK, tpDPoPProofInput{htm: http.MethodPost, htu: htu, jti: "wrong-key", ath: ath, iat: now})
	_, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, withProof(wrongKey, htu))
	requireReason(t, body, oauthdomain.ReasonSenderProofInvalid)

	outside := "https://evil.example/documents/d1"
	outsideProof := tpDPoPProof(t, key, jwk, tpDPoPProofInput{htm: http.MethodPost, htu: outside, jti: "outside", ath: ath, iat: now})
	_, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, withProof(outsideProof, outside))
	requireReason(t, body, oauthdomain.ReasonSenderProofInvalid)

	proof := tpDPoPProof(t, key, jwk, tpDPoPProofInput{htm: http.MethodPost, htu: htu, jti: "good", ath: ath, iat: now})
	_, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, withProof(proof, htu))
	if body["decision"] != true {
		t.Fatalf("正しい証明: %v", body)
	}
	_, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, withProof(proof, htu))
	requireReason(t, body, oauthdomain.ReasonSenderProofInvalid)
}
