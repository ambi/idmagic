// Package scim is the SCIM protocol feature slice for outbound provisioning
// (ADR-128 decision 2): it depends on backend/provisioning/domain (the
// protocol-agnostic core) and owns the SCIM wire client. It does not import
// backend/scim (the inbound server); the two share no code (ADR-128 decision 3).
package scim

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ambi/idmagic/backend/provisioning/domain"
)

// MappingOperation distinguishes create from update for AttributeMappingRule's
// apply_on filtering (spec/contexts/provisioning.yaml models.AttributeApplyOn).
type MappingOperation int

const (
	ApplyOnCreate MappingOperation = iota
	ApplyOnUpdate
)

// AttributeResolver resolves an idmagic-side attribute key to its current value.
// ok is false when the attribute is unset (distinct from a present-but-empty value).
type AttributeResolver func(key string) (value any, ok bool)

var multiValuedFilterPath = regexp.MustCompile(`^([A-Za-z0-9_]+)\[([A-Za-z0-9_]+)\s+eq\s+"([^"]*)"\]\.([A-Za-z0-9_]+)$`)

// BuildResource applies rules against resolve to build a SCIM resource document
// (spec/contexts/provisioning.yaml models.AttributeMappingRule). op selects
// create vs update: create_only rules are skipped when op is ApplyOnUpdate.
// A required rule that cannot resolve any value (no attribute, no constant, no
// default) fails closed with an error rather than sending a partial resource.
func BuildResource(rules []domain.AttributeMappingRule, resolve AttributeResolver, op MappingOperation) (map[string]any, error) {
	doc := map[string]any{}
	for _, rule := range rules {
		if rule.ApplyOn == domain.ApplyCreateOnly && op == ApplyOnUpdate {
			continue
		}
		value, resolved := resolveRuleValue(rule, resolve)
		if !resolved {
			if rule.Required {
				return nil, fmt.Errorf("provisioning/scim: required attribute mapping %q could not be resolved", rule.TargetPath)
			}
			continue
		}
		if err := setPath(doc, rule.TargetPath, value); err != nil {
			return nil, fmt.Errorf("provisioning/scim: target_path %q: %w", rule.TargetPath, err)
		}
	}
	return doc, nil
}

func resolveRuleValue(rule domain.AttributeMappingRule, resolve AttributeResolver) (any, bool) {
	switch rule.SourceKind {
	case domain.SourceKindConstant:
		if rule.ConstantValue != nil {
			return rule.ConstantValue, true
		}
	case domain.SourceKindAttribute:
		if resolve != nil {
			if value, ok := resolve(rule.SourceKey); ok && !isEmptyValue(value) {
				return value, true
			}
		}
	}
	if rule.DefaultValue != nil {
		return rule.DefaultValue, true
	}
	return nil, false
}

func isEmptyValue(v any) bool {
	s, ok := v.(string)
	return ok && strings.TrimSpace(s) == ""
}

// setPath writes value into doc at path. Two forms are supported:
//   - dot path: "name.givenName" -> nested object.
//   - multi-valued filter path: `emails[type eq "work"].value` -> a single-element
//     array whose object has the filter attribute, the target field, and
//     primary=true. This covers wi-45's default mapping table; it does not
//     implement the full SCIM filter grammar (that lives in the inbound
//     backend/scim/domain/filter.go, a different concern: parsing an incoming
//     query filter, not constructing an outbound multi-valued attribute).
func setPath(doc map[string]any, path string, value any) error {
	if m := multiValuedFilterPath.FindStringSubmatch(path); m != nil {
		arrayField, filterAttr, filterValue, targetField := m[1], m[2], m[3], m[4]
		doc[arrayField] = []any{
			map[string]any{filterAttr: filterValue, targetField: value, "primary": true},
		}
		return nil
	}
	segments := strings.Split(path, ".")
	for _, s := range segments {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("invalid path")
		}
	}
	cursor := doc
	for _, segment := range segments[:len(segments)-1] {
		next, ok := cursor[segment].(map[string]any)
		if !ok {
			next = map[string]any{}
			cursor[segment] = next
		}
		cursor = next
	}
	cursor[segments[len(segments)-1]] = value
	return nil
}
