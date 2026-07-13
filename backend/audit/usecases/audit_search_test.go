package usecases

import (
	"testing"

	"github.com/ambi/idmagic/backend/audit/ports"
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

func TestExtractSearchAttributes(t *testing.T) {
	rec := &ports.AuditEventRecord{
		Type: "UserAuthenticated",
		Payload: map[string]any{
			"userId":    "user-1",
			"clientId":  "client-1",
			"sessionId": "sess-1",
			"ip":        "203.0.113.9",
		},
	}
	attrs := ExtractSearchAttributes(rec)
	want := map[string]string{
		"event.type": "UserAuthenticated",
		"outcome":    "success",
		"actor.id":   "user-1",
		"client.id":  "client-1",
		"session.id": "sess-1",
		"client.ip":  "203.0.113.9",
	}
	for k, v := range want {
		if attrs[k] != v {
			t.Errorf("attr %q = %q, want %q", k, attrs[k], v)
		}
	}
	// wi-147: 実アカウントが確定するイベントは username を payload に持たない。
	// 検索時に username -> user_id を解決する設計のため、actor.username は空。
	if _, ok := attrs["actor.username"]; ok {
		t.Fatalf("UserAuthenticated should not carry actor.username, got %q", attrs["actor.username"])
	}
}

func TestExtractSearchAttributesFailureOutcome(t *testing.T) {
	// ADR-104 (ADR-046 の username 条項を撤回): 実アカウントが確定しない可能性のある
	// AuthenticationFailed は平文 username をそのまま検索属性として使う。
	rec := &ports.AuditEventRecord{
		Type:    "AuthenticationFailed",
		Payload: map[string]any{"username": "someone"},
	}
	attrs := ExtractSearchAttributes(rec)
	if attrs["outcome"] != "failure" {
		t.Fatalf("outcome = %q, want failure", attrs["outcome"])
	}
	if attrs["actor.username"] != "someone" {
		t.Fatalf("actor.username = %q, want plaintext %q", attrs["actor.username"], "someone")
	}
}
