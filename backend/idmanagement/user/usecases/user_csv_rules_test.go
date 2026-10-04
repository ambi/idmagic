package usecases

import (
	"errors"
	"slices"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// userCSVRulesDeps は組み込みの拡張属性 `department` とテナント定義の属性 `cost_code` の
// 両方を実効スキーマに持つ。片方しか無いと、接頭辞の取り違えが観測できない。
func userCSVRulesDeps(repo *usermemory.UserRepository, guard perUserImportOwnershipGuard) UserImportPlanDeps {
	deps := importPlannerDeps(repo, guard)
	deps.SchemaReader = importSchemaReader{defs: []userdomain.UserAttributeDef{
		{Key: "department", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityPrivate},
		{Key: "cost_code", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityPrivate},
	}}
	return deps
}

func rowCodes(plan userdomain.UserImportPlan) []string {
	out := make([]string, len(plan.Rows))
	for i, row := range plan.Rows {
		out[i] = string(row.Action)
		if row.Error != nil {
			out[i] += ":" + string(row.Error.Code)
		}
	}
	return out
}

//spec:covers EX-IDMANAGEMENT-055-01: 組み込みの拡張属性を attr:<key>、テナント定義の属性を custom:<key> の列で読むこと。
func TestUserCSVReadsBuiltinAttributesWithAttrAndTenantAttributesWithCustom(t *testing.T) {
	repo := usermemory.NewUserRepository()
	plan, err := planUserImportForTest(importPlannerContext(), userCSVRulesDeps(repo, perUserImportOwnershipGuard{}),
		"preferred_username,attr:department,custom:cost_code\ndave,Platform,CC-1\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Rows) != 1 || plan.Rows[0].Action != userdomain.UserImportCreate {
		t.Fatalf("rows=%v, want one create", rowCodes(plan))
	}
	attrs := plan.Rows[0].User.Attributes
	if *attrs["department"].String != "Platform" || *attrs["cost_code"].String != "CC-1" {
		t.Fatalf("attributes=%+v", attrs)
	}
}

//spec:covers EX-IDMANAGEMENT-055-02: 組み込みの拡張属性を custom:<key> で指定したファイルを invalid_header で拒否すること。
func TestUserCSVRejectsABuiltinAttributeUnderTheCustomPrefix(t *testing.T) {
	repo := usermemory.NewUserRepository()
	_, err := planUserImportForTest(importPlannerContext(), userCSVRulesDeps(repo, perUserImportOwnershipGuard{}),
		"preferred_username,custom:department\ndave,Platform\n")
	var csvErr *idmdomain.CSVError
	if !errors.As(err, &csvErr) || csvErr.Code != idmdomain.CSVErrorInvalidHeader {
		t.Fatalf("err=%v, want invalid_header", err)
	}
}

//spec:covers EX-IDMANAGEMENT-056-01, EX-IDMANAGEMENT-056-02: テナントにない id の行を作成にせず target_not_found とし、同じユーザー名の後の行と、大文字と小文字だけが異なるユーザー名の後の行を duplicate_username として前の行を残すこと。
func TestUserCSVResolvesByIDFirstAndKeepsTheEarlierDuplicate(t *testing.T) {
	repo := usermemory.NewUserRepository()
	plan, err := planUserImportForTest(importPlannerContext(), userCSVRulesDeps(repo, perUserImportOwnershipGuard{}),
		"id,preferred_username\nmissing-id,\n,dave\n,dave\n,Dave\n")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"rejected:target_not_found", "created", "rejected:duplicate_username", "rejected:duplicate_username"}
	if got := rowCodes(plan); !slices.Equal(got, want) {
		t.Fatalf("rows=%v, want %v", got, want)
	}
}

