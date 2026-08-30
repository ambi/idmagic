package bootstrap

import (
	"strings"
	"testing"
)

// TestRenderConfigReferenceDescribesEveryFieldRead covers REQ-SYSTEM-017: the
// reference is generated from the same Load*Config calls that parse the
// values, and rendering fails rather than silently omitting a key that has
// no description or describing a key no process reads.
func TestRenderConfigReferenceDescribesEveryFieldRead(t *testing.T) {
	t.Parallel()
	rendered, err := RenderConfigReference()
	if err != nil {
		t.Fatalf("RenderConfigReference: %v", err)
	}
	for _, section := range configReferenceSections {
		l := NewConfigLoader(stubEnv(nil))
		section.Load(l)
		for _, field := range l.Fields() {
			if !strings.Contains(rendered, "| `"+field.Key+"` |") {
				t.Errorf("%s is read by the %s config but is missing from the reference", field.Key, section.Title)
			}
		}
	}
}

// TestRenderConfigReferenceOmitsSecretValues covers REQ-SYSTEM-017's secret
// rule: a secret key is listed so an operator knows to set it, with no value
// and no default.
func TestRenderConfigReferenceOmitsSecretValues(t *testing.T) {
	t.Parallel()
	rendered, err := RenderConfigReference()
	if err != nil {
		t.Fatalf("RenderConfigReference: %v", err)
	}
	l := NewConfigLoader(stubEnv(nil))
	for _, section := range configReferenceSections {
		section.Load(l)
	}
	secrets := 0
	for _, field := range l.Fields() {
		if !field.Secret {
			continue
		}
		secrets++
		row := referenceRow(t, rendered, field.Key)
		if !strings.Contains(row, "| secret |") {
			t.Errorf("row for %s does not declare the value secret: %s", field.Key, row)
		}
		if !strings.Contains(row, "| — |") {
			t.Errorf("row for %s renders a default for a secret: %s", field.Key, row)
		}
	}
	if secrets == 0 {
		t.Fatal("expected at least one secret field in the reference")
	}
}

func TestRenderConfigReferenceRecordsTypeDefaultAndRequirement(t *testing.T) {
	t.Parallel()
	rendered, err := RenderConfigReference()
	if err != nil {
		t.Fatalf("RenderConfigReference: %v", err)
	}
	for key, want := range map[string]string{
		"PERSISTENCE":            "enum: `memory`, `postgres`",
		"TRUSTED_FORWARDED_HOPS": "integer (>= 0)",
		"JOB_POLL_INTERVAL":      "duration (> 0)",
	} {
		row := referenceRow(t, rendered, key)
		if !strings.Contains(row, want) {
			t.Errorf("row for %s = %s, want it to state %q", key, row, want)
		}
	}
	if row := referenceRow(t, rendered, "ADDR"); !strings.Contains(row, "| `:8080` |") {
		t.Errorf("row for ADDR = %s, want the :8080 default", row)
	}
	if row := referenceRow(t, rendered, "DATABASE_URL"); !strings.Contains(row, "| when `PERSISTENCE=postgres` |") {
		t.Errorf("row for DATABASE_URL = %s, want its conditional requirement", row)
	}
}

// TestRenderConfigReferenceKeepsProcessSpecificDefaults covers
// REQ-SYSTEM-017's process ownership rule. A key read by more than one
// process must appear in each process section because its default may differ
// (OTEL_SERVICE_NAME is idmagic for API and idmagic-worker for Worker).
func TestRenderConfigReferenceKeepsProcessSpecificDefaults(t *testing.T) {
	t.Parallel()
	rendered, err := RenderConfigReference()
	if err != nil {
		t.Fatalf("RenderConfigReference: %v", err)
	}
	row := referenceRowInSection(t, rendered, "Worker", "OTEL_SERVICE_NAME")
	if !strings.Contains(row, "| `idmagic-worker` |") {
		t.Errorf("Worker OTEL_SERVICE_NAME row = %s, want the worker-specific default", row)
	}
}

func TestRenderFeatureRegistryReference_REQ_SYSTEM_017(t *testing.T) {
	t.Parallel()
	registry := FeatureRegistry{{
		ID: "preview-v2", Name: "preview", Version: "2", Maturity: FeaturePreview, DefaultEnablement: FeatureDisabled,
		Dependencies: []FeatureID{"base-v1"}, UpdatePolicy: UpdateRecreateOnVersionChange, SpecificationRef: "REQ-SYSTEM-016",
	}}

	rendered := RenderFeatureRegistryReference(registry)
	want := "| `preview-v2` | `2` | preview | disabled | `base-v1` | recreate_on_version_change | `REQ-SYSTEM-016` |"
	if !strings.Contains(rendered, want) {
		t.Fatalf("feature registry reference =\n%s\nwant row %s", rendered, want)
	}
}

func referenceRow(t *testing.T, rendered, key string) string {
	t.Helper()
	for line := range strings.SplitSeq(rendered, "\n") {
		if strings.HasPrefix(line, "| `"+key+"` |") {
			return line
		}
	}
	t.Fatalf("%s has no row in the generated reference", key)
	return ""
}

func referenceRowInSection(t *testing.T, rendered, section, key string) string {
	t.Helper()
	heading := "## " + section + "\n"
	_, tail, ok := strings.Cut(rendered, heading)
	if !ok {
		t.Fatalf("reference has no %s section", section)
	}
	if next, _, found := strings.Cut(tail, "\n## "); found {
		tail = next
	}
	return referenceRow(t, tail, key)
}
