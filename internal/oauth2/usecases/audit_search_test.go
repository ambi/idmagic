package usecases

import (
	"testing"

	"github.com/ambi/idmagic/internal/oauth2/ports"
	"github.com/ambi/idmagic/internal/shared/spec"
)

func TestParseAuditFilterAcceptsAllowlisted(t *testing.T) {
	exprs, err := ParseAuditFilter([]RawFilter{
		{Field: "event.type", Operator: "eq", Values: []string{"UserAuthenticated"}},
		{Field: "actor.id", Operator: "in", Values: []string{"u1", "u2"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exprs) != 2 {
		t.Fatalf("expected 2 expressions, got %d", len(exprs))
	}
}

func TestParseAuditFilterRejects(t *testing.T) {
	cases := []struct {
		name string
		raw  RawFilter
	}{
		{"unknown field", RawFilter{Field: "payload.password", Operator: "eq", Values: []string{"x"}}},
		{"operator not allowed", RawFilter{Field: "actor.username", Operator: "contains", Values: []string{"x"}}},
		{"unknown operator", RawFilter{Field: "actor.id", Operator: "regex", Values: []string{"x"}}},
		{"eq wrong cardinality", RawFilter{Field: "actor.id", Operator: "eq", Values: []string{"a", "b"}}},
		{"in empty", RawFilter{Field: "actor.id", Operator: "in", Values: nil}},
		{"empty value", RawFilter{Field: "actor.id", Operator: "eq", Values: []string{""}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseAuditFilter([]RawFilter{tc.raw}); err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}

func TestTransformFilterValuesHashesUsername(t *testing.T) {
	salt := []byte("tenant-salt")
	exprs, err := ParseAuditFilter([]RawFilter{
		{Field: "actor.username", Operator: "eq", Values: []string{"Alice"}},
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := TransformFilterValues(exprs, salt)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	// username は lowercased で salted hash され、平文は残らない。
	want := spec.SaltedHash(salt, spec.NormalizeUsername("Alice"))
	if got[0].Values[0] != want {
		t.Fatalf("username not hashed as expected: got %q want %q", got[0].Values[0], want)
	}
	if got[0].Values[0] == "Alice" || got[0].Values[0] == "alice" {
		t.Fatal("plaintext username leaked into filter value")
	}
}

func TestTransformFilterValuesTruncatesIP(t *testing.T) {
	exprs, err := ParseAuditFilter([]RawFilter{
		{Field: "client.ip", Operator: "eq", Values: []string{"203.0.113.9"}},
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := TransformFilterValues(exprs, nil)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if got[0].Values[0] != "203.0.113.0/24" {
		t.Fatalf("ip not truncated: %q", got[0].Values[0])
	}
}

func TestTransformFilterValuesRejectsBadIP(t *testing.T) {
	exprs, _ := ParseAuditFilter([]RawFilter{
		{Field: "client.ip", Operator: "eq", Values: []string{"not-an-ip"}},
	})
	if _, err := TransformFilterValues(exprs, nil); err == nil {
		t.Fatal("expected error for malformed IP")
	}
}

func TestTransformFilterValuesLeavesRawAttributes(t *testing.T) {
	exprs, _ := ParseAuditFilter([]RawFilter{
		{Field: "actor.id", Operator: "eq", Values: []string{"user-123"}},
	})
	got, err := TransformFilterValues(exprs, []byte("salt"))
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if got[0].Values[0] != "user-123" {
		t.Fatalf("raw attribute value should be unchanged, got %q", got[0].Values[0])
	}
}

func TestExtractSearchAttributes(t *testing.T) {
	rec := &ports.AuditEventRecord{
		Type: "UserAuthenticated",
		Payload: map[string]any{
			"userId":       "user-1",
			"clientId":     "client-1",
			"sessionId":    "sess-1",
			"usernameHash": "hash-alice",
			"ipTruncated":  "203.0.113.0/24",
		},
	}
	attrs := ExtractSearchAttributes(rec)
	want := map[string]string{
		"event.type":     "UserAuthenticated",
		"outcome":        "success",
		"actor.id":       "user-1",
		"client.id":      "client-1",
		"session.id":     "sess-1",
		"actor.username": "hash-alice",
		"client.ip":      "203.0.113.0/24",
	}
	for k, v := range want {
		if attrs[k] != v {
			t.Errorf("attr %q = %q, want %q", k, attrs[k], v)
		}
	}
}

func TestExtractSearchAttributesFailureOutcome(t *testing.T) {
	rec := &ports.AuditEventRecord{
		Type:    "AuthenticationFailed",
		Payload: map[string]any{"username": "someone"},
	}
	attrs := ExtractSearchAttributes(rec)
	if attrs["outcome"] != "failure" {
		t.Fatalf("outcome = %q, want failure", attrs["outcome"])
	}
	// 平文 username は sidecar 検索属性に載らない (payload 側にのみ残す)。
	for k, v := range attrs {
		if v == "someone" {
			t.Fatalf("plaintext username leaked into search attribute %q", k)
		}
	}
}

func TestBuildAuthenticationEventAttributes(t *testing.T) {
	salt := []byte("tenant-a")
	got := BuildAuthenticationEventAttributes(salt, " Alice ", "203.0.113.9", "UA/1")
	if got.UsernameHash != spec.SaltedHash(salt, "alice") {
		t.Fatalf("username hash = %q", got.UsernameHash)
	}
	if got.IPTruncated != "203.0.113.0/24" {
		t.Fatalf("ip truncated = %q", got.IPTruncated)
	}
	if got.IPHash != spec.SaltedHash(salt, "203.0.113.9") {
		t.Fatalf("ip hash = %q", got.IPHash)
	}
	if got.UAHash != spec.SaltedHash(salt, "UA/1") {
		t.Fatalf("ua hash = %q", got.UAHash)
	}
}

func TestBuildAuthenticationEventAttributesSeparatesTenants(t *testing.T) {
	a := BuildAuthenticationEventAttributes([]byte("tenant-a"), "alice", "203.0.113.9", "")
	b := BuildAuthenticationEventAttributes([]byte("tenant-b"), "alice", "203.0.113.9", "")
	if a.UsernameHash == b.UsernameHash {
		t.Fatal("username hash must differ by tenant salt")
	}
	if a.IPHash == b.IPHash {
		t.Fatal("ip hash must differ by tenant salt")
	}
	if a.IPTruncated != b.IPTruncated {
		t.Fatal("ip truncation should not depend on tenant salt")
	}
}
