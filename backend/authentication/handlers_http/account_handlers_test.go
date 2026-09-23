package handlers_http_test

// backend/authentication/handlers_http が直接所有するハンドラ
// (account context / consents / security / signin_activity / admin signin_activity)
// の正常系・未認証系をカバーする。パッケージ内の他ハンドラは各自の feature
// handlers_http (mfa/webauthn/session/...) が個別にテストする。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/audit"
	auditmemory "github.com/ambi/idmagic/backend/audit/db_memory"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	recoverymemory "github.com/ambi/idmagic/backend/authentication/recovery/db_memory"
	recoverydomain "github.com/ambi/idmagic/backend/authentication/recovery/domain"
	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/db_memory"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	webauthnmemory "github.com/ambi/idmagic/backend/authentication/webauthn/db_memory"
	webauthndomain "github.com/ambi/idmagic/backend/authentication/webauthn/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	consentmemory "github.com/ambi/idmagic/backend/oauth2/consent/db_memory"
	consentdomain "github.com/ambi/idmagic/backend/oauth2/consent/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

type accountHandlerFixture struct {
	e        *echo.Echo
	users    *usermemory.UserRepository
	consents *consentmemory.ConsentRepository
	factors  *totpmemory.MfaFactorRepository
	audit    *auditmemory.AuditEventStore
	webauthn *webauthnmemory.WebAuthnCredentialRepository
	recovery *recoverymemory.RecoveryCodeRepository
	policies *appmemory.DefaultSignInPolicyRepository
}

func newAccountHandlerFixture(t *testing.T) *accountHandlerFixture {
	t.Helper()
	users := usermemory.NewUserRepository()
	consents := consentmemory.NewConsentRepository()
	factors := totpmemory.NewMfaFactorRepository()
	auditStore := auditmemory.NewAuditEventStore(0)
	webauthnRepo := webauthnmemory.NewWebAuthnCredentialRepository()
	recoveryRepo := recoverymemory.NewRecoveryCodeRepository()
	policies := appmemory.NewDefaultSignInPolicyRepository()

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:                 "http://idp.test",
		UserRepo:               users,
		AuthnResolver:          authusecases.DemoHeaderResolver{},
		OAuth2:                 oauth2.Module{ConsentRepo: consents},
		Audit:                  audit.Module{AuditEventRepo: auditStore},
		MfaFactorRepo:          factors,
		WebAuthnCredentialRepo: webauthnRepo,
		RecoveryCodeRepo:       recoveryRepo,
		Application:            application.Module{DefaultSignInPolicyRepo: policies},
	})
	return &accountHandlerFixture{
		e: e, users: users, consents: consents, factors: factors, audit: auditStore,
		webauthn: webauthnRepo, recovery: recoveryRepo, policies: policies,
	}
}

func (f *accountHandlerFixture) seedUser(t *testing.T, sub string, roles ...string) {
	t.Helper()
	now := time.Now().UTC()
	user := &userdomain.User{
		ID: sub, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: sub,
		PasswordHash: "unused", Roles: roles,
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now, UpdatedAt: now,
	}
	f.users.Seed(user)
}

func (f *accountHandlerFixture) request(path, sub string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, defaultRealmPath(path), http.NoBody)
	if sub != "" {
		request.Header.Set("X-Demo-Sub", sub)
	}
	response := httptest.NewRecorder()
	f.e.ServeHTTP(response, request)
	return response
}

