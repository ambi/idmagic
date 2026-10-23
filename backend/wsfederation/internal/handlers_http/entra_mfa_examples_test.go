package handlers_http_test

// Entra の条件付きアクセスが MFA を求めるとき、IdMagic が済ませた MFA を Entra に伝え、Entra が
// MFA を求めて戻した要求には第二要素を求めることを、実物のログインセッションの上で観測する。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	recoverydomain "github.com/ambi/idmagic/backend/authentication/recovery/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

const (
	multipleAuthn        = "http://schemas.microsoft.com/claims/multipleauthn"
	saml11PasswordMethod = "urn:oasis:names:tc:SAML:1.0:am:password"
	entraRecoveryCode    = "abcd-efgh-ij"
)

// sessionOrActor は、Cookie が指すログインセッションがあればそれを、なければテストが差し替える
// 行為者を返す。Entra の構成は管理者の行為者で行い、サインインは実物のセッションを通す。
type sessionOrActor struct {
	sessions *authentication.SessionManager
	current  **authdomain.AuthenticationContext
}

func (r sessionOrActor) Resolve(ctx context.Context, headers authdomain.Headers) (*authdomain.AuthenticationContext, error) {
	if authn, err := r.sessions.Resolve(ctx, headers); err != nil || authn != nil {
		return authn, err
	}
	return *r.current, nil
}

type entraMFAServer struct {
	e        *echo.Echo
	events   *[]spec.DomainEvent
	sessions *authentication.SessionManager
}

// newEntraMFAServer は、contoso.com をフェデレーションし、alice に未使用の復旧コードを持たせ、
// carol には第二要素を持たせないサーバーを返す。
func newEntraMFAServer(t *testing.T) entraMFAServer {
	t.Helper()
	sessions := authentication.Module{SessionStore: authentication.NewMemoryModule(nil).SessionStore}.NewSessionManager()
	codes := authentication.NewMemoryModule(nil).RecoveryCodeRepo
	sum := sha256.Sum256([]byte(strings.ReplaceAll(entraRecoveryCode, "-", "")))
	if err := codes.ReplaceAll(context.Background(), entraAlice.id, []*recoverydomain.RecoveryCode{{
		UserID: entraAlice.id, CodeHash: hex.EncodeToString(sum[:]), GeneratedAt: time.Now().UTC(),
	}}); err != nil {
		t.Fatal(err)
	}
	var current *authdomain.AuthenticationContext
	s := newEntraServerConfigured(t, protectedSourceAnchor(), func(d *httpadapter.Deps) {
		current = &authdomain.AuthenticationContext{UserID: entraAdmin.id, AuthTime: time.Now().Unix()}
		d.AuthnResolver = sessionOrActor{sessions: sessions, current: &current}
		d.SessionManager = sessions
		d.RecoveryCodeRepo = codes
	}, entraAlice, entraCarol)
	if rec := configureEntra(s.e, `{"domain":"contoso.com","source_anchor_attribute":"object_guid"}`); rec.Code != http.StatusCreated {
		t.Fatalf("configure: status=%d body=%s", rec.Code, rec.Body.String())
	}
	current = nil
	*s.events = nil
	return entraMFAServer{e: s.e, events: s.events, sessions: sessions}
}

