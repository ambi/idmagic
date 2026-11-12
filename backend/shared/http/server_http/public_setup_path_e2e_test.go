package server_http_test

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	ratememory "github.com/ambi/idmagic/backend/shared/ratelimit/db_memory"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
)

// テナント、署名鍵、初期管理者とそのセッションだけを基盤として置く。
// 検証対象の利用者、クライアント、Agent と業務設定は公開 API の応答を引き継ぐ。
type publicSetupPath struct {
	s     *stack.Stack
	admin *http.Cookie
}

func newPublicSetupPath(t *testing.T, extra ...stack.Option) *publicSetupPath {
	t.Helper()
	options := append([]stack.Option{
		stack.WithAuthorizationCodeFlow(), stack.WithTokenIssuance(),
		stack.WithAgentRevocation(), stack.WithAccountApi(), stack.WithRateLimiter(ratememory.NewRateLimiter(rlports.RateLimitConfigs{
			"resource_access":            {MaxRequests: 100_000, WindowSeconds: 60},
			"authorize":                  {MaxRequests: 100_000, WindowSeconds: 60},
			"login":                      {MaxRequests: 100_000, WindowSeconds: 60},
			"token":                      {MaxRequests: 100_000, WindowSeconds: 60},
			"backchannel_authentication": {MaxRequests: 100_000, WindowSeconds: 60},
			"device_authorization":       {MaxRequests: 100_000, WindowSeconds: 60},
		})),
	}, extra...)
	s := stack.New(t, options...)
	return &publicSetupPath{s: s, admin: s.AdminSessionCookie(t, "default")}
}

func (f *publicSetupPath) request(t *testing.T, method, path string, body any, cookie *http.Cookie, want int, bearer ...string) map[string]any {
	t.Helper()
	var raw bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&raw).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, stack.Issuer+"/realms/default"+path, &raw)
	req.Header.Set("Content-Type", "application/json")
	if strings.Contains(path, "/scim/v2/") {
		req.Header.Set("Content-Type", "application/scim+json")
	}
	req.Header.Set("Origin", stack.Issuer)
	req.Header.Set("X-Csrf-Token", "public-setup-csrf")
	req.AddCookie(&http.Cookie{Name: "idmagic_csrf", Value: "public-setup-csrf"})
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if len(bearer) > 0 {
		req.Header.Set("Authorization", "Bearer "+bearer[0])
	}
	rec := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(rec, req)
	if rec.Code != want {
		t.Fatalf("%s %s: status=%d body=%s, want %d", method, path, rec.Code, rec.Body.String(), want)
	}
	result := map[string]any{}
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatalf("%s %s: decode response: %v", method, path, err)
		}
	}
	return result
}

func setupString(t *testing.T, body map[string]any, field string) string {
	t.Helper()
	value, _ := body[field].(string)
	if value == "" {
		t.Fatalf("response has no %s: %v", field, body)
	}
	return value
}

func (f *publicSetupPath) signedForm(t *testing.T, key *rsa.PrivateKey, clientID, path string, form url.Values, want int) map[string]any {
	t.Helper()
	now := time.Now().UTC()
	assertion := setupJWT(t, key, map[string]any{
		"iss": clientID, "sub": clientID, "aud": stack.Issuer + "/realms/default/token",
		"jti": rand.Text(), "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
	})
	form.Set("client_id", clientID)
	form.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	form.Set("client_assertion", assertion)
	req := httptest.NewRequest(http.MethodPost, stack.Issuer+"/realms/default"+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(rec, req)
	result := map[string]any{}
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
	}
	if rec.Code != want {
		t.Fatalf("POST %s: status=%d body=%v, want %d", path, rec.Code, result, want)
	}
	return result
}

