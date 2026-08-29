package domain

import (
	"errors"
	"io"
	"strings"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

func testGroupAttributeDefs() []GroupAttributeDef {
	return []GroupAttributeDef{
		{Key: "cost_center", Type: idmdomain.AttributeTypeString},
		{Key: "headcount", Type: idmdomain.AttributeTypeNumber},
		{Key: "tags", Type: idmdomain.AttributeTypeStringArray, MultiValued: true},
	}
}

func collectGroupCSV(t *testing.T, document string) ([]idmdomain.CSVRow, error) {
	t.Helper()
	schema, err := NewGroupCSVSchema(testGroupAttributeDefs())
	if err != nil {
		t.Fatal(err)
	}
	reader, err := idmdomain.NewCSVReader(strings.NewReader(document), schema.Accepts, idmdomain.DefaultCSVTransferPolicy())
	if err != nil {
		return nil, err
	}
	var rows []idmdomain.CSVRow
	for {
		record, err := reader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return rows, err
		}
		if record.Row != nil {
			rows = append(rows, *record.Row)
		}
	}
	return rows, nil
}

// scenario REQ-IDMANAGEMENT-026: 機械キーの語彙は閉じており、順序と部分集合は
// 自由、列の欠落と空セルは別の意味を持つ。
func TestGroupCSVSchemaIsAClosedMachineKeyVocabulary(t *testing.T) {
	schema, err := NewGroupCSVSchema(testGroupAttributeDefs())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"id", "name", "description", "email", "membership_type", "roles",
		"dynamic_rule_expression", "dynamic_rule_enabled", "lifecycle_action",
		"created_at", "updated_at",
		"custom:cost_center", "custom:headcount", "custom:tags",
	}
	got := schema.Columns()
	if len(got) != len(want) {
		t.Fatalf("columns = %v, want %v", got, want)
	}
	for i, key := range want {
		if got[i].Key != key {
			t.Fatalf("columns[%d] = %q, want %q", i, got[i].Key, key)
		}
	}
	// Group には union すべき組み込みカタログが無いので `attr:<key>` は持たない。
	// スキーマに無い属性キーも受理しない。
	for _, key := range []string{"attributes", "attr:cost_center", "custom:unknown", "password"} {
		if schema.Accepts(key) {
			t.Fatalf("schema accepts %q, but the Group allow list is closed", key)
		}
	}
	if schema.Column("custom:cost_center").Attribute == nil {
		t.Fatal("a custom column must carry the attribute definition that fixes its lexical form")
	}
	if !schema.Column("lifecycle_action").WriteOnly {
		t.Fatal("lifecycle_action must be write-only so an export never carries an intent to delete")
	}
	if !schema.Column("created_at").ReadOnly || !schema.Column("updated_at").ReadOnly {
		t.Fatal("timestamps must be read-only")
	}
}

// scenario REQ-IDMANAGEMENT-026: 列の欠落 (維持) と存在する空セル (clear) を
// 別々に保持する。
func TestGroupCSVRowSeparatesAbsentColumnsFromEmptyCells(t *testing.T) {
	rows, err := collectGroupCSV(t, "id,name,description\ngroup-1,engineering,\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if cell, ok := rows[0].Cell("description"); !ok || !cell.Present || cell.Raw != "" {
		t.Fatalf("description = %+v present=%v; want present and empty", cell, ok)
	}
	if _, ok := rows[0].Cell("roles"); ok {
		t.Fatal("roles must be absent when its header is absent")
	}
}

// scenario REQ-IDMANAGEMENT-028: `lifecycle_action` の語彙は閉じている。未知の値は
// 既知の値へ丸めず拒否し、空と列欠落はどちらも lifecycle を変えない。
func TestGroupCSVLifecycleActionVocabularyIsClosed(t *testing.T) {
	for raw, want := range map[string]GroupCSVLifecycleAction{
		"":       GroupCSVLifecycleNone,
		"delete": GroupCSVLifecycleDelete,
	} {
		got, err := ParseGroupCSVLifecycleAction(raw)
		if err != nil || got != want {
			t.Fatalf("ParseGroupCSVLifecycleAction(%q) = %q, %v; want %q", raw, got, err, want)
		}
	}
	for _, raw := range []string{"purge", "DELETE", "soft_delete", "disable", "delete ", "restore"} {
		if got, err := ParseGroupCSVLifecycleAction(raw); err == nil {
			t.Fatalf("ParseGroupCSVLifecycleAction(%q) = %q, want a refusal rather than a rounded value", raw, got)
		}
	}
}

// scenario REQ-IDMANAGEMENT-026: membership_type の語彙も閉じており、空は manual。
func TestGroupCSVMembershipTypeVocabularyIsClosed(t *testing.T) {
	for raw, want := range map[string]GroupMembershipType{
		"":        GroupMembershipManual,
		"manual":  GroupMembershipManual,
		"dynamic": GroupMembershipDynamic,
	} {
		got, err := ParseGroupCSVMembershipType(raw)
		if err != nil || got != want {
			t.Fatalf("ParseGroupCSVMembershipType(%q) = %q, %v; want %q", raw, got, err, want)
		}
	}
	for _, raw := range []string{"Manual", "static", "rule", " dynamic"} {
		if got, err := ParseGroupCSVMembershipType(raw); err == nil {
			t.Fatalf("ParseGroupCSVMembershipType(%q) = %q, want a refusal", raw, got)
		}
	}
}

