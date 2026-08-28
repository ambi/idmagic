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
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	approvalmemory "github.com/ambi/idmagic/backend/oauth2/approval/db_memory"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/approval/domain"
	oauthmemory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

// REQ-OAUTH2-041: an authenticated confidential client can create a backchannel request.
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

// REQ-OAUTH2-043: a stale session cannot decide even with a valid CSRF token.
func TestApprovalDecisionRequiresRecentStepUp(t *testing.T) {
	fix := newApprovalHandlerFixture(t)
	fix.authn.ctx.AuthTime = time.Now().Add(-time.Hour).Unix()
	fix.authn.ctx.StepUpAt = 0
	req := approvalDecisionRequest(fix.id, "approve", true)
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

// REQ-OAUTH2-043: the decision endpoint rejects a missing CSRF proof.
func TestApprovalDecisionRequiresCSRF(t *testing.T) {
	fix := newApprovalHandlerFixture(t)
	req := approvalDecisionRequest(fix.id, "approve", false)
	rec := httptest.NewRecorder()
	fix.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

// REQ-OAUTH2-043: an authenticated, stepped-up owner can decide once.
func TestApprovalDecisionSucceedsForOwner(t *testing.T) {
	fix := newApprovalHandlerFixture(t)
	req := approvalDecisionRequest(fix.id, "approve", true)
	rec := httptest.NewRecorder()
	fix.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func approvalDecisionRequest(id, decision string, csrf bool) *http.Request {
	req := httptest.NewRequest(
		http.MethodPost,
		"/realms/default/api/account/v1/approval-requests/"+id+"/decision",
		strings.NewReader(`{"decision":"`+decision+`"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://test")
	if csrf {
		req.Header.Set("X-Csrf-Token", "csrf-value")
		req.Header.Set("Cookie", "idmagic_csrf=csrf-value")
	}
	return req
}
