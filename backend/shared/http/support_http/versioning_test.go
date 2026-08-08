package support_http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

// ADR-156 / wi-297 T004: current management API paths are the v1 alias.
// RegisterVersionAliases must mirror every route under a versioned prefix at
// its "<prefix>/v1/..." path without requiring individual handlers to
// register twice, and must leave unversioned prefixes untouched.
func TestRegisterVersionAliases_MirrorsAdminAndAccountPrefixes(t *testing.T) {
	e := echo.New()
	RegisterVersionAliases(e)
	e.GET("/api/admin/widgets", func(c *echo.Context) error {
		return c.String(http.StatusOK, "admin-widgets")
	})
	e.GET("/api/account/profile", func(c *echo.Context) error {
		return c.String(http.StatusOK, "account-profile")
	})
	e.GET("/api/auth/login", func(c *echo.Context) error {
		return c.String(http.StatusOK, "auth-login")
	})

	cases := []struct {
		path       string
		wantStatus int
		wantBody   string
	}{
		{"/api/admin/widgets", http.StatusOK, "admin-widgets"},
		{"/api/admin/v1/widgets", http.StatusOK, "admin-widgets"},
		{"/api/account/profile", http.StatusOK, "account-profile"},
		{"/api/account/v1/profile", http.StatusOK, "account-profile"},
		{"/api/auth/login", http.StatusOK, "auth-login"},
		{"/api/auth/v1/login", http.StatusNotFound, ""},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, http.NoBody))
		if rec.Code != tc.wantStatus {
			t.Errorf("%s: status = %d, want %d", tc.path, rec.Code, tc.wantStatus)
		}
		if tc.wantStatus == http.StatusOK && rec.Body.String() != tc.wantBody {
			t.Errorf("%s: body = %q, want %q", tc.path, rec.Body.String(), tc.wantBody)
		}
	}
}

// A route registered directly at "<prefix>/v1/..." must not be re-aliased
// into "<prefix>/v1/v1/...".
func TestRegisterVersionAliases_DoesNotDoubleAliasVersionedPaths(t *testing.T) {
	e := echo.New()
	RegisterVersionAliases(e)
	e.GET("/api/admin/v1/widgets", func(c *echo.Context) error {
		return c.String(http.StatusOK, "explicit-v1")
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/admin/v1/v1/widgets", http.NoBody))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (must not double-alias)", rec.Code, http.StatusNotFound)
	}
}

// Tenant path-style routing prefixes routes with /realms/:tenant_id before
// the OnAddRoute hook observes them; the alias must still be inserted after
// the tenant segment, not before it.
func TestRegisterVersionAliases_HonorsGroupPrefix(t *testing.T) {
	e := echo.New()
	RegisterVersionAliases(e)
	g := e.Group("/realms/:tenant_id")
	g.GET("/api/admin/widgets", func(c *echo.Context) error {
		return c.String(http.StatusOK, "tenant-admin-widgets")
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/admin/v1/widgets", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "tenant-admin-widgets" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "tenant-admin-widgets")
	}
}
