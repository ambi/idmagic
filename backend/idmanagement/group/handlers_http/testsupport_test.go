package handlers_http_test

// adminCSRF/adminJSONRequest are small HTTP test helpers with no
// feature-specific logic. Go test files cannot be imported across packages,
// so this file duplicates them from
// user/handlers_http/admin_user_handler_test.go for this package's own
// tests (ADR-130 Phase 2).

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

func adminCSRF(t *testing.T, e *echo.Echo) (string, *http.Cookie) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/realms/default/api/auth/account", http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("account status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	cookies := response.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("csrf cookie missing")
	}
	// path style のテナントは cookie path でテナント境界を作る (ADR-033 §1)。
	// bare path が正規ロケーションでなくなった今 (ADR-144)、cookie path は
	// テナントの URL prefix と一致する。
	if cookies[0].Path != "/realms/default" {
		t.Fatalf("csrf cookie path=%q, want /realms/default", cookies[0].Path)
	}
	return body.CSRFToken, cookies[0]
}

// this package's own tests happen to only exercise POST today.
//
//nolint:unparam // generic helper duplicated from user/handlers_http (ADR-130 Phase 2);
func adminJSONRequest(
	t *testing.T,
	e *echo.Echo,
	method, path, csrf string,
	cookie *http.Cookie,
	body any,
) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, defaultRealmPath(path), bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", csrf)
	request.Header.Set("X-Demo-Sub", "admin")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

// defaultRealmPath は bare path を default テナントの正規ロケーション配下へ移す。
// ADR-144 で bare path はどのテナントの正規ロケーションでもなくなったため、
// テストのリクエスト先も /realms/default 配下でなければ 404 になる。
func defaultRealmPath(path string) string {
	if strings.HasPrefix(path, "/realms/") {
		return path
	}
	return "/realms/default" + path
}
