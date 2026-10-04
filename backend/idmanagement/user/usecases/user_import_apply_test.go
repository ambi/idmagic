package usecases

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
)

type importTestHasher struct{ err error }

func (h importTestHasher) Hash(string) (string, error)         { return "import-password-hash", h.err }
func (h importTestHasher) Verify(string, string) (bool, error) { return false, nil }

type importRowCommitter struct {
	mutations    []userports.UserImportRowMutation
	failUsername string
}

func (c *importRowCommitter) CommitUserImportRow(_ context.Context, mutation userports.UserImportRowMutation) error {
	if mutation.After.PreferredUsername == c.failUsername {
		return errors.New("commit failed")
	}
	c.mutations = append(c.mutations, mutation)
	return nil
}

func applyUserImportForTest(
	ctx context.Context,
	deps UserImportApplyDeps,
	input string,
) (userdomain.UserImportPlanSummary, []userdomain.UserImportRowPlan, error) {
	var rows []userdomain.UserImportRowPlan
	summary, err := ApplyUserImport(ctx, deps, strings.NewReader(input), idmdomain.DefaultCSVTransferPolicy(), "admin", time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC), func(row userdomain.UserImportRowPlan) error {
		rows = append(rows, row)
		return nil
	})
	return summary, rows, err
}

// 失敗した行は確定境界へ 1 度も渡らない。**行の一部だけが残る形を落とすのが
// この具体例である。** 確定の回数だけでなく、失敗した行の対象を読み直して
// プロフィールもロールも必須操作もカスタム属性も動いていないことを見る。
//
//spec:covers EX-IDMANAGEMENT-004-08: 1 行の確定が途中で失敗したとき、その行のプロフィール・ロール・必須操作・カスタム属性が一部も保存されず、他の有効な行は適用され続けること。
func TestApplyUserImportCommitsEachRowAtomicallyAndContinuesAfterFailure(t *testing.T) {
	repo := usermemory.NewUserRepository()
	repo.Seed(importPlannerUser("user-alice", "alice"))
	repo.Seed(importPlannerUser("user-bob", "bob"))
	committer := &importRowCommitter{failUsername: "bob"}
	deps := UserImportApplyDeps{
		Plan:           importPlannerDeps(repo, perUserImportOwnershipGuard{}),
		Committer:      committer,
		PasswordHasher: importTestHasher{}, DynamicGroups: importDynamicGroups(repo),
	}
	csv := "id,email,roles,required_actions,attr:department\n" +
		"user-alice,new-alice@example.com,admin|support,verify_email,Engineering\n" +
		"user-bob,new-bob@example.com,admin,update_password,Sales\n"

	summary, rows, err := applyUserImportForTest(importPlannerContext(), deps, csv)
	if err != nil {
		t.Fatal(err)
	}
	if summary.UpdatedRows != 1 || summary.RejectedRows != 1 || len(rows) != 2 {
		t.Fatalf("summary=%+v rows=%+v", summary, rows)
	}
	if rows[1].Error == nil || rows[1].Error.Code != "apply_failed" {
		t.Fatalf("failed row=%+v", rows[1])
	}
	if len(committer.mutations) != 1 {
		t.Fatalf("commits=%d, want 1", len(committer.mutations))
	}
	mutation := committer.mutations[0]
	if mutation.Before == nil || mutation.Before.ID != "user-alice" || mutation.After.ID != "user-alice" {
		t.Fatalf("mutation identity=%+v", mutation)
	}
	if mutation.After.Email == nil || *mutation.After.Email != "new-alice@example.com" ||
		len(mutation.After.Roles) != 2 || len(mutation.After.Lifecycle.RequiredActions) != 1 {
		t.Fatalf("after=%+v", mutation.After)
	}
	if got := mutation.After.Attributes["department"].String; got == nil || *got != "Engineering" {
		t.Fatalf("department=%+v", got)
	}
	if mutation.ActorUserID != "admin" || mutation.AuditEventType != "UserUpdated" || mutation.Now.IsZero() {
		t.Fatalf("audit metadata=%+v", mutation)
	}
	// 失敗した行は不可分である。4 つの書き込み先のどれにも痕跡が残ってはいけない。
	bob, err := repo.FindBySub(importPlannerContext(), "user-bob")
	if err != nil || bob == nil {
		t.Fatalf("FindBySub=(%+v,%v)", bob, err)
	}
	if bob.Email == nil || *bob.Email != "bob@example.com" {
		t.Fatalf("失敗した行の email が保存された: %v", bob.Email)
	}
	if len(bob.Roles) != 1 || bob.Roles[0] != "support" {
		t.Fatalf("失敗した行の roles が保存された: %v", bob.Roles)
	}
	if len(bob.Lifecycle.RequiredActions) != 0 {
		t.Fatalf("失敗した行の required_actions が保存された: %v", bob.Lifecycle.RequiredActions)
	}
	if got := bob.Attributes["department"].String; got == nil || *got != "Old" {
		t.Fatalf("失敗した行のカスタム属性が保存された: %+v", got)
	}
}

