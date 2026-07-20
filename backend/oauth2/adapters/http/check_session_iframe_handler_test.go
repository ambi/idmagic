package http_test

// SCL interface CheckSessionIframe (OIDC Session Management 1.0, adoption: optional,
// ADR-127 決定8)。session_state の salted hash 相関は実装せず、現在の browser cookie が
// 有効な LoginSession に解決できるかどうかだけを静的ページに埋め込んで返す。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/adapters/persistence/memory"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	httpadapter "github.com/ambi/idmagic/backend/shared/adapters/http/server"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

func newCheckSessionIframeServer() (*echo.Echo, *sessionmemory.SessionStore) {
	store := sessionmemory.NewSessionStore()
	sm := sessionusecases.NewSessionManager(store)
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps:           support.Deps{Issuer: "http://idp.test"},
		Authentication: authentication.Module{SessionManager: sm, AuthnResolver: sm},
	})
	return e, store
}

func TestCheckSessionIframe_noSession_respondsChanged(t *testing.T) {
	e, _ := newCheckSessionIframeServer()
	req := httptest.NewRequest(http.MethodGet, "/session/check", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct == "" || ct[:9] != "text/html" {
		t.Fatalf("expected text/html content-type, got %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"changed"`) {
		t.Fatalf("expected embedded status changed, got body=%s", body)
	}
	if strings.Contains(body, `"unchanged"`) {
		t.Fatalf("did not expect unchanged status without a session, got body=%s", body)
	}
}

func TestCheckSessionIframe_validSession_respondsUnchanged(t *testing.T) {
	e, store := newCheckSessionIframeServer()
	sess := &sessiondomain.LoginSession{
		ID: "sess-1", TenantID: tenancydomain.DefaultTenantID, UserID: "user-1",
		AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
		ACR:       authusecases.DeriveACR([]string{"pwd"}),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := store.Save(context.Background(), sess); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/session/check", http.NoBody)
	req.AddCookie(&http.Cookie{Name: sessionusecases.SessionCookie, Value: sess.ID})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"unchanged"`) {
		t.Fatalf("expected embedded status unchanged for a valid session, got body=%s", body)
	}
}
