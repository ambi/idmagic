package usecases

import (
	"testing"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
)

//spec:covers EX-IDMANAGEMENT-072-01: name による照合と行の重複の判定が、前後の空白と大文字と小文字を区別しないこと。
func TestGroupCSVMatchesNamesCaseInsensitively(t *testing.T) {
	f := newGroupImportFixture(t)
	f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual)
	rows := planRows(t, f, "name\n ENGINEERING \nEngineering\n")
	first := rowByNumber(t, rows, 2)
	if first.Before == nil || first.Before.ID != "group-1" {
		t.Fatalf("row 2 = %+v, want the existing engineering", first)
	}
	if second := rowByNumber(t, rows, 3); second.Error == nil || second.Error.Code != "duplicate_name" {
		t.Fatalf("row 3 = %+v, want duplicate_name", second)
	}
}

//spec:covers EX-IDMANAGEMENT-072-02: email のセルの表示名付きの形式を invalid_email で拒否し、アドレスだけのセルを小文字にすること。
func TestGroupCSVEmailCellAcceptsOnlyABareAddress(t *testing.T) {
	f := newGroupImportFixture(t)
	rows := planRows(t, f, "name,email\nsales,Sales <sales@example.test>\nsupport,Support@Example.TEST\n")
	if row := rowByNumber(t, rows, 2); row.Error == nil || row.Error.Code != "invalid_email" {
		t.Fatalf("row 2 = %+v, want invalid_email", row)
	}
	if row := rowByNumber(t, rows, 3); row.Group == nil || row.Group.Email == nil || *row.Group.Email != "support@example.test" {
		t.Fatalf("row 3 = %+v, want support@example.test", row)
	}
}

//spec:covers REQ-IDMANAGEMENT-072: 空の dynamic_rule_enabled のセルが現在の有効か無効かを保ち、式のセルの前後の空白を除くこと。
func TestGroupCSVEmptyEnabledCellKeepsTheCurrentState(t *testing.T) {
	f := newGroupImportFixture(t)
	f.seedGroup(t, "group-1", "dyn", groupdomain.GroupMembershipDynamic)
	if err := f.groupRepo.SaveDynamicRule(f.ctx, &groupdomain.DynamicGroupRule{
		GroupID: "group-1", TenantID: "acme", Expression: `user.preferred_username.size() > 0`, Enabled: true, Version: 4,
		ReferencedAttributes: []string{"preferred_username"}, CreatedAt: f.now, UpdatedAt: f.now,
	}); err != nil {
		t.Fatal(err)
	}
	rows := planRows(t, f, "id,dynamic_rule_expression,dynamic_rule_enabled\ngroup-1,  user.preferred_username.size() > 0  ,\n")
	row := rowByNumber(t, rows, 2)
	if row.Error != nil || row.Action != groupdomain.GroupImportUnchanged {
		t.Fatalf("row = %+v, want unchanged", row)
	}
	if row.Rule == nil || !row.Rule.Enabled || row.Rule.Expression != `user.preferred_username.size() > 0` {
		t.Fatalf("rule = %+v, want the trimmed expression still enabled", row.Rule)
	}
}
