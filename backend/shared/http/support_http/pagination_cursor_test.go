package support_http

import (
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
