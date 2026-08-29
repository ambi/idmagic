package usecases

// 計画器の内側の単体境界。受入テストが通す経路のうち、行ごとの判定規則だけを
// リポジトリの memory アダプター越しに直接確かめる。

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func planRows(t *testing.T, f *groupImportFixture, document string) []groupdomain.GroupImportRowPlan {
	t.Helper()
	_, rows := f.preview(t, document)
	return rows
}

// scenario REQ-IDMANAGEMENT-026: `id` で解決し、`name` が別の Group を指す行は
// identifier_mismatch。一致すれば改名として計画する。
func TestGroupImportPlannerResolvesByIDThenName(t *testing.T) {
	f := newGroupImportFixture(t)
	f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual, "catalog:read")
	f.seedGroup(t, "group-2", "sales", groupdomain.GroupMembershipManual)
	f.seedGroup(t, "group-3", "ops", groupdomain.GroupMembershipManual)

	rows := planRows(t, f, "id,name\ngroup-1,platform\ngroup-3,sales\n,ops\n,fresh\nunknown-id,x\n")
	if got := rowByNumber(t, rows, 2); got.Action != groupdomain.GroupImportUpdate {
		t.Fatalf("rename row = %+v, want updated", got)
	}
	if got := rowByNumber(t, rows, 3); got.Error == nil || got.Error.Code != "identifier_mismatch" {
		t.Fatalf("row 3 = %+v, want identifier_mismatch", got.Error)
	}
	if got := rowByNumber(t, rows, 4); got.Action != groupdomain.GroupImportUnchanged {
		t.Fatalf("name-resolved row = %+v, want unchanged", got)
	}
	if got := rowByNumber(t, rows, 5); got.Action != groupdomain.GroupImportCreate {
		t.Fatalf("unknown name row = %+v, want created", got)
	}
	if got := rowByNumber(t, rows, 6); got.Error == nil || got.Error.Code != "target_not_found" {
		t.Fatalf("unknown id row = %+v, want target_not_found", got.Error)
	}
}

// scenario REQ-IDMANAGEMENT-026: ファイル内で同じ対象や同じ最終 name を複数行が
// 指せば拒否する。リポジトリを引かずに判定できる衝突である。
func TestGroupImportPlannerRefusesDuplicateTargetsWithinOneFile(t *testing.T) {
	f := newGroupImportFixture(t)
	f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual)

	rows := planRows(t, f, "id,name\ngroup-1,engineering\ngroup-1,engineering\n,fresh\n,fresh\n")
	if got := rowByNumber(t, rows, 3); got.Error == nil || got.Error.Code != "duplicate_target" {
		t.Fatalf("row 3 = %+v, want duplicate_target", got.Error)
	}
	if got := rowByNumber(t, rows, 5); got.Error == nil || got.Error.Code != "duplicate_name" {
		t.Fatalf("row 5 = %+v, want duplicate_name", got.Error)
	}
}

// scenario REQ-IDMANAGEMENT-027: 列が無ければ維持、optional 列の空は clear、
// roles の空は空集合。読み取り専用列は受理して無視する。
func TestGroupImportPlannerHonoursColumnPresence(t *testing.T) {
	f := newGroupImportFixture(t)
	description := "the platform team"
	group := f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual, "catalog:read")
	group.Description = &description
	if err := f.groupRepo.Save(f.ctx, group); err != nil {
		t.Fatal(err)
	}

	rows := planRows(t, f, "id,created_at,updated_at\ngroup-1,2001-01-01T00:00:00Z,2001-01-01T00:00:00Z\n")
	planned := rowByNumber(t, rows, 2)
	if planned.Action != groupdomain.GroupImportUnchanged {
		t.Fatalf("read-only-only row = %+v, want unchanged", planned)
	}

	rows = planRows(t, f, "id,description,roles\ngroup-1,,\n")
	planned = rowByNumber(t, rows, 2)
	if planned.Action != groupdomain.GroupImportUpdate {
		t.Fatalf("clearing row = %+v, want updated", planned)
	}
	if planned.Group.Description != nil {
		t.Fatalf("description = %v, want cleared by the present-empty cell", *planned.Group.Description)
	}
	if len(planned.Group.Roles) != 0 {
		t.Fatalf("roles = %v, want the empty set", planned.Group.Roles)
	}
}