// signIn は amr で認証したログインセッションを作り、その Cookie を返す。
func (s entraMFAServer) signIn(t *testing.T, userID string, amr ...string) (string, string) {
	t.Helper()
	authn, err := s.sessions.Create(defaultTenantContext(), userID, amr, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return authn.SessionID, authentication.SessionCookie + "=" + authn.SessionID
}

func (s entraMFAServer) storedSession(t *testing.T, sessionID string) *authdomain.AuthenticationContext {
	t.Helper()
	authn, err := s.sessions.Resolve(defaultTenantContext(), authdomain.HTTPHeadersAdapter{H: http.Header{
		"Cookie": []string{authentication.SessionCookie + "=" + sessionID},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return authn
}

func defaultTenantContext() context.Context {
	return tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
}

func getWithCookie(e *echo.Echo, target, cookie string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, http.NoBody)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// entraSignInPath は、Entra が送る形の受動的な要求のパスである。wauth が空なら付けない。
func entraSignInPath(wauth string) string {
	query := url.Values{"wa": {"wsignin1.0"}, "wtrealm": {entraRealm}, "wctx": {"entra-ctx"}}
	if wauth != "" {
		query.Set("wauth", wauth)
	}
	return defaultRealmPath("/wsfed?" + query.Encode())
}

func authenticationMethod(t *testing.T, body string) string {
	t.Helper()
	statement := entraAssertion(t, body, true).FindElement("./AuthenticationStatement")
	if statement == nil {
		t.Fatalf("AuthenticationStatement missing: %s", body)
	}
	return statement.SelectAttrValue("AuthenticationMethod", "")
}

//spec:covers REQ-WSFEDERATION-002: 記憶済みの端末を除く第二要素で認証したログインセッションだけが、Entra への Assertion の認証方式を multipleauthn にすること。
//spec:covers ENTRA-MFA-AUTHENTICATION-METHOD: TOTP で認証したセッションの SAML 1.1 の AuthenticationStatement の AuthenticationMethod が multipleauthn で、パスワードだけと記憶済みの端末のセッションでは password であること。
func TestEntraMFA_ReportsTheSecondFactorAsMultipleAuthn(t *testing.T) {
	for _, tc := range []struct {
		name string
		amr  []string
		want string
	}{
		{"password and TOTP", []string{authdomain.AMRPassword, authdomain.AMROTP}, multipleAuthn},
		{"password only", []string{authdomain.AMRPassword}, saml11PasswordMethod},
		{"password and trusted device", []string{authdomain.AMRPassword, authdomain.AMRTrustedDevice}, saml11PasswordMethod},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newEntraMFAServer(t)
			_, cookie := s.signIn(t, entraAlice.id, tc.amr...)
			rec := getWithCookie(s.e, entraSignInPath(""), cookie)
			if rec.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if got := authenticationMethod(t, rec.Body.String()); got != tc.want {
				t.Fatalf("AuthenticationMethod = %q, want %q", got, tc.want)
			}
		})
	}
}

//spec:covers REQ-WSFEDERATION-002: MFA の wauth を受けたパスワードだけのセッションを、トークンを返さずに第二要素の照合の待ちにして第二要素の画面へ 303 で送り、照合の後に戻った要求へ multipleauthn の Assertion を返すこと。
//spec:covers ADFS-MFA-WAUTH: wauth の multipleauthn に、第二要素を済ませたセッションでだけ発行すること。
func TestEntraMFA_RequiresASecondFactorWhenEntraAsksForMFA(t *testing.T) {
	s := newEntraMFAServer(t)
	sessionID, cookie := s.signIn(t, entraAlice.id, authdomain.AMRPassword)
	signIn := entraSignInPath(multipleAuthn)

	rec := getWithCookie(s.e, signIn, cookie)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d body=%s, want 303 to the second factor", rec.Code, rec.Body.String())
	}
	if want := "/realms/default/totp?return_to=" + url.QueryEscape(signIn); rec.Header().Get("Location") != want {
		t.Fatalf("Location = %q, want %q", rec.Header().Get("Location"), want)
	}
	if hasEvent(*s.events, "WsFedSignInIssued") || strings.Contains(rec.Body.String(), "RequestSecurityTokenResponse") {
		t.Fatalf("token issued before the second factor: events=%v body=%s", *s.events, rec.Body.String())
	}
	if stored := s.storedSession(t, sessionID); stored == nil || !stored.AuthenticationPending {
		t.Fatalf("session = %+v, want pending a second factor", stored)
	}

	code := postJSONWithCookie(s.e, "/realms/default/api/auth/recovery-code", cookie,
		`{"code":"`+entraRecoveryCode+`","return_to":"`+signIn+`"}`)
	if code.Code != http.StatusOK {
		t.Fatalf("recovery code: status=%d body=%s", code.Code, code.Body.String())
	}
	var next struct {
		RedirectTo string `json:"redirect_to"`
	}
	if err := json.Unmarshal(code.Body.Bytes(), &next); err != nil || next.RedirectTo != signIn {
		t.Fatalf("recovery code redirect_to = %q (%v), want %q", next.RedirectTo, err, signIn)
	}

	back := getWithCookie(s.e, next.RedirectTo, cookie)
	if back.Code != http.StatusOK {
		t.Fatalf("after the second factor: status=%d body=%s", back.Code, back.Body.String())
	}
	if got := authenticationMethod(t, back.Body.String()); got != multipleAuthn {
		t.Fatalf("AuthenticationMethod = %q, want %q", got, multipleAuthn)
	}
}

//spec:covers REQ-WSFEDERATION-002: 第二要素を持たない利用者への MFA の wauth を、400 と WsFedSignInRejected で拒否し、トークンを発行せず、セッションを待ちにしないこと。
func TestEntraMFA_RefusesMFAForAUserWithoutASecondFactor(t *testing.T) {
	s := newEntraMFAServer(t)
	sessionID, cookie := s.signIn(t, entraCarol.id, authdomain.AMRPassword)

	rec := getWithCookie(s.e, entraSignInPath(multipleAuthn), cookie)
	if rec.Code != http.StatusBadRequest || !hasEvent(*s.events, "WsFedSignInRejected") {
		t.Fatalf("status=%d body=%s events=%v, want 400 and WsFedSignInRejected", rec.Code, rec.Body.String(), *s.events)
	}
	if hasEvent(*s.events, "WsFedSignInIssued") || strings.Contains(rec.Body.String(), "RequestSecurityTokenResponse") {
		t.Fatalf("token issued: events=%v body=%s", *s.events, rec.Body.String())
	}
	if stored := s.storedSession(t, sessionID); stored == nil || stored.AuthenticationPending {
		t.Fatalf("session = %+v, want the unchanged authenticated session", stored)
	}
}

func postJSONWithCookie(e *echo.Echo, target, cookie, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://idp.example")
	req.Header.Set("X-Csrf-Token", "csrf-val")
	req.Header.Set("Cookie", cookie+"; idmagic_csrf=csrf-val")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}
