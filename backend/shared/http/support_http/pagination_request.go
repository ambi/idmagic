package support_http

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
)

// cursorTTL is how long an issued pagination cursor stays valid (ADR-158).
const cursorTTL = 1 * time.Hour

// ErrBadPageRequest wraps any page-request validation failure (invalid
// limit, invalid/expired/tenant-or-query-mismatched cursor) that a handler
// should map to 400 InvalidRequestError, as opposed to a downstream
// repository error. Check with errors.Is.
var ErrBadPageRequest = errors.New("support_http: invalid page request")

// PageRequest is a parsed, validated cursor+limit ready to hand to a
// ports.*Repository.ListPage-shaped method.
type PageRequest struct {
	// AfterPrimary/AfterID are the decoded keyset ("", "" for the first
	// page): the (primary sort field, id tie-break) of the last row the
	// caller already saw.
	AfterPrimary string
	AfterID      string
	Limit        int
}

// ParsePageRequest parses "limit" (ParseLimit) and "cursor" for a keyset-
// paginated list endpoint (ADR-158). A cursor is verified against tenantID
// and queryHash (the caller's fingerprint of the interface + any sort/filter
// params) so a cursor cannot be replayed against a different tenant or a
// since-changed query, then split into its (primary, id) keyset. Every
// failure wraps ErrBadPageRequest.
func ParsePageRequest(c *echo.Context, codec *CursorCodec, tenantID, queryHash string, defLimit, maxLimit int) (PageRequest, error) {
	limit, err := ParseLimit(c, defLimit, maxLimit)
	if err != nil {
		return PageRequest{}, fmt.Errorf("%w: %w", ErrBadPageRequest, err)
	}
	page := PageRequest{Limit: limit}
	cursor := c.QueryParam("cursor")
	if cursor == "" {
		return page, nil
	}
	after, err := codec.DecodeForQuery(cursor, tenantID, queryHash)
	if err != nil {
		return PageRequest{}, fmt.Errorf("%w: %w", ErrBadPageRequest, err)
	}
	page.AfterPrimary, page.AfterID, err = splitKeyset(after)
	if err != nil {
		return PageRequest{}, fmt.Errorf("%w: %w", ErrBadPageRequest, err)
	}
	return page, nil
}

// SetNextLink encodes a fresh cursor for the (primary, id) keyset of the last
// row the caller saw and sets the Link response header (rel="next") via
// BuildNextLink — but only when hasMore is true; otherwise it is a no-op
// (the absence of a Link header marks the last page).
func SetNextLink(c *echo.Context, codec *CursorCodec, issuerFallback, tenantID, queryHash, lastPrimary, lastID string, hasMore bool) error {
	if !hasMore {
		return nil
	}
	nextCursor, err := codec.Encode(Cursor{
		TenantID: tenantID, QueryHash: queryHash,
		After: joinKeyset(lastPrimary, lastID), ExpiresAt: time.Now().Add(cursorTTL),
	})
	if err != nil {
		return err
	}
	if link := BuildNextLink(c, issuerFallback, nextCursor); link != "" {
		c.Response().Header().Set("Link", link)
	}
	return nil
}

// joinKeyset/splitKeyset encode a (primary, id) keyset into Cursor.After as
// one opaque string. primary is base64url-encoded so it can contain the "."
// separator safely (raw url base64 never produces "."), whatever characters
// the underlying sort field allows.
func joinKeyset(primary, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(primary)) + "." + id
}

func splitKeyset(after string) (primary, id string, err error) {
	encodedPrimary, id, ok := strings.Cut(after, ".")
	if !ok {
		return "", "", errors.New("malformed keyset")
	}
	raw, err := base64.RawURLEncoding.DecodeString(encodedPrimary)
	if err != nil {
		return "", "", err
	}
	return string(raw), id, nil
}
