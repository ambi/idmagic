package bootstrap

import (
	"fmt"
	"strings"
)

type (
	FeatureID         string
	FeatureVersion    string
	FeatureMaturity   string
	DefaultEnablement string
	UpdatePolicy      string
)

const (
	FeatureExperimental FeatureMaturity = "experimental"
	FeaturePreview      FeatureMaturity = "preview"
	FeatureSupported    FeatureMaturity = "supported"
	FeatureDeprecated   FeatureMaturity = "deprecated"

	FeatureDisabled DefaultEnablement = "disabled"
	FeatureEnabled  DefaultEnablement = "enabled"

	UpdateRolling                 UpdatePolicy = "rolling"
	UpdateRecreateOnVersionChange UpdatePolicy = "recreate_on_version_change"
	UpdateRecreateAlways          UpdatePolicy = "recreate_always"
)

// FeatureDefinition は製品ビルドが静的に登録する実行時機能の不変な定義。
type FeatureDefinition struct {
	ID                FeatureID
	Name              string
	Version           FeatureVersion
	Maturity          FeatureMaturity
	DefaultEnablement DefaultEnablement
	Dependencies      []FeatureID
	UpdatePolicy      UpdatePolicy
	SpecificationRef  string
}

type FeatureRegistry []FeatureDefinition

type FeatureResolution struct {
	Enabled  []FeatureDefinition
	Warnings []FeatureWarning
	Metadata FeatureRuntimeMetadata
}

type FeatureWarning struct {
	ID       FeatureID
	Maturity FeatureMaturity
	Reason   string
}

type RuntimeFeatureMetadata struct {
	ID           FeatureID       `json:"id"`
	Version      FeatureVersion  `json:"version"`
	Maturity     FeatureMaturity `json:"maturity"`
	UpdatePolicy UpdatePolicy    `json:"update_policy"`
}

type FeatureRuntimeMetadata struct {
	SchemaVersion string                   `json:"schema_version"`
	Enabled       []RuntimeFeatureMetadata `json:"enabled"`
}

type FeatureError struct {
	Feature FeatureID
	Problem string
}

func (e FeatureError) Error() string {
	if e.Feature == "" {
		return e.Problem
	}
	return fmt.Sprintf("%s: %s", e.Feature, e.Problem)
}

type FeatureErrors []FeatureError

func (e FeatureErrors) Error() string {
	parts := make([]string, 0, len(e))
	for _, item := range e {
		parts = append(parts, item.Error())
	}
	return strings.Join(parts, "; ")
}

