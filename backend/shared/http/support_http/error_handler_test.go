package support_http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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

type fakeQuotaExceeded struct {
	resource string
	tenantID string
}

func (e *fakeQuotaExceeded) Error() string         { return "quota exceeded for " + e.resource }
func (e *fakeQuotaExceeded) IsQuotaExceeded() bool { return true }
func (e *fakeQuotaExceeded) GetResource() string   { return e.resource }
func (e *fakeQuotaExceeded) GetTenantID() string   { return e.tenantID }
