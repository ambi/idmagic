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

	"github.com/ambi/idmagic/backend/signinpolicy"

	"github.com/ambi/idmagic/backend/idmanagement"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"

	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"

	"github.com/ambi/idmagic/backend/audit"
	policydomain "github.com/ambi/idmagic/backend/signinpolicy/domain"
	policyports "github.com/ambi/idmagic/backend/signinpolicy/ports"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	recoverymemory "github.com/ambi/idmagic/backend/authentication/internal/recovery/db_memory"
	totpmemory "github.com/ambi/idmagic/backend/authentication/internal/totp/db_memory"
	authusecases "github.com/ambi/idmagic/backend/authentication/internal/usecases"
	webauthnmemory "github.com/ambi/idmagic/backend/authentication/internal/webauthn/db_memory"
	recoverydomain "github.com/ambi/idmagic/backend/authentication/recovery/domain"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	webauthndomain "github.com/ambi/idmagic/backend/authentication/webauthn/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	consentdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

type accountHandlerFixture struct {
	e        *echo.Echo
	users    userports.UserRepository
	consents oauthports.ConsentRepository
	factors  *totpmemory.MfaFactorRepository
	audit    auditports.AuditEventRepository
	webauthn *webauthnmemory.WebAuthnCredentialRepository
	recovery *recoverymemory.RecoveryCodeRepository
	policies policyports.DefaultSignInPolicyRepository
	quotas   tenantports.QuotaRepository
}

func newAccountHandlerFixture(t *testing.T) *accountHandlerFixture {
	t.Helper()
	users := idmanagement.NewMemoryModule().UserRepo
	consents := oauth2.NewMemoryModule().ConsentRepo
	factors := totpmemory.NewMfaFactorRepository()
	auditStore := audit.NewAuditEventStore(0)
	webauthnRepo := webauthnmemory.NewWebAuthnCredentialRepository()
	recoveryRepo := recoverymemory.NewRecoveryCodeRepository()
	policies := signinpolicy.NewMemoryModule().DefaultRepo
	quotas := tenancy.NewMemoryModule().QuotaRepo

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:                 "http://idp.test",
		UserRepo:               users,
		AuthnResolver:          authusecases.DemoHeaderResolver{},
		OAuth2:                 oauth2.Module{ConsentRepo: consents},
		Tenancy:                tenancy.Module{QuotaRepo: quotas},
		Audit:                  audit.Module{AuditEventRepo: auditStore},
		MfaFactorRepo:          factors,
		WebAuthnCredentialRepo: webauthnRepo,
		RecoveryCodeRepo:       recoveryRepo,
		SignInPolicy:           signinpolicy.Module{DefaultRepo: policies},
	})
	return &accountHandlerFixture{
		e: e, users: users, consents: consents, factors: factors, audit: auditStore,
		webauthn: webauthnRepo, recovery: recoveryRepo, policies: policies, quotas: quotas,
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
	_ = f.users.Save(context.Background(), user)
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

//verify:operation ListMyConsents: 未認証時の 401 Problem Details
func TestHandleListAccountConsentsRequiresAuthentication(t *testing.T) {
	f := newAccountHandlerFixture(t)
	resp := f.request("/api/account/v1/consents", "")
	authContractProblem(t, resp, http.StatusUnauthorized, "authentication_required")
}

//verify:operation ListMyConsents: 200 の JSON 応答に許可済みの同意だけが含まれること
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
	body := authContractJSON(t, resp, http.StatusOK)
	consents, ok := body["consents"].([]any)
	if !ok || len(consents) != 1 {
		t.Fatalf("consents=%v", body)
	}
	consent := consents[0].(map[string]any)
	if consent["client_id"] != "client-granted" || consent["client_name"] != "client-granted" || consent["state"] != "granted" {
		t.Fatalf("consent=%v", consent)
	}
	scopes, ok := consent["scopes"].([]any)
	if !ok || len(scopes) != 1 || scopes[0] != "openid" {
		t.Fatalf("scopes=%v", consent)
	}
	authContractTime(t, consent, "granted_at")
	authContractTime(t, consent, "expires_at")
}

//verify:operation RevokeMyConsent: CSRF 拒否の 403 Problem Details
func TestHandleRevokeAccountConsentRequiresBrowserVerification(t *testing.T) {
	f := newAccountHandlerFixture(t)
	f.seedUser(t, "alice")
	now := time.Now().UTC()
	if err := f.consents.Save(context.Background(), tenancydomain.DefaultTenantID, &consentdomain.Consent{
		UserID: "alice", ClientID: "some-client", Scopes: []string{"openid"},
		State: consentdomain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, defaultRealmPath("/api/account/v1/consents/some-client/revoke"), http.NoBody)
	request.Header.Set("X-Demo-Sub", "alice")
	response := httptest.NewRecorder()
	f.e.ServeHTTP(response, request)
	authContractProblem(t, response, http.StatusForbidden, "invalid_origin")
	stored, err := f.consents.Find(context.Background(), tenancydomain.DefaultTenantID, "alice", "some-client")
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || stored.State != consentdomain.ConsentGranted {
		t.Fatalf("refused revocation changed consent: %#v", stored)
	}
}

//verify:operation RevokeMyConsent: 204 の空本文と保存した同意の失効
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
	authContractEmpty(t, response)
	stored, err := f.consents.Find(context.Background(), tenancydomain.DefaultTenantID, "alice", "client-1")
	if err != nil || stored == nil || stored.State != consentdomain.ConsentRevoked || stored.RevokedAt == nil {
		t.Fatalf("revoked consent=%+v err=%v", stored, err)
	}
}

