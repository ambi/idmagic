package support

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/labstack/echo/v5"
)

// AutoSubmitScript is the exact inline script body used by the SAML / WS-Fed
// auto-POST forms. It is fixed, so its CSP hash (autoSubmitScriptHash) can pin
// it in a strict script-src without 'unsafe-inline' or a per-request nonce. Any
// handler that renders this script must emit exactly these bytes.
const AutoSubmitScript = "document.forms[0].submit()"

// autoSubmitScriptHash is the CSP source expression ('sha256-…') that admits
// exactly AutoSubmitScript. Because the script is a fixed literal, a static hash
// is as strict as a nonce here while needing no per-request state.
var autoSubmitScriptHash = cspSHA256(AutoSubmitScript)

// SecurityHeadersConfig configures the security response headers applied to every
// backend response (SecurityResponseHeaders / FrameAncestorsPolicy objectives).
// Zero value is a safe enforce-mode policy with HSTS disabled (the dev-http
// default); operators opt into HSTS and report-only via env at the boundary.
type SecurityHeadersConfig struct {
	// ReportOnly emits Content-Security-Policy-Report-Only instead of the
	// enforcing header, so a strict policy can be rolled out by first observing
	// violations before enforcing.
	ReportOnly bool
	// ReportURI, when set, is added to the CSP as report-uri so violation
	// reports are collected during staged rollout.
	ReportURI string
	// HSTSEnabled gates Strict-Transport-Security. HSTS is owned by the TLS
	// terminator, so it stays off by default (dev http) and an operator turns it
	// on only when TLS is terminated at or ahead of this hop, or leaves it to an
	// edge proxy.
	HSTSEnabled           bool
	HSTSMaxAgeSeconds     int
	HSTSIncludeSubdomains bool
}

// cspContextKey carries the resolved CSP header name and report-uri so a handler
// rendering an auto-submitting cross-origin POST form can override the response
// CSP (SetAutoPostFormCSP) without re-deriving the middleware's configuration.
type cspContextKey struct{}

type cspContext struct {
	headerName string
	reportURI  string
}

// SecurityHeadersMiddleware applies the security response headers to every
// backend response (SecurityResponseHeaders / FrameAncestorsPolicy). It is
// secure-by-default: it pins default-src / base-uri to 'none', forbids framing
// (frame-ancestors 'none' + X-Frame-Options: DENY), constrains form-action to
// 'self', stops MIME sniffing (nosniff) and referrer leakage (no-referrer). The
// base CSP has no script-src, so default-src 'none' denies all script. HSTS is
// emitted only when explicitly enabled because the TLS terminator owns it.
//
// The resolved CSP header name and report-uri are stored in the request context
// so a handler rendering an auto-submitting cross-origin POST form can override
// the CSP with a form-action allowance and the pinned auto-submit script hash
// (SetAutoPostFormCSP).
func SecurityHeadersMiddleware(cfg SecurityHeadersConfig) echo.MiddlewareFunc {
	headerName := "Content-Security-Policy"
	if cfg.ReportOnly {
		headerName = "Content-Security-Policy-Report-Only"
	}
	reportURI := strings.TrimSpace(cfg.ReportURI)
	baseCSP := buildCSP("form-action 'self'", "", reportURI)
	hsts := cfg.hstsValue()
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()
			ctx := context.WithValue(req.Context(), cspContextKey{}, cspContext{headerName: headerName, reportURI: reportURI})
			c.SetRequest(req.WithContext(ctx))

			h := c.Response().Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("X-Frame-Options", "DENY")
			h.Set(headerName, baseCSP)
			if hsts != "" {
				h.Set("Strict-Transport-Security", hsts)
			}
			return next(c)
		}
	}
}

// SetAutoPostFormCSP overrides the response CSP for an auto-submitting POST form
// that targets a cross-origin action URL (SAML ACS / WS-Fed ReplyURL). It keeps
// frame-ancestors 'none' (no clickjacking of the auto-submit) while allowing the
// specific destination in form-action and the pinned auto-submit script hash in
// script-src. It writes the same header name (enforce or report-only) the
// middleware chose.
func SetAutoPostFormCSP(c *echo.Context, actionURL string) {
	st, _ := c.Request().Context().Value(cspContextKey{}).(cspContext)
	headerName := st.headerName
	if headerName == "" {
		// Middleware inactive (e.g. in isolated tests): fall back to enforce.
		headerName = "Content-Security-Policy"
	}
	policy := buildCSP("form-action "+cspFormActionSource(actionURL), "script-src "+autoSubmitScriptHash, st.reportURI)
	c.Response().Header().Set(headerName, policy)
}

// buildCSP assembles the policy from the always-on secure base plus the given
// form-action and optional script-src directives, appending report-uri last.
func buildCSP(formAction, scriptSrc, reportURI string) string {
	directives := []string{
		"default-src 'none'",
		"base-uri 'none'",
		"frame-ancestors 'none'",
		formAction,
	}
	if scriptSrc != "" {
		directives = append(directives, scriptSrc)
	}
	if reportURI != "" {
		directives = append(directives, "report-uri "+reportURI)
	}
	return strings.Join(directives, "; ")
}

// hstsValue renders the Strict-Transport-Security value, or "" when HSTS is
// disabled so the header is omitted entirely.
func (cfg SecurityHeadersConfig) hstsValue() string {
	if !cfg.HSTSEnabled {
		return ""
	}
	maxAge := cfg.HSTSMaxAgeSeconds
	if maxAge <= 0 {
		maxAge = 31536000
	}
	v := fmt.Sprintf("max-age=%d", maxAge)
	if cfg.HSTSIncludeSubdomains {
		v += "; includeSubDomains"
	}
	return v
}

// cspFormActionSource reduces a full action URL to a CSP source expression
// (scheme://host[:port]); a malformed or relative value degrades to 'self'
// rather than widening the policy.
func cspFormActionSource(actionURL string) string {
	u, err := url.Parse(strings.TrimSpace(actionURL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "'self'"
	}
	return u.Scheme + "://" + u.Host
}

// cspSHA256 returns the CSP source expression ('sha256-…' base64) for an inline
// script body.
func cspSHA256(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}
