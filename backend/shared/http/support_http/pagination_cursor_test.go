package support_http

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	codec := NewCursorCodec([]byte("test-secret"))
	cur := Cursor{
		TenantID:  "tenant-1",
		QueryHash: "sort=id;filter=none",
		After:     "0193abcd",
		ExpiresAt: time.Now().Add(1 * time.Hour).UTC().Truncate(time.Second),
	}
	token, err := codec.Encode(cur)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	got, err := codec.Decode(token)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if got.TenantID != cur.TenantID || got.QueryHash != cur.QueryHash || got.After != cur.After {
		t.Fatalf("decoded cursor mismatch: got %+v, want %+v", got, cur)
	}
	if !got.ExpiresAt.Equal(cur.ExpiresAt) {
		t.Fatalf("decoded expiry mismatch: got %v, want %v", got.ExpiresAt, cur.ExpiresAt)
	}
}

func TestDecodeRejectsTamperedPayload(t *testing.T) {
	codec := NewCursorCodec([]byte("test-secret"))
	token, err := codec.Encode(Cursor{
		TenantID: "tenant-1", After: "1", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("expected token to have payload.signature shape, got %q", token)
	}
	tampered := parts[0] + "x." + parts[1]
	if _, err := codec.Decode(tampered); err == nil {
		t.Fatal("expected Decode to reject a tampered payload")
	}
}

func TestDecodeRejectsWrongSecret(t *testing.T) {
	encoder := NewCursorCodec([]byte("secret-a"))
	decoder := NewCursorCodec([]byte("secret-b"))
	token, err := encoder.Encode(Cursor{
		TenantID: "tenant-1", After: "1", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if _, err := decoder.Decode(token); err == nil {
		t.Fatal("expected Decode to reject a token signed with a different secret")
	}
}

func TestDecodeRejectsExpiredCursor(t *testing.T) {
	codec := NewCursorCodec([]byte("test-secret"))
	token, err := codec.Encode(Cursor{
		TenantID: "tenant-1", After: "1", ExpiresAt: time.Now().Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if _, err := codec.Decode(token); err == nil {
		t.Fatal("expected Decode to reject an expired cursor")
	}
}

func TestDecodeRejectsMalformedToken(t *testing.T) {
	codec := NewCursorCodec([]byte("test-secret"))
	for _, bad := range []string{"", "not-a-token", "a.b.c", "a."} {
		if _, err := codec.Decode(bad); err == nil {
			t.Fatalf("expected Decode(%q) to fail", bad)
		}
	}
}

func TestDecodeForQueryRejectsTenantMismatch(t *testing.T) {
	codec := NewCursorCodec([]byte("test-secret"))
	token, err := codec.Encode(Cursor{
		TenantID: "tenant-1", QueryHash: "q1", After: "1", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if _, err := codec.DecodeForQuery(token, "tenant-2", "q1"); err == nil {
		t.Fatal("expected DecodeForQuery to reject a cursor issued for a different tenant")
	}
}

func TestDecodeForQueryRejectsQueryMismatch(t *testing.T) {
	codec := NewCursorCodec([]byte("test-secret"))
	token, err := codec.Encode(Cursor{
		TenantID: "tenant-1", QueryHash: "q1", After: "1", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if _, err := codec.DecodeForQuery(token, "tenant-1", "q2"); err == nil {
		t.Fatal("expected DecodeForQuery to reject a cursor issued for a different sort/filter query")
	}
}

func TestDecodeForQueryReturnsAfterOnMatch(t *testing.T) {
	codec := NewCursorCodec([]byte("test-secret"))
	token, err := codec.Encode(Cursor{
		TenantID: "tenant-1", QueryHash: "q1", After: "keyset-value", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	after, err := codec.DecodeForQuery(token, "tenant-1", "q1")
	if err != nil {
		t.Fatalf("DecodeForQuery failed: %v", err)
	}
	if after != "keyset-value" {
		t.Fatalf("got after=%q, want %q", after, "keyset-value")
	}
}

func TestV3CursorRoundTripsCompactPageAndEndAnchor(t *testing.T) {
	codec := NewCursorCodec([]byte("test-secret"))
	cur := Cursor{
		Version: cursorVersion, TenantID: "tenant-1", QueryHash: "ListThings;status=active",
		After:     joinKeyset(strings.Repeat("primary-value-", 12), "018f0d4e-7b5a-7c21-9d0a-123456789abc"),
		Direction: PageBackward, Anchor: PageAnchorKeyset, Page: 42,
	}
	token, err := codec.Encode(cur)
	if err != nil {
		t.Fatalf("Encode v3: %v", err)
	}
	if !strings.HasPrefix(token, "v3.") {
		t.Fatalf("token = %q, want v3 prefix", token)
	}
	got, err := codec.DecodeCursorForQuery(token, cur.TenantID, cur.QueryHash)
	if err != nil {
		t.Fatalf("DecodeCursorForQuery v3: %v", err)
	}
	if got.After != cur.After || got.Direction != cur.Direction || got.Anchor != cur.Anchor || got.Page != cur.Page {
		t.Fatalf("decoded cursor = %+v, want %+v", got, cur)
	}
}

func TestV3CursorIsAtMostSixtyPercentOfV2Fixture(t *testing.T) {
	codec := NewCursorCodec([]byte("test-secret"))
	boundary := joinKeyset(strings.Repeat("max-primary-", 24), "018f0d4e-7b5a-7c21-9d0a-123456789abc")
	v2, err := codec.Encode(Cursor{
		Version: 2, TenantID: "tenant-with-a-realistically-long-identifier", QueryHash: strings.Repeat("filter-fingerprint-", 4),
		After: boundary, Direction: PageForward,
	})
	if err != nil {
		t.Fatalf("Encode v2: %v", err)
	}
	v3, err := codec.Encode(Cursor{
		Version: cursorVersion, TenantID: "tenant-with-a-realistically-long-identifier", QueryHash: strings.Repeat("filter-fingerprint-", 4),
		After: boundary, Direction: PageForward, Anchor: PageAnchorKeyset, Page: 12,
	})
	if err != nil {
		t.Fatalf("Encode v3: %v", err)
	}
	if len(v3)*100 > len(v2)*60 {
		t.Fatalf("v3 length = %d, v2 length = %d; want v3 <= 60%%", len(v3), len(v2))
	}
}

func TestV3CursorRejectsTamperTenantQueryAndUnknownVersion(t *testing.T) {
	codec := NewCursorCodec([]byte("test-secret"))
	token, err := codec.Encode(Cursor{
		Version: cursorVersion, TenantID: "tenant-1", QueryHash: "query-1",
		After:     joinKeyset("alpha", "018f0d4e-7b5a-7c21-9d0a-123456789abc"),
		Direction: PageForward, Anchor: PageAnchorKeyset, Page: 2,
	})
	if err != nil {
		t.Fatalf("Encode v3: %v", err)
	}
	if _, err := codec.DecodeCursorForQuery(token, "tenant-2", "query-1"); err == nil {
		t.Fatal("expected tenant mismatch rejection")
	}
	if _, err := codec.DecodeCursorForQuery(token, "tenant-1", "query-2"); err == nil {
		t.Fatal("expected query mismatch rejection")
	}
	parts := strings.Split(token, ".")
	replacement := "A"
	if strings.HasSuffix(parts[1], replacement) {
		replacement = "B"
	}
	parts[1] = parts[1][:len(parts[1])-1] + replacement
	if _, err := codec.DecodeCursorForQuery(strings.Join(parts, "."), "tenant-1", "query-1"); err == nil {
		t.Fatal("expected payload tamper rejection")
	}
	unknown := "v4." + url.PathEscape(parts[1]) + "." + parts[2]
	if _, err := codec.DecodeCursorForQuery(unknown, "tenant-1", "query-1"); err == nil {
		t.Fatal("expected unknown version rejection")
	}
}

func FuzzCursorDecodeForQueryNeverPanics(f *testing.F) {
	codec := NewCursorCodec([]byte("test-secret"))
	valid, err := codec.Encode(Cursor{
		Version: cursorVersion, TenantID: "tenant-1", QueryHash: "query-1",
		After:     joinKeyset("alpha", "018f0d4e-7b5a-7c21-9d0a-123456789abc"),
		Direction: PageForward, Anchor: PageAnchorKeyset, Page: 2,
	})
	if err != nil {
		f.Fatalf("encode seed: %v", err)
	}
	for _, seed := range []string{"", "v3..", "v4.a.b", "legacy.invalid", valid} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, token string) {
		_, _ = codec.DecodeCursorForQuery(token, "tenant-1", "query-1")
	})
}