func TestApplyUserImportCreateIncludesCredentialHistoryRequiredActionAndQuotaInOneCommit(t *testing.T) {
	repo := usermemory.NewUserRepository()
	committer := &importRowCommitter{}
	deps := UserImportApplyDeps{
		Plan:           importPlannerDeps(repo, perUserImportOwnershipGuard{}),
		Committer:      committer,
		PasswordHasher: importTestHasher{}, DynamicGroups: importDynamicGroups(repo),
	}

	summary, rows, err := applyUserImportForTest(importPlannerContext(), deps, "preferred_username,email\nalice,alice@example.com\n")
	if err != nil {
		t.Fatal(err)
	}
	if summary.CreatedRows != 1 || len(rows) != 1 || len(committer.mutations) != 1 {
		t.Fatalf("summary=%+v rows=%+v commits=%d", summary, rows, len(committer.mutations))
	}
	mutation := committer.mutations[0]
	if mutation.Before != nil || mutation.After.ID == "" || mutation.After.PasswordHash != "import-password-hash" {
		t.Fatalf("create mutation=%+v", mutation)
	}
	if mutation.PasswordHistoryHash != mutation.After.PasswordHash || !mutation.ConsumesUserQuota {
		t.Fatalf("credential/quota metadata=%+v", mutation)
	}
	if !containsRequiredAction(mutation.After.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword) {
		t.Fatalf("required actions=%v", mutation.After.Lifecycle.RequiredActions)
	}
	if mutation.AuditEventType != "UserCreated" {
		t.Fatalf("audit event=%q", mutation.AuditEventType)
	}
}

//spec:covers REQ-IDMANAGEMENT-004, REQ-IDMANAGEMENT-089: CSV の適用が、作成、更新、変更なしの各行の User を確定後に動的グループの規則で評価し、規則に一致する User だけを所属させること。
func TestApplyUserImportEvaluatesDynamicGroups(t *testing.T) {
	ctx := importPlannerContext()
	repo := usermemory.NewUserRepository()
	unchanged := importPlannerUser("user-uma", "uma")
	engineering := "Engineering"
	unchanged.Attributes["department"] = userdomain.AttributeValue{Type: idmdomain.AttributeTypeString, String: &engineering}
	repo.Seed(unchanged)
	repo.Seed(importPlannerUser("user-alice", "alice"))
	groups := groupmemory.NewGroupRepository()
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	if err := groups.Save(ctx, &groupdomain.Group{
		ID: "dyn-eng", TenantID: "acme", Name: "dyn-eng", MembershipType: groupdomain.GroupMembershipDynamic, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := groups.SaveDynamicRule(ctx, &groupdomain.DynamicGroupRule{
		GroupID: "dyn-eng", TenantID: "acme", Expression: `user.department == "Engineering"`,
		Enabled: true, Version: 1, ReferencedAttributes: []string{"department"}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	committer := &importRowCommitter{}
	deps := UserImportApplyDeps{
		Plan:           importPlannerDeps(repo, perUserImportOwnershipGuard{}),
		Committer:      committer,
		PasswordHasher: importTestHasher{},
		DynamicGroups:  groupusecases.DynamicGroupDeps{GroupRepo: groups, UserRepo: repo},
	}
	csv := "preferred_username,attr:department\n" +
		"carol,Engineering\n" + // 作成
		"alice,Engineering\n" + // 更新
		"uma,Engineering\n" + // 変更なし
		"dave,Sales\n" // 作成するが規則に一致しない

	summary, _, err := applyUserImportForTest(ctx, deps, csv)
	if err != nil {
		t.Fatal(err)
	}
	if summary.CreatedRows != 2 || summary.UpdatedRows != 1 || summary.UnchangedRows != 1 {
		t.Fatalf("summary=%+v", summary)
	}
	members, err := groups.ListMembersByGroup(ctx, "acme", "dyn-eng")
	if err != nil {
		t.Fatal(err)
	}
	carolID := committer.mutations[0].After.ID
	got := make([]string, 0, len(members))
	for _, member := range members {
		got = append(got, member.UserID)
	}
	slices.Sort(got)
	want := []string{carolID, "user-alice", "user-uma"}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("members=%v, want %v", got, want)
	}
}

func containsRequiredAction(actions []idmdomain.RequiredAction, want idmdomain.RequiredAction) bool {
	return slices.Contains(actions, want)
}

// importDynamicGroups は動的グループの規則を持たないテナントの評価の依存を返す。
func importDynamicGroups(repo *usermemory.UserRepository) groupusecases.DynamicGroupDeps {
	return groupusecases.DynamicGroupDeps{GroupRepo: groupmemory.NewGroupRepository(), UserRepo: repo}
}