// scenario REQ-IDMANAGEMENT-026: dynamic rule は片方の列だけを与えた行でも、
// 維持された相方と組み合わせた最終状態として検証する。
func TestGroupImportPlannerValidatesTheDynamicRuleAsAFinalState(t *testing.T) {
	f := newGroupImportFixture(t)
	f.plan.SchemaRepo = ruleSchemaRepo{}
	f.seedGroup(t, "manual-group", "engineering", groupdomain.GroupMembershipManual)
	f.seedGroup(t, "dynamic-group", "sales", groupdomain.GroupMembershipDynamic)
	f.seedGroup(t, "other-dynamic-group", "support", groupdomain.GroupMembershipDynamic)

	rows := planRows(t, f, "id,dynamic_rule_expression,dynamic_rule_enabled\n"+
		"manual-group,\"user.department == \"\"Eng\"\"\",false\n"+
		"dynamic-group,,true\n"+
		"other-dynamic-group,\"user.unknown_attribute == \"\"x\"\"\",false\n")
	for number, want := range map[int]idmdomain.CSVErrorCode{2: "invalid_dynamic_rule", 3: "invalid_dynamic_rule", 4: "invalid_dynamic_rule"} {
		got := rowByNumber(t, rows, number)
		if got.Error == nil || got.Error.Code != want {
			t.Fatalf("row %d = %+v, want %q", number, got.Error, want)
		}
	}

	// 有効化の列だけを与えた行も、維持された式 (ここでは規則が無いので空) と
	// 組み合わせた最終状態として検証する。列ごとに見ると通ってしまう組み合わせである。
	rows = planRows(t, f, "id,dynamic_rule_enabled\ndynamic-group,true\n")
	if got := rowByNumber(t, rows, 2); got.Error == nil || got.Error.Code != "invalid_dynamic_rule" {
		t.Fatalf("enabling a rule that has no expression = %+v, want invalid_dynamic_rule", got.Error)
	}

	rows = planRows(t, f, "id,dynamic_rule_expression,dynamic_rule_enabled\ndynamic-group,\"user.department == \"\"Eng\"\"\",true\n")
	planned := rowByNumber(t, rows, 2)
	if planned.Action != groupdomain.GroupImportUpdate || planned.Rule == nil {
		t.Fatalf("row = %+v, want an updated row carrying a rule", planned)
	}
	if !planned.Rule.Enabled || planned.Rule.Expression != `user.department == "Eng"` {
		t.Fatalf("rule = %+v", planned.Rule)
	}
	if len(planned.Rule.ReferencedAttributes) != 1 || planned.Rule.ReferencedAttributes[0] != "department" {
		t.Fatalf("referenced attributes = %v, want [department]", planned.Rule.ReferencedAttributes)
	}

	// 有効化された規則を持つ dynamic group の式だけを変えた行は、維持された
	// enabled=true と組み合わせて検証されたうえで受理される。
	f.seedGroup(t, "ruled-group", "ops", groupdomain.GroupMembershipDynamic)
	if err := f.groupRepo.SaveDynamicRule(f.ctx, &groupdomain.DynamicGroupRule{
		GroupID: "ruled-group", TenantID: "acme", Expression: `user.department == "Ops"`,
		Enabled: true, Version: 1, CreatedAt: f.now, UpdatedAt: f.now,
	}); err != nil {
		t.Fatal(err)
	}
	rows = planRows(t, f, "id,dynamic_rule_expression\nruled-group,\"user.department == \"\"Platform\"\"\"\n")
	planned = rowByNumber(t, rows, 2)
	if planned.Action != groupdomain.GroupImportUpdate || planned.Rule == nil || !planned.Rule.Enabled {
		t.Fatalf("row = %+v, want the retained enabled=true carried into the final state", planned)
	}
	if planned.Rule.Version != 2 {
		t.Fatalf("rule version = %d, want the retained version bumped", planned.Rule.Version)
	}
}