// ResolveFeatures は明示指定、既定値、依存閉包から有効機能を決定する純粋計算。
func ResolveFeatures(registry FeatureRegistry, explicitEnable, explicitDisable []FeatureID) (FeatureResolution, error) {
	if errs := validateFeatureRegistry(registry); len(errs) > 0 {
		return FeatureResolution{}, errs
	}
	byID := make(map[FeatureID]FeatureDefinition, len(registry))
	for _, feature := range registry {
		byID[feature.ID] = feature
	}
	if errs := validateFeatureSelection(byID, explicitEnable, explicitDisable); len(errs) > 0 {
		return FeatureResolution{}, errs
	}
	disabled := make(map[FeatureID]bool, len(explicitDisable))
	for _, id := range explicitDisable {
		disabled[id] = true
	}
	enabled := make(map[FeatureID]bool, len(explicitEnable))
	explicitlyEnabled := make(map[FeatureID]bool, len(explicitEnable))
	for _, id := range explicitEnable {
		explicitlyEnabled[id] = true
	}
	ordered := make([]FeatureDefinition, 0, len(registry))

	var errs FeatureErrors
	var include func(FeatureID, FeatureID)
	include = func(id, requiredBy FeatureID) {
		if disabled[id] {
			errs = append(errs, FeatureError{Feature: requiredBy, Problem: fmt.Sprintf("requires explicitly disabled feature %s", id)})
			return
		}
		if enabled[id] {
			return
		}
		enabled[id] = true
		feature, ok := byID[id]
		if !ok {
			return
		}
		for _, dependency := range feature.Dependencies {
			include(dependency, id)
		}
		ordered = append(ordered, feature)
	}
	for _, feature := range registry {
		if feature.DefaultEnablement == FeatureEnabled && !disabled[feature.ID] {
			include(feature.ID, feature.ID)
		}
	}
	for _, id := range explicitEnable {
		include(id, id)
	}
	if len(errs) > 0 {
		return FeatureResolution{}, errs
	}

	metadata := FeatureRuntimeMetadata{SchemaVersion: "1", Enabled: make([]RuntimeFeatureMetadata, 0, len(ordered))}
	warnings := make([]FeatureWarning, 0)
	for _, feature := range ordered {
		metadata.Enabled = append(metadata.Enabled, RuntimeFeatureMetadata{
			ID: feature.ID, Version: feature.Version, Maturity: feature.Maturity, UpdatePolicy: feature.UpdatePolicy,
		})
		if explicitlyEnabled[feature.ID] && (feature.Maturity == FeatureExperimental || feature.Maturity == FeaturePreview) {
			warnings = append(warnings, FeatureWarning{ID: feature.ID, Maturity: feature.Maturity, Reason: fmt.Sprintf("explicitly enabled %s feature", feature.Maturity)})
		}
		if feature.Maturity == FeatureDeprecated {
			warnings = append(warnings, FeatureWarning{ID: feature.ID, Maturity: feature.Maturity, Reason: "deprecated feature is enabled"})
		}
	}
	return FeatureResolution{Enabled: ordered, Warnings: warnings, Metadata: metadata}, nil
}

// ProductFeatureRegistry はこの製品ビルドで実行時選択できる機能を返す。
// 常時提供する機能は登録しないため、選択可能な機能が無いビルドでは空になる。
func ProductFeatureRegistry() FeatureRegistry {
	return FeatureRegistry{}
}

// LoadFeatureConfig は環境由来の明示指定を純粋な resolver へ渡し、拒否を既存の
// 起動時集約エラーへ接続する。
func LoadFeatureConfig(loader *ConfigLoader, registry FeatureRegistry) FeatureResolution {
	allowed := make([]string, 0, len(registry))
	for _, feature := range registry {
		allowed = append(allowed, string(feature.ID))
	}
	enableValues := loader.EnumList("FEATURES_ENABLE", nil, allowed...)
	disableValues := loader.EnumList("FEATURES_DISABLE", nil, allowed...)
	enabled := make([]FeatureID, len(enableValues))
	for i, value := range enableValues {
		enabled[i] = FeatureID(value)
	}
	disabled := make([]FeatureID, len(disableValues))
	for i, value := range disableValues {
		disabled[i] = FeatureID(value)
	}
	resolution, err := ResolveFeatures(registry, enabled, disabled)
	if err != nil {
		loader.fail("FEATURES_ENABLE", err.Error())
		return FeatureResolution{}
	}
	return resolution
}

func validateFeatureSelection(byID map[FeatureID]FeatureDefinition, explicitEnable, explicitDisable []FeatureID) FeatureErrors {
	var errs FeatureErrors
	enabled := make(map[FeatureID]bool, len(explicitEnable))
	disabled := make(map[FeatureID]bool, len(explicitDisable))
	for _, id := range explicitEnable {
		enabled[id] = true
		if _, exists := byID[id]; !exists {
			errs = append(errs, FeatureError{Feature: id, Problem: "explicit enable references unknown feature"})
		}
	}
	for _, id := range explicitDisable {
		disabled[id] = true
		if _, exists := byID[id]; !exists {
			errs = append(errs, FeatureError{Feature: id, Problem: "explicit disable references unknown feature"})
		}
	}
	reportedOverlap := make(map[FeatureID]bool, len(enabled))
	for _, id := range explicitEnable {
		if disabled[id] && !reportedOverlap[id] {
			errs = append(errs, FeatureError{Feature: id, Problem: "feature is both explicitly enabled and disabled"})
			reportedOverlap[id] = true
		}
	}
	return errs
}