func TestHandleAccountContextRequiresAuthentication(t *testing.T) {
	f := newAccountHandlerFixture(t)
	resp := f.request("/api/auth/account", "")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestHandleAccountContextReturnsProfileAndCSRFCookie(t *testing.T) {
	f := newAccountHandlerFixture(t)
	f.seedUser(t, "alice", "admin")
	resp := f.request("/api/auth/account", "alice")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if len(resp.Result().Cookies()) == 0 {
		t.Fatal("expected a CSRF cookie to be issued")
	}
	body := resp.Body.String()
	if !strings.Contains(body, `"preferred_username":"alice"`) || !strings.Contains(body, `"admin"`) {
		t.Fatalf("unexpected body=%s", body)
	}
}

func TestHandleListAccountConsentsRequiresAuthentication(t *testing.T) {
	f := newAccountHandlerFixture(t)
	resp := f.request("/api/account/v1/consents", "")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestHandleListAccountConsentsReturnsGrantedConsentsOnly(t *testing.T) {
	f := newAccountHandlerFixture(t)
	f.seedUser(t, "alice")
	now := time.Now().UTC()
	if err := f.consents.Save(context.Background(), tenancydomain.DefaultTenantID, &consentdomain.Consent{
		UserID: "alice", ClientID: "client-granted", Scopes: []string{"openid"},
		State: consentdomain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.consents.Save(context.Background(), tenancydomain.DefaultTenantID, &consentdomain.Consent{
		UserID: "alice", ClientID: "client-revoked", Scopes: []string{"openid"},
		State: consentdomain.ConsentRevoked, GrantedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	resp := f.request("/api/account/v1/consents", "alice")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, "client-granted") || strings.Contains(body, "client-revoked") {
		t.Fatalf("expected only the granted consent, got body=%s", body)
	}
}

func TestHandleRevokeAccountConsentRequiresBrowserVerification(t *testing.T) {
	f := newAccountHandlerFixture(t)
	f.seedUser(t, "alice")
	request := httptest.NewRequest(http.MethodPost, defaultRealmPath("/api/account/v1/consents/some-client/revoke"), http.NoBody)
	request.Header.Set("X-Demo-Sub", "alice")
	response := httptest.NewRecorder()
	f.e.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestHandleRevokeAccountConsentSucceeds(t *testing.T) {
	f := newAccountHandlerFixture(t)
	f.seedUser(t, "alice")
	now := time.Now().UTC()
	if err := f.consents.Save(context.Background(), tenancydomain.DefaultTenantID, &consentdomain.Consent{
		UserID: "alice", ClientID: "client-1", Scopes: []string{"openid"},
		State: consentdomain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	// Fetch a CSRF cookie/token through an authenticated GET first, then
	// present both on the POST (double-submit) along with a matching Origin.
	ctxReq := httptest.NewRequest(http.MethodGet, defaultRealmPath("/api/auth/account"), http.NoBody)
	ctxReq.Header.Set("X-Demo-Sub", "alice")
	ctxResp := httptest.NewRecorder()
	f.e.ServeHTTP(ctxResp, ctxReq)
	cookies := ctxResp.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected a CSRF cookie from account context, status=%d body=%s", ctxResp.Code, ctxResp.Body.String())
	}
	csrfCookie := cookies[0]

	request := httptest.NewRequest(http.MethodPost, defaultRealmPath("/api/account/v1/consents/client-1/revoke"), http.NoBody)
	request.Header.Set("X-Demo-Sub", "alice")
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", csrfCookie.Value)
	request.AddCookie(csrfCookie)
	response := httptest.NewRecorder()
	f.e.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestHandleGetAccountSecurityRequiresAuthentication(t *testing.T) {
	f := newAccountHandlerFixture(t)
	resp := f.request("/api/account/v1/security", "")
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestHandleGetAccountSecurityReturnsEnrolledFactors(t *testing.T) {
	f := newAccountHandlerFixture(t)
	f.seedUser(t, "alice")
	now := time.Now().UTC()
	if err := f.factors.Save(context.Background(), &totpdomain.MfaFactor{
		UserID: "alice", Type: spec.MfaFactorTOTP, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.webauthn.Save(context.Background(), &webauthndomain.WebAuthnCredential{
		CredentialID: "cred-1", UserID: "alice", PublicKey: "pub", Transports: []string{"usb"}, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	consumedAt := now.Add(-time.Hour)
	if err := f.recovery.ReplaceAll(context.Background(), "alice", []*recoverydomain.RecoveryCode{
		{UserID: "alice", CodeHash: "hash-1", GeneratedAt: now},
		{UserID: "alice", CodeHash: "hash-2", GeneratedAt: now, ConsumedAt: &consumedAt},
	}); err != nil {
		t.Fatal(err)
	}
	resp := f.request("/api/account/v1/security", "alice")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, `"totp_enrolled":true`) {
		t.Fatalf("expected totp_enrolled=true, got body=%s", body)
	}
	if !strings.Contains(body, `"cred-1"`) {
		t.Fatalf("expected webauthn credential in body=%s", body)
	}
	if !strings.Contains(body, `"total":2`) || !strings.Contains(body, `"remaining":1`) {
		t.Fatalf("expected recovery code counts total=2 remaining=1, got body=%s", body)
	}
}

//spec:covers REQ-AUTHENTICATION-019, EX-AUTHENTICATION-019-01: 強制開始前のテナントデフォルトポリシーの開始日時を、認証要素が未登録の利用者にだけ返すこと
func TestHandleGetAccountSecurityAnnouncesUpcomingMfaEnforcement(t *testing.T) {
	f := newAccountHandlerFixture(t)
	f.seedUser(t, "alice")
	f.seedUser(t, "bob")
	f.seedUser(t, "carol")
	now := time.Now().UTC()
	start := now.Add(72 * time.Hour).Truncate(time.Second)
	grace := 3600
	if err := f.policies.Save(context.Background(), &appdomain.TenantDefaultSignInPolicy{
		TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
		Rules: []appdomain.SignInRule{{
			RuleID: "default", Name: "Require MFA", Enabled: true,
			RequiredAuthn: appdomain.RequiredAuthnLevel{Strength: appdomain.RequiredAuthnMfa},
			MfaEnrollment: &appdomain.MfaEnrollmentPolicy{
				EnforcementStartAt: &start, GracePeriodSeconds: &grace, AllowAdminBypass: true,
			},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.factors.Save(context.Background(), &totpdomain.MfaFactor{
		UserID: "bob", Type: spec.MfaFactorTOTP, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.webauthn.Save(context.Background(), &webauthndomain.WebAuthnCredential{
		CredentialID: "carol-key", UserID: "carol", PublicKey: "pub", CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	enforcementStart := func(sub string) *time.Time {
		t.Helper()
		resp := f.request("/api/account/v1/security", sub)
		if resp.Code != http.StatusOK {
			t.Fatalf("%s: status=%d body=%s", sub, resp.Code, resp.Body.String())
		}
		var body struct {
			MfaEnforcementStartAt *time.Time `json:"mfa_enforcement_start_at"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.MfaEnforcementStartAt
	}
	if got := enforcementStart("alice"); got == nil || !got.Equal(start) {
		t.Fatalf("未登録の利用者への強制開始日時 = %v, want %v", got, start)
	}
	if got := enforcementStart("bob"); got != nil {
		t.Fatalf("TOTP を登録済みの利用者へ強制開始日時 %v を返した", got)
	}
	if got := enforcementStart("carol"); got != nil {
		t.Fatalf("パスキーを登録済みの利用者へ強制開始日時 %v を返した", got)
	}
}

func (f *accountHandlerFixture) seedSignIn(t *testing.T, sub string, amr []string, at time.Time) {
	t.Helper()
	event := &authdomain.UserAuthenticated{At: at, TenantID: tenancydomain.DefaultTenantID, UserID: sub, AMR: amr}
	amrAny := make([]any, len(amr))
	for i, v := range amr {
		amrAny[i] = v
	}
	if err := f.audit.Append(context.Background(), &auditports.AuditEventRecord{
		ID: sub + "-" + at.Format(time.RFC3339Nano), TenantID: tenancydomain.DefaultTenantID,
		Type: event.EventType(), OccurredAt: at,
		Payload: map[string]any{"userId": sub, "amr": amrAny},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestHandleListSignInActivityReturnsSeededEvent(t *testing.T) {
	f := newAccountHandlerFixture(t)
	f.seedUser(t, "alice")
	f.seedSignIn(t, "alice", []string{"pwd"}, time.Now().UTC())
	resp := f.request("/api/account/v1/signin-activity", "alice")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), `"pwd"`) {
		t.Fatalf("expected amr=pwd in body=%s", resp.Body.String())
	}
}

func TestHandleGetUserSignInActivityRequiresAdmin(t *testing.T) {
	f := newAccountHandlerFixture(t)
	f.seedUser(t, "alice")
	resp := f.request("/api/admin/v1/users/alice/signin-activity", "alice")
	if resp.Code == http.StatusOK {
		t.Fatalf("expected non-admin caller to be rejected, status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestHandleGetUserSignInActivitySucceedsForAdmin(t *testing.T) {
	f := newAccountHandlerFixture(t)
	f.seedUser(t, "alice")
	f.seedUser(t, "admin", "admin")
	f.seedSignIn(t, "alice", []string{"pwd", "otp"}, time.Now().UTC())
	resp := f.request("/api/admin/v1/users/alice/signin-activity", "admin")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), `"otp"`) {
		t.Fatalf("expected amr=otp in body=%s", resp.Body.String())
	}
}
