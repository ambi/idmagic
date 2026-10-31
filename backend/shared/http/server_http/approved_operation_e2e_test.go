package server_http_test

// 人が承認した操作の実行を、/bc-authorize、/token、評価、別配布の参照 PEP を通して確かめる。
// 業務作用は PEP の台帳が呼ぶ処理の回数で読む。token の発行回数は操作の回数の証拠にならない。

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	pep "github.com/ambi/idmagic/examples/resource-server-pep"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

const (
	approvedDetailType = "payment"
	approvedParameters = `{"amount":100,"currency":"JPY"}`
)

type approvedOperationFixture struct {
	*resourceAccessFixture
	clientID string
}

func newApprovedOperationFixture(t *testing.T) *approvedOperationFixture {
	t.Helper()
	return newApprovedOperationFixtureWithLimiter(t, newResourceAccessLimiter(100_000))
}

func newApprovedOperationFixtureWithLimiter(t *testing.T, limiter rlports.RateLimiter) *approvedOperationFixture {
	t.Helper()
	f := &approvedOperationFixture{
		resourceAccessFixture: newResourceAccessFixtureWithLimiter(t, limiter, "A1"),
		clientID:              agentClientID("A1"),
	}
	f.grantD1(t)
	ctx := context.Background()
	client, err := f.s.Clients.FindByID(ctx, tenancydomain.DefaultTenantID, f.clientID)
	if err != nil || client == nil {
		t.Fatalf("client: %v %v", client, err)
	}
	client.GrantTypes = append(client.GrantTypes, spec.GrantCiba)
	client.Scope = "openid tools:read"
	if err := f.s.Clients.Save(ctx, client); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := f.s.AuthzDetailTypes.Save(ctx, &oauthdomain.AuthorizationDetailType{
		TenantID: tenancydomain.DefaultTenantID, Type: approvedDetailType, Description: "payment",
		DisplayTemplate: "{actions}", State: oauthdomain.DetailTypeEnabled,
		Schema: oauthdomain.AuthorizationDetailsSchema{Rules: []oauthdomain.AuthorizationDetailFieldRule{
			{Name: "actions", Semantics: oauthdomain.DetailFieldSet, Required: true},
			{Name: "locations", Semantics: oauthdomain.DetailFieldSet, Required: true},
			{Name: "identifier", Semantics: oauthdomain.DetailFieldExact, Required: true},
			{Name: "amount", Semantics: oauthdomain.DetailFieldAtMost, Required: true},
			{Name: "currency", Semantics: oauthdomain.DetailFieldExact, Required: true},
		}},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	server, err := f.s.McpResourceServers.FindByResource(ctx, tenancydomain.DefaultTenantID, agentRevocationResource)
	if err != nil || server == nil {
		t.Fatalf("resource server: %v %v", server, err)
	}
	server.AccessRules = []oauthdomain.ResourceAccessRule{{
		ResourceType: "document", Relation: "viewer", Scopes: []string{"tools:read"},
		AuthorizationDetailType: approvedDetailType, RequiresApprovedOperation: true,
	}}
	if err := f.s.McpResourceServers.Save(ctx, server); err != nil {
		t.Fatal(err)
	}
	return f
}

//spec:covers REQ-OAUTH2-062, REQ-OAUTH2-065: 超過時に承認の使用権を消費せず、別の操作キーで最初の許可を得られる
func TestResourceAccessRateLimitDoesNotClaimApproval(t *testing.T) {
	limiter := newResourceAccessLimiter(1)
	f := newApprovedOperationFixtureWithLimiter(t, limiter)
	token := f.approvedToken(t)
	if w, _ := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, approvedForm("invalid", "burn", approvedParameters)); w.Code != http.StatusOK {
		t.Fatalf("予算の消費: %d", w.Code)
	}
	before := len(f.s.Events.All())
	if w, _ := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, approvedForm(token, "blocked", approvedParameters)); w.Code != http.StatusTooManyRequests {
		t.Fatalf("超過: %d", w.Code)
	}
	if len(f.s.Events.All()) != before {
		t.Fatal("超過した評価がイベントを発行した")
	}
	limiter.now = limiter.now.Add(time.Minute)
	w, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, approvedForm(token, "first-use", approvedParameters))
	approved, _ := body["approved_operation"].(map[string]any)
	if w.Code != http.StatusOK || body["decision"] != true || approved["replayed"] != false {
		t.Fatalf("未消費の承認: %d %v", w.Code, body)
	}
}

