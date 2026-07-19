package scim

import (
	"reflect"
	"testing"

	"github.com/ambi/idmagic/backend/provisioning/domain"
)

func TestBuildResource_SimplePath(t *testing.T) {
	rules := []domain.AttributeMappingRule{
		{TargetPath: "userName", SourceKind: domain.SourceKindAttribute, SourceKey: "preferred_username", ApplyOn: domain.ApplyCreateAndUpdate},
	}
	doc, err := BuildResource(rules, resolverFromMap(map[string]any{"preferred_username": "alice"}), ApplyOnCreate)
	if err != nil {
		t.Fatalf("BuildResource() error = %v", err)
	}
	if doc["userName"] != "alice" {
		t.Errorf("doc[userName] = %v, want alice", doc["userName"])
	}
}

func TestBuildResource_NestedPath(t *testing.T) {
	rules := []domain.AttributeMappingRule{
		{TargetPath: "name.givenName", SourceKind: domain.SourceKindAttribute, SourceKey: "given_name", ApplyOn: domain.ApplyCreateAndUpdate},
		{TargetPath: "name.familyName", SourceKind: domain.SourceKindAttribute, SourceKey: "family_name", ApplyOn: domain.ApplyCreateAndUpdate},
	}
	doc, err := BuildResource(rules, resolverFromMap(map[string]any{"given_name": "Alice", "family_name": "Smith"}), ApplyOnCreate)
	if err != nil {
		t.Fatalf("BuildResource() error = %v", err)
	}
	name, ok := doc["name"].(map[string]any)
	if !ok {
		t.Fatalf("doc[name] = %v, want a nested object", doc["name"])
	}
	if name["givenName"] != "Alice" || name["familyName"] != "Smith" {
		t.Errorf("doc[name] = %+v, want givenName=Alice familyName=Smith", name)
	}
}

func TestBuildResource_MultiValuedFilterPath(t *testing.T) {
	rules := []domain.AttributeMappingRule{
		{TargetPath: `emails[type eq "work"].value`, SourceKind: domain.SourceKindAttribute, SourceKey: "email", ApplyOn: domain.ApplyCreateAndUpdate},
	}
	doc, err := BuildResource(rules, resolverFromMap(map[string]any{"email": "alice@example.com"}), ApplyOnCreate)
	if err != nil {
		t.Fatalf("BuildResource() error = %v", err)
	}
	want := map[string]any{
		"emails": []any{
			map[string]any{"type": "work", "value": "alice@example.com", "primary": true},
		},
	}
	if !reflect.DeepEqual(doc, want) {
		t.Errorf("doc = %+v, want %+v", doc, want)
	}
}

func TestBuildResource_ConstantSource(t *testing.T) {
	rules := []domain.AttributeMappingRule{
		{TargetPath: "active", SourceKind: domain.SourceKindConstant, ConstantValue: true, ApplyOn: domain.ApplyCreateAndUpdate},
	}
	doc, err := BuildResource(rules, resolverFromMap(nil), ApplyOnCreate)
	if err != nil {
		t.Fatalf("BuildResource() error = %v", err)
	}
	if doc["active"] != true {
		t.Errorf("doc[active] = %v, want true", doc["active"])
	}
}

func TestBuildResource_DefaultValueWhenSourceEmpty(t *testing.T) {
	rules := []domain.AttributeMappingRule{
		{TargetPath: "displayName", SourceKind: domain.SourceKindAttribute, SourceKey: "display_name", DefaultValue: "unknown", ApplyOn: domain.ApplyCreateAndUpdate},
	}
	doc, err := BuildResource(rules, resolverFromMap(nil), ApplyOnCreate)
	if err != nil {
		t.Fatalf("BuildResource() error = %v", err)
	}
	if doc["displayName"] != "unknown" {
		t.Errorf("doc[displayName] = %v, want unknown (default)", doc["displayName"])
	}
}

func TestBuildResource_RequiredMissingFailsClosed(t *testing.T) {
	rules := []domain.AttributeMappingRule{
		{TargetPath: "userName", SourceKind: domain.SourceKindAttribute, SourceKey: "preferred_username", Required: true, ApplyOn: domain.ApplyCreateAndUpdate},
	}
	_, err := BuildResource(rules, resolverFromMap(nil), ApplyOnCreate)
	if err == nil {
		t.Error("BuildResource() with a required unresolved attribute should return an error")
	}
}

func TestBuildResource_CreateOnlySkippedOnUpdate(t *testing.T) {
	rules := []domain.AttributeMappingRule{
		{TargetPath: "externalId", SourceKind: domain.SourceKindAttribute, SourceKey: "id", ApplyOn: domain.ApplyCreateOnly},
		{TargetPath: "userName", SourceKind: domain.SourceKindAttribute, SourceKey: "preferred_username", ApplyOn: domain.ApplyCreateAndUpdate},
	}
	values := map[string]any{"id": "user-1", "preferred_username": "alice"}
	doc, err := BuildResource(rules, resolverFromMap(values), ApplyOnUpdate)
	if err != nil {
		t.Fatalf("BuildResource() error = %v", err)
	}
	if _, ok := doc["externalId"]; ok {
		t.Error("BuildResource() on update should skip create_only rules")
	}
	if doc["userName"] != "alice" {
		t.Errorf("doc[userName] = %v, want alice", doc["userName"])
	}
}
