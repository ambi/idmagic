package http_test

// mockEmailSender/adminCSRF/adminJSONRequest are small HTTP test helpers with
// no feature-specific logic. Go test files cannot be imported across
// packages, so this file duplicates them from
// user/adapters/http/admin_user_handler_test.go for this package's own
// integration tests (ADR-130 Phase 2).

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	sharednotification "github.com/ambi/idmagic/backend/shared/notification"

	"github.com/labstack/echo/v5"
)

type mockEmailSender struct{}

func (m mockEmailSender) SendEmail(ctx context.Context, message sharednotification.EmailMessage) bool {
	return true
}

func adminCSRF(t *testing.T, e *echo.Echo) (string, *http.Cookie) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/auth/account", http.NoBody)
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
	if cookies[0].Path != "/" {
		t.Fatalf("csrf cookie path=%q, want /", cookies[0].Path)
	}
	return body.CSRFToken, cookies[0]
}

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
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", csrf)
	request.Header.Set("X-Demo-Sub", "admin")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}