func validateFeatureRegistry(registry FeatureRegistry) FeatureErrors {
	var errs FeatureErrors
	byID := make(map[FeatureID]FeatureDefinition, len(registry))
	byName := make(map[string]FeatureID, len(registry))
	for _, feature := range registry {
		if _, exists := byID[feature.ID]; exists {
			errs = append(errs, FeatureError{Feature: feature.ID, Problem: "duplicate feature id"})
		} else {
			byID[feature.ID] = feature
		}
		if previous, exists := byName[feature.Name]; exists {
			errs = append(errs, FeatureError{Feature: feature.ID, Problem: fmt.Sprintf("duplicate feature name %q also used by %s", feature.Name, previous)})
		} else {
			byName[feature.Name] = feature.ID
		}
		if !validFeatureMaturity(feature.Maturity) {
			errs = append(errs, FeatureError{Feature: feature.ID, Problem: fmt.Sprintf("unknown maturity %q", feature.Maturity)})
		}
		if !validDefaultEnablement(feature.DefaultEnablement) {
			errs = append(errs, FeatureError{Feature: feature.ID, Problem: fmt.Sprintf("unknown default enablement %q", feature.DefaultEnablement)})
		}
		if !validUpdatePolicy(feature.UpdatePolicy) {
			errs = append(errs, FeatureError{Feature: feature.ID, Problem: fmt.Sprintf("unknown update policy %q", feature.UpdatePolicy)})
		}
		if feature.DefaultEnablement == FeatureEnabled && feature.Maturity == FeatureExperimental {
			errs = append(errs, FeatureError{Feature: feature.ID, Problem: "experimental feature must be disabled by default"})
		}
		if feature.DefaultEnablement == FeatureEnabled && feature.Maturity == FeatureDeprecated {
			errs = append(errs, FeatureError{Feature: feature.ID, Problem: "deprecated feature must not be enabled by default"})
		}
	}
	for _, feature := range registry {
		for _, dependency := range feature.Dependencies {
			if _, exists := byID[dependency]; !exists {
				errs = append(errs, FeatureError{Feature: feature.ID, Problem: fmt.Sprintf("depends on unknown feature %s", dependency)})
			}
		}
	}
	if cycle := featureDependencyCycle(registry, byID); len(cycle) > 0 {
		names := make([]string, len(cycle))
		for i, id := range cycle {
			names[i] = string(id)
		}
		errs = append(errs, FeatureError{Feature: cycle[0], Problem: "dependency cycle: " + strings.Join(names, " -> ")})
	}
	return errs
}

func validFeatureMaturity(value FeatureMaturity) bool {
	return value == FeatureExperimental || value == FeaturePreview || value == FeatureSupported || value == FeatureDeprecated
}

func validDefaultEnablement(value DefaultEnablement) bool {
	return value == FeatureDisabled || value == FeatureEnabled
}

func validUpdatePolicy(value UpdatePolicy) bool {
	return value == UpdateRolling || value == UpdateRecreateOnVersionChange || value == UpdateRecreateAlways
}

func featureDependencyCycle(registry FeatureRegistry, byID map[FeatureID]FeatureDefinition) []FeatureID {
	const (
		unvisited = iota
		visiting
		visited
	)
	state := make(map[FeatureID]int, len(byID))
	stack := make([]FeatureID, 0, len(byID))
	var visit func(FeatureID) []FeatureID
	visit = func(id FeatureID) []FeatureID {
		state[id] = visiting
		stack = append(stack, id)
		for _, dependency := range byID[id].Dependencies {
			if _, exists := byID[dependency]; !exists {
				continue
			}
			switch state[dependency] {
			case visiting:
				start := 0
				for stack[start] != dependency {
					start++
				}
				return append(append([]FeatureID(nil), stack[start:]...), dependency)
			case unvisited:
				if cycle := visit(dependency); len(cycle) > 0 {
					return cycle
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[id] = visited
		return nil
	}
	for _, feature := range registry {
		id := feature.ID
		if state[id] == unvisited {
			if cycle := visit(id); len(cycle) > 0 {
				return cycle
			}
		}
	}
	return nil
}
