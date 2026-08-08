package support_http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/labstack/echo/v5"
)

// ADR-156 / wi-297 T005: interfaces marked deprecated_since must return the
// Deprecation header (RFC 9745), and Sunset (RFC 8594) when sunset_at is
// also set. Non-deprecated interfaces must not gain either header.
func TestDeprecationHeadersMiddleware(t *testing.T) {
	scl := &spec.SCL{
		Interfaces: map[string]spec.Interface{
			"OldWidgets": {
				Stability:       "stable",
				DeprecatedSince: "2026-01-01",
				SunsetAt:        "2027-01-01",
				Successor:       "NewWidgets",
				Bindings: []spec.Binding{
					{"kind": "http", "method": "GET", "path": "/api/admin/old-widgets"},
				},
			},
			"NewWidgets": {
				Stability: "stable",
				Bindings: []spec.Binding{
					{"kind": "http", "method": "GET", "path": "/api/admin/new-widgets"},
				},
			},
		},
	}

	e := echo.New()
	RegisterVersionAliases(e)
	e.Use(DeprecationHeadersMiddleware(scl))
	tenantGroup := e.Group("/realms/:tenant_id")
	tenantGroup.GET("/api/admin/old-widgets", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.GET("/api/admin/new-widgets", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	cases := []struct {
		name           string
		path           string
		wantDeprecated string
		wantSunset     string
	}{
		{"canonical path", "/realms/acme/api/admin/old-widgets", "Thu, 01 Jan 2026 00:00:00 GMT", "Fri, 01 Jan 2027 00:00:00 GMT"},
		{"v1 alias path", "/realms/acme/api/admin/v1/old-widgets", "Thu, 01 Jan 2026 00:00:00 GMT", "Fri, 01 Jan 2027 00:00:00 GMT"},
		{"non-deprecated interface", "/api/admin/new-widgets", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, http.NoBody))
			if got := rec.Header().Get("Deprecation"); got != tc.wantDeprecated {
				t.Errorf("Deprecation = %q, want %q", got, tc.wantDeprecated)
			}
			if got := rec.Header().Get("Sunset"); got != tc.wantSunset {
				t.Errorf("Sunset = %q, want %q", got, tc.wantSunset)
			}
		})
	}
}

func TestDeprecationHeadersMiddleware_NilSCLIsNoop(t *testing.T) {
	e := echo.New()
	e.Use(DeprecationHeadersMiddleware(nil))
	e.GET("/api/admin/widgets", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/admin/widgets", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Deprecation"); got != "" {
		t.Errorf("Deprecation = %q, want empty", got)
	}
}
