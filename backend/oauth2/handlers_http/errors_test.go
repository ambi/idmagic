package handlers_http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ambi/idmagic/backend/oauth2/usecases"

	"github.com/labstack/echo/v5"
)

// RFC6750-INVALID-TOKEN / REQ-OAUTH2-020: UserInfo 固有のエラー写像は、
// invalid_token を OAuth の認可サーバー用 400 写像へ流さず、保護リソース用の
// 401 と Bearer challenge へ写す。
func TestWriteUserInfoErrorMapsInvalidTokenToBearerChallenge(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/realms/default/userinfo", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := writeUserInfoError(c, usecases.NewOAuthError("invalid_token", "The token is invalid.")); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if got := rec.Header().Get("WWW-Authenticate"); got != `Bearer error="invalid_token"` {
		t.Fatalf("WWW-Authenticate=%q", got)
	}
}
