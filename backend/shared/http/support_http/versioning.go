package support_http

import (
	"strings"

	"github.com/labstack/echo/v5"
)

// VersionedPrefixes are the management API path prefixes that carry a "v1"
// alias (ADR-156). The current unversioned path is the v1 route; a future
// breaking change is introduced as a new "/v2/" prefix, never by mutating
// these paths.
var VersionedPrefixes = []string{"/api/admin", "/api/account"}

// RegisterVersionAliases installs an Echo.OnAddRoute hook that mirrors every
// route registered under a VersionedPrefixes entry at its "<prefix>/v1/..."
// path, so individual handlers keep registering their existing unversioned
// path exactly once and gain a versioned alias for free (ADR-156, wi-297
// T004). Call it before any routes are registered.
func RegisterVersionAliases(e *echo.Echo) {
	e.OnAddRoute = func(route echo.Route) error {
		aliasPath, ok := versionAliasPath(route.Path)
		if !ok {
			return nil
		}
		alias := route
		alias.Path = aliasPath
		alias.Name = ""
		_, err := e.AddRoute(alias)
		return err
	}
}

// versionAliasPath returns the "v1" alias for path if a VersionedPrefixes
// entry occurs anywhere in path (e.g. after a "/realms/:tenant_id" group
// prefix) and isn't already followed by a version segment.
func versionAliasPath(path string) (string, bool) {
	for _, prefix := range VersionedPrefixes {
		before, rest, found := strings.Cut(path, prefix)
		if !found {
			continue
		}
		if rest != "" && rest[0] != '/' {
			continue // e.g. prefix "/api/admin" must not match "/api/administrators"
		}
		if segment, _, _ := strings.Cut(strings.TrimPrefix(rest, "/"), "/"); segment == "v1" {
			return "", false
		}
		return before + prefix + "/v1" + rest, true
	}
	return "", false
}