func setupJWT(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "PS256", "kid": "setup-key"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

const (
	setupPassword = "public-setup-password-1234"
	setupRedirect = "https://console.example/callback"
	setupVerifier = "public-setup-pkce-verifier-01234567890123456789"
	setupResource = "https://documents.example/api"
)

func (f *publicSetupPath) user(t *testing.T, username string) string {
	t.Helper()
	user := f.request(t, http.MethodPost, "/api/admin/v1/users", map[string]any{
		"preferred_username": username, "password": setupPassword, "email": username + "@example.com",
	}, f.admin, http.StatusCreated)
	return setupString(t, user, "id")
}

func (f *publicSetupPath) keyClient(t *testing.T, scope string, grants ...string) (string, *rsa.PrivateKey) {
	t.Helper()
	key, jwk, _ := tpDPoPKey(t)
	jwk["kid"] = "setup-key"
	registered := f.request(t, http.MethodPost, "/api/admin/v1/clients", map[string]any{
		"client_type": "confidential", "redirect_uris": []string{setupRedirect},
		"response_types": []string{"code"}, "grant_types": grants, "scope": scope,
		"token_endpoint_auth_method": "private_key_jwt", "jwks": map[string]any{"keys": []any{jwk}},
	}, f.admin, http.StatusCreated)
	client, ok := registered["client"].(map[string]any)
	if !ok {
		t.Fatalf("registration has no client: %v", registered)
	}
	return setupString(t, client, "client_id"), key
}

func (f *publicSetupPath) agent(t *testing.T, owner, clientID, kind string) string {
	t.Helper()
	agent := f.request(t, http.MethodPost, "/api/admin/v1/agents", map[string]any{
		"name": "Public agent " + clientID, "owner_user_id": owner, "kind": kind,
	}, f.admin, http.StatusCreated)
	id := setupString(t, agent, "id")
	f.request(t, http.MethodPost, "/api/admin/v1/agents/"+id+"/credentials",
		map[string]any{"client_id": clientID}, f.admin, http.StatusNoContent)
	return id
}

func (f *publicSetupPath) login(t *testing.T, username, clientID, scope string) (string, *http.Cookie) {
	t.Helper()
	browser := f.s.Browser(t, "default")
	overrides := map[string]string{
		"client_id": clientID, "redirect_uri": setupRedirect, "scope": scope,
	}
	response := browser.Authorize(t, stack.AuthorizationQuery(setupVerifier, overrides))
	raw, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil || (response.StatusCode != http.StatusFound && response.StatusCode != http.StatusSeeOther) {
		t.Fatalf("authorize status=%d body=%s error=%v", response.StatusCode, raw, err)
	}
	browser.SignIn(t, username, setupPassword)
	redirect, err := url.Parse(browser.Consent(t, "allow"))
	if err != nil {
		t.Fatal(err)
	}
	code := redirect.Query().Get("code")
	if code == "" || redirect.Query().Get("state") != "opaque-state" {
		t.Fatalf("authorization did not return a code and state: %s", redirect)
	}
	return code, &http.Cookie{Name: authentication.SessionCookie, Value: browser.SessionCookie(t)}
}

func (f *publicSetupPath) resource(t *testing.T, evaluator string) {
	t.Helper()
	f.request(t, http.MethodPost, "/api/admin/v1/mcp-resource-servers", map[string]any{
		"name": "Public documents", "resource": setupResource, "scopes": []string{"read"}, "client_id": evaluator,
		"access_rules": []map[string]any{{"resource_type": "document", "relation": "viewer", "scopes": []string{"read"}}},
	}, f.admin, http.StatusCreated)
	f.request(t, http.MethodPost, "/api/admin/v1/authorization/model", map[string]any{"expected_version": 0, "resource_types": []map[string]any{
		{"name": "user"},
		{"name": "agent"},
		{"name": "document", "relations": []map[string]any{{"name": "viewer", "rewrites": []map[string]any{
			{"kind": "direct", "direct_subject_types": []string{"user", "agent"}},
		}}}},
	}}, f.admin, http.StatusCreated)
}

func setupTuple(subjectType, subjectID string) map[string]any {
	return map[string]any{"resource_type": "document", "resource_id": "d1", "relation": "viewer", "subject_type": subjectType, "subject_id": subjectID}
}

func (f *publicSetupPath) relation(t *testing.T, operation, subjectType, subjectID string) {
	t.Helper()
	f.request(t, http.MethodPost, "/api/admin/v1/authorization/relation-tuples",
		map[string]any{operation: []map[string]any{setupTuple(subjectType, subjectID)}}, f.admin, http.StatusOK)
}

func (f *publicSetupPath) evaluate(t *testing.T, key *rsa.PrivateKey, evaluator, token string, want bool) {
	t.Helper()
	result := f.signedForm(t, key, evaluator, "/resource-access/evaluate", url.Values{
		"token": {token}, "resource_type": {"document"}, "resource_id": {"d1"}, "relation": {"viewer"},
	}, http.StatusOK)
	if result["decision"] != want {
		t.Fatalf("resource access=%v, want decision=%v", result, want)
	}
}

//spec:covers REQ-OAUTH2-005, REQ-OAUTH2-001, REQ-OAUTH2-006: 登録した User とクライアントでログインと PKCE を通し、本人 API、リフレッシュのローテーションと失効を観測する。
func TestRegisteredUserUsesAndRevokesOAuthTokens(t *testing.T) {
	f := newPublicSetupPath(t)
	userID := f.user(t, "public-human")
	const scope = "openid profile account:read offline_access"
	clientID, key := f.keyClient(t, scope, "authorization_code", "refresh_token")
	code, _ := f.login(t, "public-human", clientID, scope)
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {setupRedirect}, "code_verifier": {setupVerifier}}
	wrong := maps.Clone(form)
	wrong.Set("code_verifier", strings.Repeat("x", 43))
	refused := f.signedForm(t, key, clientID, "/token", wrong, http.StatusBadRequest)
	if refused["error"] != "invalid_grant" || refused["access_token"] != nil {
		t.Fatalf("wrong PKCE verifier=%v", refused)
	}
	issued := f.signedForm(t, key, clientID, "/token", form, http.StatusOK)
	access := setupString(t, issued, "access_token")
	profile := f.request(t, http.MethodGet, "/api/account/v1/profile", nil, nil, http.StatusOK, access)
	if profile["id"] != userID {
		t.Fatalf("profile id=%v, want %s", profile["id"], userID)
	}
	refresh := setupString(t, issued, "refresh_token")
	rotated := f.signedForm(t, key, clientID, "/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresh}}, http.StatusOK)
	current := setupString(t, rotated, "refresh_token")
	if current == refresh {
		t.Fatalf("refresh token did not rotate")
	}
	f.signedForm(t, key, clientID, "/revoke", url.Values{"token": {current}, "token_type_hint": {"refresh_token"}}, http.StatusOK)
	denied := f.signedForm(t, key, clientID, "/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {current}}, http.StatusBadRequest)
	if denied["error"] != "invalid_grant" || denied["access_token"] != nil {
		t.Fatalf("revoked refresh token was accepted: %v", denied)
	}
}

//spec:covers REQ-IDMANAGEMENT-074, REQ-OAUTH2-055, REQ-OAUTH2-062, REQ-PLATFORM-001: 公開登録と束縛から鍵でトークンを取得し、同じトークンの認可が関係削除と Agent 停止で拒否に変わる。
func TestRegisteredAutonomousAgentUsesResourceAuthorization(t *testing.T) {
	f := newPublicSetupPath(t, stack.WithAccessControl())
	owner := f.user(t, "public-owner")
	clientID, key := f.keyClient(t, "read", "client_credentials")
	agentID := f.agent(t, owner, clientID, "autonomous")
	evaluator, evaluatorKey := f.keyClient(t, "read", "client_credentials")
	f.resource(t, evaluator)
	issued := f.signedForm(t, key, clientID, "/token", url.Values{
		"grant_type": {"client_credentials"}, "scope": {"read"}, "resource": {setupResource},
	}, http.StatusOK)
	token := setupString(t, issued, "access_token")
	f.evaluate(t, evaluatorKey, evaluator, token, false)
	f.relation(t, "writes", "agent", agentID)
	f.evaluate(t, evaluatorKey, evaluator, token, true)
	f.relation(t, "deletes", "agent", agentID)
	f.evaluate(t, evaluatorKey, evaluator, token, false)
	f.relation(t, "writes", "agent", agentID)
	f.evaluate(t, evaluatorKey, evaluator, token, true)
	f.request(t, http.MethodPost, "/api/admin/v1/agents/"+agentID+"/kill", nil, f.admin, http.StatusNoContent)
	f.evaluate(t, evaluatorKey, evaluator, token, false)
}

//spec:covers REQ-OAUTH2-060, REQ-OAUTH2-062: 管理 API で作成した利用者のトークンは、本人の付与後だけ Agent に委譲でき、主体と行為者を保持し、両者の関係を評価する。
func TestRegisteredAgentRequiresUserDelegation(t *testing.T) {
	f := newPublicSetupPath(t, stack.WithAccessControl())
	userID := f.user(t, "public-delegator")
	clientID, key := f.keyClient(t, "read", "urn:ietf:params:oauth:grant-type:token-exchange")
	agentID := f.agent(t, userID, clientID, "autonomous")
	evaluator, evaluatorKey := f.keyClient(t, "read", "client_credentials")
	f.resource(t, evaluator)
	f.relation(t, "writes", "user", userID)
	f.relation(t, "writes", "agent", agentID)
	browserClient, browserKey := f.keyClient(t, "openid read", "authorization_code")
	code, cookie := f.login(t, "public-delegator", browserClient, "openid read")
	subject := f.signedForm(t, browserKey, browserClient, "/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {setupRedirect},
		"code_verifier": {setupVerifier},
	}, http.StatusOK)
	exchange := func(want int) map[string]any {
		return f.signedForm(t, key, clientID, "/token", url.Values{
			"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
			"subject_token":      {setupString(t, subject, "access_token")}, "scope": {"read"}, "resource": {setupResource},
		}, want)
	}
	denied := exchange(http.StatusBadRequest)
	if denied["error"] != "invalid_grant" || denied["access_token"] != nil {
		t.Fatalf("exchange without a delegation=%v", denied)
	}
	f.request(t, http.MethodPost, "/api/account/v1/agent-delegations",
		map[string]any{"agent_id": agentID, "scopes": []string{"read"}}, cookie, http.StatusCreated)
	issued := exchange(http.StatusOK)
	token := setupString(t, issued, "access_token")
	claims := unverifiedClaims(t, token)
	actor, ok := claims["act"].(map[string]any)
	if !ok || claims["sub"] != userID || actor["sub"] != clientID || claims["scope"] != "read" {
		t.Fatalf("delegated identity=%v, want user=%s actor=%s and read", claims, userID, clientID)
	}
	f.evaluate(t, evaluatorKey, evaluator, token, true)
	f.relation(t, "deletes", "user", userID)
	f.evaluate(t, evaluatorKey, evaluator, token, false)
	f.request(t, http.MethodPost, "/api/account/v1/agent-delegations/"+agentID+"/revoke", nil, cookie, http.StatusNoContent)
	denied = exchange(http.StatusBadRequest)
	if denied["error"] != "invalid_grant" || denied["access_token"] != nil {
		t.Fatalf("exchange after withdrawing the delegation=%v", denied)
	}
}

//spec:covers REQ-WORKLOADIDENTITY-001, REQ-WORKLOADIDENTITY-006, REQ-OAUTH2-046: 信頼設定と関連付けを公開登録して実署名のアテステーションを交換し、未登録の信頼と停止した Agent では発行しない。停止したクライアント自身は共通入口で拒否する。
func TestRegisteredWorkloadTrustControlsTokenExchange(t *testing.T) {
	f := newPublicSetupPath(t, stack.WithAccessControl())
	owner := f.user(t, "public-workload-owner")
	clientID, key := f.keyClient(t, "read", "urn:ietf:params:oauth:grant-type:token-exchange")
	agentID := f.agent(t, owner, clientID, "autonomous")
	evaluator, _ := f.keyClient(t, "read", "client_credentials")
	f.resource(t, evaluator)
	externalKey, externalJWK, _ := tpDPoPKey(t)
	externalJWK["kid"] = "setup-key"
	now := time.Now().UTC()
	attestation := setupJWT(t, externalKey, map[string]any{
		"iss": "https://workloads.example", "sub": "spiffe://cluster/public-bot",
		"aud": "idmagic-workload", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
	})
	exchange := func(want int) map[string]any {
		return f.signedForm(t, key, clientID, "/token", url.Values{
			"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"subject_token_type": {"urn:ietf:params:oauth:token-type:jwt"}, "subject_token": {attestation},
			"scope": {"read"}, "resource": {setupResource},
		}, want)
	}
	denied := exchange(http.StatusBadRequest)
	if denied["error"] != "invalid_grant" || denied["access_token"] != nil {
		t.Fatalf("unregistered workload trust=%v", denied)
	}
	bundle := f.request(t, http.MethodPost, "/api/admin/v1/workload-identity/trust-bundles", map[string]any{
		"name": "Public workload trust", "trust_domain": "cluster", "issuer": "https://workloads.example",
		"accepted_audiences": []string{"idmagic-workload"}, "jwks": map[string]any{"keys": []any{externalJWK}},
	}, f.admin, http.StatusCreated)
	f.request(t, http.MethodPost, "/api/admin/v1/workload-identity/trust-bundles/"+setupString(t, bundle, "id")+"/bindings",
		map[string]any{"subject_pattern": "spiffe://cluster/*", "agent_id": agentID}, f.admin, http.StatusCreated)
	issued := exchange(http.StatusOK)
	claims := unverifiedClaims(t, setupString(t, issued, "access_token"))
	if claims["sub"] != clientID {
		t.Fatalf("workload token subject=%v, want registered client %s", claims["sub"], clientID)
	}
	f.request(t, http.MethodPost, "/api/admin/v1/agents/"+agentID+"/kill", nil, f.admin, http.StatusNoContent)
	denied = exchange(http.StatusUnauthorized)
	if denied["error"] != "invalid_client" || denied["access_token"] != nil {
		t.Fatalf("killed workload agent=%v", denied)
	}
	// 未束縛のクライアントなら共通入口を通り、アテステーションの対応先の停止で拒否する。
	unboundClient, unboundKey := f.keyClient(t, "read", "urn:ietf:params:oauth:grant-type:token-exchange")
	denied = f.signedForm(t, unboundKey, unboundClient, "/token", url.Values{
		"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:jwt"}, "subject_token": {attestation},
		"scope": {"read"}, "resource": {setupResource},
	}, http.StatusBadRequest)
	if denied["error"] != "invalid_grant" || denied["access_token"] != nil {
		t.Fatalf("killed workload subject agent=%v", denied)
	}
}

