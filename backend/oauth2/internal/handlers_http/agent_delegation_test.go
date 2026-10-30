package handlers_http_test

// Agent への代行の付与を、本番と同じ組み立ての HTTP の入口から確かめる。交換の判定
// (REQ-OAUTH2-060) とイントロスペクションの判定 (REQ-OAUTH2-061) を別々に直接呼ぶと、
// 入口へ付与の保管先を渡し忘れる故障を見逃す。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/oauth2/testing_clientauth"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/signingkeys"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

const (
	delegationIssuer   = "http://idp.test"
	delegationClientID = "agent-app"
	delegationAgentID  = "agent-1"
	delegationResource = "https://api.example/orders"
	delegationBase     = "/realms/default"
)

type delegationFixture struct {
	e          *echo.Echo
	consents   *oauth2memory.ConsentRepository
	quotas     tenantports.QuotaRepository
	signer     *oauth2.JWTSigner
	credential *testing_clientauth.PrivateKeyJWT
	events     *[]spec.DomainEvent
}

func delegationRoles(id string) []string {
	if id == "admin" {
		return []string{"admin"}
	}
	return nil
}

func newDelegationFixture(t *testing.T) *delegationFixture {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	users := idmanagement.NewMemoryModule().UserRepo
	for _, id := range []string{"owner-1", "alice", "bob", "admin"} {
		if err := users.Save(ctx, &userdomain.User{
			ID: id, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: id,
			Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive, StatusChangedAt: &now},
			Roles:     delegationRoles(id),
			CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	agents := idmanagement.NewMemoryModule().AgentRepo
	for _, agent := range []*agentdomain.Agent{
		{ID: delegationAgentID, Name: "Order bot", Status: idmdomain.AgentStatusActive},
		{ID: "agent-stopped", Name: "Stopped bot", Status: idmdomain.AgentStatusDisabled},
	} {
		agent.TenantID = tenancydomain.DefaultTenantID
		agent.Kind = idmdomain.AgentKindAutonomous
		agent.OwnerUserID = "owner-1"
		agent.CreatedAt, agent.UpdatedAt = now, now
		if err := agents.Save(ctx, agent); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := agents.AddBinding(ctx, &agentdomain.AgentCredentialBinding{
		AgentID: delegationAgentID, ClientID: delegationClientID, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	credential := testing_clientauth.NewPrivateKeyJWT(t, delegationClientID)
	clients := oauth2memory.NewClientRepository()
	clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: delegationClientID,
		ClientType: spec.ClientConfidential,
		GrantTypes: []spec.GrantType{spec.GrantTokenExchange}, TokenEndpointAuthMethod: domain.AuthMethodPrivateKeyJwt,
		JWKS: credential.JWKS(), Scope: "read write", FapiProfile: domain.FapiNone, CreatedAt: now,
	})
	resources := oauth2memory.NewMcpResourceServerRepository()
	resources.Seed(&domain.McpResourceServer{
		ID: "rs-orders", Resource: delegationResource, Name: "Orders API", TenantID: tenancydomain.DefaultTenantID,
		Scopes: []string{"read", "write"}, State: domain.McpResourceServerActive,
	})
	keyStore, err := signingkeys.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := oauth2.NewJWTSigner(delegationIssuer, keyStore)
	tenancyModule := tenancy.NewMemoryModule()
	if err := tenancyModule.TenantRepo.Save(ctx, &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, Status: tenancydomain.TenantStatusActive,
	}); err != nil {
		t.Fatal(err)
	}
	consents := oauth2memory.NewConsentRepository()
	events := &[]spec.DomainEvent{}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:     delegationIssuer,
		TenantRepo: tenancyModule.TenantRepo,
		Tenancy:    tenancy.Module{QuotaRepo: tenancyModule.QuotaRepo},
		Emit:       func(event spec.DomainEvent) { *events = append(*events, event) },
		OAuth2: oauth2.Module{
			ClientRepo: clients, ConsentRepo: consents, RefreshStore: oauth2memory.NewRefreshTokenStore(),
			AccessTokenDenylist: oauth2memory.NewAccessTokenDenylist(), McpResourceServerRepo: resources,
			ClientAssertionReplayStore: oauth2memory.NewClientAssertionReplayStore(),
		},
		IdManagement:      idmanagement.Module{AgentRepo: agents, UserRepo: users},
		UserRepo:          users,
		AuthnResolver:     authentication.DemoHeaderResolver{},
		KeyStore:          keyStore,
		TokenIssuer:       signer,
		TokenIntrospector: signer,
	})
	return &delegationFixture{
		e: e, consents: consents, quotas: tenancyModule.QuotaRepo, signer: signer, credential: credential, events: events,
	}
}

