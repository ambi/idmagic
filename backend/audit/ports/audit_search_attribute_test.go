package ports

import "testing"

func TestAuditSearchRegistryIntegrity(t *testing.T) {
	if len(AuditSearchRegistry) == 0 {
		t.Fatal("registry is empty")
	}
	for key, attr := range AuditSearchRegistry {
		if attr.Field != key {
			t.Errorf("registry key %q != attr.Field %q", key, attr.Field)
		}
		if len(attr.AllowedOperators) == 0 {
			t.Errorf("attr %q has no allowed operators", key)
		}
		for _, op := range attr.AllowedOperators {
			switch op {
			case OpEq, OpIn, OpContains, OpTimeRange:
			default:
				t.Errorf("attr %q has unknown operator %q", key, op)
			}
		}
		// ADR-104 (ADR-046 の username/IP 条項を撤回) により、現状は全属性が平文 raw_storable。
		if !attr.RawStorable {
			t.Errorf("attr %q is not raw_storable, but no PII transform exists anymore (ADR-104)", key)
		}
		if attr.Transform != TransformNone {
			t.Errorf("attr %q declares a transform, but TransformNone is the only transform (ADR-104)", key)
		}
	}
}

func TestAuditSearchRegistryHasCoreAttributes(t *testing.T) {
	// wi-145 の初期 registry が主要な検索軸を含むことを固定する。
	want := []string{
		"event.type", "outcome", "actor.id", "actor.username", "target.id",
		"client.id", "client.ip", "session.id", "transaction.id",
		"correlation.id", "request.id",
	}
	for _, field := range want {
		if _, ok := LookupSearchAttribute(field); !ok {
			t.Errorf("registry missing expected attribute %q", field)
		}
	}
}

func TestAllowsOperator(t *testing.T) {
	attr, ok := LookupSearchAttribute("actor.username")
	if !ok {
		t.Fatal("actor.username missing")
	}
	if !attr.AllowsOperator(OpEq) {
		t.Error("actor.username should allow eq")
	}
	if attr.AllowsOperator(OpContains) {
		t.Error("actor.username must not allow contains")
	}
}

func TestActorUsernameAndClientIPAreVisibleAndRawStorable(t *testing.T) {
	// ADR-104 (ADR-046 の username/IP 条項を撤回): 平文のまま sidecar に保存・検索する。
	for _, field := range []string{"actor.username", "client.ip"} {
		attr, ok := LookupSearchAttribute(field)
		if !ok {
			t.Fatalf("%s missing", field)
		}
		if !attr.UIVisible {
			t.Fatalf("%s should be available in the admin search builder", field)
		}
		if !attr.RawStorable {
			t.Fatalf("%s should store plaintext in the sidecar (ADR-104)", field)
		}
	}
}
