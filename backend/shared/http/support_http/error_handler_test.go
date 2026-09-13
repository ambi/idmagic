package support_http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"unicode"

	"github.com/labstack/echo/v5"
)

func TestErrorHandler_FallbackWritesProblemDetailsForHTTPError(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler(nil, nil)
	e.GET("/probe", func(c *echo.Context) error {
		return echo.NewHTTPError(http.StatusNotFound, "no such widget")
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe", http.NoBody))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if ct := rec.Header().Get("Content-Type"); ct != ProblemContentType {
		t.Fatalf("Content-Type = %q, want %q", ct, ProblemContentType)
	}
	var body Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v (body=%s)", err, rec.Body.String())
	}
	if body.Status != http.StatusNotFound {
		t.Errorf("status field = %d, want %d", body.Status, http.StatusNotFound)
	}
	if body.Detail != "no such widget" {
		t.Errorf("detail = %q, want %q", body.Detail, "no such widget")
	}
	if body.Type != "urn:idmagic:error:not_found" {
		t.Errorf("type = %q, want %q", body.Type, "urn:idmagic:error:not_found")
	}
}

func TestErrorHandler_FallbackWritesProblemDetailsForPlainError(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler(nil, nil)
	e.GET("/probe", func(c *echo.Context) error {
		return errors.New("boom")
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe", http.NoBody))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if ct := rec.Header().Get("Content-Type"); ct != ProblemContentType {
		t.Fatalf("Content-Type = %q, want %q", ct, ProblemContentType)
	}
	var body Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v (body=%s)", err, rec.Body.String())
	}
	if body.Detail != http.StatusText(http.StatusInternalServerError) {
		t.Errorf("detail = %q, want status text %q", body.Detail, http.StatusText(http.StatusInternalServerError))
	}
}

func TestErrorHandler_QuotaExceededWritesProblemDetails422(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler(nil, nil)
	e.GET("/probe", func(c *echo.Context) error {
		return &fakeQuotaExceeded{resource: "users", tenantID: "tenant-1"}
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe", http.NoBody))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	if ct := rec.Header().Get("Content-Type"); ct != ProblemContentType {
		t.Fatalf("Content-Type = %q, want %q", ct, ProblemContentType)
	}
	var body Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v (body=%s)", err, rec.Body.String())
	}
	if body.Type != "urn:idmagic:error:quota_exceeded" {
		t.Errorf("type = %q, want %q", body.Type, "urn:idmagic:error:quota_exceeded")
	}
}

// 未知の内部エラーは、ステータスとエラーコードを保ったまま英語の本文を返す。「英語である」
// ことを機械で読む方法は 1 つしかない。人が読む文が非 ASCII を含まないことである。
//
// 元のエラー文を日本語で与えているのは、本文がそれを写していないことを同時に読むためである。
// 写す実装は、ロケールに関係なく利用者へ内部の文を出し、UI の辞書からは直せない。
// UI が翻訳できるのは stable なエラーコードだけであり (REQ-SYSTEM-011)、`detail` は
// そのまま表示される。
//
//spec:covers EX-SYSTEM-013-01: 未知の内部エラーがステータスとエラーコードを保ち、元のエラー文を写さずに英語の本文を返すこと。
func TestErrorHandlerKeepsUnknownErrorTextEnglish(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler(nil, nil)
	e.GET("/probe", func(c *echo.Context) error {
		return errors.New("予期しない内部エラー")
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe", http.NoBody))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	var body Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v (body=%s)", err, rec.Body.String())
	}
	if body.Type != "urn:idmagic:error:internal_server_error" {
		t.Errorf("type = %q, want %q", body.Type, "urn:idmagic:error:internal_server_error")
	}
	if body.Status != http.StatusInternalServerError {
		t.Errorf("status field = %d, want %d", body.Status, http.StatusInternalServerError)
	}
	for field, value := range map[string]string{"title": body.Title, "detail": body.Detail} {
		if value == "" {
			t.Errorf("%s is empty; there is no human-readable text to be in English", field)
			continue
		}
		if !isASCII(value) {
			t.Errorf("%s = %q, want English text with no non-ASCII rune", field, value)
		}
	}
}

// isASCII は、人が読む文が非 ASCII を含まないことを判定する。
func isASCII(value string) bool {
	for _, r := range value {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

type fakeQuotaExceeded struct {
	resource string
	tenantID string
}

func (e *fakeQuotaExceeded) Error() string         { return "quota exceeded for " + e.resource }
func (e *fakeQuotaExceeded) IsQuotaExceeded() bool { return true }
func (e *fakeQuotaExceeded) GetResource() string   { return e.resource }
func (e *fakeQuotaExceeded) GetTenantID() string   { return e.tenantID }