// 同じアテステーションを、Agent に束縛したクライアントと、Token Exchange を許可されただけの
// 束縛のないクライアントがそれぞれ提示する。JWT-SVID は Bearer なので、要求のクライアントを
// 束縛と照合しなければ、手に入れた JWT-SVID でどのクライアントも Agent の資格情報を得る。
//
//spec:covers REQ-WORKLOADIDENTITY-001, RFC8693-SUBJECT-TOKEN: 束縛先のクライアントの交換は sub と client_id がそのクライアントのトークンを得て、束縛のないクライアントの交換は 400 と invalid_grant でトークンを得ない。
func TestWorkloadTokenExchangeRequiresTheBoundClient(t *testing.T) {
	f := newPublicSetupPath(t, stack.WithAccessControl())
	owner := f.user(t, "bound-workload-owner")
	clientID, key := f.keyClient(t, "read", "urn:ietf:params:oauth:grant-type:token-exchange")
	agentID := f.agent(t, owner, clientID, "autonomous")
	otherClient, otherKey := f.keyClient(t, "read", "urn:ietf:params:oauth:grant-type:token-exchange")
	evaluator, _ := f.keyClient(t, "read", "client_credentials")
	f.resource(t, evaluator)
	externalKey, externalJWK, _ := tpDPoPKey(t)
	externalJWK["kid"] = "setup-key"
	bundle := f.request(t, http.MethodPost, "/api/admin/v1/workload-identity/trust-bundles", map[string]any{
		"name": "Bound workload trust", "trust_domain": "cluster", "issuer": "https://workloads.example",
		"accepted_audiences": []string{"idmagic-workload"}, "jwks": map[string]any{"keys": []any{externalJWK}},
	}, f.admin, http.StatusCreated)
	f.request(t, http.MethodPost, "/api/admin/v1/workload-identity/trust-bundles/"+setupString(t, bundle, "id")+"/bindings",
		map[string]any{"subject_pattern": "spiffe://cluster/*", "agent_id": agentID}, f.admin, http.StatusCreated)
	now := time.Now().UTC()
	attestation := setupJWT(t, externalKey, map[string]any{
		"iss": "https://workloads.example", "sub": "spiffe://cluster/bound-bot",
		"aud": "idmagic-workload", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
	})
	exchange := func(signer *rsa.PrivateKey, requester string, want int) map[string]any {
		return f.signedForm(t, signer, requester, "/token", url.Values{
			"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"subject_token_type": {"urn:ietf:params:oauth:token-type:jwt"}, "subject_token": {attestation},
			"scope": {"read"}, "resource": {setupResource},
		}, want)
	}

	denied := exchange(otherKey, otherClient, http.StatusBadRequest)
	if denied["error"] != "invalid_grant" || denied["access_token"] != nil {
		t.Fatalf("exchange by a client not bound to the agent=%v, want invalid_grant without a token", denied)
	}
	issued := exchange(key, clientID, http.StatusOK)
	claims := unverifiedClaims(t, setupString(t, issued, "access_token"))
	if claims["sub"] != clientID || claims["client_id"] != clientID {
		t.Fatalf("workload token sub=%v client_id=%v, want the bound client %s", claims["sub"], claims["client_id"], clientID)
	}
}