// scenario REQ-IDMANAGEMENT-027: roles の字句形は `|` 区切りで、空セルは空集合、
// 空要素は拒否する。export と import は同じ字句形を共有する。
func TestGroupCSVRolesLexicalFormRoundTrips(t *testing.T) {
	roles, err := ParseGroupCSVRoles("")
	if err != nil || len(roles) != 0 {
		t.Fatalf("empty roles = %v, %v; want an empty set", roles, err)
	}
	roles, err = ParseGroupCSVRoles("catalog:read|invoice:read")
	if err != nil || len(roles) != 2 {
		t.Fatalf("roles = %v, %v", roles, err)
	}
	if FormatGroupCSVRoles(roles) != "catalog:read|invoice:read" {
		t.Fatalf("format = %q", FormatGroupCSVRoles(roles))
	}
	for _, raw := range []string{"|", "a||b", "a|"} {
		if _, err := ParseGroupCSVRoles(raw); err == nil {
			t.Fatalf("ParseGroupCSVRoles(%q) accepted an empty element", raw)
		}
	}
}

// scenario REQ-IDMANAGEMENT-026: 行は `id` を優先し、無ければ `name` で識別する。
// どちらも無い行は識別できない。
func TestGroupCSVIdentifierPrefersIDAndFallsBackToName(t *testing.T) {
	rows, err := collectGroupCSV(t, "id,name\ngroup-1,engineering\n,sales\n,\n")
	if err != nil {
		t.Fatal(err)
	}
	if id, code := GroupCSVIdentifierOf(rows[0]); code != "" || id.ID != "group-1" || id.Name != "engineering" {
		t.Fatalf("row 1 identifier = %+v, code=%q", id, code)
	}
	if id, code := GroupCSVIdentifierOf(rows[1]); code != "" || id.ID != "" || id.Name != "sales" {
		t.Fatalf("row 2 identifier = %+v, code=%q", id, code)
	}
	if _, code := GroupCSVIdentifierOf(rows[2]); code != "missing_identifier" {
		t.Fatalf("row 3 code = %q, want missing_identifier", code)
	}
}

// scenario REQ-IDMANAGEMENT-026: `email` は書き込み可能列であり、空セルは連絡先を
// 消す。形式を満たさない値は拒否する。`User.email` より機構は少ないが、字句形の
// 扱いは同じである。
func TestGroupCSVEmailCellIsWritableAndClearable(t *testing.T) {
	value, cleared, err := ParseGroupCSVEmailCell("")
	if err != nil || !cleared || value != nil {
		t.Fatalf("empty email = %v cleared=%v err=%v; want a clear", value, cleared, err)
	}
	value, cleared, err = ParseGroupCSVEmailCell(" Sales@Example.test ")
	if err != nil || cleared || value == nil || *value != "sales@example.test" {
		t.Fatalf("email = %v cleared=%v err=%v; want the normalized address", value, cleared, err)
	}
	for _, raw := range []string{"not-an-address", "a@", "@example.test", "a b@example.test"} {
		if _, _, err := ParseGroupCSVEmailCell(raw); err == nil {
			t.Fatalf("ParseGroupCSVEmailCell(%q) accepted a malformed address", raw)
		}
	}
}

// scenario REQ-IDMANAGEMENT-027: `custom:<key>` のセルは宣言された型の正規字句形で
// 解釈し、export はその逆射影を書く。往復して元の値に戻る。
func TestGroupCSVAttributeCellRoundTripsThroughItsDeclaredType(t *testing.T) {
	defs := testGroupAttributeDefs()
	byKey := map[string]GroupAttributeDef{}
	for _, def := range defs {
		byKey[def.Key] = def
	}
	for raw, key := range map[string]string{
		"CC-100":    "cost_center",
		"42":        "headcount",
		`["a","b"]`: "tags",
	} {
		def := byKey[key]
		value, cleared, err := ParseGroupCSVAttributeCell(raw, def)
		if err != nil || cleared {
			t.Fatalf("ParseGroupCSVAttributeCell(%q, %q) = cleared=%v err=%v", raw, key, cleared, err)
		}
		formatted, err := FormatGroupCSVAttributeCell(value, def)
		if err != nil || formatted != raw {
			t.Fatalf("round trip of %q = %q, err=%v", raw, formatted, err)
		}
	}

	// 型に合わない値は拒否する。既知の型へ丸めない。
	for key, raw := range map[string]string{"headcount": "many", "tags": "a,b", "cost_center": ""} {
		def := byKey[key]
		value, cleared, err := ParseGroupCSVAttributeCell(raw, def)
		if raw == "" {
			if err != nil || !cleared {
				t.Fatalf("optional empty %q: cleared=%v err=%v; want a clear", key, cleared, err)
			}
			continue
		}
		if err == nil {
			t.Fatalf("ParseGroupCSVAttributeCell(%q, %q) accepted %v", raw, key, value)
		}
	}

	// required な属性の空セルは拒否する。維持と消去を取り違えないためである。
	required := GroupAttributeDef{Key: "cost_center", Type: idmdomain.AttributeTypeString, Required: true}
	if _, _, err := ParseGroupCSVAttributeCell("", required); err == nil {
		t.Fatal("clearing a required attribute must be refused")
	}
}