func (f *approvedOperationFixture) tenantContext() context.Context {
	return tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, stack.Issuer, "/realms/default")
}

// approve は /bc-authorize で起票し、本人の判断を記録して、記録と未交換の auth_req_id を返す。
// 判断の入口そのものは REQ-OAUTH2-043 のテストが通す。
func (f *approvedOperationFixture) approve(t *testing.T, location string) (*oauthdomain.ApprovalRequest, string) {
	t.Helper()
	user, err := f.s.Users.FindBySub(context.Background(), stack.UserID)
	if err != nil || user == nil {
		t.Fatalf("user: %v %v", user, err)
	}
	details, err := json.Marshal([]map[string]any{{
		"type": approvedDetailType, "locations": []string{location}, "actions": []string{"viewer"},
		"identifier": "d1", "fields": map[string]any{"amount": 100, "currency": "JPY"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	status, body := f.post(t, "bc-authorize", f.clientID, url.Values{
		"login_hint": {user.PreferredUsername}, "scope": {"openid tools:read"}, "authorization_details": {string(details)},
	})
	authReqID, _ := body["auth_req_id"].(string)
	if status != http.StatusOK || authReqID == "" {
		t.Fatalf("bc-authorize status=%d body=%v", status, body)
	}
	ctx := f.tenantContext()
	rec, err := f.s.ApprovalRequests.FindByAuthReqIDHash(ctx, oauthdomain.HashAuthReqID(authReqID))
	if err != nil || rec == nil {
		t.Fatalf("approval: %v %v", rec, err)
	}
	if decided, err := f.s.ApprovalRequests.Decide(ctx, rec.ID, stack.UserID, spec.ApprovalEventApprove, time.Now().UTC()); err != nil || decided == nil {
		t.Fatalf("decide: %v %v", decided, err)
	}
	return rec, authReqID
}

func (f *approvedOperationFixture) exchange(t *testing.T, authReqID string) string {
	t.Helper()
	status, body := f.post(t, "token", f.clientID, url.Values{"grant_type": {"urn:openid:params:grant-type:ciba"}, "auth_req_id": {authReqID}})
	token, _ := body["access_token"].(string)
	if status != http.StatusOK || token == "" {
		t.Fatalf("ciba exchange status=%d body=%v", status, body)
	}
	return token
}

func (f *approvedOperationFixture) approvedToken(t *testing.T) string {
	t.Helper()
	_, authReqID := f.approve(t, agentRevocationResource)
	return f.exchange(t, authReqID)
}

func approvedForm(token, key, parameters string) url.Values {
	form := evalForm(token)
	form.Set("operation_key", key)
	form.Set("operation_parameters", parameters)
	return form
}

// approvedBusiness は参照 PEP と冪等性の台帳で包んだ送金の業務処理である。
type approvedBusiness struct {
	effects int
	ledger  pep.OperationLedger
	handler http.Handler
}

func (f *approvedOperationFixture) business() *approvedBusiness {
	server := &approvedBusiness{}
	server.handler = pep.Enforcer{
		EvaluateURL: stack.Issuer + "/realms/default/resource-access/evaluate",
		ClientID:    evalResourceServerClient, ClientSecret: evalResourceServerSecret,
		PublicBaseURL: agentRevocationResource,
		HTTPClient:    &http.Client{Transport: roundTripToEcho{f.s.Echo}},
	}.Protect(func(r *http.Request) pep.Operation {
		raw, _ := io.ReadAll(r.Body)
		return pep.Operation{
			ResourceType: "document", ResourceID: strings.TrimPrefix(r.URL.Path, "/documents/"), Relation: "viewer",
			Key: r.Header.Get("Idempotency-Key"), Parameters: string(raw),
		}
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.ledger.Run(r.Header.Get("Idempotency-Key"), func() pep.ExecuteResult {
			server.effects++
			return pep.ExecuteSucceeded
		})
		w.WriteHeader(http.StatusOK)
	}))
	return server
}

func (s *approvedBusiness) call(token, documentID, key, parameters string) int {
	request := httptest.NewRequest(http.MethodPost, agentRevocationResource+"documents/"+documentID, strings.NewReader(parameters))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Idempotency-Key", key)
	recorder := httptest.NewRecorder()
	s.handler.ServeHTTP(recorder, request)
	return recorder.Code
}

// 承認した操作は業務作用を一度起こす。対象と引数の差分は承認を消費せず、結果の不明な
// 要求の再送は作用を繰り返さず、別の業務要求としての再利用は作用を起こさない。
//
//spec:covers REQ-OAUTH2-065: 承認と同じ操作の成功、引数差分の拒否、再送と別の鍵で作用が一つであること
//spec:covers REQ-OAUTH2-041: 登録済みの locations を aud にすること
func TestExternalPEPRunsApprovedOperationOnce(t *testing.T) {
	f := newApprovedOperationFixture(t)
	token := f.approvedToken(t)
	business := f.business()

	if status := business.call(token, "d1", "k1", `{"amount":101,"currency":"JPY"}`); status != http.StatusForbidden || business.effects != 0 {
		t.Fatalf("引数の差分: status=%d effects=%d", status, business.effects)
	}
	if status := business.call(token, "d2", "k1", approvedParameters); status != http.StatusForbidden || business.effects != 0 {
		t.Fatalf("対象の差分: status=%d effects=%d", status, business.effects)
	}
	if status := business.call(token, "d1", "k1", `{"currency":"JPY","amount":100}`); status != http.StatusOK || business.effects != 1 {
		t.Fatalf("承認と同じ操作: status=%d effects=%d", status, business.effects)
	}
	if status := business.call(token, "d1", "k1", approvedParameters); status != http.StatusOK || business.effects != 1 {
		t.Fatalf("同じ鍵の再送: status=%d effects=%d", status, business.effects)
	}
	if status := business.call(token, "d1", "k2", approvedParameters); status != http.StatusForbidden || business.effects != 1 {
		t.Fatalf("別の鍵: status=%d effects=%d", status, business.effects)
	}
}

//spec:covers REQ-OAUTH2-065: 応答の相関と不許可の理由
func TestEvaluateApprovedOperationContract(t *testing.T) {
	f := newApprovedOperationFixture(t)
	rec, authReqID := f.approve(t, agentRevocationResource)
	token := f.exchange(t, authReqID)

	_, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, approvedForm(token, "k1", "{"))
	requireReason(t, body, oauthdomain.ReasonApprovedOperationMismatch)

	recorder, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, approvedForm(token, "k1", approvedParameters))
	approved, _ := body["approved_operation"].(map[string]any)
	digest, _ := approved["operation_digest"].(string)
	if recorder.Code != http.StatusOK || body["decision"] != true || approved["approval_request_id"] != rec.ID || approved["replayed"] != false || len(digest) != 64 {
		t.Fatalf("status=%d body=%v", recorder.Code, body)
	}
	_, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, approvedForm(token, "k1", approvedParameters))
	approved, _ = body["approved_operation"].(map[string]any)
	if body["decision"] != true || approved["replayed"] != true || approved["operation_digest"] != digest {
		t.Fatalf("再送 body=%v", body)
	}
	_, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, approvedForm(token, "k2", approvedParameters))
	requireReason(t, body, oauthdomain.ReasonApprovalAlreadyUsed)

	for name, form := range map[string]url.Values{
		"鍵がない":   approvedForm(token, "", approvedParameters),
		"長すぎる鍵":  approvedForm(token, strings.Repeat("k", 257), approvedParameters),
		"制御文字":   approvedForm(token, "k\n1", approvedParameters),
		"長すぎる引数": approvedForm(token, "k1", `{"a":"`+strings.Repeat("x", 64<<10)+`"}`),
	} {
		recorder, _ := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, form)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("%s: status=%d", name, recorder.Code)
		}
	}
}