// SPIFFE バンドルの `x509-svid` の鍵と `jwt-svid` の鍵に同じ `kid` を付け、どちらの鍵で
// 署名したかだけが異なる二つの JWT-SVID を交換する。組み立ての地点が信頼設定の種別で
// バンドルの読み方を選ばなければ、先に並ぶ `x509-svid` の鍵で検証してしまう。
//
//spec:covers REQ-WORKLOADIDENTITY-003, EX-WORKLOADIDENTITY-003-02, REQ-WORKLOADIDENTITY-008: 公開登録した spiffe の信頼設定で、x509-svid の鍵の署名を拒否し、jwt-svid の鍵の署名だけを交換し、再取得が jwt-svid の鍵の数を返す。
func TestSPIFFEBundleVerifiesWithJWTSVIDKeysOnly(t *testing.T) {
	f := newPublicSetupPath(t, stack.WithAccessControl())
	owner := f.user(t, "spiffe-bundle-owner")
	clientID, key := f.keyClient(t, "read", "urn:ietf:params:oauth:grant-type:token-exchange")
	agentID := f.agent(t, owner, clientID, "autonomous")
	evaluator, _ := f.keyClient(t, "read", "client_credentials")
	f.resource(t, evaluator)
	x509Key, x509JWK, _ := tpDPoPKey(t)
	x509JWK["kid"], x509JWK["use"] = "setup-key", "x509-svid"
	jwtKey, jwtJWK, _ := tpDPoPKey(t)
	jwtJWK["kid"], jwtJWK["use"] = "setup-key", "jwt-svid"
	bundle := f.request(t, http.MethodPost, "/api/admin/v1/workload-identity/trust-bundles", map[string]any{
		"name": "SPIFFE bundle", "profile": "spiffe", "trust_domain": "example.org",
		"accepted_audiences": []string{"idmagic-workload"},
		"jwks":               map[string]any{"spiffe_sequence": 1, "spiffe_refresh_hint": 300, "keys": []any{x509JWK, jwtJWK}},
	}, f.admin, http.StatusCreated)
	bundleID := setupString(t, bundle, "id")
	f.request(t, http.MethodPost, "/api/admin/v1/workload-identity/trust-bundles/"+bundleID+"/bindings",
		map[string]any{"subject_pattern": "spiffe://example.org/ns/prod/sa/*", "agent_id": agentID}, f.admin, http.StatusCreated)

	refreshed := f.request(t, http.MethodPost, "/api/admin/v1/workload-identity/trust-bundles/"+bundleID+"/refresh", nil, f.admin, http.StatusOK)
	if refreshed["reachable"] != true || refreshed["key_count"] != float64(1) {
		t.Fatalf("refresh=%v, want reachable with the one jwt-svid key", refreshed)
	}

	now := time.Now().UTC()
	exchange := func(signer *rsa.PrivateKey, want int) map[string]any {
		svid := setupJWT(t, signer, map[string]any{
			"sub": "spiffe://example.org/ns/prod/sa/worker", "aud": "idmagic-workload",
			"iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
		})
		return f.signedForm(t, key, clientID, "/token", url.Values{
			"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"subject_token_type": {"urn:ietf:params:oauth:token-type:jwt"}, "subject_token": {svid},
			"scope": {"read"}, "resource": {setupResource},
		}, want)
	}
	if denied := exchange(x509Key, http.StatusBadRequest); denied["error"] != "invalid_grant" || denied["access_token"] != nil {
		t.Fatalf("JWT-SVID signed with the x509-svid key=%v, want invalid_grant", denied)
	}
	issued := exchange(jwtKey, http.StatusOK)
	if claims := unverifiedClaims(t, setupString(t, issued, "access_token")); claims["sub"] != clientID {
		t.Fatalf("workload token subject=%v, want registered client %s", claims["sub"], clientID)
	}
}

