package handlers_http_test

// SCL シナリオ "prompt=none で session 無し" / "prompt=login" / "prompt=consent" /
// "max_age を超えた前回認証では再認証を要求する" を handler 層で検証する。
// AuthnResolver の差し替えだけで再認証フローを観測する単純構成。

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/db_memory"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type fakeAuthnResolver struct {
	ctx *authdomain.AuthenticationContext
}

func (f *fakeAuthnResolver) Resolve(_ context.Context, _ authdomain.Headers) (*authdomain.AuthenticationContext, error) {
	return f.ctx, nil
}

const (
	authClientID           = "auth-client"
	authClientSec          = "auth-client-secret"
	authRedirectURI        = "https://app.example.com/cb"
	authFirstPartyClientID = "auth-client-fp"
)

func newAuthorizeTestServer(
	t *testing.T,
	authn *authdomain.AuthenticationContext,
	consent *domain.Consent,
	options ...func(*httpadapter.Deps),
) (*echo.Echo, *[]spec.DomainEvent) {
	t.Helper()
	clientRepo := oauth2memory.NewClientRepository()
	userRepo := usermemory.NewUserRepository()
	consentRepo := oauth2memory.NewConsentRepository()
	secretHash := domain.HashClientSecret(authClientSec)
	now := time.Now().UTC()
	clientRepo.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: authClientID, ClientSecretHash: &secretHash,
		ClientType: spec.ClientConfidential, RedirectURIs: []string{authRedirectURI},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    "openid profile",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})
	// first-party クライアント : consent をスキップする検証用。
	clientRepo.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: authFirstPartyClientID, ClientType: spec.ClientPublic,
		RedirectURIs:             []string{authRedirectURI},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodNone,
		Scope:                    "openid profile idmagic.admin",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		FirstParty:               true,
		CreatedAt:                now,
	})
	if authn != nil {
		userRepo.Seed(&userdomain.User{
			ID: authn.UserID, PreferredUsername: "alice",
			TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
		})
	}
	if consent != nil {
		_ = consentRepo.Save(context.Background(), tenancydomain.DefaultTenantID, consent)
	}
	e := echo.New()
	emitted := &[]spec.DomainEvent{}
	deps := httpadapter.Deps{
		Issuer: "http://test",
		Emit:   func(e spec.DomainEvent) { *emitted = append(*emitted, e) },
		OAuth2: oauth2.Module{
			ClientRepo: clientRepo, ConsentRepo: consentRepo,
			RequestStore: oauth2memory.NewAuthorizationRequestStore(), CodeStore: oauth2memory.NewAuthorizationCodeStore(), PARStore: oauth2memory.NewPARStore(),
		},
		UserRepo: userRepo,
	}
	if authn != nil {
		deps.AuthnResolver = &fakeAuthnResolver{ctx: authn}
	}
	for _, option := range options {
		option(&deps)
	}
	httpadapter.Register(e, deps)
	return e, emitted
}

