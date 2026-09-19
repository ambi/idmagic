package usecases

// 主要ユースケース追跡: REQ-IDMANAGEMENT-004。

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type importSchemaReader struct {
	defs []userdomain.UserAttributeDef
	err  error
}

func (r importSchemaReader) EffectiveUserAttributeDefs(context.Context, string) ([]userdomain.UserAttributeDef, error) {
	return r.defs, r.err
}

type perUserImportOwnershipGuard struct {
	managed map[string]bool
	err     error
}

func (g perUserImportOwnershipGuard) SourceManagedUserIDs(_ context.Context, _ string, userIDs []string) (map[string]bool, error) {
	if g.err != nil {
		return nil, g.err
	}
	out := make(map[string]bool, len(userIDs))
	for _, userID := range userIDs {
		out[userID] = g.managed[userID]
	}
	return out, nil
}

func importPlannerContext() context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "acme"}, "", "")
}

func importPlannerUser(id, username string) *userdomain.User {
	now := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	email := username + "@example.com"
	department := "Old"
	return &userdomain.User{
		ID: id, TenantID: "acme", PreferredUsername: username, PasswordHash: "hash",
		Email: &email, Roles: []string{"support"},
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		Attributes: map[string]userdomain.AttributeValue{
			"department": {Type: idmdomain.AttributeTypeString, String: &department},
		},
		CreatedAt: now, UpdatedAt: now,
	}
}

func importPlannerDeps(repo *usermemory.UserRepository, guard perUserImportOwnershipGuard) UserImportPlanDeps {
	return UserImportPlanDeps{
		UserRepo: repo,
		SchemaReader: importSchemaReader{defs: []userdomain.UserAttributeDef{{
			Key: "department", Type: idmdomain.AttributeTypeString,
			Visibility: idmdomain.AttrVisibilityPrivate,
		}}},
		OwnershipGuard: guard,
	}
}

func planUserImportForTest(ctx context.Context, deps UserImportPlanDeps, input string) (userdomain.UserImportPlan, error) {
	plan := userdomain.UserImportPlan{}
	_, err := PlanUserImport(ctx, deps, strings.NewReader(input), idmdomain.DefaultCSVTransferPolicy(), func(row userdomain.UserImportRowPlan) error {
		plan.Rows = append(plan.Rows, row)
		return nil
	})
	return plan, err
}

// scenario: 管理者はエクスポートしたユーザー CSV を安全に再適用できる
func TestPlanUserImportCreateUpdateUnchangedAndFieldPresence(t *testing.T) {
	repo := usermemory.NewUserRepository()
	alice := importPlannerUser("user-alice", "alice")
	charlie := importPlannerUser("user-charlie", "charlie")
	repo.Seed(alice)
	repo.Seed(charlie)

	csv := "id,preferred_username,email,required_actions,attr:department\n" +
		"user-alice,alice,,update_password,Engineering\n" +
		",bob,bob@example.com,,Sales\n" +
		"user-charlie,charlie,charlie@example.com,,Old\n"
	plan, err := planUserImportForTest(importPlannerContext(), importPlannerDeps(repo, perUserImportOwnershipGuard{}), csv)
	if err != nil {
		t.Fatal(err)
	}
	if plan.CreatedRows() != 1 || plan.UpdatedRows() != 1 || plan.UnchangedRows() != 1 || plan.RejectedRows() != 0 {
		t.Fatalf("unexpected plan counts: %+v", plan)
	}
	updated := plan.Rows[0].User
	if updated.Email != nil {
		t.Fatalf("present empty email must clear, got %q", *updated.Email)
	}
	if len(updated.Roles) != 1 || updated.Roles[0] != "support" {
		t.Fatalf("absent roles must be preserved, got %v", updated.Roles)
	}
	if got := updated.Lifecycle.RequiredActions; len(got) != 1 || got[0] != idmdomain.RequiredActionUpdatePassword {
		t.Fatalf("required_actions=%v", got)
	}
	if got := updated.Attributes["department"].String; got == nil || *got != "Engineering" {
		t.Fatalf("department=%+v", got)
	}
}

