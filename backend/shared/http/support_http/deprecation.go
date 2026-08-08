package support_http

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/labstack/echo/v5"
)

var pathParamToken = regexp.MustCompile(`:[^/]+|\{[^/}]+\}`)

type deprecationEntry struct {
	deprecatedSince string
	sunsetAt        string
}

// DeprecationHeadersMiddleware returns an Echo middleware that adds the
// Deprecation (RFC 9745) and, when sunset_at is also set, Sunset (RFC 8594)
// response headers for interfaces the SCL document marks deprecated_since
// (ADR-156). Matching is by HTTP method and a path canonicalized to strip
// both tenant routing styles ("/realms/:tenant_id" prefix) and the "/v1/"
// alias segment, so it applies uniformly regardless of which route the
// request actually hit. A nil scl (e.g. in tests that build a bare Deps{})
// is a no-op.
func DeprecationHeadersMiddleware(scl *spec.SCL) echo.MiddlewareFunc {
	index := buildDeprecationIndex(scl)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if entry, ok := index[deprecationKey(c.Request().Method, c.RouteInfo().Path)]; ok {
				if date, ok := httpDate(entry.deprecatedSince); ok {
					c.Response().Header().Set("Deprecation", date)
				}
				if entry.sunsetAt != "" {
					if date, ok := httpDate(entry.sunsetAt); ok {
						c.Response().Header().Set("Sunset", date)
					}
				}
			}
			return next(c)
		}
	}
}

func buildDeprecationIndex(scl *spec.SCL) map[string]deprecationEntry {
	index := make(map[string]deprecationEntry)
	if scl == nil {
		return index
	}
	for _, iface := range scl.Interfaces {
		if iface.DeprecatedSince == "" {
			continue
		}
		for _, binding := range iface.Bindings {
			if binding.Kind() != "http" {
				continue
			}
			method := binding.String("method")
			path := binding.String("path")
			if method == "" || path == "" {
				continue
			}
			index[deprecationKey(method, path)] = deprecationEntry{
				deprecatedSince: iface.DeprecatedSince,
				sunsetAt:        iface.SunsetAt,
			}
		}
	}
	return index
}

func deprecationKey(method, path string) string {
	return strings.ToUpper(method) + " " + canonicalizeRuntimePath(path)
}

// canonicalizeRuntimePath normalizes an Echo route pattern or SCL binding
// path to a form comparable across tenant routing styles and version
// aliases: path parameters become "*", a leading tenant group prefix is
// stripped, and a "/v1" segment right after a VersionedPrefixes entry is
// stripped.
func canonicalizeRuntimePath(path string) string {
	path = pathParamToken.ReplaceAllString(path, "*")
	switch {
	case path == "/realms/*":
		path = "/"
	case strings.HasPrefix(path, "/realms/*/"):
		path = strings.TrimPrefix(path, "/realms/*")
	case path == "/realms/default":
		path = "/"
	case strings.HasPrefix(path, "/realms/default/"):
		path = strings.TrimPrefix(path, "/realms/default")
	}
	for _, prefix := range VersionedPrefixes {
		if path == prefix+"/v1" {
			return prefix
		}
		if rest, ok := strings.CutPrefix(path, prefix+"/v1/"); ok {
			return prefix + "/" + rest
		}
	}
	return path
}

// httpDate parses value (an SCL "date" or "date-time" field) and formats it
// as an HTTP-date (RFC 7231 IMF-fixdate) for use in the Deprecation/Sunset
// headers.
func httpDate(value string) (string, bool) {
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC().Format(http.TimeFormat), true
		}
	}
	return "", false
}