// 承認から発行していないトークンと、交換で派生したトークンは、承認の代わりにならない。
//
//spec:covers REQ-OAUTH2-065: CIBA の承認から発行していないトークンと派生のトークンを拒否すること
func TestApprovedOperationIsNotInheritedByOtherTokens(t *testing.T) {
	f := newApprovedOperationFixture(t)
	plain := f.resourceToken(t, f.clientID, agentRevocationResource, nil)
	_, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, approvedForm(plain, "k1", approvedParameters))
	// client_credentials のトークンは明細を持たないので、明細の検査で先に止まる。
	requireReason(t, body, oauthdomain.ReasonAuthorizationDetailMismatch)

	token := f.approvedToken(t)
	now := time.Now().UTC()
	if err := f.s.Consents.SaveAgentDelegation(context.Background(), tenancydomain.DefaultTenantID, &oauthdomain.Consent{
		UserID: stack.UserID, AgentID: "A1", Scopes: []string{"tools:read"},
		State: oauthdomain.ConsentGranted, GrantedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	// 関係の判定で止まらないように、派生のトークンの主体である利用者にも関係を与える。
	f.adminSession(t, "/api/admin/v1/authorization/relation-tuples", map[string]any{"writes": []map[string]any{{
		"resource_type": "document", "resource_id": "d1", "relation": "viewer", "subject_type": "user", "subject_id": stack.UserID,
	}}})
	status, exchanged := f.post(t, "token", f.clientID, url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:token-exchange"}, "subject_token": {token},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"}, "resource": {agentRevocationResource}, "scope": {"tools:read"},
	})
	derived, _ := exchanged["access_token"].(string)
	if status != http.StatusOK || derived == "" {
		t.Fatalf("exchange status=%d body=%v", status, exchanged)
	}
	if strings.Contains(decodeJWTPayload(t, derived), "approval_request_id") {
		t.Fatal("派生のトークンが承認を継承した")
	}
	_, body = f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, approvedForm(derived, "k1", approvedParameters))
	requireReason(t, body, oauthdomain.ReasonApprovalRequired)
}

