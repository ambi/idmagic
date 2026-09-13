package bootstrap

import (
	"strings"
	"testing"
)

// TestRenderConfigReferenceDescribesEveryFieldRead covers REQ-SYSTEM-017: the
// reference is generated from the same Load*Config calls that parse the
// values, and rendering fails rather than silently omitting a key that has
// no description or describing a key no process reads.
//
//spec:covers EX-SYSTEM-017-01: 生成物が Load*Config の読むすべてのキーを、それを読むプロセスの節と説明つきで含むこと。
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
//
//spec:covers EX-SYSTEM-017-01: 生成物がシークレットに分類されたキーの値と既定値を含まず、シークレットであることだけを示すこと。
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

//spec:covers EX-SYSTEM-017-01: 生成物が各キーの値の型、デフォルト値、必須かどうかを示すこと。
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

// 空の registry を並べて読むのは、具体例が「registry が空なら選択可能な機能が無いことを
// 示す」と言っているからである。行の形だけを見る検査は、空の registry で見出しだけを出して
// 黙る実装を通す。その生成物は、機能が無いのか生成が壊れたのかを運用者に区別させない。
//
//spec:covers EX-SYSTEM-017-01: 生成物が registry の各機能の識別子・版・成熟度・既定の有効化・依存・更新方針を含み、registry が空なら選択可能な機能が無いことを示すこと。
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

	empty := RenderFeatureRegistryReference(nil)
	if strings.Contains(empty, "| Feature ID |") {
		t.Errorf("empty registry rendered a table header:\n%s", empty)
	}
	if !strings.Contains(empty, "no runtime-selectable features") {
		t.Errorf("empty registry reference =\n%s\nwant it to state that no feature is selectable", empty)
	}
}

// 生成物と定義の突き合わせは、失敗したときに乖離したキーを名指さなければならない。
// 「一致しない」だけを返す実装では、運用者も生成器の呼び出し元も、どのキーを直すのかを
// 実装を読んで探すことになる。両方向を並べるのは、説明の欠落と読まれないキーが別の誤りで
// あり、片方だけを名指す実装が通ってしまうためである。
//
//spec:covers EX-SYSTEM-017-02: 生成物が Config の定義と一致しないとき、突き合わせが失敗して乖離したキーを報告すること。
func TestRenderConfigReferenceReportsTheDivergedKey(t *testing.T) {
	t.Parallel()
	sections := []configReferenceSection{{
		Title:     "Probe",
		Processes: "idmagic",
		Summary:   "A section that reads one key.",
		Load:      func(l *ConfigLoader) { l.Enum("PROBE_MODE", "a", "a", "b") },
	}}

	_, err := renderConfigReference(sections, map[string]string{}, nil)
	if err == nil {
		t.Fatal("rendering succeeded although the read key has no description")
	}
	if !strings.Contains(err.Error(), "PROBE_MODE") {
		t.Errorf("error = %q, want it to name the undescribed key PROBE_MODE", err)
	}

	_, err = renderConfigReference(sections, map[string]string{
		"PROBE_MODE": "The probe mode.",
		"PROBE_GONE": "A key no process reads any more.",
	}, nil)
	if err == nil {
		t.Fatal("rendering succeeded although a described key is read by no process")
	}
	if !strings.Contains(err.Error(), "PROBE_GONE") {
		t.Errorf("error = %q, want it to name the orphaned key PROBE_GONE", err)
	}
	if strings.Contains(err.Error(), "PROBE_MODE") {
		t.Errorf("error = %q names PROBE_MODE, which is described and read", err)
	}

	rendered, err := renderConfigReference(sections, map[string]string{"PROBE_MODE": "The probe mode."}, nil)
	if err != nil {
		t.Fatalf("rendering a consistent definition failed: %v", err)
	}
	if !strings.Contains(rendered, "| `PROBE_MODE` |") {
		t.Errorf("consistent definition rendered no PROBE_MODE row:\n%s", rendered)
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