func authorizeQuery(extra url.Values) url.Values {
	q := url.Values{
		"client_id":             {authClientID},
		"redirect_uri":          {authRedirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid profile"},
		"code_challenge":        {"abcdef0123456789abcdef0123456789abcdef0123ab"},
		"code_challenge_method": {"S256"},
	}
	maps.Copy(q, extra)
	return q
}

func runAuthorize(t *testing.T, e *echo.Echo, q url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/realms/default/authorize?"+q.Encode(), http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestAuthorizePromptNoneWithoutSessionReturnsLoginRequired(t *testing.T) {
	e, _ := newAuthorizeTestServer(t, nil, nil)
	rec := runAuthorize(t, e, authorizeQuery(url.Values{"prompt": {"none"}}))
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "error=login_required") ||
		!strings.Contains(rec.Header().Get("Location"), "iss=") {
		t.Fatalf("status=%d location=%q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestAuthorizePromptLoginForcesReauthentication(t *testing.T) {
	authn := &authdomain.AuthenticationContext{
		UserID: "user_alice", AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
	}
	e, _ := newAuthorizeTestServer(t, authn, nil)
	rec := runAuthorize(t, e, authorizeQuery(url.Values{"prompt": {"login"}}))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d body=%s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); !strings.HasSuffix(loc, "/login") {
		t.Fatalf("redirect Location=%q, want /login", loc)
	}
}

func TestAuthorizeMaxAgeBeyondLastAuthForcesReauthentication(t *testing.T) {
	// auth_time が 1 時間前、max_age=60 → NeedsReauthentication=true。
	authn := &authdomain.AuthenticationContext{
		UserID: "user_alice", AuthTime: time.Now().Add(-time.Hour).Unix(), AMR: []string{"pwd"},
	}
	e, _ := newAuthorizeTestServer(t, authn, nil)
	rec := runAuthorize(t, e, authorizeQuery(url.Values{"max_age": {"60"}}))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d body=%s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); !strings.HasSuffix(loc, "/login") {
		t.Fatalf("redirect Location=%q, want /login", loc)
	}
}

func TestAuthorizePromptConsentBypassesExistingConsent(t *testing.T) {
	now := time.Now().UTC()
	authn := &authdomain.AuthenticationContext{
		UserID: "user_alice", AuthTime: now.Unix(), AMR: []string{"pwd"},
	}
	// 既存 Consent。prompt=consent が無ければ即 issueCode に進む。
	consent := &domain.Consent{
		UserID: "user_alice", ClientID: authClientID,
		Scopes:    []string{"openid", "profile"},
		State:     domain.ConsentGranted,
		GrantedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	e, _ := newAuthorizeTestServer(t, authn, consent)
	rec := runAuthorize(t, e, authorizeQuery(url.Values{"prompt": {"consent"}}))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d body=%s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); !strings.HasSuffix(loc, "/consent") {
		t.Fatalf("redirect Location=%q, want /consent", loc)
	}
}

