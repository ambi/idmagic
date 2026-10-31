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
	"fmt"
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
	ratememory "github.com/ambi/idmagic/backend/shared/ratelimit/db_memory"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
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
	return newResourceAccessFixtureWithLimiter(t, newResourceAccessLimiter(100_000), agentIDs...)
}

func newResourceAccessFixtureWithLimiter(t *testing.T, limiter rlports.RateLimiter, agentIDs ...string) *resourceAccessFixture {
	t.Helper()
	f := &resourceAccessFixture{agentRevocationFixture: newAgentRevocationFixtureWith(t,
		[]stack.Option{stack.WithAuthorizationCodeFlow(), stack.WithAccessControl(), stack.WithRateLimiter(limiter)}, agentIDs...)}
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

func (f *resourceAccessFixture) evaluate(t *testing.T, clientID, secret string, values url.Values, mutate ...func(*http.Request)) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, stack.Issuer+"/realms/default/resource-access/evaluate", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(clientID, secret)
	for _, change := range mutate {
		change(request)
	}
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	body := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &body)
	return recorder, body
}

// 時刻を制御し、窓の期限に待たずに到達する。カウンター自体は本物を使う。
type resourceAccessClock struct {
	limiter     rlports.RateLimiter
	now         time.Time
	unavailable bool
}

func newResourceAccessLimiter(maxRequests int) *resourceAccessClock {
	return &resourceAccessClock{limiter: ratememory.NewRateLimiter(rlports.RateLimitConfigs{
		"resource_access":            {MaxRequests: maxRequests, WindowSeconds: 60},
		"token":                      {MaxRequests: 100_000, WindowSeconds: 60},
		"backchannel_authentication": {MaxRequests: 100_000, WindowSeconds: 60},
	}), now: time.Now().UTC()}
}

func (l *resourceAccessClock) Allow(ctx context.Context, policy, key string, _ time.Time) (rlports.RateLimitResult, error) {
	if l.unavailable && policy == "resource_access" {
		return rlports.RateLimitResult{}, errors.New("counter unavailable")
	}
	return l.limiter.Allow(ctx, policy, key, l.now)
}

//spec:covers REQ-OAUTH2-062: 上限までの許可、429 の本文と再試行指示、超過後の評価と業務作用の不実行
//verify:operation EvaluateResourceAccess: 429 の本文、no-store と Retry-After、窓の上限と復帰
func TestResourceAccessRateLimitStopsBusinessEffect(t *testing.T) {
	limiter := newResourceAccessLimiter(2)
	f := newResourceAccessFixtureWithLimiter(t, limiter, "A1")
	f.grantD1(t)
	token := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, nil)
	business := f.businessServer(roundTripToEcho{f.s.Echo})
	for i := 1; i <= 2; i++ {
		if status := business.call(t, token, "d1"); status != http.StatusOK || business.effects != i {
			t.Fatalf("上限まで: status=%d effects=%d want=%d", status, business.effects, i)
		}
	}
	eventsBefore := len(f.s.Events.All())
	w, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token))
	if w.Code != http.StatusTooManyRequests || body["error"] != "rate_limited" || body["retry_after_seconds"] != float64(60) ||
		w.Header().Get("Retry-After") != "60" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("超過: status=%d headers=%v body=%v", w.Code, w.Header(), body)
	}
	if status := business.call(t, token, "d1"); status != http.StatusServiceUnavailable || business.effects != 2 {
		t.Fatalf("超過後の PEP: status=%d effects=%d", status, business.effects)
	}
	if got := len(f.s.Events.All()); got != eventsBefore {
		t.Fatalf("超過後のイベント: %d -> %d", eventsBefore, got)
	}
	// 窓が切れる直前は拒否し、ちょうど期限に達すると同じクライアントで再開できる。
	limiter.now = limiter.now.Add(60*time.Second - time.Nanosecond)
	w, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token))
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") != "1" || body["retry_after_seconds"] != float64(1) {
		t.Fatalf("窓の直前: status=%d headers=%v body=%v", w.Code, w.Header(), body)
	}
	limiter.now = limiter.now.Add(time.Nanosecond)
	if status := business.call(t, token, "d1"); status != http.StatusOK || business.effects != 3 {
		t.Fatalf("窓の復帰: status=%d effects=%d", status, business.effects)
	}
}

