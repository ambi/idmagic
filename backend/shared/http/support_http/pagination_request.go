package support_http

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
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
	Anchor       PageAnchor
	Limit        int
	CurrentPage  int
	LegacyCursor bool
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
// paginated list endpoint. A cursor is verified against tenantID
// and queryHash (the caller's fingerprint of the interface + any sort/filter
// params) so a cursor cannot be replayed against a different tenant or a
// since-changed query, then split into its (primary, id) keyset. Every
// failure wraps ErrBadPageRequest.
func ParsePageRequest(c *echo.Context, codec *CursorCodec, tenantID, queryHash string, defLimit, maxLimit int) (PageRequest, error) {
	limit, err := ParseLimit(c, defLimit, maxLimit)
	if err != nil {
		return PageRequest{}, fmt.Errorf("%w: %w", ErrBadPageRequest, err)
	}
	page := PageRequest{Direction: PageForward, Anchor: PageAnchorKeyset, Limit: limit, CurrentPage: 1}
	cursor := c.QueryParam("cursor")
	if cursor == "" {
		return page, nil
	}
	decoded, err := codec.DecodeCursorForQuery(cursor, tenantID, queryHash)
	if err != nil {
		return PageRequest{}, fmt.Errorf("%w: %w", ErrBadPageRequest, err)
	}
	page.Direction = decoded.Direction
	page.Anchor = decoded.Anchor
	page.CurrentPage = decoded.Page
	if decoded.Version != cursorVersion {
		page.LegacyCursor = true
		// Legacy cursors carry no navigation position. Preserve their keyset
		// behavior and start newly issued v3 links from the nearest useful page.
		if page.Direction == PageForward {
			page.CurrentPage = 2
		} else {
			page.CurrentPage = 1
		}
	}
	if page.Anchor == PageAnchorEnd {
		return page, nil
	}
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
	currentPage ...int,
) error {
	page := 1
	if len(currentPage) > 0 && currentPage[0] > 0 {
		page = currentPage[0]
	}
	previousCursor, err := encodePageCursor(codec, tenantID, queryHash, firstPrimary, firstID, PageBackward, max(1, page-1), PageAnchorKeyset, hasPrevious)
	if err != nil {
		return err
	}
	nextCursor, err := encodePageCursor(codec, tenantID, queryHash, lastPrimary, lastID, PageForward, page+1, PageAnchorKeyset, hasNext)
	if err != nil {
		return err
	}
	if link := BuildPageLinks(c, issuerFallback, previousCursor, nextCursor); link != "" {
		c.Response().Header().Set("Link", link)
	}
	return nil
}

func encodePageCursor(codec *CursorCodec, tenantID, queryHash, primary, id string, direction PageDirection, page int, anchor PageAnchor, present bool) (string, error) {
	if !present {
		return "", nil
	}
	after := ""
	if anchor == PageAnchorKeyset {
		after = joinKeyset(primary, id)
	}
	return codec.Encode(Cursor{
		Version: cursorVersion, TenantID: tenantID, QueryHash: queryHash,
		After: after, Direction: direction, Anchor: anchor, Page: page,
	})
}

// PaginationMetadata is emitted as response headers so existing body contracts
// and picker clients remain backward compatible.
type PaginationMetadata struct {
	TotalItems  int64
	TotalPages  int
	CurrentPage int
	PageSize    int
}

func CalculatePaginationMetadata(totalItems int64, page PageRequest) PaginationMetadata {
	metadata := PaginationMetadata{TotalItems: totalItems, PageSize: page.Limit}
	if totalItems <= 0 || page.Limit <= 0 {
		return metadata
	}
	metadata.TotalPages = int((totalItems + int64(page.Limit) - 1) / int64(page.Limit))
	metadata.CurrentPage = page.CurrentPage
	if metadata.CurrentPage <= 0 {
		metadata.CurrentPage = 1
	}
	if page.Anchor == PageAnchorEnd {
		metadata.CurrentPage = metadata.TotalPages
	}
	return metadata
}

func SetPaginationHeaders(c *echo.Context, metadata PaginationMetadata) {
	header := c.Response().Header()
	header.Set("Pagination-Total-Items", strconv.FormatInt(metadata.TotalItems, 10))
	header.Set("Pagination-Total-Pages", strconv.Itoa(metadata.TotalPages))
	header.Set("Pagination-Current-Page", strconv.Itoa(metadata.CurrentPage))
	header.Set("Pagination-Page-Size", strconv.Itoa(metadata.PageSize))
}

// SetPaginationLinks emits first/prev/next/last links for an exact-count page.
func SetPaginationLinks(
	c *echo.Context,
	codec *CursorCodec,
	issuerFallback, tenantID, queryHash string,
	page PageRequest,
	firstPrimary, firstID, lastPrimary, lastID string,
	hasPrevious, hasNext bool,
	totalPages int,
) error {
	if totalPages <= 0 {
		c.Response().Header().Del("Link")
		return nil
	}
	currentPage := page.CurrentPage
	if page.Anchor == PageAnchorEnd {
		currentPage = totalPages
	}
	if currentPage <= 0 {
		currentPage = 1
	}
	previousCursor, err := encodePageCursor(codec, tenantID, queryHash, firstPrimary, firstID, PageBackward, max(1, currentPage-1), PageAnchorKeyset, hasPrevious)
	if err != nil {
		return err
	}
	nextCursor, err := encodePageCursor(codec, tenantID, queryHash, lastPrimary, lastID, PageForward, currentPage+1, PageAnchorKeyset, hasNext)
	if err != nil {
		return err
	}
	lastCursor, err := encodePageCursor(codec, tenantID, queryHash, "", "", PageBackward, totalPages, PageAnchorEnd, currentPage != totalPages)
	if err != nil {
		return err
	}
	link := BuildPaginationLinks(c, issuerFallback, currentPage > 1, previousCursor, nextCursor, lastCursor)
	if link == "" {
		c.Response().Header().Del("Link")
	} else {
		c.Response().Header().Set("Link", link)
	}
	return nil
}

// TrimEndPage converts a reverse scan of the final limit rows into the true
// remainder page without serializing count before the page query.
func TrimEndPage[T any](items []T, totalItems int64, limit int) []T {
	if limit <= 0 || totalItems <= 0 {
		return nil
	}
	if len(items) > limit {
		items = items[len(items)-limit:]
	}
	want := int(totalItems % int64(limit))
	if want == 0 {
		want = limit
	}
	if len(items) > want {
		items = items[len(items)-want:]
	}
	return items
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
