package handlers_http_test

// SCL interface CheckSessionIframe (OIDC Session Management 1.0, adoption: optional,
// 決定8)。session_state の salted hash 相関は実装せず、現在の browser cookie が
// 有効な LoginSession に解決できるかどうかだけを静的ページに埋め込んで返す。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"

	"github.com/ambi/idmagic/backend/authentication"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"

	"github.com/labstack/echo/v5"
)

func newCheckSessionIframeServer() (*echo.Echo, sessionports.SessionStore) {
	store := authentication.NewMemoryModule(nil).SessionStore
	sm := authentication.Module{SessionStore: store}.NewSessionManager()
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:         "http://idp.test",
		Authentication: authentication.Module{SessionManager: sm, AuthnResolver: sm},
	})
	return e, store
}

// OP セッションを解決できない状態を changed として送信元 origin へ返すことを固定する。
//
//spec:covers OIDC-SESSION-MGMT-CHECK-IFRAME: RP の postMessage を受ける iframe が、
func TestCheckSessionIframe_noSession_respondsChanged(t *testing.T) {
	e, _ := newCheckSessionIframeServer()
	req := httptest.NewRequest(http.MethodGet, "/realms/default/session/check", http.NoBody)
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
	if !strings.Contains(body, `window.addEventListener("message"`) {
		t.Fatalf("message listener is missing: body=%s", body)
	}
	if !strings.Contains(body, `e.source.postMessage(STATUS, e.origin)`) {
		t.Fatalf("postMessage response to the sender origin is missing: body=%s", body)
	}
}

//spec:covers OIDC-SESSION-MGMT-CHECK-IFRAME: 有効な OP セッションを unchanged として返すことを固定する。
func TestCheckSessionIframe_validSession_respondsUnchanged(t *testing.T) {
	e, store := newCheckSessionIframeServer()
	sess := &sessiondomain.LoginSession{
		ID: "sess-1", TenantID: tenancydomain.DefaultTenantID, UserID: "user-1",
		AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
		ACR:       authdomain.DeriveACR([]string{"pwd"}),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := store.Save(context.Background(), sess); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/realms/default/session/check", http.NoBody)
	req.AddCookie(&http.Cookie{Name: authentication.SessionCookie, Value: sess.ID})
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
