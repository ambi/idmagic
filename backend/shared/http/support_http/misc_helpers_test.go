package support_http

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	consentusecases "github.com/ambi/idmagic/backend/oauth2/consent/usecases"

	"github.com/labstack/echo/v5"
)

func TestWriteConsentError(t *testing.T) {
	t.Run("maps ErrConsentNotFound to 404", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/x", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		d := Deps{}
		if err := d.WriteConsentError(c, consentusecases.ErrConsentNotFound); err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status=%d", rec.Code)
		}
	})

	t.Run("passes through an unmapped error", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/x", http.NoBody)
		c := e.NewContext(req, httptest.NewRecorder())
		d := Deps{}
		other := errors.New("boom")
		if err := d.WriteConsentError(c, other); !errors.Is(err, other) {
			t.Fatalf("err=%v, want passthrough", err)
		}
	})
}

func TestWriteServerError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := WriteServerError(c, errors.New("db exploded")); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "internal_server_error") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestDecodeJSON(t *testing.T) {
	t.Run("decodes a valid body", func(t *testing.T) {
		var dest struct{ Name string }
		req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString(`{"Name":"alice"}`))
		if err := DecodeJSON(req, &dest); err != nil {
			t.Fatal(err)
		}
		if dest.Name != "alice" {
			t.Fatalf("dest=%+v", dest)
		}
	})

	t.Run("rejects unknown fields", func(t *testing.T) {
		var dest struct{ Name string }
		req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString(`{"Name":"alice","Extra":1}`))
		if err := DecodeJSON(req, &dest); err == nil {
			t.Fatal("expected error for an unknown field")
		}
	})

	t.Run("rejects a body over the size limit", func(t *testing.T) {
		var dest struct{ Name string }
		huge := `{"Name":"` + strings.Repeat("a", 70<<10) + `"}`
		req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString(huge))
		if err := DecodeJSON(req, &dest); err == nil {
			t.Fatal("expected error for an oversized body")
		}
	})
}

func TestLoggingMiddleware(t *testing.T) {
	cases := []struct {
		name   string
		status int
	}{
		{"2xx", http.StatusOK},
		{"400", http.StatusBadRequest},
		{"4xx non-400", http.StatusForbidden},
		{"5xx", http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			e.Use(LoggingMiddleware())
			e.GET("/x", func(c *echo.Context) error {
				return c.JSON(tc.status, map[string]string{"message": "because reasons"})
			})
			req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status=%d, want %d", rec.Code, tc.status)
			}
		})
	}

	t.Run("infers status from a returned handler error", func(t *testing.T) {
		e := echo.New()
		e.Use(LoggingMiddleware())
		e.GET("/x", func(*echo.Context) error {
			return echo.NewHTTPError(http.StatusTeapot, "nope")
		})
		req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusTeapot {
			t.Fatalf("status=%d", rec.Code)
		}
	})
}

func TestExtractClientIP(t *testing.T) {
	t.Run("nil request or non-positive trusted hops returns empty", func(t *testing.T) {
		if got := ExtractClientIP(nil, 1); got != "" {
			t.Fatalf("got=%q", got)
		}
		req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
		req.Header.Set("X-Forwarded-For", "1.1.1.1")
		if got := ExtractClientIP(req, 0); got != "" {
			t.Fatalf("got=%q", got)
		}
	})

	t.Run("picks the client IP the configured number of hops back", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
		req.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.2, 10.0.0.1")
		if got := ExtractClientIP(req, 2); got != "203.0.113.5" {
			t.Fatalf("got=%q", got)
		}
	})

	t.Run("returns empty when there are not enough hops", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
		req.Header.Set("X-Forwarded-For", "10.0.0.2, 10.0.0.1")
		if got := ExtractClientIP(req, 5); got != "" {
			t.Fatalf("got=%q", got)
		}
	})
}

func TestApplicationGateClientIP(t *testing.T) {
	g := &ApplicationGate{GateTrustedForwardedHops: 1}
	req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.1")
	if got := g.ClientIP(req); got != "203.0.113.5" {
		t.Fatalf("got=%q", got)
	}

	zero := &ApplicationGate{}
	if got := zero.ClientIP(req); got != "" {
		t.Fatalf("got=%q, want empty with GateTrustedForwardedHops=0", got)
	}
	if got := g.ClientIP(nil); got != "" {
		t.Fatalf("got=%q, want empty for a nil request", got)
	}
}

func TestIsAdminAPIPath(t *testing.T) {
	if !IsAdminAPIPath("/api/admin/v1/users") {
		t.Fatal("expected admin API path to match")
	}
	if IsAdminAPIPath("/api/account/v1/profile") {
		t.Fatal("expected non-admin path to not match")
	}
}

func TestDetachedCompletionContext(t *testing.T) {
	t.Run("nil request context falls back to Background", func(t *testing.T) {
		d := Deps{}
		ctx, cancel := d.DetachedCompletionContext(nil) //nolint:staticcheck // exercising the nil-context fallback
		defer cancel()
		if ctx == nil {
			t.Fatal("expected a non-nil context")
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("expected a deadline to be set")
		}
	})

	t.Run("uses the configured timeout", func(t *testing.T) {
		d := Deps{DetachedCompletionTimeout: 0}
		ctx, cancel := d.DetachedCompletionContext(context.Background())
		defer cancel()
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("expected the default timeout to apply")
		}
	})

	t.Run("survives cancellation of the parent context", func(t *testing.T) {
		d := Deps{}
		parent, parentCancel := context.WithCancel(context.Background())
		parentCancel()
		ctx, cancel := d.DetachedCompletionContext(parent)
		defer cancel()
		if err := ctx.Err(); err != nil {
			t.Fatalf("expected the detached context to survive parent cancellation, err=%v", err)
		}
	})
}

func TestRecordDetachedCompletionFailureNilSafe(t *testing.T) {
	d := Deps{}
	d.RecordDetachedCompletionFailure() // must not panic with no AbortMetrics wired
}

func TestInsufficientScopeAndInvalidTokenErrorMessages(t *testing.T) {
	scopeErr := &InsufficientScopeError{Required: "admin:read"}
	if scopeErr.Error() != "insufficient scope: admin:read" {
		t.Fatalf("Error()=%q", scopeErr.Error())
	}
	tokenErr := &InvalidTokenError{}
	if tokenErr.Error() != "invalid access token" {
		t.Fatalf("Error()=%q", tokenErr.Error())
	}
}
