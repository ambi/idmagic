package support_http

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
)

func newPageCtx(t *testing.T, query string) *echo.Context {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/users?"+query, http.NoBody)
	return e.NewContext(req, httptest.NewRecorder())
}

func TestParsePageRequestFirstPage(t *testing.T) {
	codec := NewCursorCodec([]byte("secret"))
	c := newPageCtx(t, "limit=25")
	page, err := ParsePageRequest(c, codec, "tenant-1", "ListThings", 50, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.Limit != 25 || page.AfterPrimary != "" || page.AfterID != "" {
		t.Fatalf("unexpected page request: %+v", page)
	}
}

func TestParsePageRequestRejectsBadLimit(t *testing.T) {
	codec := NewCursorCodec([]byte("secret"))
	c := newPageCtx(t, "limit=not-a-number")
	if _, err := ParsePageRequest(c, codec, "tenant-1", "ListThings", 50, 200); !errors.Is(err, ErrBadPageRequest) {
		t.Fatalf("expected ErrBadPageRequest, got %v", err)
	}
}

func TestParsePageRequestRejectsInvalidCursor(t *testing.T) {
	codec := NewCursorCodec([]byte("secret"))
	c := newPageCtx(t, "cursor=not-a-real-cursor")
	if _, err := ParsePageRequest(c, codec, "tenant-1", "ListThings", 50, 200); !errors.Is(err, ErrBadPageRequest) {
		t.Fatalf("expected ErrBadPageRequest, got %v", err)
	}
}

func TestParsePageRequestRoundTripsWithSetNextLink(t *testing.T) {
	codec := NewCursorCodec([]byte("secret"))
	c := newPageCtx(t, "limit=2")
	if err := SetNextLink(c, codec, "http://idp.test", "tenant-1", "ListThings", "bravo", "id-2", true); err != nil {
		t.Fatalf("SetNextLink failed: %v", err)
	}
	link := c.Response().Header().Get("Link")
	if link == "" {
		t.Fatal("expected Link header to be set")
	}
	linkURL := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	query := strings.TrimPrefix(linkURL, "http://idp.test/api/admin/v1/users?")

	c2 := newPageCtx(t, query)
	page, err := ParsePageRequest(c2, codec, "tenant-1", "ListThings", 50, 200)
	if err != nil {
		t.Fatalf("ParsePageRequest on next link failed: %v", err)
	}
	if page.AfterPrimary != "bravo" || page.AfterID != "id-2" {
		t.Fatalf("unexpected round-tripped keyset: %+v", page)
	}
}

func TestParsePageRequestRejectsCursorFromDifferentQuery(t *testing.T) {
	codec := NewCursorCodec([]byte("secret"))
	cursor, err := codec.Encode(Cursor{
		TenantID: "tenant-1", QueryHash: "ListOtherThings", After: "x.y",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	c := newPageCtx(t, "cursor="+cursor)
	if _, err := ParsePageRequest(c, codec, "tenant-1", "ListThings", 50, 200); !errors.Is(err, ErrBadPageRequest) {
		t.Fatalf("expected ErrBadPageRequest, got %v", err)
	}
}

func TestSetNextLinkNoopWithoutMore(t *testing.T) {
	codec := NewCursorCodec([]byte("secret"))
	c := newPageCtx(t, "limit=2")
	if err := SetNextLink(c, codec, "http://idp.test", "tenant-1", "ListThings", "bravo", "id-2", false); err != nil {
		t.Fatalf("SetNextLink failed: %v", err)
	}
	if link := c.Response().Header().Get("Link"); link != "" {
		t.Fatalf("expected no Link header, got %q", link)
	}
}

func TestSetNextLinkIssuesVersionedForwardCursorWithoutExpiry(t *testing.T) {
	codec := NewCursorCodec([]byte("secret"))
	c := newPageCtx(t, "limit=2")
	if err := SetNextLink(c, codec, "http://idp.test", "tenant-1", "ListThings", "bravo", "id-2", true); err != nil {
		t.Fatalf("SetNextLink failed: %v", err)
	}

	link := c.Response().Header().Get("Link")
	linkURL := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	parsed, err := url.Parse(linkURL)
	if err != nil {
		t.Fatalf("parse Link URL: %v", err)
	}
	token := parsed.Query().Get("cursor")
	payloadPart, _, ok := strings.Cut(token, ".")
	if !ok {
		t.Fatalf("cursor has no signed payload: %q", token)
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		t.Fatalf("decode cursor payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("unmarshal cursor payload: %v", err)
	}
	if claims["v"] != float64(2) {
		t.Fatalf("cursor version = %#v, want 2", claims["v"])
	}
	if claims["d"] != "forward" {
		t.Fatalf("cursor direction = %#v, want forward", claims["d"])
	}
	if _, exists := claims["exp"]; exists {
		t.Fatalf("new cursor unexpectedly contains expiry: %s", payload)
	}
}

func TestParsePageRequestPreservesBackwardDirection(t *testing.T) {
	codec := NewCursorCodec([]byte("secret"))
	cursor, err := codec.Encode(Cursor{
		Version: cursorVersion, TenantID: "tenant-1", QueryHash: "ListThings",
		After: joinKeyset("bravo", "id-2"), Direction: PageBackward,
	})
	if err != nil {
		t.Fatalf("encode backward cursor: %v", err)
	}
	c := newPageCtx(t, "cursor="+url.QueryEscape(cursor))
	page, err := ParsePageRequest(c, codec, "tenant-1", "ListThings", 50, 200)
	if err != nil {
		t.Fatalf("ParsePageRequest failed: %v", err)
	}
	direction := reflect.ValueOf(page).FieldByName("Direction")
	if !direction.IsValid() {
		t.Fatal("PageRequest does not preserve the signed cursor direction")
	}
	if got := direction.String(); got != string(PageBackward) {
		t.Fatalf("direction = %q, want %q", got, PageBackward)
	}
}

func TestSetPageLinksSignsPreviousAndNextBoundaries(t *testing.T) {
	codec := NewCursorCodec([]byte("secret"))
	c := newPageCtx(t, "limit=2&status=active")
	if err := SetPageLinks(
		c, codec, "http://idp.test", "tenant-1", "ListThings;status=active",
		"alpha", "id-1", "bravo", "id-2", true, true,
	); err != nil {
		t.Fatalf("SetPageLinks failed: %v", err)
	}

	link := c.Response().Header().Get("Link")
	parts := strings.Split(link, ", ")
	if len(parts) != 2 {
		t.Fatalf("Link parts = %d, want 2: %q", len(parts), link)
	}
	assertCursor := func(part string, wantDirection PageDirection, wantKeyset string) {
		t.Helper()
		linkURL := part[strings.Index(part, "<")+1 : strings.Index(part, ">")]
		parsed, err := url.Parse(linkURL)
		if err != nil {
			t.Fatalf("parse Link URL: %v", err)
		}
		cursor, err := codec.DecodeCursorForQuery(parsed.Query().Get("cursor"), "tenant-1", "ListThings;status=active")
		if err != nil {
			t.Fatalf("decode cursor: %v", err)
		}
		if cursor.Direction != wantDirection || cursor.After != wantKeyset || !cursor.ExpiresAt.IsZero() {
			t.Fatalf("cursor = %+v, want direction=%s keyset=%q no expiry", cursor, wantDirection, wantKeyset)
		}
	}
	assertCursor(parts[0], PageBackward, joinKeyset("alpha", "id-1"))
	assertCursor(parts[1], PageForward, joinKeyset("bravo", "id-2"))
}

func TestTrimPageForwardDropsLookaheadAndExposesPrevious(t *testing.T) {
	page := PageRequest{AfterPrimary: "boundary", Direction: PageForward, Limit: 2}
	items, hasPrevious, hasNext := TrimPage([]string{"bravo", "charlie", "delta"}, page)
	if strings.Join(items, ",") != "bravo,charlie" || !hasPrevious || !hasNext {
		t.Fatalf("items=%v previous=%v next=%v", items, hasPrevious, hasNext)
	}
}

func TestTrimPageBackwardDropsFarthestLookaheadAndKeepsCanonicalOrder(t *testing.T) {
	page := PageRequest{AfterPrimary: "boundary", Direction: PageBackward, Limit: 2}
	items, hasPrevious, hasNext := TrimPage([]string{"alpha", "bravo", "charlie"}, page)
	if strings.Join(items, ",") != "bravo,charlie" || !hasPrevious || !hasNext {
		t.Fatalf("items=%v previous=%v next=%v", items, hasPrevious, hasNext)
	}
}