//spec:covers REQ-OAUTH2-062: 入力不正と認証失敗は予算を消費せず、不許可の評価は消費し、提示トークンや資源では予算を分けない
func TestResourceAccessRateLimitCountsOnlyAuthenticatedValidRequests(t *testing.T) {
	f := newResourceAccessFixtureWithLimiter(t, newResourceAccessLimiter(2), "A1")
	f.grantD1(t)
	token := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, nil)
	if w, _ := f.evaluate(t, evalResourceServerClient, "wrong", evalForm(token)); w.Code != http.StatusUnauthorized {
		t.Fatalf("認証失敗: %d", w.Code)
	}
	if w, _ := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, url.Values{"token": {token}}); w.Code != http.StatusBadRequest {
		t.Fatalf("入力不正: %d", w.Code)
	}
	if w, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token)); w.Code != http.StatusOK || body["decision"] != true {
		t.Fatalf("許可: %d %v", w.Code, body)
	}
	denied := evalForm(token)
	denied.Set("resource_id", "d2")
	if w, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, denied); w.Code != http.StatusOK || body["decision"] != false {
		t.Fatalf("不許可: %d %v", w.Code, body)
	}
	other := evalForm("another-presented-token")
	other.Set("resource_id", "d3")
	if w, _ := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, other, func(r *http.Request) {
		r.RemoteAddr = "203.0.113.10:1234"
		r.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.1")
	}); w.Code != http.StatusTooManyRequests {
		t.Fatalf("別トークンと資源: %d", w.Code)
	}
	// 同じ認証情報でも、別テナントの予算は独立する。登録がなくても超過の 429 にはならない。
	now := time.Now().UTC()
	hash := oauthdomain.HashClientSecret(evalResourceServerSecret)
	if err := f.s.Clients.Save(context.Background(), &oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: "another-evaluator", ClientType: spec.ClientConfidential,
		TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic, ClientSecretHash: &hash, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if w, _ := f.evaluate(t, "another-evaluator", evalResourceServerSecret, evalForm(token)); w.Code != http.StatusForbidden {
		t.Fatalf("別クライアント: %d %s", w.Code, w.Body.String())
	}
	if err := f.s.Clients.Save(context.Background(), &oauthdomain.OAuth2Client{
		TenantID: stack.OtherRealm, ClientID: evalResourceServerClient,
		ClientType: spec.ClientConfidential, TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic,
		ClientSecretHash: &hash, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if w, _ := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token), func(r *http.Request) { r.URL.Path = "/realms/acme/resource-access/evaluate" }); w.Code != http.StatusForbidden {
		t.Fatalf("別テナント: %d %s", w.Code, w.Body.String())
	}
}

//spec:covers REQ-OAUTH2-062, REQ-OAUTH2-063: 超過で止めた要求の DPoP 証明は次の窓で一度だけ使える
func TestResourceAccessRateLimitDoesNotConsumeDPoPProof(t *testing.T) {
	limiter := newResourceAccessLimiter(1)
	f := newResourceAccessFixtureWithLimiter(t, limiter, "A1")
	f.grantD1(t)
	key, jwk, _ := tpDPoPKey(t)
	now := time.Now().UTC()
	token := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, func(r *http.Request) {
		r.Header.Set("DPoP", tpDPoPProof(t, key, jwk, tpDPoPProofInput{
			htm: http.MethodPost,
			htu: stack.Issuer + "/realms/default/token", jti: "rate-limit-issue", iat: now,
		}))
	})
	if w, _ := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm("invalid")); w.Code != http.StatusOK {
		t.Fatalf("予算の消費: %d", w.Code)
	}
	form := evalForm(token)
	sum := sha256.Sum256([]byte(token))
	htu := agentRevocationResource + "documents/d1"
	form.Set("dpop_proof", tpDPoPProof(t, key, jwk, tpDPoPProofInput{htm: http.MethodPost, htu: htu, jti: "rate-limited-proof", ath: base64.RawURLEncoding.EncodeToString(sum[:]), iat: now}))
	form.Set("htm", http.MethodPost)
	form.Set("htu", htu)
	before := len(f.s.Events.All())
	if w, _ := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, form); w.Code != http.StatusTooManyRequests {
		t.Fatalf("超過: %d", w.Code)
	}
	if len(f.s.Events.All()) != before {
		t.Fatal("超過した評価がイベントを発行した")
	}
	limiter.now = limiter.now.Add(time.Minute)
	if w, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, form); w.Code != http.StatusOK || body["decision"] != true {
		t.Fatalf("未消費の証明: %d %v", w.Code, body)
	}
	limiter.now = limiter.now.Add(time.Minute)
	_, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, form)
	requireReason(t, body, oauthdomain.ReasonSenderProofInvalid)
}

//spec:covers REQ-OAUTH2-062: カウンターの障害と未配線で業務処理と評価イベントを起こさない
//verify:operation EvaluateResourceAccess: カウンター障害時の 429 と未配線時の 503 のエラー本文
func TestResourceAccessRateLimitUnavailableStopsEvaluation(t *testing.T) {
	for _, unavailable := range []bool{false, true} {
		t.Run(fmt.Sprint(unavailable), func(t *testing.T) {
			var limiter rlports.RateLimiter
			if unavailable {
				clock := newResourceAccessLimiter(2)
				clock.unavailable = true
				limiter = clock
			}
			f := newResourceAccessFixtureWithLimiter(t, limiter, "A1")
			f.grantD1(t)
			token := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, nil)
			before := len(f.s.Events.All())
			w, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, evalForm(token))
			want, code := http.StatusServiceUnavailable, "temporarily_unavailable"
			if unavailable {
				want, code = http.StatusTooManyRequests, "rate_limited"
			}
			if w.Code != want || body["error"] != code {
				t.Fatalf("評価不能: status=%d body=%v", w.Code, body)
			}
			if unavailable && (w.Header().Get("Retry-After") != "30" || body["retry_after_seconds"] != float64(30)) {
				t.Fatalf("障害時の再試行指示: headers=%v body=%v", w.Header(), body)
			}
			business := f.businessServer(roundTripToEcho{f.s.Echo})
			if status := business.call(t, token, "d1"); status != http.StatusServiceUnavailable || business.effects != 0 {
				t.Fatalf("PEP: status=%d effects=%d", status, business.effects)
			}
			if got := len(f.s.Events.All()); got != before {
				t.Fatalf("イベント: %d -> %d", before, got)
			}
		})
	}
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

func (l *resourceAccessClock) AllowWithin(ctx context.Context, policy, key string, _ rlports.RateLimitPolicyConfig, now time.Time) (rlports.RateLimitResult, error) {
	return l.Allow(ctx, policy, key, now)
}