// 取消、Agent の停止、期限の後は、一致する操作でも業務作用を起こさない。
//
//spec:covers REQ-OAUTH2-065: 期限切れ、失効、停止、承認の記録の消失で実行しないこと
func TestApprovedOperationStopsAfterRevocationKillAndExpiry(t *testing.T) {
	t.Run("期限", func(t *testing.T) {
		f := newApprovedOperationFixture(t)
		rec, authReqID := f.approve(t, agentRevocationResource)
		token := f.exchange(t, authReqID)
		ctx := f.tenantContext()
		stored, err := f.s.ApprovalRequests.FindByID(ctx, rec.ID)
		if err != nil || stored == nil {
			t.Fatalf("approval: %v %v", stored, err)
		}
		stored.ExpiresAt = time.Now().UTC().Add(-time.Second)
		if err := f.s.ApprovalRequests.Save(ctx, stored); err != nil {
			t.Fatal(err)
		}
		_, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, approvedForm(token, "k1", approvedParameters))
		requireReason(t, body, oauthdomain.ReasonApprovalExpired)
	})
	t.Run("記録の消失", func(t *testing.T) {
		f := newApprovedOperationFixture(t)
		token := f.approvedToken(t)
		if err := f.s.ApprovalRequests.DeleteAllForSub(f.tenantContext(), stack.UserID); err != nil {
			t.Fatal(err)
		}
		_, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, approvedForm(token, "k1", approvedParameters))
		requireReason(t, body, oauthdomain.ReasonApprovalExpired)
	})
	t.Run("Agent の停止", func(t *testing.T) {
		f := newApprovedOperationFixture(t)
		token := f.approvedToken(t)
		f.admin(t, http.MethodPost, "/api/admin/v1/agents/A1/kill", nil)
		business := f.business()
		if status := business.call(token, "d1", "k1", approvedParameters); status != http.StatusForbidden || business.effects != 0 {
			t.Fatalf("status=%d effects=%d", status, business.effects)
		}
	})
	t.Run("トークンの失効", func(t *testing.T) {
		f := newApprovedOperationFixture(t)
		token := f.approvedToken(t)
		if status, body := f.post(t, "revoke", f.clientID, url.Values{"token": {token}}); status != http.StatusOK {
			t.Fatalf("revoke status=%d body=%v", status, body)
		}
		business := f.business()
		if status := business.call(token, "d1", "k1", approvedParameters); status != http.StatusForbidden || business.effects != 0 {
			t.Fatalf("status=%d effects=%d", status, business.effects)
		}
	})
}

// 未登録の location を aud にすると、承認を経て任意の宛先のトークンを作れてしまう。
//
//spec:covers REQ-OAUTH2-041: 未登録の locations を aud にしないこと
func TestCIBATokenAudienceIgnoresUnregisteredLocation(t *testing.T) {
	f := newApprovedOperationFixture(t)
	_, authReqID := f.approve(t, evalOtherResource)
	var claims struct {
		Aud string `json:"aud"`
	}
	if err := json.Unmarshal([]byte(decodeJWTPayload(t, f.exchange(t, authReqID))), &claims); err != nil {
		t.Fatal(err)
	}
	if claims.Aud != f.clientID {
		t.Fatalf("aud=%q, want %q", claims.Aud, f.clientID)
	}
}

func decodeJWTPayload(t *testing.T, token string) string {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	return string(payload)
}
