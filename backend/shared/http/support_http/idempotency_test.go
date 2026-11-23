package support_http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ambi/idmagic/backend/shared/idempotency"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/labstack/echo/v5"
)

type brokenResponseStore struct{ idempotency.Store }

func (brokenResponseStore) Complete(context.Context, idempotency.Record) error {
	return errors.New("injected response storage failure")
}

// HTTP の判断だけを検査する。AEAD と AAD は Ledger の実暗号を使うテストが担当する。
type responseTestCipher struct{}

func (responseTestCipher) Encrypt(_ context.Context, _, _, _, _, _, plain string) (int, []byte, error) {
	return 1, []byte(plain), nil
}

func (responseTestCipher) Decrypt(_ context.Context, _, _, _, _, _ string, _ int, value []byte) (string, error) {
	return string(value), nil
}

func idempotentRequest(t *testing.T, ledger *idempotency.Ledger, key, principal, body string, handler echo.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler(nil, nil)
	e.POST("/api/admin/v1/resources", func(c *echo.Context) error {
		return WithIdempotency(c, Deps{Idempotency: ledger}, principal, handler)
	})
	request := httptest.NewRequestWithContext(tenantports.WithTenantID(t.Context(), "tenant-a"), http.MethodPost, "/api/admin/v1/resources", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if key != "" {
		request.Header.Set("Idempotency-Key", key)
	}
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, request)
	return recorder
}

//spec:covers REQ-SYSTEM-036: 応答を保存できなくても受付を消さず、同じ要求の再送を 409 で拒否して作用を繰り返さない。
func TestIdempotencyKeepsIndeterminateRequestsReserved(t *testing.T) {
	ledger := idempotency.New(brokenResponseStore{Store: idempotency.NewMemoryStore()}, responseTestCipher{})
	executions := 0
	handler := func(c *echo.Context) error {
		executions++
		return c.JSON(http.StatusCreated, map[string]string{"secret": "once"})
	}
	first := idempotentRequest(t, ledger, `"key"`, "admin", `{}`, handler)
	second := idempotentRequest(t, ledger, `"key"`, "admin", `{}`, handler)
	if first.Code != http.StatusInternalServerError || second.Code != http.StatusConflict || executions != 1 {
		t.Fatalf("first=%d second=%d executions=%d, want 500, 409, 1", first.Code, second.Code, executions)
	}
	if strings.Contains(first.Body.String(), "once") || !strings.Contains(second.Body.String(), "idempotency_key_in_progress") {
		t.Fatalf("first=%s second=%s", first.Body, second.Body)
	}
}

//spec:covers REQ-SYSTEM-036: 認証済みの主体ごとに応答を分離し、500 を含む保存済み応答を作用なしで再生する。
func TestIdempotencyReplaysFailuresAndSeparatesPrincipals(t *testing.T) {
	ledger := idempotency.New(idempotency.NewMemoryStore(), responseTestCipher{})
	executions := 0
	handler := func(c *echo.Context) error { executions++; return errors.New("injected dependency failure") }
	first := idempotentRequest(t, ledger, `"key"`, "admin", `{}`, handler)
	second := idempotentRequest(t, ledger, `"key"`, "admin", `{}`, handler)
	if first.Code != 500 || second.Code != 500 || !strings.Contains(first.Body.String(), "internal_server_error") || !strings.Contains(second.Body.String(), "internal_server_error") || executions != 1 {
		t.Fatalf("first=%d %s second=%d %s executions=%d", first.Code, first.Body, second.Code, second.Body, executions)
	}
	idempotentRequest(t, ledger, `"key"`, "other-admin", `{}`, handler)
	if executions != 2 {
		t.Fatalf("other principal executions=%d, want 2", executions)
	}
}

//spec:covers REQ-SYSTEM-036: 初回が実行中の同じ要求は待たずに 409、異なる要求は 422 で拒否する。
func TestIdempotencyRefusesConcurrentRequests(t *testing.T) {
	ledger := idempotency.New(idempotency.NewMemoryStore(), responseTestCipher{})
	started, finish := make(chan struct{}), make(chan struct{})
	var executions atomic.Int32
	handler := func(c *echo.Context) error {
		executions.Add(1)
		close(started)
		<-finish
		return c.NoContent(http.StatusCreated)
	}
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- idempotentRequest(t, ledger, `"key"`, "admin", `{}`, handler) }()
	<-started
	second := idempotentRequest(t, ledger, `"key"`, "admin", `{}`, handler)
	mismatch := idempotentRequest(t, ledger, `"key"`, "admin", `{"different":true}`, handler)
	close(finish)
	first := <-done
	if first.Code != 201 || second.Code != 409 || mismatch.Code != 422 || executions.Load() != 1 {
		t.Fatalf("first=%d second=%d mismatch=%d executions=%d", first.Code, second.Code, mismatch.Code, executions.Load())
	}
}

