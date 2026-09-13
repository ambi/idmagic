package bootstrap

import (
	"reflect"
	"strings"
	"testing"
)

// TestResolveFeatures_REQ_SYSTEM_016 は REQ-SYSTEM-016 に対し、明示的に無効化した依存を resolver が
// 暗黙に有効化して起動を続ける退行を拒否する。
func TestResolveFeatures_REQ_SYSTEM_016(t *testing.T) {
	t.Parallel()
	registry := FeatureRegistry{
		{ID: "base-v1", Name: "base", Version: "1", Maturity: FeatureSupported, DefaultEnablement: FeatureDisabled, UpdatePolicy: UpdateRolling},
		{ID: "preview-v1", Name: "preview", Version: "1", Maturity: FeaturePreview, DefaultEnablement: FeatureDisabled, Dependencies: []FeatureID{"base-v1"}, UpdatePolicy: UpdateRecreateOnVersionChange},
	}

	resolution, err := ResolveFeatures(registry, []FeatureID{"preview-v1"}, []FeatureID{"base-v1"})
	if err == nil {
		t.Fatal("ResolveFeatures succeeded although an enabled feature requires an explicitly disabled dependency")
	}
	if !strings.Contains(err.Error(), "preview-v1") || !strings.Contains(err.Error(), "base-v1") {
		t.Fatalf("ResolveFeatures error = %q, want both the enabled feature and disabled dependency", err)
	}
	if len(resolution.Enabled) != 0 {
		t.Fatalf("resolution.Enabled = %#v, want no partial resolution on error", resolution.Enabled)
	}
}

// TestFeatureResolutionMetadata_REQ_SYSTEM_017 は REQ-SYSTEM-017 に対し、無効な機能が運用メタデータへ
// 混入する退行と、更新方針を落とす退行を拒否する。
func TestFeatureResolutionMetadata_REQ_SYSTEM_017(t *testing.T) {
	t.Parallel()
	registry := FeatureRegistry{
		{ID: "unused-v1", Name: "unused", Version: "1", Maturity: FeatureExperimental, DefaultEnablement: FeatureDisabled, UpdatePolicy: UpdateRecreateAlways},
		{ID: "preview-v2", Name: "preview", Version: "2", Maturity: FeaturePreview, DefaultEnablement: FeatureDisabled, Dependencies: []FeatureID{"base-v1"}, UpdatePolicy: UpdateRecreateOnVersionChange},
		{ID: "base-v1", Name: "base", Version: "1", Maturity: FeatureSupported, DefaultEnablement: FeatureDisabled, UpdatePolicy: UpdateRolling},
	}

	resolution, err := ResolveFeatures(registry, []FeatureID{"preview-v2"}, nil)
	if err != nil {
		t.Fatalf("ResolveFeatures: %v", err)
	}
	want := FeatureRuntimeMetadata{
		SchemaVersion: "1",
		Enabled: []RuntimeFeatureMetadata{
			{ID: "base-v1", Version: "1", Maturity: "supported", UpdatePolicy: "rolling"},
			{ID: "preview-v2", Version: "2", Maturity: "preview", UpdatePolicy: "recreate_on_version_change"},
		},
	}
	if !reflect.DeepEqual(resolution.Metadata, want) {
		t.Fatalf("resolution.Metadata = %#v, want %#v", resolution.Metadata, want)
	}
}

// 6 種類の registry の誤りを同時に置いて、報告がそのすべてを名指すことと、解決結果に
// 部分的な有効化が残らないことを読む。最初の 1 件で止まる実装は運用者に起動試行を
// 繰り返させ、部分的な結果を返す実装は、エラーを無視した呼び出し元に半端な機能集合で
// 起動させる。
//
//spec:covers EX-SYSTEM-016-05: FeatureRegistry の識別子と未版名の重複、存在しない依存、依存循環、実験的機能と非推奨機能の既定有効化をすべて報告し、部分的な解決を返さないこと。
func TestResolveFeaturesRejectsEveryInvalidRegistryEntry_REQ_SYSTEM_016(t *testing.T) {
	t.Parallel()
	registry := FeatureRegistry{
		{ID: "duplicate-v1", Name: "duplicate", Version: "1", Maturity: FeatureSupported, DefaultEnablement: FeatureDisabled, UpdatePolicy: UpdateRolling},
		{ID: "duplicate-v1", Name: "other", Version: "1", Maturity: FeatureSupported, DefaultEnablement: FeatureDisabled, UpdatePolicy: UpdateRolling},
		{ID: "same-name-v1", Name: "same-name", Version: "1", Maturity: FeatureSupported, DefaultEnablement: FeatureDisabled, UpdatePolicy: UpdateRolling},
		{ID: "same-name-v2", Name: "same-name", Version: "2", Maturity: FeatureSupported, DefaultEnablement: FeatureDisabled, UpdatePolicy: UpdateRolling},
		{ID: "missing-v1", Name: "missing", Version: "1", Maturity: FeatureSupported, DefaultEnablement: FeatureDisabled, Dependencies: []FeatureID{"unknown-v1"}, UpdatePolicy: UpdateRolling},
		{ID: "cycle-a-v1", Name: "cycle-a", Version: "1", Maturity: FeatureSupported, DefaultEnablement: FeatureDisabled, Dependencies: []FeatureID{"cycle-b-v1"}, UpdatePolicy: UpdateRolling},
		{ID: "cycle-b-v1", Name: "cycle-b", Version: "1", Maturity: FeatureSupported, DefaultEnablement: FeatureDisabled, Dependencies: []FeatureID{"cycle-a-v1"}, UpdatePolicy: UpdateRolling},
		{ID: "experimental-v1", Name: "experimental", Version: "1", Maturity: FeatureExperimental, DefaultEnablement: FeatureEnabled, UpdatePolicy: UpdateRolling},
		{ID: "deprecated-v1", Name: "deprecated", Version: "1", Maturity: FeatureDeprecated, DefaultEnablement: FeatureEnabled, UpdatePolicy: UpdateRolling},
	}

	resolution, err := ResolveFeatures(registry, nil, nil)
	if err == nil {
		t.Fatal("ResolveFeatures accepted an invalid registry")
	}
	for _, want := range []string{"duplicate feature id", "duplicate feature name", "unknown-v1", "dependency cycle", "experimental", "deprecated"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ResolveFeatures error = %q, want %q", err, want)
		}
	}
	if len(resolution.Enabled) != 0 {
		t.Fatalf("resolution.Enabled = %#v, want no partial resolution on error", resolution.Enabled)
	}
}

