package support_http

import (
	"errors"
	"net/http"
	"net/http/httptest"
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
	url := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	query := strings.TrimPrefix(url, "http://idp.test/api/admin/v1/users?")

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
