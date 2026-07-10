package support

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/ambi/idmagic/backend/shared/logging"

	"github.com/labstack/echo/v5"
)

var hex32 = regexp.MustCompile(`^[0-9a-f]{32}$`)

// serve runs a single request through the middleware chain and returns the
// recorder plus the request_id the handler observed in its context.
func serve(t *testing.T, mw echo.MiddlewareFunc, req *http.Request) (*httptest.ResponseRecorder, string) {
	t.Helper()
	e := echo.New()
	e.Use(mw)
	var seen string
	e.GET("/probe", func(c *echo.Context) error {
		seen = logging.RequestIDFromContext(c.Request().Context())
		return c.NoContent(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec, seen
}

func TestRequestIDMiddleware_GeneratesWhenAbsent(t *testing.T) {
	rec, seen := serve(t, RequestIDMiddleware(false), httptest.NewRequest(http.MethodGet, "/probe", http.NoBody))

	got := rec.Header().Get(RequestIDHeader)
	if !hex32.MatchString(got) {
		t.Fatalf("generated request id = %q, want 32 hex chars", got)
	}
	if seen != got {
		t.Fatalf("handler saw %q, response header %q; want equal", seen, got)
	}
}

func TestRequestIDMiddleware_IgnoresInboundWhenNotTrusted(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/probe", http.NoBody)
	req.Header.Set(RequestIDHeader, "attacker-supplied-id")

	rec, seen := serve(t, RequestIDMiddleware(false), req)

	got := rec.Header().Get(RequestIDHeader)
	if got == "attacker-supplied-id" {
		t.Fatal("untrusted inbound request id was reused (spoofable)")
	}
	if !hex32.MatchString(got) {
		t.Fatalf("request id = %q, want freshly generated 32 hex chars", got)
	}
	if seen != got {
		t.Fatalf("handler saw %q, response header %q; want equal", seen, got)
	}
}

func TestRequestIDMiddleware_ReusesSafeInboundWhenTrusted(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/probe", http.NoBody)
	req.Header.Set(RequestIDHeader, "edge-req-01.abc_DEF")

	rec, seen := serve(t, RequestIDMiddleware(true), req)

	if got := rec.Header().Get(RequestIDHeader); got != "edge-req-01.abc_DEF" {
		t.Fatalf("trusted inbound request id = %q, want reused", got)
	}
	if seen != "edge-req-01.abc_DEF" {
		t.Fatalf("handler saw %q, want reused inbound", seen)
	}
}

func TestRequestIDMiddleware_RegeneratesUnsafeInboundWhenTrusted(t *testing.T) {
	// Even from a trusted hop, a value with injection characters or excess
	// length is rejected and replaced (defense in depth).
	cases := map[string]string{
		"newline injection": "abc\r\ndef",
		"space":             "abc def",
		"too long":          string(make([]byte, maxRequestIDLen+1)),
	}
	for name, inbound := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/probe", http.NoBody)
			req.Header.Set(RequestIDHeader, inbound)

			rec, _ := serve(t, RequestIDMiddleware(true), req)
			if got := rec.Header().Get(RequestIDHeader); !hex32.MatchString(got) {
				t.Fatalf("unsafe inbound %q yielded %q, want freshly generated", inbound, got)
			}
		})
	}
}
