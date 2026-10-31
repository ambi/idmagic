package handlers_http_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	signinports "github.com/ambi/idmagic/backend/authentication/signin/ports"
	fixture "github.com/ambi/idmagic/backend/authentication/testing_kerberos"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	policydomain "github.com/ambi/idmagic/backend/signinpolicy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
	"github.com/labstack/echo/v5"
	"github.com/otuschhoff/gokrb5/v8/gssapi"
)

type negotiateTokenRevoker struct{}

func (negotiateTokenRevoker) RevokeTokensBySid(context.Context, string) error { return nil }

type negotiateAuthorizations struct{}

func (negotiateAuthorizations) Find(*echo.Context) (signinports.Authorization, error) {
	return nil, errors.New("no authorization transaction")
}

type negotiateLimited struct{ err error }

func (l negotiateLimited) Allow(context.Context, string, string, time.Time) (rlports.RateLimitResult, error) {
	return rlports.RateLimitResult{Allowed: false, RetryAfterSeconds: 30}, l.err
}

//spec:covers REQ-AUTHENTICATION-042: レート制限とカウンター障害ではチャレンジとセッションの作成より前に拒否する。
func TestNegotiateRateLimitFailsClosed(t *testing.T) {
	for _, failure := range []error{nil, errors.New("counter unavailable")} {
		f := newNegotiateFixture(t, func(deps *authentication.RouteDeps) { deps.RateLimiter = negotiateLimited{err: failure} })
		response := f.request(nil, nil)
		if response.Code != 429 || response.Header().Get("Retry-After") != "30" || response.Header().Get("WWW-Authenticate") != "" {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
		sessions, _ := f.module.SessionStore.ListBySub(f.ctx, "alice-id")
		if len(sessions) != 0 || len(f.events) != 0 {
			t.Fatalf("sessions=%d events=%v", len(sessions), f.events)
		}
	}
}

type negotiateFixture struct {
	module   authentication.Module
	e        *echo.Echo
	kerberos *fixture.Fixture
	ctx      context.Context
	events   []string
}

func newNegotiateFixture(t *testing.T, options ...func(*authentication.RouteDeps)) *negotiateFixture {
	t.Helper()
	ctx := testing_tenant.Default(context.Background())
	users := idmanagement.NewMemoryModule().UserRepo
	now := time.Now().UTC()
	user := &userdomain.User{
		ID: "alice-id", TenantID: tenantports.TenantID(ctx), PreferredUsername: "local-user",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now,
	}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	m := authentication.NewMemoryModule(users)
	m.SessionManager = m.NewSessionManager()
	m.AuthnResolver = m.SessionManager
	f := &negotiateFixture{module: m, kerberos: fixture.New(t), e: echo.New(), ctx: ctx}
	if _, err := f.module.ConfigureNegotiate(f.kerberos.WriteConfiguration(t, user.TenantID, true), users); err != nil {
		t.Fatal(err)
	}
	deps := authentication.RouteDeps{
		Issuer: "https://sso.example.com", Emit: func(event spec.DomainEvent) { f.events = append(f.events, event.EventType()) },
		UserRepo: users, ClientIP: func(*http.Request) string { return "203.0.113.10" },
	}
	for _, configure := range options {
		configure(&deps)
	}
	f.module.RegisterRoutes(f.e.Group(""), deps, negotiateTokenRevoker{}, negotiateAuthorizations{})
	return f
}

type negotiateMfaPolicy struct{}

func (negotiateMfaPolicy) Get(_ context.Context, tenantID string) (*policydomain.TenantDefaultSignInPolicy, error) {
	return &policydomain.TenantDefaultSignInPolicy{TenantID: tenantID, Rules: []policydomain.SignInRule{{Enabled: true, RequiredAuthn: policydomain.RequiredAuthnLevel{Strength: policydomain.RequiredAuthnMfa}}}}, nil
}

func (negotiateMfaPolicy) Save(context.Context, *policydomain.TenantDefaultSignInPolicy) error {
	return nil
}

//spec:covers REQ-AUTHENTICATION-042: Kerberos の成功だけでは MFA 必須のポリシーを満たさず、要素未登録ならセッションを失効させる。
func TestNegotiateCannotBypassRequiredMFA(t *testing.T) {
	f := newNegotiateFixture(t, func(deps *authentication.RouteDeps) { deps.DefaultSignInPolicyRepo = negotiateMfaPolicy{} })
	challenge := f.request(nil, nil)
	response := f.request(challenge.Result().Cookies(), f.kerberos.Token(t, "alice", f.kerberos.Config.Realm, time.Now().Add(time.Hour), f.kerberos.Certificate))
	if response.Code != http.StatusForbidden {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "idmagic_session" && cookie.MaxAge >= 0 {
			t.Fatal("policy refusal issued a usable session cookie")
		}
	}
	sessions, err := f.module.SessionStore.ListBySub(f.ctx, "alice-id")
	if err != nil {
		t.Fatal(err)
	}
	for _, session := range sessions {
		if session.RevokedAt == nil {
			t.Fatalf("unrevoked session=%+v", session)
		}
	}
	for _, event := range f.events {
		if event == "UserAuthenticated" {
			t.Fatal("policy refusal completed authentication")
		}
	}
}

//spec:covers REQ-AUTHENTICATION-042: 既存の第二要素があるときは、その要素の照合へ進み、Kerberos だけで認証完了にしない。
func TestNegotiateContinuesRequiredSecondFactor(t *testing.T) {
	f := newNegotiateFixture(t, func(deps *authentication.RouteDeps) { deps.DefaultSignInPolicyRepo = negotiateMfaPolicy{} })
	secret := "JBSWY3DPEHPK3PXP"
	if err := f.module.MfaFactorRepo.Save(f.ctx, &totpdomain.MfaFactor{UserID: "alice-id", Type: spec.MfaFactorTOTP, Secret: &secret, CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	challenge := f.request(nil, nil)
	response := f.request(challenge.Result().Cookies(), f.kerberos.Token(t, "alice", f.kerberos.Config.Realm, time.Now().Add(time.Hour), f.kerberos.Certificate))
	if response.Code != 200 || !strings.Contains(response.Body.String(), "/totp?return_to=%2Fadmin") {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
	var sessionID string
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "idmagic_session" {
			sessionID = cookie.Value
		}
	}
	if sessionID == "" {
		t.Fatal("pending session cookie missing")
	}
	session, err := f.module.SessionStore.Find(f.ctx, sessionID)
	if err != nil || session == nil || session.UserID != "alice-id" || !session.AuthenticationPending || session.RevokedAt != nil {
		t.Fatalf("session=%+v err=%v", session, err)
	}
	for _, event := range f.events {
		if event == "UserAuthenticated" {
			t.Fatal("Kerberos bypassed MFA")
		}
	}
}

func (f *negotiateFixture) request(cookies []*http.Cookie, token []byte, configure ...func(*http.Request)) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/auth/negotiate", bytes.NewBufferString(`{"return_to":"/admin"}`)).WithContext(f.ctx)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://sso.example.com")
	r.Header.Set(support.CSRFHeader, "csrf-test")
	r.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "csrf-test"})
	for _, cookie := range cookies {
		r.AddCookie(cookie)
	}
	if token != nil {
		r.Header.Set("Authorization", "Negotiate "+base64.StdEncoding.EncodeToString(token))
	}
	w := httptest.NewRecorder()
	for _, apply := range configure {
		apply(r)
	}
	f.e.ServeHTTP(w, r)
	return w
}