// scenario REQ-IDMANAGEMENT-026: create 行は membership_type を選べ、空と列欠落は
// manual になる。
func TestGroupImportPlannerChoosesMembershipTypeOnlyAtCreation(t *testing.T) {
	f := newGroupImportFixture(t)
	rows := planRows(t, f, "name,membership_type\nfresh-dynamic,dynamic\nfresh-default,\nfresh-bogus,static\n")
	if got := rowByNumber(t, rows, 2); got.Action != groupdomain.GroupImportCreate || got.Group.MembershipType != groupdomain.GroupMembershipDynamic {
		t.Fatalf("row 2 = %+v, want a created dynamic group", got)
	}
	if got := rowByNumber(t, rows, 3); got.Action != groupdomain.GroupImportCreate || got.Group.MembershipType != groupdomain.GroupMembershipManual {
		t.Fatalf("row 3 = %+v, want a created manual group", got)
	}
	if got := rowByNumber(t, rows, 4); got.Error == nil || got.Error.Code != "invalid_membership_type" {
		t.Fatalf("row 4 = %+v, want invalid_membership_type", got.Error)
	}
}

// scenario REQ-IDMANAGEMENT-026: 適用は古い計画を実行せず、現在状態から再計画する。
func TestGroupImportApplyReplansAgainstCurrentState(t *testing.T) {
	f := newGroupImportFixture(t)
	group := f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual, "catalog:read")
	document := "id,name,roles\ngroup-1,engineering,catalog:read|invoice:read\n"

	summary, _ := f.preview(t, document)
	if summary.UpdatedRows != 1 {
		t.Fatalf("preview summary = %+v, want 1 update", summary)
	}

	// 別の操作が、プレビューが計画したのと同じ最終状態を先に作る。
	group.Roles = []string{"catalog:read", "invoice:read"}
	if err := f.groupRepo.Save(f.ctx, group); err != nil {
		t.Fatal(err)
	}

	applied, err := ApplyGroupImport(f.ctx, f.apply, strings.NewReader(document), idmdomain.DefaultCSVTransferPolicy(),
		"operator", f.now.Add(time.Hour), nil)
	if err != nil {
		t.Fatal(err)
	}
	if applied.UnchangedRows != 1 || applied.UpdatedRows != 0 {
		t.Fatalf("apply summary = %+v, want the stale plan replaced by unchanged", applied)
	}
}

// ruleSchemaRepo は動的規則の式が参照できる属性定義だけを供給する読み取り専用の
// スタブ。CSV 経路の検証対象は式の妥当性であり、スキーマの永続化ではない。
type ruleSchemaRepo struct{}

func (ruleSchemaRepo) FindByTenant(context.Context, string) (*userdomain.TenantUserAttributeSchema, error) {
	department := userdomain.UserAttributeDef{Key: "department", Type: idmdomain.AttributeTypeString}
	return &userdomain.TenantUserAttributeSchema{TenantID: "acme", Attributes: []userdomain.UserAttributeDef{department}}, nil
}

func (ruleSchemaRepo) Save(context.Context, *userdomain.TenantUserAttributeSchema) error {
	return errReadOnlySchemaRepo
}

func (ruleSchemaRepo) Delete(context.Context, string) error { return errReadOnlySchemaRepo }

var errReadOnlySchemaRepo = errors.New("the group import test schema repository is read-only")

var _ tenantports.TenantUserAttributeSchemaRepository = ruleSchemaRepo{}

