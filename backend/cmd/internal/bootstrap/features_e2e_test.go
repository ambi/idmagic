package bootstrap

import (
	"reflect"
	"testing"
)

// TestLoadFeatureConfig_REQ_SYSTEM_016 は REQ-SYSTEM-016 の正式な起動設定アダプターから resolver までを通し、
// FEATURES_ENABLE を読み落として Operator の選択を無視する誤配線を検出する。
func TestLoadFeatureConfig_REQ_SYSTEM_016(t *testing.T) {
	t.Parallel()
	registry := FeatureRegistry{
		{ID: "base-v1", Name: "base", Version: "1", Maturity: FeatureSupported, DefaultEnablement: FeatureDisabled, UpdatePolicy: UpdateRolling},
		{ID: "preview-v1", Name: "preview", Version: "1", Maturity: FeaturePreview, DefaultEnablement: FeatureDisabled, Dependencies: []FeatureID{"base-v1"}, UpdatePolicy: UpdateRecreateOnVersionChange},
	}
	loader := NewConfigLoader(stubEnv(map[string]string{"FEATURES_ENABLE": "preview-v1"}))

	resolution := LoadFeatureConfig(loader, registry)
	if err := loader.Err(); err != nil {
		t.Fatalf("LoadFeatureConfig: %v", err)
	}
	var ids []FeatureID
	for _, feature := range resolution.Enabled {
		ids = append(ids, feature.ID)
	}
	if !reflect.DeepEqual(ids, []FeatureID{"base-v1", "preview-v1"}) {
		t.Fatalf("enabled ids = %#v", ids)
	}
	if len(resolution.Warnings) != 1 || resolution.Warnings[0].ID != "preview-v1" {
		t.Fatalf("warnings = %#v, want an explicit preview warning", resolution.Warnings)
	}
}