// 004-04 が並べる識別子の誤りのうち、食い違いと欠落をここが持つ。重複は
// TestPlanUserImportRefusesDuplicateTargetsAndFinalUsernames が持つ。
//
//spec:covers EX-IDMANAGEMENT-004-04: id と preferred_username が別の User を指す行、識別子を 1 つも持たない行が、それぞれ安定コードで rejected になること。
func TestPlanUserImportRejectsIdentifierMismatchInvalidTypesAndMissingIdentifier(t *testing.T) {
	repo := usermemory.NewUserRepository()
	repo.Seed(importPlannerUser("user-alice", "alice"))
	repo.Seed(importPlannerUser("user-bob", "bob"))
	repo.Seed(importPlannerUser("user-carol", "carol"))
	csv := "id,preferred_username,email_verified,attr:department\n" +
		"user-alice,bob,true,Engineering\n" +
		"user-carol,carol,TRUE,Engineering\n" +
		",,,Engineering\n"

	plan, err := planUserImportForTest(importPlannerContext(), importPlannerDeps(repo, perUserImportOwnershipGuard{}), csv)
	if err != nil {
		t.Fatal(err)
	}
	want := []idmdomain.CSVErrorCode{"identifier_mismatch", "invalid_boolean", "missing_identifier"}
	if plan.RejectedRows() != len(want) {
		t.Fatalf("plan=%+v", plan)
	}
	for i, code := range want {
		if plan.Rows[i].Error == nil || plan.Rows[i].Error.Code != code {
			t.Fatalf("row[%d]=%+v, want %q", i, plan.Rows[i], code)
		}
	}
}

// ガードが読めなかった場合も、ガードが配線されていない場合も同じ拒否になる。
// 読めないことを「管理外」と読み替える実装は、外部管理の User を上書きしてしまう。
//
//spec:covers EX-IDMANAGEMENT-004-07: 外部の取り込み元が管理する User への行が source_managed で rejected になり、その User が変更されないこと。
func TestPlanUserImportFailsClosedForSourceManagedUsers(t *testing.T) {
	for name, guard := range map[string]perUserImportOwnershipGuard{
		"managed":       {managed: map[string]bool{"user-alice": true}},
		"guard failure": {err: errors.New("source unavailable")},
		"missing guard": {},
	} {
		t.Run(name, func(t *testing.T) {
			repo := usermemory.NewUserRepository()
			repo.Seed(importPlannerUser("user-alice", "alice"))
			deps := importPlannerDeps(repo, guard)
			if name == "missing guard" {
				deps.OwnershipGuard = nil
			}
			plan, err := planUserImportForTest(importPlannerContext(), deps, "id,email\nuser-alice,new@example.com\n")
			if err != nil {
				t.Fatal(err)
			}
			if plan.RejectedRows() != 1 || plan.Rows[0].Error == nil || plan.Rows[0].Error.Code != "source_managed" {
				t.Fatalf("plan=%+v", plan)
			}
			// 具体例は「`User` は変更されない」まで言う。計画は保存層を動かさない。
			stored, err := repo.FindBySub(importPlannerContext(), "user-alice")
			if err != nil || stored == nil {
				t.Fatalf("FindBySub=(%+v,%v)", stored, err)
			}
			if stored.Email == nil || *stored.Email != "alice@example.com" {
				t.Fatalf("拒否が email を変えた: %v", stored.Email)
			}
		})
	}
}

