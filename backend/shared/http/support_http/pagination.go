package support_http

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/labstack/echo/v5"
)

// ParseLimit parses the "limit" query parameter for a keyset-paginated list
// endpoint (ADR-158): absent falls back to def, present must be a positive
// integer (else an error the caller maps to InvalidRequestError), clamped to
// max rather than rejected so a too-large limit degrades gracefully.
func ParseLimit(c *echo.Context, def, maxLimit int) (int, error) {
	raw := c.QueryParam("limit")
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, errors.New("limit must be a positive integer")
	}
	if n > maxLimit {
		n = maxLimit
	}
	return n, nil
}

// BuildPageLinks builds RFC 8288 rel="prev" / rel="next" links for the
// directions that exist. It returns an empty value when neither direction is
// available.
//
// Each URL reuses the current request's query parameters (limit, sort,
// filter, ...) so a client that just follows the URL never needs to
// reconstruct them, and only the cursor is replaced.
func BuildPageLinks(c *echo.Context, issuerFallback, previousCursor, nextCursor string) string {
	links := make([]string, 0, 2)
	if previousCursor != "" {
		links = append(links, buildPageLink(c, issuerFallback, previousCursor, "prev"))
	}
	if nextCursor != "" {
		links = append(links, buildPageLink(c, issuerFallback, nextCursor, "next"))
	}
	return strings.Join(links, ", ")
}

// BuildNextLink is retained for forward-only callers during the bidirectional
// migration. New handlers should use BuildPageLinks.
func BuildNextLink(c *echo.Context, issuerFallback, nextCursor string) string {
	return BuildPageLinks(c, issuerFallback, "", nextCursor)
}

func buildPageLink(c *echo.Context, issuerFallback, cursor, rel string) string {
	q := url.Values{}
	for k, vs := range c.Request().URL.Query() {
		q[k] = append([]string(nil), vs...)
	}
	q.Set("cursor", cursor)
	// path style では RequestIssuer 自体が /realms/{realm} を含むため、request path
	// からもテナント prefix を落としてから継ぐ (TenantURL の doc 参照、二重 prefix 防止)。
	path := strings.TrimPrefix(c.Request().URL.Path, tenancy.URLPrefix(c.Request().Context()))
	pageURL := TenantURL(c, path, issuerFallback) + "?" + q.Encode()
	return `<` + pageURL + `>; rel="` + rel + `"`
}