//spec:covers REQ-SYSTEM-036: キーの形式と長さの両側を検査し、キーなしの要求は通常どおり実行する。
func TestIdempotencyKeyValidation(t *testing.T) {
	for _, input := range []struct {
		values []string
		want   string
		valid  bool
	}{
		{[]string{`"key"`}, "key", true},
		{[]string{` "a\\b\"c" `}, `a\b"c`, true},
		{[]string{`"` + strings.Repeat("a", 255) + `"`}, strings.Repeat("a", 255), true},
		{[]string{`"` + strings.Repeat("a", 256) + `"`}, "", false},
		{[]string{`""`}, "", false},
		{[]string{"key"}, "", false},
		{[]string{`"one"`, `"two"`}, "", false},
		{[]string{`"one", "two"`}, "", false},
		{[]string{`"a\nb"`}, "", false},
		{[]string{`"あ"`}, "", false},
	} {
		got, valid := parseIdempotencyKey(input.values)
		if got != input.want || valid != input.valid {
			t.Errorf("parse %q = %q %v, want %q %v", input.values, got, valid, input.want, input.valid)
		}
	}
	called := 0
	handler := func(c *echo.Context) error { called++; return c.NoContent(201) }
	for range 2 {
		if response := idempotentRequest(t, nil, "", "admin", `{}`, handler); response.Code != 201 {
			t.Fatal(response.Code)
		}
	}
	if called != 2 {
		t.Fatal(called)
	}
	bad := idempotentRequest(t, nil, "unquoted", "admin", `{}`, handler)
	if bad.Code != 400 || called != 2 {
		t.Fatalf("malformed key status=%d executions=%d", bad.Code, called)
	}
}

