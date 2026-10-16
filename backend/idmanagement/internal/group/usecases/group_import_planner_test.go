package usecases

// 計画器の内側の単体境界。受入テストが通す経路のうち、行ごとの判定規則だけを
// リポジトリの memory アダプター越しに直接確かめる。

import (
	"context"
	"errors"
	"slices"
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

// 026-04 が並べる 3 つの識別子の誤りのうち、食い違いをここが持つ。重複は
// TestGroupImportPlannerRefusesDuplicateTargetsWithinOneFile、欠落は
// TestGroupImportPlannerRefusesRowsWithNoIdentifier が持つ。
//
//spec:covers EX-IDMANAGEMENT-026-04: id と name が別の Group を指す行が identifier_mismatch で rejected になること。
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

// リポジトリを引かずに判定できる衝突である。対象の重複と最終 name の重複は
// 別のコードを持つので、両方を並べて読む。
//
//spec:covers EX-IDMANAGEMENT-026-04: 同じ対象または同じ最終 name を複数行が指すファイルが、duplicate_target と duplicate_name で rejected になること。
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

// 列が無ければ維持、optional 列の空は clear、roles の空は空集合。
//
//spec:covers EX-IDMANAGEMENT-027-04: 読み取り専用列だけを編集した行が受理されたうえで unchanged になること。
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

// 片方の列だけを与えた行でも、維持された相方と組み合わせた最終状態として検証する。
// 列ごとに見ると通ってしまう組み合わせがあるので、そこが具体例の要点である。
//
//spec:covers EX-IDMANAGEMENT-026-09: manual グループへの式、式の無いまま有効化する行、未定義の属性や許可外の関数を参照する式が、いずれも invalid_dynamic_rule で rejected になること。
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

// 別の操作がプレビューと同じ最終状態を先に作ったとき、適用は updated ではなく
// unchanged になる。判定が保存済みの計画からではなく現在状態から出ている証拠である。
//
//spec:covers EX-IDMANAGEMENT-026-06: プレビュー後に Group の状態が別の操作で変わったとき、適用が古い計画を実行せず現在状態から再判定すること。
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

// `email` と `custom:<key>` は他の書き込み可能列と同じ規則に従う。列が無ければ維持、
// 空セルは消去、値が不正なら行を拒否して連絡先も属性も変更しない。
//
//spec:covers EX-IDMANAGEMENT-026-10: email の形式違反が invalid_email で rejected になり、連絡先もカスタム属性も変更されないこと。
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

// 未検証の属性が CSV から入る余地を作らない。行ごとの拒否ではなくヘッダーの時点で
// ファイルごと落ちるので、1 行も読まれないことがこの拒否の効果である。
//
//spec:covers EX-IDMANAGEMENT-026-10: テナントスキーマに無い custom:<key> 列を持つファイルが invalid_header でファイルごと拒否されること。
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

// 予約ロールを新しく加える行は rejected になり、適用まで進めても Group は変わらない。
//
// テナントは acme、すなわち制御面ではない。プレビューだけを直して適用の再計画を
// 直さない実装と区別するため、同じ文書を適用にも通して Group を読み直す。
//
//spec:covers EX-IDMANAGEMENT-032-07: 制御面テナント以外のテナントの Group CSV の行が `system_admin` を新しく加えると `roles` 列を指す invalid_roles で rejected になり、その Group のロールも他の項目も変更されないこと。
func TestGroupImportPlannerRefusesTheReservedRole(t *testing.T) {
	f := newGroupImportFixture(t)
	f.seedGroup(t, "group-engineering", "engineering", groupdomain.GroupMembershipManual, "catalog:read")

	document := "id,name,description,roles\n" +
		"group-engineering,engineering,escalation path,system_admin\n"

	rows := planRows(t, f, document)
	if len(rows) != 1 {
		t.Fatalf("rows=%+v", rows)
	}
	if rows[0].Action != groupdomain.GroupImportRejected {
		t.Fatalf("Action=%s, want rejected", rows[0].Action)
	}
	if rows[0].Error == nil || rows[0].Error.Code != "invalid_roles" || rows[0].Error.Column != "roles" {
		t.Fatalf("Error=%+v, want column=roles code=invalid_roles", rows[0].Error)
	}

	// 適用も同じ計画器を通ること。ロールだけでなく同じ行が与えた description も入らない。
	if summary := f.applyCSV(t, document); summary.RejectedRows != 1 || summary.UpdatedRows != 0 {
		t.Fatalf("apply summary=%+v", summary)
	}
	stored, err := f.groupRepo.FindByID(f.ctx, "acme", "group-engineering")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(stored.Roles, []string{"catalog:read"}) {
		t.Fatalf("拒否されたのに roles=%v が変わった", stored.Roles)
	}
	if stored.Description != nil {
		t.Fatalf("拒否されたのに description=%q が入った", *stored.Description)
	}

	// 対照: 予約ロール以外の名前なら同じ行が通り、description も入る。
	accepted := "id,name,description,roles\n" +
		"group-engineering,engineering,escalation path,catalog:write\n"
	if summary := f.applyCSV(t, accepted); summary.UpdatedRows != 1 {
		t.Fatalf("前提が壊れている: 通常ロールの適用 summary=%+v", summary)
	}
}
