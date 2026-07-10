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
		// PII (非 raw_storable) 属性は transform を要し、hash なら tenant salt を要する。
		if !attr.RawStorable {
			if attr.Transform == TransformNone {
				t.Errorf("attr %q is not raw_storable but has no transform", key)
			}
			if attr.Transform == TransformHash && !attr.TenantSaltReq {
				t.Errorf("attr %q hashes but does not require tenant salt", key)
			}
		}
		// raw_storable 属性は変換不要。
		if attr.RawStorable && attr.Transform != TransformNone {
			t.Errorf("attr %q is raw_storable but declares a transform", key)
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
		t.Error("actor.username (PII) must not allow contains")
	}
}

func TestPIIAttributesAreVisibleButNotRawStorable(t *testing.T) {
	for _, field := range []string{"actor.username", "client.ip"} {
		attr, ok := LookupSearchAttribute(field)
		if !ok {
			t.Fatalf("%s missing", field)
		}
		if !attr.UIVisible {
			t.Fatalf("%s should be available in the admin search builder", field)
		}
		if attr.RawStorable {
			t.Fatalf("%s must not store plaintext in the sidecar", field)
		}
	}
}
