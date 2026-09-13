package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	approvalmemory "github.com/ambi/idmagic/backend/oauth2/approval/db_memory"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/approval/domain"
	oauthmemory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

//spec:covers REQ-OAUTH2-041: an authenticated confidential client can create a backchannel request.
func TestBackchannelAuthenticateCreatesPendingRequest(t *testing.T) {
	tenantRepo := tenancymemory.NewTenantRepository()
	_ = tenantRepo.Save(context.Background(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
		Status: tenancydomain.TenantStatusActive,
	})
	clientRepo := oauthmemory.NewClientRepository()
	secretHash := oauthdomain.HashClientSecret("client-secret")
	clientRepo.Seed(&oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: "agent-app",
		ClientSecretHash: &secretHash, ClientType: spec.ClientConfidential,
		GrantTypes: []spec.GrantType{spec.GrantCiba}, Scope: "openid payments.write",
		TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic,
	})
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{
		ID: "alice-id", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})
	store := approvalmemory.NewApprovalRequestStore()
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://test", TenantRepo: tenantRepo,
		OAuth2:       oauth2.Module{ClientRepo: clientRepo, ApprovalRequestStore: store},
		IdManagement: idmanagement.Module{UserRepo: userRepo},
	})
	form := url.Values{"login_hint": {"alice"}, "scope": {"openid payments.write"}}
	req := httptest.NewRequest(
		http.MethodPost,
		"/realms/default/bc-authorize",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("agent-app", "client-secret")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response struct {
		AuthReqID string `json:"auth_req_id"`
		ExpiresIn int    `json:"expires_in"`
		Interval  int    `json:"interval"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.AuthReqID == "" || response.ExpiresIn != 300 || response.Interval != 5 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

// 応答だけでは、承認を経ずに発行する実装と、記録が残らない実装を見分けられない。
//
//spec:covers REQ-OAUTH2-041, EX-OAUTH2-041-01: 要求は Pending → Approved → Consumed と進み、各段で BackchannelAuthRequested、BackchannelAuthApproved、AccessTokenIssued が発行され、交換は一度だけ成立する。
func TestBackchannelApprovalIssuesTokenOnce(t *testing.T) {
	tenantRepo := tenancymemory.NewTenantRepository()
	if err := tenantRepo.Save(context.Background(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, Status: tenancydomain.TenantStatusActive,
	}); err != nil {
		t.Fatal(err)
	}
	clientRepo := oauthmemory.NewClientRepository()
	secretHash := oauthdomain.HashClientSecret("client-secret")
	clientRepo.Seed(&oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: "agent-app", ClientSecretHash: &secretHash, ClientType: spec.ClientConfidential,
		GrantTypes: []spec.GrantType{spec.GrantCiba}, Scope: "openid", TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic,
	})
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{
		ID: "alice-id", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})
	store := approvalmemory.NewApprovalRequestStore()
	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	tokenIssuer := tokens_jose.NewJWTSigner("http://test", keyStore)
	authn := &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{UserID: "alice-id", AuthTime: time.Now().Unix(), StepUpAt: time.Now().Unix()}}
	events := &[]spec.DomainEvent{}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://test", TenantRepo: tenantRepo, AuthnResolver: authn,
		Emit:         func(event spec.DomainEvent) { *events = append(*events, event) },
		OAuth2:       oauth2.Module{ClientRepo: clientRepo, ApprovalRequestStore: store},
		IdManagement: idmanagement.Module{UserRepo: userRepo}, KeyStore: keyStore, TokenIssuer: tokenIssuer,
	})

	startForm := url.Values{"login_hint": {"alice"}, "scope": {"openid"}}
	startReq := httptest.NewRequest(http.MethodPost, "/realms/default/bc-authorize", strings.NewReader(startForm.Encode()))
	startReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	startReq.SetBasicAuth("agent-app", "client-secret")
	startRec := httptest.NewRecorder()
	e.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("backchannel start status=%d body=%s", startRec.Code, startRec.Body.String())
	}
	var started struct {
		AuthReqID string `json:"auth_req_id"`
	}
	if err := json.Unmarshal(startRec.Body.Bytes(), &started); err != nil || started.AuthReqID == "" {
		t.Fatalf("backchannel response=%+v err=%v", started, err)
	}
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
	pending, err := store.ListPendingForUser(ctx, "alice-id")
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending approvals=%#v err=%v", pending, err)
	}
	if pending[0].State != spec.ApprovalPending {
		t.Fatalf("state after the request=%v, want %v", pending[0].State, spec.ApprovalPending)
	}
	assertEventEmitted(t, *events, "BackchannelAuthRequested")

	decision := approvalDecisionRequest(pending[0].ID, true)
	decisionRec := httptest.NewRecorder()
	e.ServeHTTP(decisionRec, decision)
	if decisionRec.Code != http.StatusNoContent {
		t.Fatalf("approval status=%d body=%s", decisionRec.Code, decisionRec.Body.String())
	}
	approved, err := store.FindByID(ctx, pending[0].ID)
	if err != nil || approved.State != spec.ApprovalApproved {
		t.Fatalf("state after the decision=%v err=%v", approved.State, err)
	}
	assertEventEmitted(t, *events, "BackchannelAuthApproved")

	exchange := func() *httptest.ResponseRecorder {
		form := url.Values{"grant_type": {"urn:openid:params:grant-type:ciba"}, "auth_req_id": {started.AuthReqID}}
		request := httptest.NewRequest(http.MethodPost, "/realms/default/token", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.SetBasicAuth("agent-app", "client-secret")
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)
		return response
	}
	issued := exchange()
	if issued.Code != http.StatusOK {
		t.Fatalf("approved exchange status=%d body=%s", issued.Code, issued.Body.String())
	}
	var tokenBody map[string]any
	if err := json.Unmarshal(issued.Body.Bytes(), &tokenBody); err != nil {
		t.Fatalf("approved exchange body=%s err=%v", issued.Body.String(), err)
	}
	if tokenBody["access_token"] == "" || tokenBody["id_token"] == "" || tokenBody["scope"] != "openid" {
		t.Fatalf("approved exchange body=%v", tokenBody)
	}
	consumed, err := store.FindByID(ctx, pending[0].ID)
	if err != nil || consumed.State != spec.ApprovalConsumed {
		t.Fatalf("state after the exchange=%v err=%v", consumed.State, err)
	}
	if consumed.UserID != "alice-id" {
		t.Fatalf("consumed user=%q, want the approving user", consumed.UserID)
	}
	assertEventEmitted(t, *events, "AccessTokenIssued")

	if replay := exchange(); replay.Code == http.StatusOK {
		t.Fatalf("consumed auth_req_id was replayed: %s", replay.Body.String())
	}
}

type approvalHandlerFixture struct {
	e     *echo.Echo
	store *approvalmemory.ApprovalRequestStore
	authn *fakeAuthnResolver
	id    string
}

func newApprovalHandlerFixture(t *testing.T) approvalHandlerFixture {
	t.Helper()
	tenantRepo := tenancymemory.NewTenantRepository()
	_ = tenantRepo.Save(context.Background(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
		Status: tenancydomain.TenantStatusActive,
	})
	store := approvalmemory.NewApprovalRequestStore()
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
	now := time.Now().UTC()
	id, err := approvaldomain.NewApprovalRequestID()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, &approvaldomain.ApprovalRequest{
		ID: id, ClientID: "agent-app", UserID: "alice", Scopes: []string{"openid"},
		State: spec.ApprovalPending, AuthReqIDHash: approvaldomain.HashAuthReqID("secret"),
		IntervalSeconds: 5, RequestedAt: now, ExpiresAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	authn := &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{
		UserID: "alice", AuthTime: now.Unix(), StepUpAt: now.Unix(),
	}}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://test", TenantRepo: tenantRepo,
		OAuth2:        oauth2.Module{ApprovalRequestStore: store},
		AuthnResolver: authn,
	})
	return approvalHandlerFixture{e: e, store: store, authn: authn, id: id}
}

//spec:covers REQ-OAUTH2-043, EX-OAUTH2-043-02: a stale session cannot decide even with a valid CSRF token, and the refused request stays Pending.
func TestApprovalDecisionRequiresRecentStepUp(t *testing.T) {
	fix := newApprovalHandlerFixture(t)
	fix.authn.ctx.AuthTime = time.Now().Add(-time.Hour).Unix()
	fix.authn.ctx.StepUpAt = 0
	req := approvalDecisionRequest(fix.id, true)
	rec := httptest.NewRecorder()
	fix.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "step_up_required") {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
	stored, err := fix.store.FindByID(ctx, fix.id)
	if err != nil || stored.State != spec.ApprovalPending {
		t.Fatalf("stored state = %v, err = %v", stored.State, err)
	}
}

//spec:covers REQ-OAUTH2-043, EX-OAUTH2-043-03: the decision endpoint rejects a decision whose CSRF proof is missing, and the request stays Pending.
func TestApprovalDecisionRequiresCSRF(t *testing.T) {
	fix := newApprovalHandlerFixture(t)
	req := approvalDecisionRequest(fix.id, false)
	rec := httptest.NewRecorder()
	fix.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
	stored, err := fix.store.FindByID(ctx, fix.id)
	if err != nil || stored.State != spec.ApprovalPending {
		t.Fatalf("stored state = %v, err = %v", stored.State, err)
	}
}

//spec:covers REQ-OAUTH2-043: an authenticated, stepped-up owner can decide once.
func TestApprovalDecisionSucceedsForOwner(t *testing.T) {
	fix := newApprovalHandlerFixture(t)
	req := approvalDecisionRequest(fix.id, true)
	rec := httptest.NewRecorder()
	fix.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func approvalDecisionRequest(id string, csrf bool) *http.Request {
	req := httptest.NewRequest(
		http.MethodPost,
		"/realms/default/api/account/v1/approval-requests/"+id+"/decision",
		strings.NewReader(`{"decision":"approve"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://test")
	if csrf {
		req.Header.Set("X-Csrf-Token", "csrf-value")
		req.Header.Set("Cookie", "idmagic_csrf=csrf-value")
	}
	return req
}
