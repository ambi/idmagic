package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestDynamicGroupRuleReconcilesMembership(t *testing.T) {
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "acme"}, "", "")
	groups := groupmemory.NewGroupRepository()
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	department := "Engineering"
	group := &groupdomain.Group{ID: "g1", TenantID: "acme", Name: "engineering", MembershipType: groupdomain.GroupMembershipDynamic, Roles: []string{}, CreatedAt: now, UpdatedAt: now}
	if err := groups.Save(ctx, group); err != nil {
		t.Fatal(err)
	}
	user := &userdomain.User{ID: "u1", TenantID: "acme", PreferredUsername: "alice", PasswordHash: "x", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, Attributes: map[string]userdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &department}}, CreatedAt: now, UpdatedAt: now}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	deps := groupusecases.DynamicGroupDeps{GroupRepo: groups, UserRepo: users}
	rule, err := groupusecases.UpdateDynamicGroupRule(ctx, deps, "admin", "g1", `user.department == "Engineering"`, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = groupusecases.SetDynamicGroupRuleEnabled(ctx, deps, "admin", "g1", true, now); err != nil {
		t.Fatal(err)
	}
	members, _ := groups.ListMembersByGroup(ctx, "acme", "g1")
	if len(members) != 1 || members[0].Source != groupdomain.MembershipSourceDynamicRule || members[0].RuleVersion == nil || *members[0].RuleVersion == rule.Version {
		t.Fatalf("unexpected members: %+v", members)
	}
	if err := groupusecases.AddMember(ctx, groupusecases.AdminGroupDeps{GroupRepo: groups, UserRepo: users}, "admin", "g1", "u1", now); !errors.Is(err, groupusecases.ErrDynamicMembershipManaged) {
		t.Fatalf("manual add err=%v", err)
	}
}

// dynamicFixtureGroupID は dynamicRuleFixture が建てる動的グループ。
const dynamicFixtureGroupID = "g1"

