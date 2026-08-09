package support_http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func newLimitCtx(t *testing.T, query string) *echo.Context {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/users?"+query, http.NoBody)
	return e.NewContext(req, httptest.NewRecorder())
}

func TestParseLimitDefaultsWhenAbsent(t *testing.T) {
	c := newLimitCtx(t, "")
	got, err := ParseLimit(c, 50, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 50 {
		t.Fatalf("got %d, want 50", got)
	}
}

func TestParseLimitClampsToMax(t *testing.T) {
	c := newLimitCtx(t, "limit=10000")
	got, err := ParseLimit(c, 50, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 200 {
		t.Fatalf("got %d, want 200 (clamped)", got)
	}
}

func TestParseLimitUsesExplicitValue(t *testing.T) {
	c := newLimitCtx(t, "limit=25")
	got, err := ParseLimit(c, 50, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 25 {
		t.Fatalf("got %d, want 25", got)
	}
}

func TestParseLimitRejectsNonPositive(t *testing.T) {
	for _, raw := range []string{"0", "-1", "not-a-number"} {
		c := newLimitCtx(t, "limit="+raw)
		if _, err := ParseLimit(c, 50, 200); err == nil {
			t.Fatalf("expected error for limit=%q", raw)
		}
	}
}
