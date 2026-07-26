package handlers_http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/labstack/echo/v5"
)

type testingT interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}

func adminCSRF(t testingT, e *echo.Echo) (string, *http.Cookie) {
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
	if cookies[0].Path != "/realms/default" {
		t.Fatalf("csrf cookie path=%q, want /realms/default", cookies[0].Path)
	}
	return body.CSRFToken, cookies[0]
}

func adminJSONRequest(
	t testingT,
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