// テナント既定のサインインポリシーが拒否したとき、/authorize が実際に返す応答を測る。
// Token とは違い Authorize には 403 の経路が存在するが、本文は OAuth のエラー形
// (`{"error": ...}` + `application/json`) ではなく RFC 9457 Problem Details である。
// 契約の AuthorizeError403 はこの実測に一致していなければならない。
func TestAuthorizeDeniedBySignInPolicyReturnsProblemDetails403(t *testing.T) {
	now := time.Now().UTC()
	authn := &authdomain.AuthenticationContext{
		UserID: "user_alice", AuthTime: now.Unix(), AMR: []string{"pwd"},
	}
	policyRepo := appmemory.NewDefaultSignInPolicyRepository()
	// 到達不能な CIDR だけを許可することで、どの試験用の client IP でも拒否させる。
	if err := policyRepo.Save(context.Background(), &appdomain.TenantDefaultSignInPolicy{
		TenantID: tenancydomain.DefaultTenantID,
		Rules: []appdomain.SignInRule{{
			RuleID: "deny-all-networks", Name: "office only", Enabled: true,
			Condition: appdomain.AccessCondition{NetworkAllowCIDRs: []string{"203.0.113.0/32"}},
		}},
	}); err != nil {
		t.Fatal(err)
	}

	e, emitted := newAuthorizeTestServer(t, authn, nil, func(deps *httpadapter.Deps) {
		deps.Application.DefaultSignInPolicyRepo = policyRepo
	})
	q := authorizeQuery(url.Values{})
	q.Set("client_id", authFirstPartyClientID)
	q.Set("scope", "openid profile idmagic.admin")
	rec := runAuthorize(t, e, q)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
		t.Fatalf("Content-Type=%q, want application/problem+json", ct)
	}
	var problem struct {
		Type   string `json:"type"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v (body=%s)", err, rec.Body.String())
	}
	if problem.Type != "urn:idmagic:error:access_denied" || problem.Status != http.StatusForbidden {
		t.Fatalf("problem=%+v, want type urn:idmagic:error:access_denied and status 403", problem)
	}
	if strings.Contains(rec.Body.String(), `"error"`) {
		t.Fatalf("403 body must not be the OAuth error shape, got %s", rec.Body.String())
	}
	// 拒否が実際に何も通していないこと。状態を読み戻して、認可コードが 1 つも
	// 発行されていないことを確かめる。状態と文言の一方だけを見る試験は、拒否を
	// 書いてから操作を続行する実装にも通ってしまう。
	for _, event := range *emitted {
		if _, ok := event.(*domain.AuthorizationCodeIssued); ok {
			t.Fatalf("denied authorization issued an authorization code: %+v", event)
		}
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("denied authorization must not redirect, got Location=%q", loc)
	}
}

// MFA を要求するポリシーに対し、第二要素を 1 つも持たず登録も開始できない利用者は
// 403 で止まる。ここも応答を書いたあとに拒否を伝えない形になっていた分岐である。
func TestAuthorizeMfaRequiredWithoutSecondFactorIssuesNoCode(t *testing.T) {
	now := time.Now().UTC()
	authn := &authdomain.AuthenticationContext{
		UserID: "user_alice", AuthTime: now.Unix(), AMR: []string{"pwd"},
	}
	policyRepo := appmemory.NewDefaultSignInPolicyRepository()
	if err := policyRepo.Save(context.Background(), &appdomain.TenantDefaultSignInPolicy{
		TenantID: tenancydomain.DefaultTenantID,
		Rules: []appdomain.SignInRule{{
			RuleID: "mfa-required", Name: "mfa", Enabled: true,
			RequiredAuthn: appdomain.RequiredAuthnLevel{Strength: appdomain.RequiredAuthnMfa},
		}},
	}); err != nil {
		t.Fatal(err)
	}

	e, emitted := newAuthorizeTestServer(t, authn, nil, func(deps *httpadapter.Deps) {
		deps.Application.DefaultSignInPolicyRepo = policyRepo
	})
	q := authorizeQuery(url.Values{})
	q.Set("client_id", authFirstPartyClientID)
	q.Set("scope", "openid profile idmagic.admin")
	rec := runAuthorize(t, e, q)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", rec.Code, rec.Body.String())
	}
	for _, event := range *emitted {
		if _, ok := event.(*domain.AuthorizationCodeIssued); ok {
			t.Fatalf("refused authorization issued an authorization code: %+v", event)
		}
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("refused authorization must not redirect, got Location=%q", loc)
	}
}

// 第二要素は持っているがセッションが引けず、ステップアップを開始できない場合。
// 401 を書いたあとに拒否を伝えない形になっていた 3 つめの分岐である。
func TestAuthorizeStepUpWithExpiredSessionIssuesNoCode(t *testing.T) {
	now := time.Now().UTC()
	// AuthnResolver は認証済みの文脈を返すが、SessionManager の store は空なので
	// RequireFactor は「セッションなし」を返す。ステップアップを始められない状態。
	authn := &authdomain.AuthenticationContext{
		UserID: "user_alice", SessionID: "sess-gone", AuthTime: now.Unix(), AMR: []string{"pwd"},
	}
	policyRepo := appmemory.NewDefaultSignInPolicyRepository()
	if err := policyRepo.Save(context.Background(), &appdomain.TenantDefaultSignInPolicy{
		TenantID: tenancydomain.DefaultTenantID,
		Rules: []appdomain.SignInRule{{
			RuleID: "mfa-required", Name: "mfa", Enabled: true,
			RequiredAuthn: appdomain.RequiredAuthnLevel{Strength: appdomain.RequiredAuthnMfa},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	// TOTP を登録済みにして、secondFactorMethods を空でなくする。
	factorRepo := totpmemory.NewMfaFactorRepository()
	secret := "JBSWY3DPEHPK3PXP"
	if err := factorRepo.Save(context.Background(), &totpdomain.MfaFactor{
		UserID: "user_alice", Type: spec.MfaFactorTOTP, Secret: &secret, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	e, emitted := newAuthorizeTestServer(t, authn, nil, func(deps *httpadapter.Deps) {
		deps.Application.DefaultSignInPolicyRepo = policyRepo
		deps.MfaFactorRepo = factorRepo
		deps.SessionManager = sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())
	})
	q := authorizeQuery(url.Values{})
	q.Set("client_id", authFirstPartyClientID)
	q.Set("scope", "openid profile idmagic.admin")
	rec := runAuthorize(t, e, q)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s, want 401", rec.Code, rec.Body.String())
	}
	for _, event := range *emitted {
		if _, ok := event.(*domain.AuthorizationCodeIssued); ok {
			t.Fatalf("refused authorization issued an authorization code: %+v", event)
		}
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("refused authorization must not redirect, got Location=%q", loc)
	}
}

// first-party クライアントは consent 画面をスキップし、同意レコードが
// 無くても即 authorization code を発行する (redirect_uri へ code 付きで 303)。
func TestAuthorizeFirstPartyClientSkipsConsent(t *testing.T) {
	now := time.Now().UTC()
	authn := &authdomain.AuthenticationContext{
		UserID: "user_alice", AuthTime: now.Unix(), AMR: []string{"pwd"},
	}
	e, emitted := newAuthorizeTestServer(t, authn, nil)
	q := authorizeQuery(url.Values{})
	q.Set("client_id", authFirstPartyClientID)
	q.Set("scope", "openid profile idmagic.admin")
	rec := runAuthorize(t, e, q)
	// 認可コード発行は redirect_uri へ 302 (Found)。/consent への 303 ではない。
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if strings.Contains(loc, "/consent") {
		t.Fatalf("first-party client must skip consent, got Location=%q", loc)
	}
	if !strings.HasPrefix(loc, authRedirectURI) || !strings.Contains(loc, "code=") {
		t.Fatalf("expected redirect to %s with code, got Location=%q", authRedirectURI, loc)
	}
	found := false
	for _, e := range *emitted {
		if _, ok := e.(*domain.AuthorizationCodeIssued); ok {
			found = true
		}
	}
	if !found {
		t.Fatal("expected AuthorizationCodeIssued to be emitted")
	}
}

// Federated authentication completes outside the password handler. The resume
// endpoint must continue the authorization transaction using the newly created
// session and issue a code for a first-party client.
func TestAuthorizeResumeCompletesFederatedSession(t *testing.T) {
	now := time.Now().UTC()
	authn := &authdomain.AuthenticationContext{
		UserID: "user_alice", SessionID: "session_federated",
		AuthTime: now.Unix(), AMR: []string{"federated"},
	}
	e, _ := newAuthorizeTestServer(t, authn, nil)
	q := authorizeQuery(url.Values{"prompt": {"login"}})
	q.Set("client_id", authFirstPartyClientID)
	q.Set("scope", "openid profile idmagic.admin")

	start := runAuthorize(t, e, q)
	if start.Code != http.StatusSeeOther {
		t.Fatalf("start status=%d body=%s", start.Code, start.Body.String())
	}
	var transactionCookie *http.Cookie
	for _, cookie := range start.Result().Cookies() {
		if strings.Contains(cookie.Name, "idmagic_transaction") {
			transactionCookie = cookie
			break
		}
	}
	if transactionCookie == nil {
		t.Fatal("authorization transaction cookie was not set")
	}

	req := httptest.NewRequest(http.MethodGet, "/realms/default/authorize/resume", http.NoBody)
	req.AddCookie(transactionCookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("resume status=%d body=%s", rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); !strings.HasPrefix(location, authRedirectURI) ||
		!strings.Contains(location, "code=") {
		t.Fatalf("resume Location=%q, want redirect URI with authorization code", location)
	}
}