//spec:covers REQ-SOURCING-002, REQ-SOURCING-005, REQ-PLATFORM-001: 公開発行した SCIM 資格情報で User と所属グループを取り込み、名前の変更と所有者の無効化を管理 API と Agent のアクセスへ反映する。
func TestSCIMPublicCredentialsAndOwnerLifecycle(t *testing.T) {
	f := newPublicSetupPath(t, stack.WithSCIM())
	credentials := f.request(t, http.MethodPost, "/api/admin/v1/api-tokens", map[string]any{
		"description": "Public SCIM setup", "expiry_days": 1,
		"scopes": []string{"scim:users:read", "scim:users:write", "scim:groups:read", "scim:groups:write"},
	}, f.admin, http.StatusCreated)
	bearer := setupString(t, credentials, "token")
	user := f.request(t, http.MethodPost, "/scim/v2/Users", map[string]any{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "public-scim", "active": true, "displayName": "SCIM owner",
	}, nil, http.StatusCreated, bearer)
	scimUserID := setupString(t, user, "id")
	group := f.request(t, http.MethodPost, "/scim/v2/Groups", map[string]any{
		"schemas": []string{"urn:ietf:params:scim:schemas:core:2.0:Group"}, "displayName": "Public SCIM group",
		"members": []map[string]any{{"value": scimUserID}},
	}, nil, http.StatusCreated, bearer)
	scimGroupID := setupString(t, group, "id")
	users := f.request(t, http.MethodGet, "/api/admin/v1/users?query=public-scim", nil, f.admin, http.StatusOK)
	owner := scimOnlyResult(t, users, "users")
	ownerID := setupString(t, owner, "id")
	if owner["status"] != "active" || owner["name"] != "SCIM owner" {
		t.Fatalf("SCIM user did not reach the administrative view: %v", owner)
	}
	groups := f.request(t, http.MethodGet, "/api/admin/v1/groups?query=Public+SCIM+group", nil, f.admin, http.StatusOK)
	adminGroup := scimOnlyResult(t, groups, "groups")
	groupID := setupString(t, adminGroup, "id")
	detail := f.request(t, http.MethodGet, "/api/admin/v1/groups/"+groupID, nil, f.admin, http.StatusOK)
	member := scimOnlyResult(t, detail, "members")
	if member["user_id"] != ownerID {
		t.Fatalf("SCIM group member=%v, want %s", member, ownerID)
	}
	patchSchema := []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"}
	f.request(t, http.MethodPatch, "/scim/v2/Users/"+scimUserID, map[string]any{
		"schemas": patchSchema, "Operations": []map[string]any{{"op": "replace", "path": "displayName", "value": "Changed SCIM owner"}},
	}, nil, http.StatusOK, bearer)
	changed := f.request(t, http.MethodGet, "/api/admin/v1/users/"+ownerID, nil, f.admin, http.StatusOK)
	if changed["name"] != "Changed SCIM owner" {
		t.Fatalf("SCIM update did not reach the user: %v", changed)
	}
	clientID, key := f.keyClient(t, "read", "client_credentials")
	f.agent(t, ownerID, clientID, "autonomous")
	issued := f.signedForm(t, key, clientID, "/token", url.Values{
		"grant_type": {"client_credentials"}, "scope": {"read"},
	}, http.StatusOK)
	token := setupString(t, issued, "access_token")
	active := f.signedForm(t, key, clientID, "/introspect", url.Values{"token": {token}}, http.StatusOK)
	if active["active"] != true {
		t.Fatalf("active SCIM owner's agent token=%v", active)
	}
	f.request(t, http.MethodPatch, "/scim/v2/Groups/"+scimGroupID, map[string]any{
		"schemas": patchSchema, "Operations": []map[string]any{{"op": "replace", "path": "members", "value": []any{}}},
	}, nil, http.StatusOK, bearer)
	detail = f.request(t, http.MethodGet, "/api/admin/v1/groups/"+groupID, nil, f.admin, http.StatusOK)
	members, ok := detail["members"].([]any)
	if !ok || len(members) != 0 {
		t.Fatalf("SCIM group removal did not reach the administrative view: %v", detail)
	}
	f.request(t, http.MethodPatch, "/scim/v2/Users/"+scimUserID, map[string]any{
		"schemas": patchSchema, "Operations": []map[string]any{{"op": "replace", "path": "active", "value": false}},
	}, nil, http.StatusOK, bearer)
	disabled := f.request(t, http.MethodGet, "/api/admin/v1/users/"+ownerID, nil, f.admin, http.StatusOK)
	if disabled["status"] != "disabled" {
		t.Fatalf("SCIM disabling did not reach the user: %v", disabled)
	}
	inactive := f.signedForm(t, key, clientID, "/introspect", url.Values{"token": {token}}, http.StatusOK)
	if inactive["active"] != false {
		t.Fatalf("disabled SCIM owner's agent token remains active: %v", inactive)
	}
}

