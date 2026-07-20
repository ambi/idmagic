package http_test

// SCL interfaces "ListSessions"/"RevokeSession"/"RevokeUserSessions" (admin, wi-28 T007,
// ADR-127 決定9) を /api/admin/users/{sub}/sessions 経由で検証する。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	authmemory "github.com/ambi/idmagic/backend/authentication/session/adapters/persistence/memory"
	authdomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	authusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	demousecases "github.com/ambi/idmagic/backend/authentication/usecases"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/adapters/persistence/memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	httpadapter "github.com/ambi/idmagic/backend/shared/adapters/http/server"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

type adminSessionsFixture struct {
	e            *echo.Echo
	sessionStore *authmemory.SessionStore
	refreshStore *oauth2memory.RefreshTokenStore
}

func newAdminSessionsFixture(t *testing.T) adminSessionsFixture {
	t.Helper()
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	users.Seed(&userdomain.User{
		ID: "alice", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice",
		PasswordHash: "unused", CreatedAt: now, UpdatedAt: now,
	})

	sessionStore := authmemory.NewSessionStore()
	refreshStore := oauth2memory.NewRefreshTokenStore()

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps:          support.Deps{Issuer: "http://idp.test"},
		UserRepo:      users,
		AuthnResolver: demousecases.DemoHeaderResolver{},
		Authentication: authentication.Module{
			SessionManager: authusecases.NewSessionManager(sessionStore),
		},
		OAuth2: oauth2.Module{RefreshStore: refreshStore},
	})
	return adminSessionsFixture{e: e, sessionStore: sessionStore, refreshStore: refreshStore}
}

func (f adminSessionsFixture) seedSession(t *testing.T, sid, userID string, authTime time.Time) {
	t.Helper()
	if err := f.sessionStore.Save(context.Background(), &authdomain.LoginSession{
		ID: sid, TenantID: tenancydomain.DefaultTenantID, UserID: userID, AuthTime: authTime.Unix(),
		AMR: []string{"pwd"}, ACR: "urn:mace:incommon:iap:silver", ExpiresAt: authTime.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
}

func (f adminSessionsFixture) seedRefreshToken(t *testing.T, sid, clientID string) {
	t.Helper()
	if err := f.refreshStore.Save(context.Background(), &oauthdomain.RefreshTokenRecord{
		ID: clientID + "-rt", Hash: "hash-" + clientID, FamilyID: clientID + "-fam",
		ClientID: clientID, UserID: "alice", Scopes: []string{"openid", "offline_access"},
		IssuedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Hour), AbsoluteExpiresAt: time.Now().Add(24 * time.Hour),
		Sid: &sid,
	}); err != nil {
		t.Fatal(err)
	}
}

// sessionTestCSRF は /api/auth/password_reset_context (CSRF cookie 発行専用の GET) を叩いて
// CSRF token/cookie を得る。password feature の password_reset_handler_test.go の
// passwordResetCSRF と同じ実装だが、_test.go はパッケージを跨げないため複製する
// (ADR-130 Phase 2 と同方針)。
func sessionTestCSRF(t *testing.T, e *echo.Echo) (string, *http.Cookie) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/auth/password_reset_context", http.NoBody)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("context status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	result := response.Result()
	defer result.Body.Close()
	cookies := result.Cookies()
	if len(cookies) != 1 || body.CSRFToken == "" {
		t.Fatalf("csrf=%q cookies=%v", body.CSRFToken, cookies)
	}
	return body.CSRFToken, cookies[0]
}

func adminRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, http.NoBody)
	req.Header.Set("X-Demo-Sub", "admin")
	return req
}

// adminMutationRequest builds a POST request satisfying VerifyBrowserRequest
// (matching Origin + double-submit CSRF cookie/header), mirroring the
// existing passwordResetCSRF helper used elsewhere in this package.
func adminMutationRequest(t *testing.T, e *echo.Echo, method, path string) *http.Request {
	t.Helper()
	csrf, cookie := sessionTestCSRF(t, e)
	req := httptest.NewRequest(method, path, http.NoBody)
	req.Header.Set("X-Demo-Sub", "admin")
	req.Header.Set("Origin", "http://idp.test")
	req.Header.Set("X-Csrf-Token", csrf)
	req.AddCookie(cookie)
	return req
}

func TestAdminListSessionsReturnsTargetUserSessions(t *testing.T) {
	f := newAdminSessionsFixture(t)
	base := time.Now().UTC().Truncate(time.Second)
	f.seedSession(t, "s1", "alice", base)
	f.seedSession(t, "s2", "bob", base)

	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, adminRequest(http.MethodGet, "/api/admin/users/alice/sessions"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if want := `"id":"s1"`; !strings.Contains(rec.Body.String(), want) {
		t.Fatalf("expected %s in body=%s", want, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"id":"s2"`) {
		t.Fatalf("bob's session leaked into alice's list: %s", rec.Body.String())
	}
}

func TestAdminRevokeSessionCascadesToRefreshTokens(t *testing.T) {
	f := newAdminSessionsFixture(t)
	base := time.Now().UTC().Truncate(time.Second)
	f.seedSession(t, "s1", "alice", base)
	f.seedRefreshToken(t, "s1", "web-app")

	req := adminMutationRequest(t, f.e, http.MethodPost, "/api/admin/users/alice/sessions/s1/revoke")
	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if sess, _ := f.sessionStore.Find(context.Background(), "s1"); sess != nil {
		t.Fatal("session was not revoked")
	}
	rt, _ := f.refreshStore.FindByHash(context.Background(), "hash-web-app")
	if rt == nil || !rt.Revoked {
		t.Fatal("refresh token sharing the sid was not revoked")
	}
}

func TestAdminRevokeSessionRejectsMismatchedUser(t *testing.T) {
	f := newAdminSessionsFixture(t)
	base := time.Now().UTC().Truncate(time.Second)
	f.seedSession(t, "s1", "alice", base)

	// bob という URL に対して alice のセッション id を指定しても 404 になる。
	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, adminMutationRequest(t, f.e, http.MethodPost, "/api/admin/users/bob/sessions/s1/revoke"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if sess, _ := f.sessionStore.Find(context.Background(), "s1"); sess == nil {
		t.Fatal("alice's session must not be revoked via mismatched user_id")
	}
}

func TestAdminRevokeAllSessionsRevokesEveryTargetSession(t *testing.T) {
	f := newAdminSessionsFixture(t)
	base := time.Now().UTC().Truncate(time.Second)
	f.seedSession(t, "s1", "alice", base)
	f.seedSession(t, "s2", "alice", base.Add(time.Minute))
	f.seedSession(t, "s3", "bob", base)

	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, adminMutationRequest(t, f.e, http.MethodPost, "/api/admin/users/alice/sessions/revoke_all"))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if sess, _ := f.sessionStore.Find(context.Background(), "s1"); sess != nil {
		t.Fatal("s1 was not revoked")
	}
	if sess, _ := f.sessionStore.Find(context.Background(), "s2"); sess != nil {
		t.Fatal("s2 was not revoked")
	}
	if sess, _ := f.sessionStore.Find(context.Background(), "s3"); sess == nil {
		t.Fatal("bob's session must not be revoked")
	}
}

func TestAdminSessionEndpointsRequireAdminRole(t *testing.T) {
	f := newAdminSessionsFixture(t)
	base := time.Now().UTC().Truncate(time.Second)
	f.seedSession(t, "s1", "alice", base)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users/alice/sessions", http.NoBody)
	req.Header.Set("X-Demo-Sub", "alice")
	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", rec.Code, rec.Body.String())
	}
}
