package usecases

// 主要ユースケース追跡: REQ-AUDIT-001。

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
		if got := attrs[k]; len(got) != 1 || got[0] != v {
			t.Errorf("attr %q = %v, want [%q]", k, got, v)
		}
	}
	// wi-147: 実アカウントが確定するイベントは username を payload に持たない。
	// 検索時に username -> user_id を解決する設計のため、actor.username は空。
	if _, ok := attrs["actor.username"]; ok {
		t.Fatalf("UserAuthenticated should not carry actor.username, got %v", attrs["actor.username"])
	}
}

func TestExtractSearchAttributesFailureOutcome(t *testing.T) {
	// 実アカウントが確定しない可能性のある
	// AuthenticationFailed は平文 username をそのまま検索属性として使う。
	rec := &ports.AuditEventRecord{
		Type:    "AuthenticationFailed",
		Payload: map[string]any{"username": "someone"},
	}
	attrs := ExtractSearchAttributes(rec)
	if got := attrs["outcome"]; len(got) != 1 || got[0] != "failure" {
		t.Fatalf("outcome = %v, want [failure]", got)
	}
	if got := attrs["actor.username"]; len(got) != 1 || got[0] != "someone" {
		t.Fatalf("actor.username = %v, want plaintext [someone]", got)
	}
}

// REQ-AUDIT-005: エージェントが代行した操作は、行為者の種別とエージェントの識別子を持ち、
// actor.id が利用者へ読み替えられない。
func TestExtractSearchAttributesAgentActorDoesNotFallBackToTheUser(t *testing.T) {
	rec := &ports.AuditEventRecord{
		Type: "TokenExchanged",
		Payload: map[string]any{
			"agentId":        "agent-a",
			"actorUserId":    "agent-client",
			"subjectUserId":  "user-alice",
			"userId":         "user-alice",
			"delegationMode": "on_behalf_of",
			"actorChain":     []any{"agent-client", "user-alice"},
		},
	}
	attrs := ExtractSearchAttributes(rec)
	if got := attrs["actor.type"]; len(got) != 1 || got[0] != ports.ActorTypeAgent {
		t.Fatalf("actor.type = %v, want [%s]", got, ports.ActorTypeAgent)
	}
	if got := attrs["agent.id"]; len(got) != 1 || got[0] != "agent-a" {
		t.Fatalf("agent.id = %v, want [agent-a]", got)
	}
	if got := attrs["actor.id"]; len(got) != 1 || got[0] != "agent-client" {
		t.Fatalf("actor.id = %v, want [agent-client]; it must not fall back to the user", got)
	}
	if got := attrs["target.id"]; len(got) != 1 || got[0] != "user-alice" {
		t.Fatalf("target.id = %v, want [user-alice]; the delegated user is the target, not the actor", got)
	}
	if got := attrs["delegation.mode"]; len(got) != 1 || got[0] != "on_behalf_of" {
		t.Fatalf("delegation.mode = %v, want [on_behalf_of]", got)
	}
}

// REQ-AUDIT-005: agentId を持つイベントでも、Agent が操作の対象であるものは
// 行為者を Agent としない。管理者による Agent の登録がその例。
func TestExtractSearchAttributesAgentAsTargetKeepsTheHumanActor(t *testing.T) {
	rec := &ports.AuditEventRecord{
		Type: "AgentRegistered",
		Payload: map[string]any{
			"actorUserId": "admin-1",
			"agentId":     "agent-a",
		},
	}
	attrs := ExtractSearchAttributes(rec)
	if got := attrs["actor.type"]; len(got) != 1 || got[0] != ports.ActorTypeUser {
		t.Fatalf("actor.type = %v, want [%s]", got, ports.ActorTypeUser)
	}
	if got := attrs["actor.id"]; len(got) != 1 || got[0] != "admin-1" {
		t.Fatalf("actor.id = %v, want [admin-1]", got)
	}
	if got := attrs["agent.id"]; len(got) != 1 || got[0] != "agent-a" {
		t.Fatalf("agent.id = %v, want [agent-a]; the axis names the Agent the event concerns", got)
	}
}

