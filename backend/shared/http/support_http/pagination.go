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

// BuildNextLink builds the RFC 8288 Link response header value (rel="next")
// for a keyset-paginated list response (ADR-158). It returns "" when there is
// no next page (nextCursor == ""), signaling the caller not to set the header
// at all — the absence of a Link header marks the last page.
//
// The next URL reuses the current request's query parameters (limit, sort,
// filter, ...) so a client that just follows the URL never needs to
// reconstruct them, and only the cursor is replaced.
func BuildNextLink(c *echo.Context, issuerFallback, nextCursor string) string {
	if nextCursor == "" {
		return ""
	}
	q := url.Values{}
	for k, vs := range c.Request().URL.Query() {
		q[k] = append([]string(nil), vs...)
	}
	q.Set("cursor", nextCursor)
	// path style では RequestIssuer 自体が /realms/{realm} を含むため、request path
	// からもテナント prefix を落としてから継ぐ (TenantURL の doc 参照、二重 prefix 防止)。
	path := strings.TrimPrefix(c.Request().URL.Path, tenancy.URLPrefix(c.Request().Context()))
	next := TenantURL(c, path, issuerFallback) + "?" + q.Encode()
	return `<` + next + `>; rel="next"`
}
