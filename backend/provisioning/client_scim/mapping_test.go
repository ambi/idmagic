package client_scim

import (
	"reflect"
	"testing"

	"github.com/ambi/idmagic/backend/provisioning/domain"
)

// RFC7643-OUT-CORE-RESOURCES: 単純パスは同名の属性になる。
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

// RFC7643-OUT-CORE-RESOURCES: `.` 区切りのパスは入れ子のオブジェクトになる。
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

// RFC7643-OUT-CORE-RESOURCES: 多値フィルターパスは `emails[type eq "work"].value` の 1 段だけを解釈する。
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

// RFC7643-OUT-CORE-RESOURCES: 定数の対応付けは属性の解決を経ずに値になる。
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

// RFC7643-OUT-CORE-RESOURCES: 属性が空のときは既定値が入る。
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

// RFC7643-OUT-CORE-RESOURCES: required の対応付けが解決できない配信は、部分的な本文を送らずに失敗する。
func TestBuildResource_RequiredMissingFailsClosed(t *testing.T) {
	rules := []domain.AttributeMappingRule{
		{TargetPath: "userName", SourceKind: domain.SourceKindAttribute, SourceKey: "preferred_username", Required: true, ApplyOn: domain.ApplyCreateAndUpdate},
	}
	_, err := BuildResource(rules, resolverFromMap(nil), ApplyOnCreate)
	if err == nil {
		t.Error("BuildResource() with a required unresolved attribute should return an error")
	}
}

// RFC7643-OUT-EXTERNAL-ID: externalId は作成時だけ送り、更新では送り直さない。
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

// RFC7643-OUT-SCHEMA-EXTENSIONS: 拡張スキーマの属性を送らない。
// 対象パスは `.` で区切った単純パスと 1 段の多値フィルターパスしか表現できないため、
// 拡張スキーマの URN で修飾したパスを書いても、URN を鍵とする拡張オブジェクトにはならない。
func TestBuildResource_OmitsExtensionSchemaAttributes(t *testing.T) {
	const enterpriseURN = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
	rules := []domain.AttributeMappingRule{
		{TargetPath: enterpriseURN + ":employeeNumber", SourceKind: domain.SourceKindAttribute, SourceKey: "employee_number", ApplyOn: domain.ApplyCreateAndUpdate},
	}
	doc, err := BuildResource(rules, resolverFromMap(map[string]any{"employee_number": "E-1"}), ApplyOnCreate)
	if err != nil {
		t.Fatalf("BuildResource() error = %v", err)
	}
	if _, ok := doc[enterpriseURN]; ok {
		t.Errorf("doc[%s] が存在する。拡張スキーマの属性は送らない", enterpriseURN)
	}
	if _, ok := doc["schemas"]; ok {
		t.Error("doc[schemas] が存在する。拡張スキーマの URN を広告しない")
	}
}