//spec:covers REQ-SYSTEM-036: JSON の空白だけを正規化し、キーと配列の順序、数値表記、パス、クエリと CSV の相違を残す。
func TestIdempotencyRequestFingerprint(t *testing.T) {
	fingerprint := func(path, media, body string) string {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		request.Header.Set("Content-Type", media)
		value, err := requestFingerprint(request)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	first := fingerprint("/realms/default/api/admin/v1/resources/a", "application/json", `{"a":1,"b":[2,3]}`)
	if got := fingerprint("/api/admin/v1/resources/a", "application/json", ` { "a": 1, "b": [2,3] } `); got != first {
		t.Fatal("JSON whitespace or tenant prefix changes the fingerprint")
	}
	for _, input := range []struct{ path, media, body string }{
		{"/api/admin/v1/resources/b", "application/json", `{"a":1,"b":[2,3]}`},
		{"/api/admin/v1/resources/a?preview=true", "application/json", `{"a":1,"b":[2,3]}`},
		{"/api/admin/v1/resources/a", "application/json", `{"a":1.0,"b":[2,3]}`},
		{"/api/admin/v1/resources/a", "application/json", `{"a":1,"b":[3,2]}`},
		{"/api/admin/v1/resources/a", "application/json", `{"b":[2,3],"a":1}`},
		{"/api/admin/v1/resources/a", "text/csv", `{"a":1,"b":[2,3]}`},
	} {
		if fingerprint(input.path, input.media, input.body) == first {
			t.Errorf("different request matched: %+v", input)
		}
	}
	if fingerprint("/api/admin/v1/resources/a", "text/csv", "a,b\n1,2\n") == fingerprint("/api/admin/v1/resources/a", "text/csv", "a,b\r\n1,2\r\n") {
		t.Fatal("CSV bytes were normalized")
	}
	for _, bodies := range [][2]string{
		{`{"x":{"a":1},"x":{"b":2}}`, `{"x":{"b":2}}`},
		{`{"name":"a","Name":"b"}`, `{"Name":"b","name":"a"}`},
	} {
		if fingerprint("/api/admin/v1/resources/a", "application/json", bodies[0]) == fingerprint("/api/admin/v1/resources/a", "application/json", bodies[1]) {
			t.Fatal("ambiguous JSON keys were normalized")
		}
	}
	if fingerprint("/api/admin/v1/resources/a", "application/json;broken", `{}`) == fingerprint("/api/admin/v1/resources/a", "application/json;other", `{}`) {
		t.Fatal("different malformed content types were normalized to the same request")
	}
}

//spec:covers REQ-SYSTEM-036: 再生には表現用のヘッダーだけを使い、Problem Details は再送の相関 ID を返す。
func TestIdempotencyResponseHeadersAndCorrelation(t *testing.T) {
	ledger := idempotency.New(idempotency.NewMemoryStore(), responseTestCipher{})
	e := echo.New()
	e.Use(RequestIDMiddleware(true))
	e.POST("/api/admin/v1/resources", func(c *echo.Context) error {
		return WithIdempotency(c, Deps{Idempotency: ledger}, "admin", func(c *echo.Context) error {
			c.Response().Header().Set("Set-Cookie", "do-not-replay=1")
			c.Response().Header().Set("Location", "/resource")
			return WriteProblem(c, 422, "invalid_request", "Invalid resource.")
		})
	})
	for _, id := range []string{"first-request", "retry-request"} {
		request := httptest.NewRequestWithContext(tenantports.WithTenantID(t.Context(), "tenant-a"), http.MethodPost, "/api/admin/v1/resources", strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", `"key"`)
		request.Header.Set(RequestIDHeader, id)
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)
		if response.Code != 422 || response.Header().Get("Location") != "/resource" || response.Header().Get("Cache-Control") != "no-store" || !strings.Contains(response.Body.String(), id) || response.Header().Get(RequestIDHeader) != id {
			t.Fatalf("id=%s response=%d %v %s", id, response.Code, response.Header(), response.Body)
		}
		if id == "retry-request" && response.Header().Get("Set-Cookie") != "" {
			t.Fatal("replay returned Set-Cookie")
		}
	}
}

//spec:covers REQ-SYSTEM-036: ハンドラーの panic の後も受付記録を保持し、同じ要求を実行し直さない。
func TestIdempotencyKeepsPanicOutcomeReserved(t *testing.T) {
	ledger := idempotency.New(idempotency.NewMemoryStore(), responseTestCipher{})
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler(nil, nil)
	e.Use(RecoverMiddleware(nil))
	executions := 0
	e.POST("/api/admin/v1/resources", func(c *echo.Context) error {
		return WithIdempotency(c, Deps{Idempotency: ledger}, "admin", func(_ *echo.Context) error { executions++; panic("injected failure") })
	})
	for _, want := range []int{500, 409} {
		request := httptest.NewRequestWithContext(tenantports.WithTenantID(t.Context(), "tenant-a"), http.MethodPost, "/api/admin/v1/resources", strings.NewReader(`{}`))
		request.Header.Set("Idempotency-Key", `"key"`)
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)
		if response.Code != want {
			t.Fatalf("response=%d %s, want %d", response.Code, response.Body, want)
		}
	}
	if executions != 1 {
		t.Fatal(executions)
	}
}

func FuzzIdempotencyKeyStructuredString(f *testing.F) {
	for _, value := range []string{`"valid"`, `"escaped\\\""`, `""`, "not-quoted", `"a\nb"`, `"a", "b"`} {
		f.Add(value)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		// 任意の入力から有効な sf-string を組み、常時拒否する実装も検出する。
		plain := []byte(raw)
		if len(plain) > 255 {
			plain = plain[:255]
		}
		for i, ch := range plain {
			plain[i] = 0x20 + ch%95
		}
		if len(plain) == 0 {
			plain = []byte("key")
		}
		if got, ok := parseIdempotencyKey([]string{strconv.Quote(string(plain))}); !ok || got != string(plain) {
			t.Fatalf("valid structured string rejected: %q -> %q", plain, got)
		}
		key, valid := parseIdempotencyKey([]string{raw})
		if !valid {
			return
		}
		if key == "" || len(key) > 255 {
			t.Fatalf("accepted key length=%d", len(key))
		}
		for _, ch := range []byte(key) {
			if ch < 0x20 || ch > 0x7e {
				t.Fatalf("accepted non-ASCII key %q", key)
			}
		}
		decoded, ok := parseIdempotencyKey([]string{strconv.Quote(key)})
		if !ok || decoded != key {
			t.Fatalf("structured string round trip: %q -> %q", key, decoded)
		}
	})
}