//spec:covers REQ-AUTHENTICATION-042: CSRF と Origin の拒否、未対応の principal、不正な試行はセッションを作らず成功イベントも残さない。
func TestNegotiateRefusalsHaveNoSession(t *testing.T) {
	for _, name := range []string{"csrf", "origin", "unmapped", "bad-attempt", "tampered-ticket"} {
		t.Run(name, func(t *testing.T) {
			f := newNegotiateFixture(t)
			cookies := f.request(nil, nil).Result().Cookies()
			principal := "alice"
			if name == "unmapped" {
				principal = "bob"
			}
			token := f.kerberos.Token(t, principal, f.kerberos.Config.Realm, time.Now().Add(time.Hour), f.kerberos.Certificate)
			if name == "bad-attempt" {
				cookies[0].Value += "x"
			}
			if name == "tampered-ticket" {
				token[len(token)-1] ^= 1
			}
			response := f.request(cookies, token, func(r *http.Request) {
				if name == "csrf" {
					r.Header.Del(support.CSRFHeader)
				}
				if name == "origin" {
					r.Header.Set("Origin", "https://attacker.example")
				}
			})
			want := 401
			if name == "csrf" || name == "origin" {
				want = 403
			}
			if response.Code != want {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if response.Header().Get("WWW-Authenticate") != "" {
				t.Fatal("a refusal issued another Negotiate challenge")
			}
			sessions, _ := f.module.SessionStore.ListBySub(f.ctx, "alice-id")
			if len(sessions) != 0 {
				t.Fatalf("refusal created %d sessions", len(sessions))
			}
			for _, event := range f.events {
				if event == "NegotiateAuthSucceeded" || event == "UserAuthenticated" {
					t.Fatalf("refusal emitted %s", event)
				}
			}
		})
	}
}

//spec:covers REQ-AUTHENTICATION-042, REQ-AUTHENTICATION-043, RFC8176-AMR-VOCABULARY: 公開の配線から実物チケットで明示的に対応付けた User のセッションが成立し、再送では増えない。
//verify:operation AuthenticateWithNegotiate: チャレンジの 401、成功の 200 と redirect_to、Cookie、再送の拒否本文と保存結果。
func TestNegotiateCreatesMappedSessionAndRejectsReplay(t *testing.T) {
	f := newNegotiateFixture(t)
	challenge := f.request(nil, nil)
	if challenge.Code != http.StatusUnauthorized || challenge.Header().Get("WWW-Authenticate") != "Negotiate" {
		t.Fatalf("challenge=%d %s", challenge.Code, challenge.Body.String())
	}
	cookies := challenge.Result().Cookies()
	token := f.kerberos.Token(t, "alice", f.kerberos.Config.Realm, time.Now().Add(time.Hour), f.kerberos.Certificate)
	response := f.request(cookies, token)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"redirect_to":"/admin"`) {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
	if !strings.HasPrefix(response.Header().Get("WWW-Authenticate"), "Negotiate ") {
		t.Fatal("mutual authentication reply missing")
	}
	sessions, err := f.module.SessionStore.ListBySub(f.ctx, "alice-id")
	if err != nil || len(sessions) != 1 || len(sessions[0].AMR) != 1 || sessions[0].AMR[0] != "wia" || sessions[0].AuthenticationPending {
		t.Fatalf("sessions=%+v err=%v", sessions, err)
	}
	if authdomain.IsMfaAMR("wia") {
		t.Fatal("Kerberos satisfied MFA")
	}
	if strings.Join(f.events, ",") != "NegotiateAuthSucceeded,UserAuthenticated" {
		t.Fatalf("events=%v", f.events)
	}
	replayed := f.request(cookies, token)
	if replayed.Code != http.StatusUnauthorized || !strings.Contains(replayed.Body.String(), "rejected") {
		t.Fatalf("replay=%d %s", replayed.Code, replayed.Body.String())
	}
	after, _ := f.module.SessionStore.ListBySub(f.ctx, "alice-id")
	if len(after) != 1 {
		t.Fatalf("replay created %d sessions", len(after))
	}
}

//spec:covers REQ-AUTHENTICATION-042, REQ-AUTHENTICATION-043: 不正な継続先と欠落した認可要求ではチャレンジせず、セッションを作らない。
func TestNegotiateRejectsInvalidContinuation(t *testing.T) {
	for _, body := range []string{`{"return_to":"https://attacker.example"}`, `{"return_to":"//attacker.example"}`, `{"return_to":"/account"}`, `{broken`, `{}`} {
		f := newNegotiateFixture(t)
		response := f.request(nil, nil, func(request *http.Request) {
			request.Body = io.NopCloser(strings.NewReader(body))
			request.ContentLength = int64(len(body))
		})
		want := http.StatusBadRequest
		if body == `{}` {
			want = http.StatusUnauthorized
		}
		if response.Code != want || response.Header().Get("WWW-Authenticate") != "" {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
		sessions, _ := f.module.SessionStore.ListBySub(f.ctx, "alice-id")
		if len(sessions) != 0 {
			t.Fatal("invalid continuation created a session")
		}
	}
}

//spec:covers REQ-AUTHENTICATION-042: 相互認証を要求しない有効な Kerberos は受理し、空の応答トークンをチャレンジとして返さない。
func TestNegotiateWithoutMutualReplyHasNoChallenge(t *testing.T) {
	f := newNegotiateFixture(t)
	f.kerberos.GSSAPIFlags = []int{gssapi.ContextFlagInteg, gssapi.ContextFlagConf}
	challenge := f.request(nil, nil)
	response := f.request(challenge.Result().Cookies(), f.kerberos.Token(t, "alice", f.kerberos.Config.Realm, time.Now().Add(time.Hour), f.kerberos.Certificate))
	if response.Code != http.StatusOK || response.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
}

//spec:covers REQ-AUTHENTICATION-042, REQ-AUTHENTICATION-043: パッシブの呼び出しも手動設定ではフォームを保ち、WIA の明示要求は一度のチャレンジか認証後の継続へ写す。
func TestPassiveNegotiatePreservesFallbackAndContinuation(t *testing.T) {
	var deps authentication.RouteDeps
	f := newNegotiateFixture(t, func(current *authentication.RouteDeps) { deps = *current })
	if _, err := f.module.ConfigureNegotiate(f.kerberos.WriteConfiguration(t, f.kerberos.Config.TenantID, false), deps.UserRepo); err != nil {
		t.Fatal(err)
	}
	passive := f.module.NegotiateSignIn(deps)
	required := false
	f.e.GET("/passive", func(c *echo.Context) error {
		if handled, err := passive(c, "/admin", required); handled || err != nil {
			return err
		}
		return c.String(http.StatusOK, "form")
	})
	request := func(cookies []*http.Cookie, token []byte) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/passive", http.NoBody).WithContext(f.ctx)
		for _, cookie := range cookies {
			r.AddCookie(cookie)
		}
		if token != nil {
			r.Header.Set("Authorization", "Negotiate "+base64.StdEncoding.EncodeToString(token))
		}
		response := httptest.NewRecorder()
		f.e.ServeHTTP(response, r)
		return response
	}
	manual := request(nil, nil)
	if manual.Code != http.StatusOK || manual.Body.String() != "form" || manual.Header().Get("WWW-Authenticate") != "" {
		t.Fatal("manual-only configuration hid normal login")
	}
	required = true
	challenge := request(nil, nil)
	if challenge.Code != http.StatusUnauthorized || challenge.Header().Get("WWW-Authenticate") != "Negotiate" {
		t.Fatal("explicit WIA did not challenge")
	}
	fallback := request(challenge.Result().Cookies(), nil)
	if fallback.Code != http.StatusOK || fallback.Body.String() != "form" || fallback.Header().Get("WWW-Authenticate") != "" {
		t.Fatal("missing credentials did not fall back")
	}
	completed := request(challenge.Result().Cookies(), f.kerberos.Token(t, "alice", f.kerberos.Config.Realm, time.Now().Add(time.Hour), f.kerberos.Certificate))
	if completed.Code != http.StatusSeeOther || completed.Header().Get("Location") != "/admin" {
		t.Fatalf("completion=%d %s", completed.Code, completed.Body.String())
	}
}

//spec:covers REQ-AUTHENTICATION-043: チャレンジにチケットを送らない端末では通常ログインへ戻り、再チャレンジしない。
func TestNegotiateWithoutTicketFallsBackWithoutLoop(t *testing.T) {
	f := newNegotiateFixture(t)
	challenge := f.request(nil, nil)
	response := f.request(challenge.Result().Cookies(), nil)
	if response.Code != 200 || strings.TrimSpace(response.Body.String()) != "{}" || response.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("fallback=%d %s headers=%v", response.Code, response.Body.String(), response.Header())
	}
	sessions, _ := f.module.SessionStore.ListBySub(f.ctx, "alice-id")
	if len(sessions) != 0 || len(f.events) != 0 {
		t.Fatalf("sessions=%d events=%v", len(sessions), f.events)
	}
}

//spec:covers REQ-AUTHENTICATION-043: 利用可能性と自動試行の真偽だけを公開し、信頼設定を返さない。
//verify:operation GetNegotiateAvailability: 公開 GET の 200 と enabled、automatic の JSON を観測する。
func TestGetNegotiateAvailabilityHTTPContract(t *testing.T) {
	f := newNegotiateFixture(t)
	r := httptest.NewRequest(http.MethodGet, "/api/auth/negotiate", http.NoBody).WithContext(f.ctx)
	w := httptest.NewRecorder()
	f.e.ServeHTTP(w, r)
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != `{"enabled":true,"automatic":true}` {
		t.Fatalf("response=%d %s", w.Code, w.Body.String())
	}
}

func (l negotiateLimited) AllowWithin(ctx context.Context, policyID, key string, _ rlports.RateLimitPolicyConfig, now time.Time) (rlports.RateLimitResult, error) {
	return l.Allow(ctx, policyID, key, now)
}