// scenario REQ-IDMANAGEMENT-026 / REQ-IDMANAGEMENT-027: `email` と `custom:<key>` は
// 他の書き込み可能列と同じ規則に従う。列が無ければ維持、空セルは消去、値が不正なら
// 行を拒否して連絡先も属性も変更しない。
func TestGroupImportPlannerAppliesEmailAndCustomAttributes(t *testing.T) {
	f := newGroupImportFixture(t)
	email := "eng@example.test"
	center := "CC-100"
	group := f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual, "catalog:read")
	group.Email = &email
	group.Attributes = map[string]userdomain.AttributeValue{
		"cost_center": {Type: idmdomain.AttributeTypeString, String: &center},
	}
	if err := f.groupRepo.Save(f.ctx, group); err != nil {
		t.Fatal(err)
	}

	// 列が無ければ維持する。
	if got := rowByNumber(t, planRows(t, f, "id\ngroup-1\n"), 2); got.Action != groupdomain.GroupImportUnchanged {
		t.Fatalf("row without the columns = %+v, want unchanged", got)
	}

	// 値を書けば更新する。
	rows := planRows(t, f, "id,email,custom:cost_center\ngroup-1,Sales@Example.test,CC-200\n")
	planned := rowByNumber(t, rows, 2)
	if planned.Action != groupdomain.GroupImportUpdate {
		t.Fatalf("row = %+v, want updated", planned)
	}
	if planned.Group.Email == nil || *planned.Group.Email != "sales@example.test" {
		t.Fatalf("email = %v, want the normalized address", planned.Group.Email)
	}
	if value := planned.Group.Attributes["cost_center"]; value.String == nil || *value.String != "CC-200" {
		t.Fatalf("cost_center = %+v, want CC-200", value)
	}

	// 存在する空セルは消す。
	rows = planRows(t, f, "id,email,custom:cost_center\ngroup-1,,\n")
	planned = rowByNumber(t, rows, 2)
	if planned.Action != groupdomain.GroupImportUpdate || planned.Group.Email != nil {
		t.Fatalf("clearing row = %+v, want the email cleared", planned)
	}
	if _, present := planned.Group.Attributes["cost_center"]; present {
		t.Fatalf("attributes = %+v, want the custom attribute cleared", planned.Group.Attributes)
	}

	// 不正な値は行を拒否し、保存済みの値には触れない。
	if got := rowByNumber(t, planRows(t, f, "id,email\ngroup-1,not-an-address\n"), 2); got.Error == nil || got.Error.Code != "invalid_email" {
		t.Fatalf("malformed email = %+v, want invalid_email", got.Error)
	}
	if got := rowByNumber(t, planRows(t, f, "id,custom:cost_center\ngroup-1,\"a\nb\"\n"), 2); got.Action != groupdomain.GroupImportUpdate {
		t.Fatalf("a quoted multi-line custom value = %+v, want it accepted verbatim", got)
	}
	f.applyCSV(t, "id,email\ngroup-1,not-an-address\n")
	after, err := f.groupRepo.FindByID(f.ctx, "acme", "group-1")
	if err != nil || after == nil {
		t.Fatal(err)
	}
	if after.Email == nil || *after.Email != "eng@example.test" {
		t.Fatalf("email = %v, want the refusal to leave it untouched", after.Email)
	}
	if value := after.Attributes["cost_center"]; value.String == nil || *value.String != "CC-100" {
		t.Fatalf("cost_center = %+v, want the refusal to leave it untouched", value)
	}
}

// scenario REQ-IDMANAGEMENT-026: テナントスキーマに無い `custom:<key>` 列は
// ヘッダーの時点でファイルごと拒否する。未検証の属性が CSV から入る余地を作らない。
func TestGroupImportPlannerRefusesUndeclaredCustomColumns(t *testing.T) {
	f := newGroupImportFixture(t)
	f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual)

	_, err := PlanGroupImport(f.ctx, f.plan, strings.NewReader("id,custom:unknown\ngroup-1,x\n"),
		idmdomain.DefaultCSVTransferPolicy(), nil)
	csvErr, ok := errors.AsType[*idmdomain.CSVError](err)
	if !ok || csvErr.Code != idmdomain.CSVErrorInvalidHeader {
		t.Fatalf("error = %v, want invalid_header for an undeclared custom column", err)
	}
}
