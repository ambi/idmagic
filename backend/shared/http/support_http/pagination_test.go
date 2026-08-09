package support_http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestBuildNextLinkReturnsEmptyWithoutCursor(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/users?limit=50", http.NoBody)
	c := e.NewContext(req, httptest.NewRecorder())
	if got := BuildNextLink(c, "https://example.com", ""); got != "" {
		t.Fatalf("expected empty Link value when there is no next cursor, got %q", got)
	}
}

func TestBuildNextLinkIncludesAbsoluteURLAndRelNext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/users?limit=50", http.NoBody)
	c := e.NewContext(req, httptest.NewRecorder())
	got := BuildNextLink(c, "https://example.com", "opaque-cursor-value")
	want := `<https://example.com/api/admin/v1/users?cursor=opaque-cursor-value&limit=50>; rel="next"`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildNextLinkOverridesExistingCursorParam(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/users?cursor=stale&limit=50", http.NoBody)
	c := e.NewContext(req, httptest.NewRecorder())
	got := BuildNextLink(c, "https://example.com", "fresh-cursor")
	want := `<https://example.com/api/admin/v1/users?cursor=fresh-cursor&limit=50>; rel="next"`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
