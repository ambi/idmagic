package support_http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

func serveWithSecurityHeaders(t *testing.T, cfg SecurityHeadersConfig, h echo.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	e.Use(SecurityHeadersMiddleware(cfg))
	e.GET("/", h)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))
	return rec
}

func TestSecurityHeadersMiddleware_DefaultsAreSecure(t *testing.T) {
	rec := serveWithSecurityHeaders(t, SecurityHeadersConfig{}, func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	h := rec.Header()

	if got := h.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := h.Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("Referrer-Policy = %q, want no-referrer", got)
	}
	if got := h.Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", got)
	}
	csp := h.Get("Content-Security-Policy")
	for _, want := range []string{"default-src 'none'", "base-uri 'none'", "frame-ancestors 'none'", "form-action 'self'"} {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP %q missing %q", csp, want)
		}
	}
	// The base CSP admits no script at all (default-src 'none' denies it); only
	// the auto-POST responses opt a pinned hash into script-src.
	if strings.Contains(csp, "script-src") {
		t.Errorf("base CSP should not carry a script-src: %q", csp)
	}
	if strings.Contains(csp, "unsafe-inline") {
		t.Errorf("CSP must not contain 'unsafe-inline': %q", csp)
	}
	// HSTS is owned by the TLS terminator and off by default (dev http).
	if got := h.Get("Strict-Transport-Security"); got != "" {
		t.Errorf("HSTS should be absent by default, got %q", got)
	}
	if got := h.Get("Content-Security-Policy-Report-Only"); got != "" {
		t.Errorf("report-only header should be absent in enforce mode, got %q", got)
	}
}

func TestSecurityHeadersMiddleware_HSTSWhenEnabled(t *testing.T) {
	rec := serveWithSecurityHeaders(t, SecurityHeadersConfig{HSTSEnabled: true, HSTSMaxAgeSeconds: 31536000, HSTSIncludeSubdomains: true}, func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if got := rec.Header().Get("Strict-Transport-Security"); got != "max-age=31536000; includeSubDomains" {
		t.Errorf("HSTS = %q", got)
	}
}

func TestSecurityHeadersMiddleware_ReportOnlyAndReportURI(t *testing.T) {
	rec := serveWithSecurityHeaders(t, SecurityHeadersConfig{ReportOnly: true, ReportURI: "/csp-report"}, func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	if got := rec.Header().Get("Content-Security-Policy"); got != "" {
		t.Errorf("enforce header should be absent in report-only mode, got %q", got)
	}
	ro := rec.Header().Get("Content-Security-Policy-Report-Only")
	if ro == "" {
		t.Fatal("report-only header missing")
	}
	if !strings.Contains(ro, "report-uri /csp-report") {
		t.Errorf("report-uri missing from %q", ro)
	}
}

func TestSetAutoPostFormCSP_AllowsDestinationAndPinnedScriptHash(t *testing.T) {
	rec := serveWithSecurityHeaders(t, SecurityHeadersConfig{ReportURI: "/csp-report"}, func(c *echo.Context) error {
		SetAutoPostFormCSP(c, "https://sp.example.com/acs?x=1")
		return c.HTML(http.StatusOK, "<html></html>")
	})
	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "form-action https://sp.example.com") {
		t.Errorf("form-action destination missing: %q", csp)
	}
	if strings.Contains(csp, "sp.example.com/acs") {
		t.Errorf("form-action should be an origin, not the full path: %q", csp)
	}
	if !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Errorf("frame-ancestors 'none' must be preserved: %q", csp)
	}
	// The pinned auto-submit script hash is admitted, without 'unsafe-inline'.
	if !strings.Contains(csp, "script-src "+autoSubmitScriptHash) {
		t.Errorf("pinned script hash missing: %q", csp)
	}
	if strings.Contains(csp, "unsafe-inline") {
		t.Errorf("must not use 'unsafe-inline': %q", csp)
	}
	if !strings.Contains(csp, "report-uri /csp-report") {
		t.Errorf("report-uri should carry over to the override: %q", csp)
	}
}

func TestSetAutoPostFormCSP_ReportOnlyOverridesSameHeader(t *testing.T) {
	rec := serveWithSecurityHeaders(t, SecurityHeadersConfig{ReportOnly: true}, func(c *echo.Context) error {
		SetAutoPostFormCSP(c, "https://sp.example.com/acs")
		return c.HTML(http.StatusOK, "<html></html>")
	})
	if got := rec.Header().Get("Content-Security-Policy"); got != "" {
		t.Errorf("enforce header should stay absent in report-only mode, got %q", got)
	}
	if !strings.Contains(rec.Header().Get("Content-Security-Policy-Report-Only"), "form-action https://sp.example.com") {
		t.Error("auto-post CSP should be written to the report-only header")
	}
}

func TestAutoSubmitScriptHash_PinsExactScript(t *testing.T) {
	// Guards against drift: the pinned hash must be the hash of the exact bytes
	// the SAML/WS-Fed templates render inside <script>…</script>.
	if autoSubmitScriptHash != cspSHA256(AutoSubmitScript) {
		t.Fatalf("hash %q does not pin AutoSubmitScript", autoSubmitScriptHash)
	}
	if !strings.HasPrefix(autoSubmitScriptHash, "'sha256-") {
		t.Fatalf("expected a CSP sha256 source expression, got %q", autoSubmitScriptHash)
	}
}

func TestCSPFormActionSource_DegradesToSelf(t *testing.T) {
	for _, bad := range []string{"", "/relative/acs", "://nohost"} {
		if got := cspFormActionSource(bad); got != "'self'" {
			t.Errorf("cspFormActionSource(%q) = %q, want 'self'", bad, got)
		}
	}
	if got := cspFormActionSource("https://sp.example.com:8443/acs"); got != "https://sp.example.com:8443" {
		t.Errorf("origin extraction = %q", got)
	}
}