//spec:covers EX-IDMANAGEMENT-057-01, EX-IDMANAGEMENT-057-02, EX-IDMANAGEMENT-057-03: roles と required_actions を | で区切って前後の空白を除き、空の値を含むセルと大文字の真偽値を拒否し、required_actions の重複を除いて昇順に並べること。
func TestUserCSVBuiltinListAndBooleanCells(t *testing.T) {
	repo := usermemory.NewUserRepository()
	plan, err := planUserImportForTest(importPlannerContext(), userCSVRulesDeps(repo, perUserImportOwnershipGuard{}),
		"preferred_username,roles,required_actions,email_verified\n"+
			"a1,support | audit,verify_email|update_password|verify_email,true\n"+
			"a2,support||audit,,false\n"+
			"a3,,update_password|,false\n"+
			"a4,,,TRUE\n"+
			",,,\n")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"created", "rejected:invalid_roles", "rejected:invalid_required_actions", "rejected:invalid_boolean", "rejected:missing_identifier"}
	if got := rowCodes(plan); !slices.Equal(got, want) {
		t.Fatalf("rows=%v, want %v", got, want)
	}
	created := plan.Rows[0].User
	if !slices.Equal(created.Roles, []string{"audit", "support"}) {
		t.Fatalf("roles=%v, want [audit support]", created.Roles)
	}
	wantActions := []idmdomain.RequiredAction{idmdomain.RequiredActionUpdatePassword, idmdomain.RequiredActionVerifyEmail}
	if !slices.Equal(created.Lifecycle.RequiredActions, wantActions) {
		t.Fatalf("required_actions=%v, want %v", created.Lifecycle.RequiredActions, wantActions)
	}
}

//spec:covers EX-IDMANAGEMENT-057-04: 空の name のセルを適用すると、その User の名前を消すこと。
func TestUserCSVEmptyNameCellClearsTheName(t *testing.T) {
	repo := usermemory.NewUserRepository()
	alice := importPlannerUser("user-alice", "alice")
	name := "Alice"
	alice.Name = &name
	repo.Seed(alice)
	committer := &importRowCommitter{}
	_, rows, err := applyUserImportForTest(importPlannerContext(), UserImportApplyDeps{
		Plan: importPlannerDeps(repo, perUserImportOwnershipGuard{}), Committer: committer, PasswordHasher: importTestHasher{}, DynamicGroups: importDynamicGroups(repo),
	}, "id,name\nuser-alice,\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Action != userdomain.UserImportUpdate || len(committer.mutations) != 1 {
		t.Fatalf("rows=%+v mutations=%d, want one update", rows, len(committer.mutations))
	}
	if after := committer.mutations[0].After; after.Name != nil {
		t.Fatalf("name=%q, want cleared", *after.Name)
	}
}

//spec:covers EX-IDMANAGEMENT-058-01: CSV で作成する User が Active で、必須操作 update_password とパスワードの記録を持ち、使用量を消費すること。
func TestApplyUserImportCreatesAUserThatMustSetAPassword(t *testing.T) {
	repo := usermemory.NewUserRepository()
	committer := &importRowCommitter{}
	if _, _, err := applyUserImportForTest(importPlannerContext(), UserImportApplyDeps{
		Plan: importPlannerDeps(repo, perUserImportOwnershipGuard{}), Committer: committer, PasswordHasher: importTestHasher{}, DynamicGroups: importDynamicGroups(repo),
	}, "preferred_username\ndave\n"); err != nil {
		t.Fatal(err)
	}
	if len(committer.mutations) != 1 {
		t.Fatalf("mutations=%d, want 1", len(committer.mutations))
	}
	mutation := committer.mutations[0]
	after := mutation.After
	if after.Lifecycle.Status != idmdomain.UserStatusActive || !slices.Contains(after.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword) {
		t.Fatalf("status=%s actions=%v", after.Lifecycle.Status, after.Lifecycle.RequiredActions)
	}
	if after.PasswordHash == "" || mutation.PasswordHistoryHash != after.PasswordHash || !mutation.ConsumesUserQuota {
		t.Fatalf("mutation=%+v, want a password record and quota consumption", mutation)
	}
}

//spec:covers EX-IDMANAGEMENT-059-01: 取り込み元の所有の判定に失敗したとき、既存の User の行を source_managed で拒否し、新しい User の行は作成として計画すること。
func TestUserCSVOwnershipFailureRejectsExistingUsersOnly(t *testing.T) {
	repo := usermemory.NewUserRepository()
	repo.Seed(importPlannerUser("user-alice", "alice"))
	plan, err := planUserImportForTest(importPlannerContext(), userCSVRulesDeps(repo, perUserImportOwnershipGuard{err: errors.New("lookup failed")}),
		"preferred_username,name\nalice,Alice\ndave,Dave\n")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"rejected:source_managed", "created"}
	if got := rowCodes(plan); !slices.Equal(got, want) {
		t.Fatalf("rows=%v, want %v", got, want)
	}
}
