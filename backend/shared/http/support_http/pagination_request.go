package support_http

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/labstack/echo/v5"
)

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
	Direction    PageDirection
	Limit        int
}

// TrimPage removes the repository lookahead row and reports which navigation
// directions exist. Backward repositories return canonical order, so their
// farthest lookahead is the first row rather than the last.
func TrimPage[T any](items []T, page PageRequest) ([]T, bool, bool) {
	hasBoundary := page.AfterPrimary != "" || page.AfterID != ""
	hasLookahead := len(items) > page.Limit
	if hasLookahead {
		if page.Direction == PageBackward {
			items = items[1:]
		} else {
			items = items[:page.Limit]
		}
	}
	if page.Direction == PageBackward {
		return items, hasLookahead, hasBoundary
	}
	return items, hasBoundary, hasLookahead
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
	page := PageRequest{Direction: PageForward, Limit: limit}
	cursor := c.QueryParam("cursor")
	if cursor == "" {
		return page, nil
	}
	decoded, err := codec.DecodeCursorForQuery(cursor, tenantID, queryHash)
	if err != nil {
		return PageRequest{}, fmt.Errorf("%w: %w", ErrBadPageRequest, err)
	}
	page.Direction = decoded.Direction
	page.AfterPrimary, page.AfterID, err = splitKeyset(decoded.After)
	if err != nil {
		return PageRequest{}, fmt.Errorf("%w: %w", ErrBadPageRequest, err)
	}
	return page, nil
}

// SetPageLinks signs the first/last row boundaries and emits only the
// directions that exist. A previous cursor reads before the first row; a next
// cursor reads after the last row.
func SetPageLinks(
	c *echo.Context,
	codec *CursorCodec,
	issuerFallback, tenantID, queryHash string,
	firstPrimary, firstID, lastPrimary, lastID string,
	hasPrevious, hasNext bool,
) error {
	previousCursor, err := encodePageCursor(codec, tenantID, queryHash, firstPrimary, firstID, PageBackward, hasPrevious)
	if err != nil {
		return err
	}
	nextCursor, err := encodePageCursor(codec, tenantID, queryHash, lastPrimary, lastID, PageForward, hasNext)
	if err != nil {
		return err
	}
	if link := BuildPageLinks(c, issuerFallback, previousCursor, nextCursor); link != "" {
		c.Response().Header().Set("Link", link)
	}
	return nil
}

func encodePageCursor(codec *CursorCodec, tenantID, queryHash, primary, id string, direction PageDirection, present bool) (string, error) {
	if !present {
		return "", nil
	}
	return codec.Encode(Cursor{
		Version: cursorVersion, TenantID: tenantID, QueryHash: queryHash,
		After: joinKeyset(primary, id), Direction: direction,
	})
}

// SetNextLink is retained for forward-only handlers during migration.
func SetNextLink(c *echo.Context, codec *CursorCodec, issuerFallback, tenantID, queryHash, lastPrimary, lastID string, hasMore bool) error {
	return SetPageLinks(c, codec, issuerFallback, tenantID, queryHash, "", "", lastPrimary, lastID, false, hasMore)
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