// 警告を DeepEqual で固定するのは、具体例が「識別子と成熟度だけを」と言っているからである。
// 部分一致で読む検査は、環境変数の生値を理由に混ぜた実装を通す。そこには DSN も入りうる。
//
//spec:covers EX-SYSTEM-016-01: 明示指定と既定値と依存閉包から有効機能を決め、明示的に有効化した preview と有効な deprecated の機能を識別子と成熟度だけの起動警告へ記録すること。
func TestResolveFeaturesAppliesDefaultsDependenciesAndWarnings_REQ_SYSTEM_016(t *testing.T) {
	t.Parallel()
	registry := FeatureRegistry{
		{ID: "base-v1", Name: "base", Version: "1", Maturity: FeatureSupported, DefaultEnablement: FeatureEnabled, UpdatePolicy: UpdateRolling},
		{ID: "preview-v1", Name: "preview", Version: "1", Maturity: FeaturePreview, DefaultEnablement: FeatureDisabled, Dependencies: []FeatureID{"base-v1"}, UpdatePolicy: UpdateRolling},
		{ID: "deprecated-v1", Name: "deprecated", Version: "1", Maturity: FeatureDeprecated, DefaultEnablement: FeatureDisabled, UpdatePolicy: UpdateRecreateAlways},
	}

	resolution, err := ResolveFeatures(registry, []FeatureID{"base-v1", "preview-v1", "deprecated-v1"}, nil)
	if err != nil {
		t.Fatalf("ResolveFeatures: %v", err)
	}
	var ids []FeatureID
	for _, feature := range resolution.Enabled {
		ids = append(ids, feature.ID)
	}
	if !reflect.DeepEqual(ids, []FeatureID{"base-v1", "preview-v1", "deprecated-v1"}) {
		t.Fatalf("enabled ids = %#v", ids)
	}
	wantWarnings := []FeatureWarning{
		{ID: "preview-v1", Maturity: FeaturePreview, Reason: "explicitly enabled preview feature"},
		{ID: "deprecated-v1", Maturity: FeatureDeprecated, Reason: "deprecated feature is enabled"},
	}
	if !reflect.DeepEqual(resolution.Warnings, wantWarnings) {
		t.Fatalf("resolution.Warnings = %#v, want %#v", resolution.Warnings, wantWarnings)
	}
}

// 存在しない識別子を有効化と無効化の両側で指定し、さらに同じ機能を両方で指定する。
// 3 件すべてが 1 回の報告に揃うこと、そして部分的な解決が残らないことを読む。
// 明示的に無効化した依存を要求する選択は TestResolveFeatures_REQ_SYSTEM_016 が持つ。
//
//spec:covers EX-SYSTEM-016-06: FEATURES_ENABLE と FEATURES_DISABLE の存在しない機能と矛盾する指定をすべて報告し、部分的な解決を返さないこと。
func TestResolveFeaturesRejectsEveryInvalidSelection_REQ_SYSTEM_016(t *testing.T) {
	t.Parallel()
	registry := FeatureRegistry{
		{ID: "known-v1", Name: "known", Version: "1", Maturity: FeatureSupported, DefaultEnablement: FeatureDisabled, UpdatePolicy: UpdateRolling},
	}

	resolution, err := ResolveFeatures(registry, []FeatureID{"known-v1", "missing-enable-v1"}, []FeatureID{"known-v1", "missing-disable-v1"})
	if err == nil {
		t.Fatal("ResolveFeatures accepted unknown and contradictory selections")
	}
	for _, want := range []string{"missing-enable-v1", "missing-disable-v1", "both explicitly enabled and disabled"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ResolveFeatures error = %q, want %q", err, want)
		}
	}
	if len(resolution.Enabled) != 0 {
		t.Fatalf("resolution.Enabled = %#v, want no partial resolution on error", resolution.Enabled)
	}
}