// REQ-AUDIT-005: 行為者が Agent で、その識別子しか無いイベントは actor.id を
// 代行された利用者で埋めない。
func TestExtractSearchAttributesAgentActorWithoutAnActorSubUsesTheAgent(t *testing.T) {
	rec := &ports.AuditEventRecord{
		Type: "BackchannelAuthRequested",
		Payload: map[string]any{
			"agentId":  "agent-a",
			"clientId": "agent-app",
			"userId":   "user-alice",
		},
	}
	attrs := ExtractSearchAttributes(rec)
	if got := attrs["actor.id"]; len(got) != 1 || got[0] != "agent-a" {
		t.Fatalf("actor.id = %v, want [agent-a]; it must not fall back to the user", got)
	}
	if got := attrs["actor.type"]; len(got) != 1 || got[0] != ports.ActorTypeAgent {
		t.Fatalf("actor.type = %v, want [%s]", got, ports.ActorTypeAgent)
	}
	if got := attrs["target.id"]; len(got) != 1 || got[0] != "user-alice" {
		t.Fatalf("target.id = %v, want [user-alice]", got)
	}
}

// REQ-AUDIT-005: エージェントを名指さないイベントの行為者は利用者であり、
// 既存の userId フォールバックはそのまま残る。
func TestExtractSearchAttributesUserActorKeepsTheExistingFallback(t *testing.T) {
	rec := &ports.AuditEventRecord{
		Type:    "UserAuthenticated",
		Payload: map[string]any{"userId": "user-alice"},
	}
	attrs := ExtractSearchAttributes(rec)
	if got := attrs["actor.id"]; len(got) != 1 || got[0] != "user-alice" {
		t.Fatalf("actor.id = %v, want [user-alice]", got)
	}
	if got := attrs["actor.type"]; len(got) != 1 || got[0] != ports.ActorTypeUser {
		t.Fatalf("actor.type = %v, want [%s]", got, ports.ActorTypeUser)
	}
	if _, ok := attrs["agent.id"]; ok {
		t.Fatalf("agent.id should be absent, got %v", attrs["agent.id"])
	}
}

// REQ-AUDIT-006: 委譲チェーンの参加者は多値の軸として並び、どの段からでも引ける。
func TestExtractSearchAttributesDelegationChainIsMultiValued(t *testing.T) {
	rec := &ports.AuditEventRecord{
		Type: "TokenExchanged",
		Payload: map[string]any{
			"actorUserId":     "agent-b",
			"subjectUserId":   "user-alice",
			"delegationDepth": float64(2),
			"actorChain":      []any{"agent-b", "agent-a", "user-alice"},
		},
	}
	attrs := ExtractSearchAttributes(rec)
	want := []string{"agent-b", "agent-a", "user-alice"}
	got := attrs["delegation.actor"]
	if len(got) != len(want) {
		t.Fatalf("delegation.actor = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("delegation.actor = %v, want %v", got, want)
		}
	}
	if d := attrs["delegation.depth"]; len(d) != 1 || d[0] != "2" {
		t.Fatalf("delegation.depth = %v, want [2]", d)
	}
}

// REQ-AUDIT-005: 委譲の軸を持たない過去のイベントは、その軸のどの値にも一致しない
// 形で保存される (値を補って埋めない)。
func TestExtractSearchAttributesLeavesDelegationAxesAbsent(t *testing.T) {
	rec := &ports.AuditEventRecord{
		Type:    "ClientCreated",
		Payload: map[string]any{"clientId": "client-1"},
	}
	attrs := ExtractSearchAttributes(rec)
	for _, field := range []string{"actor.type", "agent.id", "delegation.actor", "delegation.depth", "delegation.mode"} {
		if _, ok := attrs[field]; ok {
			t.Fatalf("%s should be absent for an event without an actor, got %v", field, attrs[field])
		}
	}
}