// subjectToken は sub が利用者の、Agent の関わらないトークンを発行する。
func (f *delegationFixture) subjectToken(t *testing.T, sub, scope string) string {
	t.Helper()
	ctx := tenantports.WithTenant(context.Background(),
		&tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm},
		delegationIssuer+delegationBase, delegationBase)
	token, _, err := f.signer.SignAccessToken(ctx, oauthports.AccessTokenInput{
		Client: &domain.OAuth2Client{ClientID: "portal"}, Sub: sub,
		Scopes: strings.Fields(scope), AuthTime: time.Now().UTC().Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func (f *delegationFixture) postClientForm(t *testing.T, path string, form url.Values) (int, map[string]any) {
	t.Helper()
	form = f.credential.AuthenticateForm(t, form, delegationIssuer+delegationBase+"/token")
	request := httptest.NewRequest(http.MethodPost, delegationBase+path, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	f.e.ServeHTTP(response, request)
	var body map[string]any
	_ = json.Unmarshal(response.Body.Bytes(), &body)
	return response.Code, body
}

func (f *delegationFixture) exchange(t *testing.T, subjectToken, scope string) (int, map[string]any) {
	t.Helper()
	return f.postClientForm(t, "/token", url.Values{
		"grant_type":    {string(spec.GrantTokenExchange)},
		"subject_token": {subjectToken},
		"resource":      {delegationResource},
		"scope":         {scope},
	})
}

func (f *delegationFixture) introspectActive(t *testing.T, token string) bool {
	t.Helper()
	code, body := f.postClientForm(t, "/introspect", url.Values{"token": {token}, "token_type_hint": {"access_token"}})
	if code != http.StatusOK {
		t.Fatalf("introspect status=%d body=%v", code, body)
	}
	active, _ := body["active"].(bool)
	return active
}

// session は sub のブラウザーセッションとして要求する。
func (f *delegationFixture) session(t *testing.T, method, path, sub, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, delegationBase+path, strings.NewReader(body))
	request.Header.Set("X-Demo-Sub", sub)
	request.Header.Set("Origin", delegationIssuer)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Csrf-Token", "csrf-value")
	request.AddCookie(&http.Cookie{Name: "idmagic_csrf", Value: "csrf-value"})
	response := httptest.NewRecorder()
	f.e.ServeHTTP(response, request)
	return response
}

func (f *delegationFixture) grant(t *testing.T, sub, scopes string) {
	t.Helper()
	response := f.session(t, http.MethodPost, "/api/account/v1/agent-delegations", sub,
		`{"agent_id":"`+delegationAgentID+`","scopes":["`+strings.Join(strings.Fields(scopes), `","`)+`"]}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("grant status=%d body=%s", response.Code, response.Body.String())
	}
}

func (f *delegationFixture) consentUsage(t *testing.T) int {
	t.Helper()
	usage, err := f.quotas.GetUsage(context.Background(), tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	return usage.Consents
}

//spec:covers REQ-OAUTH2-060: 付与のない Agent の交換と付与を超えるスコープの交換は invalid_grant で拒否し、付与の範囲の交換だけが発行されて last_used_at を残す
func TestExchangeRequiresTheUsersDelegationGrant(t *testing.T) {
	f := newDelegationFixture(t)
	subject := f.subjectToken(t, "alice", "read write")

	if code, body := f.exchange(t, subject, "read"); code != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Fatalf("exchange without a grant status=%d body=%v", code, body)
	}

	f.grant(t, "alice", "read")
	if code, body := f.exchange(t, subject, "read write"); code != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Fatalf("exchange beyond the grant status=%d body=%v", code, body)
	}
	code, body := f.exchange(t, subject, "read")
	if code != http.StatusOK || body["access_token"] == "" {
		t.Fatalf("exchange within the grant status=%d body=%v", code, body)
	}
	stored, err := f.consents.FindAgentDelegation(context.Background(), tenancydomain.DefaultTenantID, "alice", delegationAgentID)
	if err != nil || stored == nil || stored.LastUsedAt == nil {
		t.Fatalf("grant after use=%+v err=%v", stored, err)
	}
}

//spec:covers REQ-OAUTH2-060: 期限切れの付与では交換を拒否する
func TestExchangeRejectsAnExpiredDelegationGrant(t *testing.T) {
	f := newDelegationFixture(t)
	now := time.Now().UTC()
	if err := f.consents.SaveAgentDelegation(context.Background(), tenancydomain.DefaultTenantID, &domain.Consent{
		UserID: "alice", AgentID: delegationAgentID, Scopes: []string{"read"},
		State: domain.ConsentGranted, GrantedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if code, body := f.exchange(t, f.subjectToken(t, "alice", "read"), "read"); code != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Fatalf("exchange with an expired grant status=%d body=%v", code, body)
	}
}

//spec:covers REQ-OAUTH2-061: 撤回した本人の代行トークンだけが即座に active=false になり、同じ Agent の他の利用者のトークンは active=true のままで、エポックを進める RevocationEpochAdvanced も出ない
func TestRevocationStopsOnlyThatUsersDelegation(t *testing.T) {
	f := newDelegationFixture(t)
	f.grant(t, "alice", "read")
	f.grant(t, "bob", "read")
	tokens := map[string]string{}
	for _, sub := range []string{"alice", "bob"} {
		code, body := f.exchange(t, f.subjectToken(t, sub, "read"), "read")
		if code != http.StatusOK {
			t.Fatalf("%s exchange status=%d body=%v", sub, code, body)
		}
		tokens[sub], _ = body["access_token"].(string)
		if !f.introspectActive(t, tokens[sub]) {
			t.Fatalf("%s token is inactive before the revocation", sub)
		}
	}

	revoked := f.session(t, http.MethodPost, "/api/account/v1/agent-delegations/"+delegationAgentID+"/revoke", "alice", "")
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status=%d body=%s", revoked.Code, revoked.Body.String())
	}

	if f.introspectActive(t, tokens["alice"]) {
		t.Fatal("the revoked user's token stayed active")
	}
	if !f.introspectActive(t, tokens["bob"]) {
		t.Fatal("another user's token stopped with the revocation")
	}
	for _, event := range *f.events {
		if event.EventType() == "RevocationEpochAdvanced" {
			t.Fatal("the revocation advanced the agent-wide epoch")
		}
	}
}

//spec:covers REQ-OAUTH2-061: 再付与は granted_at より前に発行したトークンを復活させない
func TestRegrantDoesNotReviveEarlierTokens(t *testing.T) {
	f := newDelegationFixture(t)
	f.grant(t, "alice", "read")
	code, body := f.exchange(t, f.subjectToken(t, "alice", "read"), "read")
	if code != http.StatusOK {
		t.Fatalf("exchange status=%d body=%v", code, body)
	}
	token, _ := body["access_token"].(string)

	// 撤回と再付与を、発行より後の秒に起きたことにする。
	now := time.Now().UTC()
	if err := f.consents.SaveAgentDelegation(context.Background(), tenancydomain.DefaultTenantID, &domain.Consent{
		UserID: "alice", AgentID: delegationAgentID, Scopes: []string{"read"},
		State: domain.ConsentGranted, GrantedAt: now.Add(2 * time.Second), ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if f.introspectActive(t, token) {
		t.Fatal("a token issued before the re-grant was revived")
	}
}

//spec:covers REQ-OAUTH2-057: Bearer のトークンでの付与は 403 で拒否し、付与の行を作らない
func TestGrantRejectsBearerTokens(t *testing.T) {
	f := newDelegationFixture(t)
	request := httptest.NewRequest(http.MethodPost, delegationBase+"/api/account/v1/agent-delegations",
		strings.NewReader(`{"agent_id":"`+delegationAgentID+`","scopes":["read"]}`))
	request.Header.Set("Authorization", "Bearer "+f.subjectToken(t, "alice", "account:write"))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	f.e.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "access_denied") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	stored, err := f.consents.FindAgentDelegation(context.Background(), tenancydomain.DefaultTenantID, "alice", delegationAgentID)
	if err != nil || stored != nil {
		t.Fatalf("grant=%+v err=%v", stored, err)
	}
}

//spec:covers REQ-OAUTH2-057: 付与できる Agent は Active だけで、不正な入力と存在しないか止まった Agent への付与は拒否して使用量を変えない
//verify:operation ListDelegableAgents: Active な Agent だけの一覧
//verify:operation GrantMyAgentDelegation: 201 の付与と 400、404 の拒否
func TestGrantValidatesTheAgentAndTheInput(t *testing.T) {
	f := newDelegationFixture(t)
	listed := f.session(t, http.MethodGet, "/api/account/v1/delegable-agents", "alice", "")
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"Order bot"`) ||
		!strings.Contains(listed.Body.String(), `"owner-1"`) || strings.Contains(listed.Body.String(), "Stopped bot") {
		t.Fatalf("delegable status=%d body=%s", listed.Code, listed.Body.String())
	}
	tooFar := time.Now().UTC().Add(366 * 24 * time.Hour).Format(time.RFC3339)
	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	for name, want := range map[string]struct {
		body   string
		status int
		code   string
	}{
		"no scopes":   {`{"agent_id":"agent-1","scopes":[" "]}`, http.StatusBadRequest, "invalid_request"},
		"past expiry": {`{"agent_id":"agent-1","scopes":["read"],"expires_at":"` + past + `"}`, http.StatusBadRequest, "invalid_request"},
		"too far":     {`{"agent_id":"agent-1","scopes":["read"],"expires_at":"` + tooFar + `"}`, http.StatusBadRequest, "invalid_request"},
		"unknown":     {`{"agent_id":"ghost","scopes":["read"]}`, http.StatusNotFound, "agent_not_found"},
		"stopped":     {`{"agent_id":"agent-stopped","scopes":["read"]}`, http.StatusNotFound, "agent_not_found"},
	} {
		response := f.session(t, http.MethodPost, "/api/account/v1/agent-delegations", "alice", want.body)
		if response.Code != want.status || !strings.Contains(response.Body.String(), want.code) {
			t.Errorf("%s: status=%d body=%s", name, response.Code, response.Body.String())
		}
	}
	if got := f.consentUsage(t); got != 0 {
		t.Fatalf("usage after the refusals=%d", got)
	}
}

//spec:covers REQ-OAUTH2-057,REQ-OAUTH2-058: 付与は consents の枠を消費し、置き換えでは足さず、撤回で 1 減り、再度の撤回では変わらず、再付与で足す
//verify:operation ListMyAgentDelegations: Granted の付与の一覧
//verify:operation RevokeMyAgentDelegation: 204 の撤回と 404 の拒否
func TestDelegationSharesTheConsentQuotaAndLifecycle(t *testing.T) {
	f := newDelegationFixture(t)
	revokePath := "/api/account/v1/agent-delegations/" + delegationAgentID + "/revoke"
	if response := f.session(t, http.MethodPost, revokePath, "alice", ""); response.Code != http.StatusNotFound ||
		!strings.Contains(response.Body.String(), "consent_not_found") {
		t.Fatalf("revoke without a grant status=%d body=%s", response.Code, response.Body.String())
	}

	f.grant(t, "alice", "read")
	f.grant(t, "alice", "read write")
	if got := f.consentUsage(t); got != 1 {
		t.Fatalf("usage after grant and replacement=%d, want 1", got)
	}
	listed := f.session(t, http.MethodGet, "/api/account/v1/agent-delegations", "alice", "")
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"agent_name":"Order bot"`) ||
		!strings.Contains(listed.Body.String(), `"write"`) {
		t.Fatalf("list status=%d body=%s", listed.Code, listed.Body.String())
	}
	if other := f.session(t, http.MethodGet, "/api/account/v1/agent-delegations", "bob", ""); strings.Contains(other.Body.String(), "Order bot") {
		t.Fatalf("another user's list=%s", other.Body.String())
	}

	for range 2 {
		if response := f.session(t, http.MethodPost, revokePath, "alice", ""); response.Code != http.StatusNoContent {
			t.Fatalf("revoke status=%d body=%s", response.Code, response.Body.String())
		}
		if got := f.consentUsage(t); got != 0 {
			t.Fatalf("usage after revoke=%d, want 0", got)
		}
	}
	if listed := f.session(t, http.MethodGet, "/api/account/v1/agent-delegations", "alice", ""); strings.Contains(listed.Body.String(), "Order bot") {
		t.Fatalf("revoked grant is still listed: %s", listed.Body.String())
	}
	f.grant(t, "alice", "read")
	if got := f.consentUsage(t); got != 1 {
		t.Fatalf("usage after re-grant=%d, want 1", got)
	}
}

//spec:covers REQ-OAUTH2-058: 代行の付与は接続済みアプリの一覧に混ざらない
func TestDelegationStaysOutOfConnectedApps(t *testing.T) {
	f := newDelegationFixture(t)
	f.grant(t, "alice", "read")
	listed := f.session(t, http.MethodGet, "/api/account/v1/consents", "alice", "")
	if listed.Code != http.StatusOK || strings.Contains(listed.Body.String(), delegationAgentID) {
		t.Fatalf("connected apps status=%d body=%s", listed.Code, listed.Body.String())
	}
}

//spec:covers REQ-OAUTH2-059: 管理者は撤回済みも含めて所属テナントの付与だけを参照でき、一般の利用者は参照できず、付与と撤回の入口は存在しない
//verify:operation ListAdminAgentDelegations: テナント内の付与の一覧と権限の拒否
func TestAdminReadsTenantDelegationsOnly(t *testing.T) {
	f := newDelegationFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	f.grant(t, "alice", "read")
	f.grant(t, "bob", "read")
	if response := f.session(t, http.MethodPost, "/api/account/v1/agent-delegations/"+delegationAgentID+"/revoke", "bob", ""); response.Code != http.StatusNoContent {
		t.Fatalf("revoke status=%d", response.Code)
	}
	if err := f.consents.SaveAgentDelegation(ctx, "acme", &domain.Consent{
		UserID: "carol", AgentID: "agent-acme", Scopes: []string{"read"},
		State: domain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	path := "/api/admin/v1/agent-delegations"
	if denied := f.session(t, http.MethodGet, path, "alice", ""); denied.Code != http.StatusForbidden {
		t.Fatalf("non-admin status=%d body=%s", denied.Code, denied.Body.String())
	}
	listed := f.session(t, http.MethodGet, path, "admin", "")
	if listed.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listed.Code, listed.Body.String())
	}
	var body struct {
		Delegations []struct {
			UserID    string              `json:"user_id"`
			AgentName string              `json:"agent_name"`
			State     domain.ConsentState `json:"state"`
		} `json:"delegations"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Delegations) != 2 || body.Delegations[0].UserID != "alice" || body.Delegations[0].State != domain.ConsentGranted ||
		body.Delegations[1].UserID != "bob" || body.Delegations[1].State != domain.ConsentRevoked ||
		body.Delegations[0].AgentName != "Order bot" {
		t.Fatalf("delegations=%+v", body.Delegations)
	}
	for _, method := range []string{http.MethodPost, http.MethodDelete} {
		if response := f.session(t, method, path, "admin", `{}`); response.Code == http.StatusOK || response.Code == http.StatusCreated || response.Code == http.StatusNoContent {
			t.Fatalf("%s on the admin list status=%d", method, response.Code)
		}
	}
}