// revokeConsent は、CSRF の cookie とトークンを揃えて本人の撤回を要求する。
func (f *accountHandlerFixture) revokeConsent(t *testing.T, sub, clientID string) *httptest.ResponseRecorder {
	t.Helper()
	ctxReq := httptest.NewRequest(http.MethodGet, defaultRealmPath("/api/auth/account"), http.NoBody)
	ctxReq.Header.Set("X-Demo-Sub", sub)
	ctxResp := httptest.NewRecorder()
	f.e.ServeHTTP(ctxResp, ctxReq)
	cookies := ctxResp.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected a CSRF cookie from account context, status=%d body=%s", ctxResp.Code, ctxResp.Body.String())
	}
	csrfCookie := cookies[0]

	request := httptest.NewRequest(http.MethodPost, defaultRealmPath("/api/account/v1/consents/"+clientID+"/revoke"), http.NoBody)
	request.Header.Set("X-Demo-Sub", sub)
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", csrfCookie.Value)
	request.AddCookie(csrfCookie)
	response := httptest.NewRecorder()
	f.e.ServeHTTP(response, request)
	return response
}

// seedGrantedConsents は、付与の経路と同じく同意ごとに consents の使用量を足す。
func (f *accountHandlerFixture) seedGrantedConsents(t *testing.T, sub string, clientIDs ...string) {
	t.Helper()
	now := time.Now().UTC()
	for _, clientID := range clientIDs {
		if err := f.quotas.CheckAndIncrement(context.Background(), tenancydomain.DefaultTenantID, tenancydomain.ResourceConsents, 1); err != nil {
			t.Fatal(err)
		}
		if err := f.consents.Save(context.Background(), tenancydomain.DefaultTenantID, &consentdomain.Consent{
			UserID: sub, ClientID: clientID, Scopes: []string{"openid"},
			State: consentdomain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
	}
}

func (f *accountHandlerFixture) consentUsage(t *testing.T) int {
	t.Helper()
	usage, err := f.quotas.GetUsage(context.Background(), tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	return usage.Consents
}

//spec:covers REQ-OAUTH2-032: 本人が Granted の同意を撤回すると consents の使用量が 1 減り、ほかの同意の分は残る
func TestHandleRevokeAccountConsentReleasesConsentQuota(t *testing.T) {
	f := newAccountHandlerFixture(t)
	f.seedUser(t, "alice")
	f.seedGrantedConsents(t, "alice", "client-1", "client-2")

	authContractEmpty(t, f.revokeConsent(t, "alice", "client-1"))

	if got := f.consentUsage(t); got != 1 {
		t.Fatalf("consents usage=%d, want 1", got)
	}
}

//spec:covers REQ-OAUTH2-032: 撤回済みの同意をもう一度撤回しても consents の使用量は変わらない
func TestHandleRevokeAccountConsentAgainKeepsConsentQuota(t *testing.T) {
	f := newAccountHandlerFixture(t)
	f.seedUser(t, "alice")
	f.seedGrantedConsents(t, "alice", "client-1", "client-2")
	authContractEmpty(t, f.revokeConsent(t, "alice", "client-1"))
	usageAfterFirst := f.consentUsage(t)

	authContractEmpty(t, f.revokeConsent(t, "alice", "client-1"))

	if got := f.consentUsage(t); got != usageAfterFirst {
		t.Fatalf("consents usage=%d after the repeated revoke, want %d", got, usageAfterFirst)
	}
}

//spec:covers REQ-OAUTH2-032: 本人に同意のないクライアントの撤回は 404 と consent_not_found で拒否され、ほかの利用者の同意を変えない
//verify:operation RevokeMyConsent: 同意がないときの 404 Problem Details
func TestHandleRevokeAccountConsentWithoutConsentReturnsNotFound(t *testing.T) {
	f := newAccountHandlerFixture(t)
	f.seedUser(t, "alice")
	f.seedUser(t, "bob")
	now := time.Now().UTC()
	if err := f.consents.Save(context.Background(), tenancydomain.DefaultTenantID, &consentdomain.Consent{
		UserID: "bob", ClientID: "client-1", Scopes: []string{"openid"},
		State: consentdomain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
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
	authContractProblem(t, response, http.StatusNotFound, "consent_not_found")
	stored, err := f.consents.Find(context.Background(), tenancydomain.DefaultTenantID, "bob", "client-1")
	if err != nil || stored == nil || stored.State != consentdomain.ConsentGranted {
		t.Fatalf("other user's consent=%+v err=%v", stored, err)
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
	if err := f.policies.Save(context.Background(), &policydomain.TenantDefaultSignInPolicy{
		TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
		Rules: []policydomain.SignInRule{{
			RuleID: "default", Name: "Require MFA", Enabled: true,
			RequiredAuthn: policydomain.RequiredAuthnLevel{Strength: policydomain.RequiredAuthnMfa},
			MfaEnrollment: &policydomain.MfaEnrollmentPolicy{
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

//verify:operation ListMySignInActivity: 200 の JSON 応答に本人の sign-in event が含まれること
func TestHandleListSignInActivityReturnsSeededEvent(t *testing.T) {
	f := newAccountHandlerFixture(t)
	f.seedUser(t, "alice")
	f.seedSignIn(t, "alice", []string{"pwd"}, time.Now().UTC())
	resp := f.request("/api/account/v1/signin-activity", "alice")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	body := authContractJSON(t, resp, http.StatusOK)
	activities, ok := body["activities"].([]any)
	if !ok || len(activities) != 1 {
		t.Fatalf("activities=%v", body)
	}
	activity := activities[0].(map[string]any)
	authContractTime(t, activity, "occurred_at")
	amr, ok := activity["amr"].([]any)
	if !ok || len(amr) != 1 || amr[0] != "pwd" {
		t.Fatalf("activity=%v", activity)
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
