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
// response headers for operations with deprecation metadata. Matching is by
// HTTP method and a path canonicalized to strip
// the tenant routing group prefix ("/realms/:tenant_id"), so it applies
// uniformly regardless of which tenant routing style the request hit. A nil
// contract (e.g. in tests that build a bare Deps{}) is a no-op.
func DeprecationHeadersMiddleware(contract *spec.RuntimeContract) echo.MiddlewareFunc {
	index := buildDeprecationIndex(contract)
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

func buildDeprecationIndex(contract *spec.RuntimeContract) map[string]deprecationEntry {
	index := make(map[string]deprecationEntry)
	if contract == nil {
		return index
	}
	for name, deprecation := range contract.Deprecations {
		if deprecation.Since == "" {
			continue
		}
		operation, ok := contract.Operation(name)
		if !ok {
			continue
		}
		index[deprecationKey(operation.Method, operation.Path)] = deprecationEntry{
			deprecatedSince: deprecation.Since,
			sunsetAt:        deprecation.Sunset,
		}
	}
	return index
}

func deprecationKey(method, path string) string {
	return strings.ToUpper(method) + " " + canonicalizeRuntimePath(path)
}

// canonicalizeRuntimePath normalizes an Echo route pattern or contract path
// path to a form comparable across tenant routing styles: path parameters
// become "*" and a leading tenant group prefix is stripped.
func canonicalizeRuntimePath(path string) string {
	path = pathParamToken.ReplaceAllString(path, "*")
	switch {
	case path == "/realms/*":
		return "/"
	case strings.HasPrefix(path, "/realms/*/"):
		return strings.TrimPrefix(path, "/realms/*")
	case path == "/realms/default":
		return "/"
	case strings.HasPrefix(path, "/realms/default/"):
		return strings.TrimPrefix(path, "/realms/default")
	}
	return path
}

// httpDate parses a date or date-time value and formats it
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
