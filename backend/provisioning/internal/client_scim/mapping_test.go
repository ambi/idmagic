package client_scim

import (
	"errors"
	"reflect"
	"testing"

	provisioningdomain "github.com/ambi/idmagic/backend/provisioning/internal/domain"
	provisioningports "github.com/ambi/idmagic/backend/provisioning/internal/ports"
)

//spec:covers RFC7643-OUT-CORE-RESOURCES: 単純パスは同名の属性になる。
func TestBuildResource_SimplePath(t *testing.T) {
	rules := []provisioningdomain.AttributeMappingRule{
		{TargetPath: "userName", SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "preferred_username", ApplyOn: provisioningdomain.ApplyCreateAndUpdate},
	}
	doc, err := BuildResource(rules, resolverFromMap(map[string]any{"preferred_username": "alice"}), ApplyOnCreate)
	if err != nil {
		t.Fatalf("BuildResource() error = %v", err)
	}
	if doc["userName"] != "alice" {
		t.Errorf("doc[userName] = %v, want alice", doc["userName"])
	}
}

//spec:covers RFC7643-OUT-CORE-RESOURCES: `.` 区切りのパスは入れ子のオブジェクトになる。
func TestBuildResource_NestedPath(t *testing.T) {
	rules := []provisioningdomain.AttributeMappingRule{
		{TargetPath: "name.givenName", SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "given_name", ApplyOn: provisioningdomain.ApplyCreateAndUpdate},
		{TargetPath: "name.familyName", SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "family_name", ApplyOn: provisioningdomain.ApplyCreateAndUpdate},
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

//spec:covers RFC7643-OUT-CORE-RESOURCES: 多値フィルターパスは `emails[type eq "work"].value` の 1 段だけを解釈する。
func TestBuildResource_MultiValuedFilterPath(t *testing.T) {
	rules := []provisioningdomain.AttributeMappingRule{
		{TargetPath: `emails[type eq "work"].value`, SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "email", ApplyOn: provisioningdomain.ApplyCreateAndUpdate},
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

//spec:covers RFC7643-OUT-CORE-RESOURCES: 定数の対応付けは属性の解決を経ずに値になる。
func TestBuildResource_ConstantSource(t *testing.T) {
	rules := []provisioningdomain.AttributeMappingRule{
		{TargetPath: "active", SourceKind: provisioningdomain.SourceKindConstant, ConstantValue: true, ApplyOn: provisioningdomain.ApplyCreateAndUpdate},
	}
	doc, err := BuildResource(rules, resolverFromMap(nil), ApplyOnCreate)
	if err != nil {
		t.Fatalf("BuildResource() error = %v", err)
	}
	if doc["active"] != true {
		t.Errorf("doc[active] = %v, want true", doc["active"])
	}
}

//spec:covers RFC7643-OUT-CORE-RESOURCES: 属性が空のときは既定値が入る。
func TestBuildResource_DefaultValueWhenSourceEmpty(t *testing.T) {
	rules := []provisioningdomain.AttributeMappingRule{
		{TargetPath: "displayName", SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "display_name", DefaultValue: "unknown", ApplyOn: provisioningdomain.ApplyCreateAndUpdate},
	}
	doc, err := BuildResource(rules, resolverFromMap(nil), ApplyOnCreate)
	if err != nil {
		t.Fatalf("BuildResource() error = %v", err)
	}
	if doc["displayName"] != "unknown" {
		t.Errorf("doc[displayName] = %v, want unknown (default)", doc["displayName"])
	}
}

//spec:covers RFC7643-OUT-CORE-RESOURCES: required の対応付けが解決できないプロビジョニングタスクは、部分的な本文を送らずに失敗する。
func TestBuildResource_RequiredMissingFailsClosed(t *testing.T) {
	rules := []provisioningdomain.AttributeMappingRule{
		{TargetPath: "userName", SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "preferred_username", Required: true, ApplyOn: provisioningdomain.ApplyCreateAndUpdate},
	}
	_, err := BuildResource(rules, resolverFromMap(nil), ApplyOnCreate)
	// プロビジョニングエンジンはこの種類を見分けて、再試行せずにプロビジョニングタスクを失敗させる。
	if !errors.Is(err, provisioningports.ErrRequiredAttributeUnresolved) {
		t.Errorf("BuildResource() error = %v, want ports.ErrRequiredAttributeUnresolved", err)
	}
}

//spec:covers RFC7643-OUT-EXTERNAL-ID: externalId は作成時だけ送り、更新では送り直さない。
func TestBuildResource_CreateOnlySkippedOnUpdate(t *testing.T) {
	rules := []provisioningdomain.AttributeMappingRule{
		{TargetPath: "externalId", SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "id", ApplyOn: provisioningdomain.ApplyCreateOnly},
		{TargetPath: "userName", SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "preferred_username", ApplyOn: provisioningdomain.ApplyCreateAndUpdate},
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

// 対象パスは `.` で区切った単純パスと 1 段の多値フィルターパスしか表現できないため、
// 拡張スキーマの URN で修飾したパスを書いても、URN を鍵とする拡張オブジェクトにはならない。
//
//spec:covers RFC7643-OUT-SCHEMA-EXTENSIONS: 拡張スキーマの属性を送らない。
func TestBuildResource_OmitsExtensionSchemaAttributes(t *testing.T) {
	const enterpriseURN = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
	rules := []provisioningdomain.AttributeMappingRule{
		{TargetPath: enterpriseURN + ":employeeNumber", SourceKind: provisioningdomain.SourceKindAttribute, SourceKey: "employee_number", ApplyOn: provisioningdomain.ApplyCreateAndUpdate},
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