// 別の操作がプレビューと同じ最終状態を先に作ったとき、再計画は updated ではなく
// unchanged になる。判定が保存済みの計画からではなく現在状態から出ている証拠である。
//
//spec:covers EX-IDMANAGEMENT-004-06: プレビュー後に User の状態が別の操作で変わったとき、適用が古い計画を実行せず現在状態から再判定すること。
func TestPlanUserImportReplansAgainstCurrentRepositoryState(t *testing.T) {
	repo := usermemory.NewUserRepository()
	alice := importPlannerUser("user-alice", "alice")
	repo.Seed(alice)
	deps := importPlannerDeps(repo, perUserImportOwnershipGuard{})
	csv := "id,email\nuser-alice,new@example.com\n"

	preview, err := planUserImportForTest(importPlannerContext(), deps, csv)
	if err != nil || preview.UpdatedRows() != 1 {
		t.Fatalf("preview=%+v error=%v", preview, err)
	}
	newEmail := "new@example.com"
	alice.Email = &newEmail
	repo.Seed(alice)

	replanned, err := planUserImportForTest(importPlannerContext(), deps, csv)
	if err != nil || replanned.UnchangedRows() != 1 {
		t.Fatalf("replanned=%+v error=%v", replanned, err)
	}
}

// 予約ロールを新しく加える行は rejected になり、既に持っている値の再送は unchanged になる。
//
// テナントは acme、すなわち制御面ではない。プレビューと適用の再計画はどちらもこの
// 計画器を通るので、片方だけを直した実装はここで落ちる。
//
// 差分で判定するところまでを 1 本で固定する。拒否だけを見るテストは、絶対集合で
// 判定する実装 (既存値を持つ環境の無編集エクスポートを全行 rejected にする実装) を
// そのまま通す。
//
//spec:covers EX-IDMANAGEMENT-032-06, EX-IDMANAGEMENT-032-08: 制御面テナント以外のテナントの行が `system_admin` を新しく加えると `roles` 列を指す invalid_roles で rejected になり、その User は変更されないこと。既に `system_admin` を持つ User の行が同じ値を再送すると unchanged になること。
func TestPlanUserImportRefusesTheReservedRoleButKeepsAStoredOne(t *testing.T) {
	repo := usermemory.NewUserRepository()
	alice := importPlannerUser("user-alice", "alice")
	legacy := importPlannerUser("user-legacy", "legacy")
	legacy.Roles = []string{"system_admin"}
	repo.Seed(alice)
	repo.Seed(legacy)

	csv := "id,preferred_username,roles\n" +
		"user-alice,alice,system_admin\n" +
		"user-legacy,legacy,system_admin\n"
	plan, err := planUserImportForTest(
		importPlannerContext(), importPlannerDeps(repo, perUserImportOwnershipGuard{}), csv,
	)
	if err != nil {
		t.Fatal(err)
	}
	if plan.RejectedRows() != 1 || plan.UnchangedRows() != 1 {
		t.Fatalf("plan=%+v", plan)
	}
	refused := plan.Rows[0]
	if refused.Action != userdomain.UserImportRejected {
		t.Fatalf("row[0].Action=%s, want rejected", refused.Action)
	}
	if refused.Error == nil || refused.Error.Code != "invalid_roles" || refused.Error.Column != "roles" {
		t.Fatalf("row[0].Error=%+v, want column=roles code=invalid_roles", refused.Error)
	}
	// 拒否された行は計画に User を載せない。載せる実装は適用側で保存しうる。
	if refused.User != nil {
		t.Fatalf("拒否された行が User を載せている: %+v", refused.User)
	}
	// 保存層の User は計画では変わらない。プレビューは読み取りだけの計画である。
	if stored, err := repo.FindBySub(importPlannerContext(), "user-alice"); err != nil {
		t.Fatal(err)
	} else if slices.Contains(stored.Roles, "system_admin") {
		t.Fatalf("プレビューが User を変更した: roles=%v", stored.Roles)
	}

	kept := plan.Rows[1]
	if kept.Action != userdomain.UserImportUnchanged {
		t.Fatalf("row[1].Action=%s, want unchanged", kept.Action)
	}
	if !slices.Contains(kept.User.Roles, "system_admin") {
		t.Fatalf("row[1] の roles=%v が system_admin を失っている", kept.User.Roles)
	}
}