// dynamicRuleFixture は動的規則の具体例が要る「揃った母集団」を建てる。
// Engineering の有効な User、Sales の有効な User、Engineering だが無効な User の 3 人を
// 置く。**1 人しか居ない母集団では、全員を入れる実装と規則を評価する実装が同じ結果になる。**
func dynamicRuleFixture(
	t *testing.T,
) (context.Context, groupusecases.DynamicGroupDeps, *groupmemory.GroupRepository, *usermemory.UserRepository) {
	t.Helper()
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "acme"}, "", "")
	groups := groupmemory.NewGroupRepository()
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	if err := groups.Save(ctx, &groupdomain.Group{
		ID: "g1", TenantID: "acme", Name: "engineering",
		MembershipType: groupdomain.GroupMembershipDynamic, Roles: []string{"catalog:read"},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	for _, seed := range []struct {
		id, department string
		status         idmdomain.UserStatus
	}{
		{"u_eng", "Engineering", idmdomain.UserStatusActive},
		{"u_sales", "Sales", idmdomain.UserStatusActive},
		{"u_eng_disabled", "Engineering", idmdomain.UserStatusDisabled},
	} {
		if err := users.Save(ctx, &userdomain.User{
			ID: seed.id, TenantID: "acme", PreferredUsername: seed.id, PasswordHash: "x",
			Lifecycle: userdomain.UserLifecycle{Status: seed.status},
			Attributes: map[string]userdomain.AttributeValue{
				"department": {Type: idmdomain.AttributeTypeString, String: new(seed.department)},
			},
			CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	return ctx, groupusecases.DynamicGroupDeps{GroupRepo: groups, UserRepo: users}, groups, users
}

// dynamicMemberIDs は動的規則を由来として在籍している User の id を返す。
// 対象は dynamicRuleFixture が建てる唯一のグループである。
func dynamicMemberIDs(
	ctx context.Context, t *testing.T, groups *groupmemory.GroupRepository,
) []string {
	t.Helper()
	members, err := groups.ListMembersByGroup(ctx, "acme", dynamicFixtureGroupID)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(members))
	for _, member := range members {
		if member.Source != groupdomain.MembershipSourceDynamicRule {
			t.Fatalf("動的グループに source=%s の所属がある: %+v", member.Source, member)
		}
		ids = append(ids, member.UserID)
	}
	slices.Sort(ids)
	return ids
}

// 具体例は 2 つの `Then` を持つ。所属の集合だけでなく、実効ロールがその所属を
// 参照していることまで見る。所属だけを見るテストは、行は作るが権限へ結び付けない
// 実装を通してしまう。
//
//spec:covers EX-IDMANAGEMENT-020-01: 保存して有効化した CEL 規則の全件再評価で、条件に一致する有効な User だけが動的規則を由来として所属し、その所属が実効ロールに乗ること。
func TestDynamicGroupRuleAdmitsOnlyMatchingActiveUsers(t *testing.T) {
	ctx, deps, groups, users := dynamicRuleFixture(t)
	now := time.Now().UTC()

	if _, err := groupusecases.UpdateDynamicGroupRule(
		ctx, deps, "admin", "g1", `user.department == "Engineering"`, now,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := groupusecases.SetDynamicGroupRuleEnabled(ctx, deps, "admin", "g1", true, now); err != nil {
		t.Fatal(err)
	}

	// Sales は条件に合わず、無効な Engineering は「有効な User」でない。
	if ids := dynamicMemberIDs(ctx, t, groups); !slices.Equal(ids, []string{"u_eng"}) {
		t.Fatalf("members = %v, want [u_eng]", ids)
	}

	// 実効ロールがその所属を参照していること。
	adminDeps := groupusecases.AdminGroupDeps{GroupRepo: groups, UserRepo: users}
	view, err := groupusecases.UserGroups(ctx, adminDeps, "u_eng")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(view.EffectiveRoles, "catalog:read") {
		t.Fatalf("u_eng の実効ロール = %v", view.EffectiveRoles)
	}
	sales, err := groupusecases.UserGroups(ctx, adminDeps, "u_sales")
	if err != nil {
		t.Fatal(err)
	}
	if len(sales.EffectiveRoles) != 0 {
		t.Fatalf("u_sales の実効ロール = %v, want empty", sales.EffectiveRoles)
	}
}

// 具体例は「属性値そのものは返さない」まで言う。判定だけを見るテストは、
// 属性を応答へ載せる実装を通す。プレビューは保存前の評価なので、`Group` が
// 変わっていないことも同時に読む。
//
//spec:covers EX-IDMANAGEMENT-021-01: 未保存の CEL 式のプレビューが一致の有無と add/remove/unchanged を返し、属性値を返さず、所属も変えないこと。
func TestPreviewDynamicGroupRuleReturnsVerdictsWithoutAttributeValues(t *testing.T) {
	ctx, deps, groups, _ := dynamicRuleFixture(t)
	now := time.Now().UTC()

	// 先に別の規則で u_sales を入れておく。これが無いと remove の判定が出ない。
	if _, err := groupusecases.UpdateDynamicGroupRule(
		ctx, deps, "admin", "g1", `user.department == "Sales"`, now,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := groupusecases.SetDynamicGroupRuleEnabled(ctx, deps, "admin", "g1", true, now); err != nil {
		t.Fatal(err)
	}
	before := dynamicMemberIDs(ctx, t, groups)
	if !slices.Equal(before, []string{"u_sales"}) {
		t.Fatalf("前提が壊れている: members = %v", before)
	}

	previews, err := groupusecases.PreviewDynamicGroupRule(
		ctx, deps, "g1", `user.department == "Engineering"`,
		[]string{"u_eng", "u_sales", "u_eng_disabled"},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		matched bool
		change  string
	}{
		"u_eng":          {matched: true, change: "add"},
		"u_sales":        {matched: false, change: "remove"},
		"u_eng_disabled": {matched: false, change: "unchanged"},
	}
	if len(previews) != len(want) {
		t.Fatalf("previews = %+v", previews)
	}
	for _, preview := range previews {
		expected, ok := want[preview.UserID]
		if !ok {
			t.Fatalf("想定外の User が返った: %+v", preview)
		}
		if preview.Matched != expected.matched || preview.Change != expected.change {
			t.Fatalf("%s: matched=%v change=%q, want %v/%q",
				preview.UserID, preview.Matched, preview.Change, expected.matched, expected.change)
		}
	}
	// 「属性値そのものは返さない」。DynamicGroupPreview は判定しか運ばない形であり、
	// 応答を JSON にしても評価に使った department の値は現れない。
	encoded, err := json.Marshal(previews)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"Engineering", "Sales"} {
		if strings.Contains(string(encoded), value) {
			t.Fatalf("プレビューが属性値 %q を返した: %s", value, encoded)
		}
	}
	// 未保存の評価なので所属は動かない。
	if after := dynamicMemberIDs(ctx, t, groups); !slices.Equal(after, before) {
		t.Fatalf("プレビューが所属を変えた: before=%v after=%v", before, after)
	}
}

// evaluationFailingUserRepository は 1 人だけ、規則が評価できない形の User を混ぜる。
// 評価失敗は CEL の実行時エラーで起きる。規則が参照する属性を 1 つも持たない User が
// その形になり (activation にキーが無い)、規則の評価は `no such key` で落ちる。
type evaluationFailingUserRepository struct {
	*usermemory.UserRepository
	failing *userdomain.User
}

func (r *evaluationFailingUserRepository) FindAll(
	ctx context.Context, tenantID string,
) ([]*userdomain.User, error) {
	users, err := r.UserRepository.FindAll(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return append(users, r.failing), nil
}

// 具体例は 2 つの `Then` を持つ。**版が上がった瞬間に旧版の所属が権限から外れる**ことと、
// **再評価に失敗した User が新版の所属を得ない**ことである。前者は再評価の完了を待たない
// ので、再評価を呼ぶ前に実効ロールを読む。
//
//spec:covers EX-IDMANAGEMENT-023-01: 動的規則の版が上がった時点で旧版の所属が実効ロールから外れること、再評価に失敗した User が新版の所属を得ないこと。
func TestDynamicGroupRuleVersionBumpDropsStaleMembershipAndFailuresGrantNothing(t *testing.T) {
	ctx, deps, groups, users := dynamicRuleFixture(t)
	now := time.Now().UTC()
	adminDeps := groupusecases.AdminGroupDeps{GroupRepo: groups, UserRepo: users}

	if _, err := groupusecases.UpdateDynamicGroupRule(
		ctx, deps, "admin", "g1", `user.department == "Engineering"`, now,
	); err != nil {
		t.Fatal(err)
	}
	enabled, err := groupusecases.SetDynamicGroupRuleEnabled(ctx, deps, "admin", "g1", true, now)
	if err != nil {
		t.Fatal(err)
	}
	enabledVersion := enabled.Version
	view, err := groupusecases.UserGroups(ctx, adminDeps, "u_eng")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(view.EffectiveRoles, "catalog:read") {
		t.Fatalf("前提が壊れている: u_eng の実効ロール = %v", view.EffectiveRoles)
	}

	// 版を上げる。**ここで JobRepo を配線するのが要点である。** 未配線の
	// `scheduleDynamicGroupReconcile` は再評価をその場で走らせてしまい、
	// 「版が上がってから再評価が終わるまで」という具体例の言う瞬間が消える。
	// production は job を挟むので、こちらが本来の形である。
	deferred := deps
	deferred.JobRepo = jobsmemory.NewJobRepository()
	updated, err := groupusecases.UpdateDynamicGroupRule(
		ctx, deferred, "admin", "g1", `user.department == "Engineering" || user.department == "Sales"`,
		now.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	// **版は単調に増える。** 除外は「所属が持つ版と現在の版が違う」ことで起きるので、
	// 版が戻る実装は、いつか過去の版と一致して旧版の所属を復活させる。変異テストが
	// `existing.Version + 1` を `- 1` にしても除外は残ってしまい、ここでしか落ちない。
	if updated.Version <= enabledVersion {
		t.Fatalf("版が %d から %d へ戻った", enabledVersion, updated.Version)
	}
	// 「直ちに実効ロールから除外される」。再評価を待たずに落ちること。
	stale, err := groupusecases.UserGroups(ctx, adminDeps, "u_eng")
	if err != nil {
		t.Fatal(err)
	}
	if len(stale.EffectiveRoles) != 0 {
		t.Fatalf("版が上がったのに旧版の所属が実効ロールに残った: %v", stale.EffectiveRoles)
	}

	// 評価できない User を 1 人混ぜて再評価する。規則が読む department を持たない
	// User は、CEL の activation にそのキーが無いため実行時エラーになる。
	failing := &userdomain.User{
		ID: "u_broken", TenantID: "acme", PreferredUsername: "u_broken", PasswordHash: "x",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now, UpdatedAt: now,
	}
	failingDeps := deferred
	failingDeps.UserRepo = &evaluationFailingUserRepository{UserRepository: users, failing: failing}
	result, err := groupusecases.ReconcileDynamicGroup(ctx, failingDeps, updated, now.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if result.Errors != 1 {
		t.Fatalf("評価失敗が %d 件、want 1 (result=%+v)", result.Errors, result)
	}
	// 「新しいバージョンのメンバーシップを得ない」。
	if ids := dynamicMemberIDs(ctx, t, groups); slices.Contains(ids, "u_broken") {
		t.Fatalf("評価に失敗した User が所属を得た: %v", ids)
	}
	// 再評価は旧版の行を書き直す。**版まで見るのが要点である。** 旧版の行を
	// 「有効」と読む実装は、除外されたままの所属を残して誰にも権限を戻さない。
	members, err := groups.ListMembersByGroup(ctx, "acme", dynamicFixtureGroupID)
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range members {
		if member.RuleVersion == nil || *member.RuleVersion != updated.Version {
			t.Fatalf("再評価後の所属が新しい版を持たない: %+v (rule version=%d)", member, updated.Version)
		}
	}
	// 権限が戻っていること。除外されたままでは再評価した意味が無い。
	restored, err := groupusecases.UserGroups(ctx, adminDeps, "u_eng")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(restored.EffectiveRoles, "catalog:read") {
		t.Fatalf("再評価後も実効ロールが戻らない: %v", restored.EffectiveRoles)
	}
}

// 122: 有効化も版を上げる。**上げ方が同じ向きでなければならない。** 有効化だけが
// 版を戻す実装は、無効化と有効化を繰り返すと過去の版と一致して、規則が消えたはずの
// 所属を復活させる。
//
//spec:covers EX-IDMANAGEMENT-020-01: 動的規則の有効化が版を単調に進め、その版の所属だけが実効ロールに乗ること。
func TestEnablingADynamicRuleAdvancesTheVersionMonotonically(t *testing.T) {
	ctx, deps, groups, _ := dynamicRuleFixture(t)
	now := time.Now().UTC()

	saved, err := groupusecases.UpdateDynamicGroupRule(
		ctx, deps, "admin", dynamicFixtureGroupID, `user.department == "Engineering"`, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	enabled, err := groupusecases.SetDynamicGroupRuleEnabled(
		ctx, deps, "admin", dynamicFixtureGroupID, true, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if enabled.Version <= saved.Version {
		t.Fatalf("有効化で版が %d から %d へ戻った", saved.Version, enabled.Version)
	}
	// 所属はその版で書かれている。版が食い違えば実効ロールから外れる。
	members, err := groups.ListMembersByGroup(ctx, "acme", dynamicFixtureGroupID)
	if err != nil || len(members) != 1 {
		t.Fatalf("members = %+v, %v", members, err)
	}
	if members[0].RuleVersion == nil || *members[0].RuleVersion != enabled.Version {
		t.Fatalf("所属の版 = %v, want %d", members[0].RuleVersion, enabled.Version)
	}
}

// 193 / 238: 具体例の `Given` は「最大 100 件の User を選択している」であり、`Then` は
// 「一致の有無と判定を返す」である。上限を超えた要求は拒否され、評価に失敗した User は
// 判定ではなく誤りとして返る。どちらも判定だけを見るテストでは観測できない。
//
//spec:covers EX-IDMANAGEMENT-021-01: 100 件を超える選択のプレビューが拒否されること、評価に失敗した User が一致ではなく error_code を伴って返ること。
func TestPreviewDynamicGroupRuleBoundsTheSelectionAndReportsEvaluationErrors(t *testing.T) {
	ctx, deps, _, users := dynamicRuleFixture(t)
	expression := `user.department == "Engineering"`

	// 100 件までは受理し、101 件で拒否する。境界そのものを見る。
	ids := make([]string, 0, 101)
	for i := range 101 {
		id := fmt.Sprintf("bulk-%03d", i)
		if err := users.Save(ctx, &userdomain.User{
			ID: id, TenantID: "acme", PreferredUsername: id, PasswordHash: "x",
			Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
			Attributes: map[string]userdomain.AttributeValue{
				"department": {Type: idmdomain.AttributeTypeString, String: new("Sales")},
			},
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if _, err := groupusecases.PreviewDynamicGroupRule(
		ctx, deps, dynamicFixtureGroupID, expression, ids[:100],
	); err != nil {
		t.Fatalf("100 件のプレビューが拒否された: %v", err)
	}
	if _, err := groupusecases.PreviewDynamicGroupRule(
		ctx, deps, dynamicFixtureGroupID, expression, ids,
	); !errors.Is(err, groupusecases.ErrInvalidDynamicGroupRule) {
		t.Fatalf("101 件のプレビューが err=%v, want ErrInvalidDynamicGroupRule", err)
	}

	// 規則が読む属性を持たない User は、評価が実行時に落ちる。判定ではなく誤りとして返る。
	if err := users.Save(ctx, &userdomain.User{
		ID: "u_broken_preview", TenantID: "acme", PreferredUsername: "u_broken_preview",
		PasswordHash: "x", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	previews, err := groupusecases.PreviewDynamicGroupRule(
		ctx, deps, dynamicFixtureGroupID, expression, []string{"u_eng", "u_broken_preview"},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, preview := range previews {
		switch preview.UserID {
		case "u_eng":
			if preview.ErrorCode != nil {
				t.Fatalf("評価できる User に error_code が付いた: %+v", preview)
			}
		case "u_broken_preview":
			if preview.ErrorCode == nil {
				t.Fatalf("評価に失敗した User に error_code が無い: %+v", preview)
			}
			if preview.Matched {
				t.Fatalf("評価に失敗した User が一致として返った: %+v", preview)
			}
		}
	}
}
