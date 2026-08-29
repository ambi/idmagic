package domain

import (
	"errors"
	"io"
	"strings"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

func collectGroupCSV(t *testing.T, document string) ([]idmdomain.CSVRow, error) {
	t.Helper()
	schema := NewGroupCSVSchema()
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
	schema := NewGroupCSVSchema()
	want := []string{
		"id", "name", "description", "membership_type", "roles",
		"dynamic_rule_expression", "dynamic_rule_enabled", "lifecycle_action",
		"created_at", "updated_at",
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
	for _, key := range []string{"email", "attributes", "custom:cost_center", "password"} {
		if schema.Accepts(key) {
			t.Fatalf("schema accepts %q, but the Group allow list is closed", key)
		}
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