//spec:covers REQ-OAUTH2-035, REQ-OAUTH2-041, REQ-OAUTH2-050: 公開 API で User、CIBA クライアント、supervised Agent と束縛を作り、承認前の保留、人の承認後の発行、一回限りの交換を観測する。
func TestRegisteredSupervisedAgentRequiresApproval(t *testing.T) {
	f := newPublicSetupPath(t)
	const username = "setup-approver"
	const password = "public-setup-password-1234"
	const redirectURI = "https://console.example/callback"
	user := f.request(t, http.MethodPost, "/api/admin/v1/users", map[string]any{
		"preferred_username": username, "password": password, "email": "approver@example.com",
	}, f.admin, http.StatusCreated)
	key, jwk, _ := tpDPoPKey(t)
	jwk["kid"] = "setup-key"
	registered := f.request(t, http.MethodPost, "/api/admin/v1/clients", map[string]any{
		"client_name": "Public setup CIBA", "client_type": "confidential",
		"redirect_uris": []string{redirectURI}, "response_types": []string{"code"},
		"grant_types":                []string{"authorization_code", "client_credentials", "urn:openid:params:grant-type:ciba"},
		"token_endpoint_auth_method": "private_key_jwt", "jwks": map[string]any{"keys": []any{jwk}},
		"scope": "openid profile",
	}, f.admin, http.StatusCreated)
	client, ok := registered["client"].(map[string]any)
	if !ok {
		t.Fatalf("registration has no client: %v", registered)
	}
	clientID := setupString(t, client, "client_id")
	f.agent(t, setupString(t, user, "id"), clientID, "supervised")

	refused := f.signedForm(t, key, clientID, "/token", url.Values{
		"grant_type": {"client_credentials"}, "scope": {"profile"},
	}, http.StatusBadRequest)
	if refused["error"] != "unauthorized_client" || refused["access_token"] != nil {
		t.Fatalf("supervised agent received an unapproved token: %v", refused)
	}
	started := f.signedForm(t, key, clientID, "/bc-authorize", url.Values{
		"scope": {"openid profile"}, "login_hint": {username},
	}, http.StatusOK)
	authReqID := setupString(t, started, "auth_req_id")
	pending := f.signedForm(t, key, clientID, "/token", url.Values{
		"grant_type": {"urn:openid:params:grant-type:ciba"}, "auth_req_id": {authReqID},
	}, http.StatusBadRequest)
	if pending["error"] != "authorization_pending" || pending["access_token"] != nil {
		t.Fatalf("unapproved request: %v, want authorization_pending without a token", pending)
	}

	browser := f.s.Browser(t, "default")
	_ = browser.Authorize(t, stack.AuthorizationQuery("public-setup-pkce-verifier-01234567890123456789", map[string]string{
		"client_id": clientID, "redirect_uri": redirectURI,
	})).Body.Close()
	browser.SignIn(t, username, password)
	browser.Consent(t, "allow")
	cookie := &http.Cookie{Name: authentication.SessionCookie, Value: browser.SessionCookie(t)}
	listed := f.request(t, http.MethodGet, "/api/account/v1/approval-requests", nil, cookie, http.StatusOK)
	requests, ok := listed["approval_requests"].([]any)
	if !ok || len(requests) != 1 {
		t.Fatalf("pending approvals=%v, want one", listed)
	}
	approval, ok := requests[0].(map[string]any)
	if !ok || approval["client_id"] != clientID || approval["agent_name"] != "Public agent "+clientID {
		t.Fatalf("approval does not identify the registered client and agent: %v", approval)
	}
	f.request(t, http.MethodPost, "/api/account/v1/approval-requests/"+setupString(t, approval, "id")+"/decision",
		map[string]any{"decision": "approve"}, cookie, http.StatusNoContent)
	// 保留を一度観測したので、公開応答が指定したポーリング間隔を守る。
	interval, ok := started["interval"].(float64)
	if !ok || interval <= 0 || interval > 10 {
		t.Fatalf("unexpected polling interval: %v", started)
	}
	time.Sleep(time.Duration(interval) * time.Second)
	issued := f.signedForm(t, key, clientID, "/token", url.Values{
		"grant_type": {"urn:openid:params:grant-type:ciba"}, "auth_req_id": {authReqID},
	}, http.StatusOK)
	claims := unverifiedClaims(t, setupString(t, issued, "access_token"))
	if claims["sub"] != setupString(t, user, "id") || claims["client_id"] != clientID {
		t.Fatalf("issued token does not identify the approver and registered client: %v", claims)
	}
	_ = setupString(t, issued, "id_token")
	replayed := f.signedForm(t, key, clientID, "/token", url.Values{
		"grant_type": {"urn:openid:params:grant-type:ciba"}, "auth_req_id": {authReqID},
	}, http.StatusBadRequest)
	if replayed["error"] != "invalid_grant" || replayed["access_token"] != nil {
		t.Fatalf("consumed request was reusable: %v", replayed)
	}
}
