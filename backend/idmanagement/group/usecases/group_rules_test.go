package usecases_test

// Group と動的グループの規則が約束する細部を、保存層とイベントから読んで固定する。

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

var groupRulesNow = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

func storedGroupByID(t *testing.T, deps groupusecases.AdminGroupDeps, id string) *groupdomain.Group {
	t.Helper()
	group, err := deps.GroupRepo.FindByID(context.Background(), tenancydomain.DefaultTenantID, id)
	if err != nil || group == nil {
		t.Fatalf("FindByID(%s)=(%v,%v)", id, group, err)
	}
	return group
}

func tenantGroupNames(t *testing.T, deps groupusecases.AdminGroupDeps) []string {
	t.Helper()
	groups, err := deps.GroupRepo.ListAll(context.Background(), tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(groups))
	for i, group := range groups {
		out[i] = group.Name
	}
	slices.Sort(out)
	return out
}

//spec:covers EX-IDMANAGEMENT-061-01, EX-IDMANAGEMENT-061-02: スキーマのないテナントで属性を含む作成を拒否し、必須の属性を省いた作成も拒否して Group を作らないこと。
func TestGroupAttributesRequireATenantSchema(t *testing.T) {
	deps, _ := newGroupDeps(t)
	ctx := context.Background()
	value := "CC-100"
	attrs := map[string]userdomain.AttributeValue{"cost_center": {Type: idmdomain.AttributeTypeString, String: &value}}
	if _, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{ActorUserID: "operator", Name: "finance", Attributes: attrs, Now: groupRulesNow}); !errors.Is(err, groupusecases.ErrInvalidAttribute) {
		t.Fatalf("スキーマなし: err=%v, want ErrInvalidAttribute", err)
	}
	if err := deps.GroupAttrSchemaRepo.Save(ctx, &groupdomain.TenantGroupAttributeSchema{
		TenantID:   tenancydomain.DefaultTenantID,
		Attributes: []groupdomain.GroupAttributeDef{{Key: "cost_center", Type: idmdomain.AttributeTypeString, Required: true}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{ActorUserID: "operator", Name: "finance", Now: groupRulesNow}); !errors.Is(err, groupusecases.ErrInvalidAttribute) {
		t.Fatalf("必須の属性の欠落: err=%v, want ErrInvalidAttribute", err)
	}
	if names := tenantGroupNames(t, deps); len(names) != 0 {
		t.Fatalf("拒否した作成が Group を作った: %v", names)
	}
}

type recordingGroupNotifier struct {
	calls int
	err   error
}

func (n *recordingGroupNotifier) NotifyGroupMutation(context.Context, string, string, groupports.ProvisioningTrigger, time.Time) error {
	n.calls++
	return n.err
}

//spec:covers EX-IDMANAGEMENT-062-01: 何も変わらない Group の更新が updated_at を進めず、GroupUpdated を発行せず、下流へ通知しないこと。
func TestUpdateGroupWithoutChangesHasNoEffect(t *testing.T) {
	deps, events := newGroupDeps(t)
	notifier := &recordingGroupNotifier{}
	deps.ProvisioningNotifier = notifier
	ctx := context.Background()
	description := "Builds things"
	group, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{ActorUserID: "operator", Name: "engineering", Description: &description, Now: groupRulesNow})
	if err != nil {
		t.Fatal(err)
	}
	*events, notifier.calls = nil, 0
	if _, err := groupusecases.UpdateGroup(ctx, deps, groupusecases.UpdateGroupInput{
		ActorUserID: "operator", ID: group.ID, Description: &description, Now: groupRulesNow.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if stored := storedGroupByID(t, deps, group.ID); !stored.UpdatedAt.Equal(groupRulesNow) {
		t.Fatalf("updated_at=%v, want %v", stored.UpdatedAt, groupRulesNow)
	}
	if len(*events) != 0 || notifier.calls != 0 {
		t.Fatalf("events=%v notifications=%d, want none", eventTypes(*events), notifier.calls)
	}
}

func memberDepsWithUsers(t *testing.T) (groupusecases.AdminGroupDeps, *[]spec.DomainEvent, string) {
	t.Helper()
	deps, events := newGroupDeps(t)
	users := usermemory.NewUserRepository()
	for id, status := range map[string]idmdomain.UserStatus{
		"disabled": idmdomain.UserStatusDisabled, "pending": idmdomain.UserStatusPendingDeletion,
		"deleted": idmdomain.UserStatusDeleted, "active": idmdomain.UserStatusActive,
	} {
		users.Seed(&userdomain.User{
			ID: id, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: id, PasswordHash: "x",
			Lifecycle: userdomain.UserLifecycle{Status: status}, CreatedAt: groupRulesNow, UpdatedAt: groupRulesNow,
		})
	}
	users.Seed(&userdomain.User{ID: "foreign", TenantID: "acme", PreferredUsername: "foreign", PasswordHash: "x", CreatedAt: groupRulesNow, UpdatedAt: groupRulesNow})
	deps.UserRepo = users
	group, err := groupusecases.CreateGroup(context.Background(), deps, groupusecases.CreateGroupInput{ActorUserID: "operator", Name: "engineering", Now: groupRulesNow})
	if err != nil {
		t.Fatal(err)
	}
	*events = nil
	return deps, events, group.ID
}

func memberIDs(t *testing.T, deps groupusecases.AdminGroupDeps, groupID string) []string {
	t.Helper()
	members, err := deps.GroupRepo.ListMembersByGroup(context.Background(), tenancydomain.DefaultTenantID, groupID)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(members))
	for i, member := range members {
		out[i] = member.UserID
	}
	slices.Sort(out)
	return out
}

//spec:covers EX-IDMANAGEMENT-063-01: Disabled と PendingDeletion の User を手動グループへ追加でき、GroupMemberAdded を発行すること。
func TestAddMemberAcceptsDisabledAndPendingDeletionUsers(t *testing.T) {
	deps, events, groupID := memberDepsWithUsers(t)
	ctx := context.Background()
	for _, id := range []string{"disabled", "pending"} {
		if err := groupusecases.AddMember(ctx, deps, "operator", groupID, id, groupRulesNow); err != nil {
			t.Fatalf("%s: err=%v, want added", id, err)
		}
	}
	if got := memberIDs(t, deps, groupID); !slices.Equal(got, []string{"disabled", "pending"}) {
		t.Fatalf("members=%v", got)
	}
	if got := eventTypes(*events); !slices.Equal(got, []string{"GroupMemberAdded", "GroupMemberAdded"}) {
		t.Fatalf("events=%v", got)
	}
}

//spec:covers EX-IDMANAGEMENT-063-02: Deleted の User、別のテナントの User、存在しない User の追加を user_not_found で拒否し、既存メンバーの追加と非メンバーの除外はイベントを発行しないこと。
func TestAddMemberRefusesDeletedForeignAndUnknownUsers(t *testing.T) {
	deps, events, groupID := memberDepsWithUsers(t)
	ctx := context.Background()
	for _, id := range []string{"deleted", "foreign", "nobody"} {
		if err := groupusecases.AddMember(ctx, deps, "operator", groupID, id, groupRulesNow); !errors.Is(err, idmusecases.ErrUserNotFound) {
			t.Fatalf("%s: err=%v, want ErrUserNotFound", id, err)
		}
	}
	if got := memberIDs(t, deps, groupID); len(got) != 0 {
		t.Fatalf("拒否した追加がメンバーシップを作った: %v", got)
	}
	if err := groupusecases.AddMember(ctx, deps, "operator", groupID, "active", groupRulesNow); err != nil {
		t.Fatal(err)
	}
	*events = nil
	if err := groupusecases.AddMember(ctx, deps, "operator", groupID, "active", groupRulesNow); err != nil {
		t.Fatal(err)
	}
	if err := groupusecases.RemoveMember(ctx, deps, "operator", groupID, "disabled", groupRulesNow); err != nil {
		t.Fatal(err)
	}
	if len(*events) != 0 {
		t.Fatalf("events=%v, want none", eventTypes(*events))
	}
}

//spec:covers EX-IDMANAGEMENT-064-01: 下流への通知に失敗したメンバーの追加がエラーを返し、確定したメンバーシップと GroupMemberAdded を取り消さないこと。
func TestAddMemberKeepsTheCommittedChangeWhenNotificationFails(t *testing.T) {
	deps, events, groupID := memberDepsWithUsers(t)
	deps.ProvisioningNotifier = &recordingGroupNotifier{err: errors.New("provisioning outage")}
	if err := groupusecases.AddMember(context.Background(), deps, "operator", groupID, "active", groupRulesNow); err == nil {
		t.Fatalf("err=nil, want the notification failure")
	}
	if got := memberIDs(t, deps, groupID); !slices.Equal(got, []string{"active"}) {
		t.Fatalf("members=%v, want [active]", got)
	}
	if got := eventTypes(*events); !slices.Equal(got, []string{"GroupMemberAdded"}) {
		t.Fatalf("events=%v, want [GroupMemberAdded]", got)
	}
}

//spec:covers EX-IDMANAGEMENT-065-01, EX-IDMANAGEMENT-065-02, EX-IDMANAGEMENT-065-03: ロールの参照、真偽値を返さない式、定数でない正規表現、許可外の関数を含む規則の保存を拒否し、規則を作らないこと。
func TestDynamicRuleExpressionConstraints(t *testing.T) {
	ctx, deps, groups, _ := dynamicRuleFixture(t)
	for _, expression := range []string{
		`"admin" in user.roles`,
		`user.preferred_username`,
		`user.email.matches(user.preferred_username)`,
		`user.department.upperAscii() == "X"`,
	} {
		if _, err := groupusecases.UpdateDynamicGroupRule(ctx, deps, "admin", dynamicFixtureGroupID, expression, groupRulesNow); !errors.Is(err, groupusecases.ErrInvalidDynamicGroupRule) {
			t.Fatalf("%s: err=%v, want ErrInvalidDynamicGroupRule", expression, err)
		}
	}
	if rule, _ := groups.FindDynamicRule(ctx, "acme", dynamicFixtureGroupID); rule != nil {
		t.Fatalf("拒否した保存が規則を作った: %+v", rule)
	}
}

func reconcileJobs(t *testing.T, jobs *jobsmemory.JobRepository) []groupusecases.DynamicGroupReconcileParams {
	t.Helper()
	listed, err := jobs.ListByTenantAndKinds(context.Background(), "acme", []jobsdomain.JobKind{jobsdomain.KindDynamicGroupReconcile}, 100)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]groupusecases.DynamicGroupReconcileParams, len(listed))
	for i, job := range listed {
		if err := json.Unmarshal(job.Params, &out[i]); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

//spec:covers EX-IDMANAGEMENT-066-01, EX-IDMANAGEMENT-066-02: 初めて保存した規則が無効で版 1、有効化で版 2 となって再評価を予約し、有効な規則の式の保存が有効のまま版 3 にして再評価を予約すること。
func TestDynamicRuleVersionsAndScheduling(t *testing.T) {
	ctx, deps, _, _ := dynamicRuleFixture(t)
	jobs := jobsmemory.NewJobRepository()
	deps.JobRepo = jobs
	saved, err := groupusecases.UpdateDynamicGroupRule(ctx, deps, "admin", dynamicFixtureGroupID, `user.department == "Engineering"`, groupRulesNow)
	if err != nil || saved.Enabled || saved.Version != 1 {
		t.Fatalf("saved=%+v err=%v, want disabled v1", saved, err)
	}
	if got := reconcileJobs(t, jobs); len(got) != 0 {
		t.Fatalf("無効な規則の保存が再評価を予約した: %+v", got)
	}
	enabled, err := groupusecases.SetDynamicGroupRuleEnabled(ctx, deps, "admin", dynamicFixtureGroupID, true, groupRulesNow)
	if err != nil || !enabled.Enabled || enabled.Version != 2 {
		t.Fatalf("enabled=%+v err=%v, want enabled v2", enabled, err)
	}
	again, err := groupusecases.SetDynamicGroupRuleEnabled(ctx, deps, "admin", dynamicFixtureGroupID, true, groupRulesNow)
	if err != nil || again.Version != 2 {
		t.Fatalf("同じ状態の有効化: rule=%+v err=%v, want v2", again, err)
	}
	updated, err := groupusecases.UpdateDynamicGroupRule(ctx, deps, "admin", dynamicFixtureGroupID, `user.department == "Sales"`, groupRulesNow)
	if err != nil || !updated.Enabled || updated.Version != 3 {
		t.Fatalf("updated=%+v err=%v, want enabled v3", updated, err)
	}
	versions := []int64{}
	for _, params := range reconcileJobs(t, jobs) {
		versions = append(versions, params.RuleVersion)
	}
	slices.Sort(versions)
	if !slices.Equal(versions, []int64{2, 3}) {
		t.Fatalf("予約した再評価の版=%v, want [2 3]", versions)
	}
}

//spec:covers REQ-IDMANAGEMENT-066: 手動グループへの規則の保存を ErrDynamicMembershipManaged で拒否し、規則のない Group の有効化を ErrInvalidDynamicGroupRule で拒否すること。
func TestDynamicRuleRefusesManualGroupsAndMissingRules(t *testing.T) {
	ctx, deps, groups, _ := dynamicRuleFixture(t)
	if err := groups.Save(ctx, &groupdomain.Group{ID: "manual", TenantID: "acme", Name: "manual", MembershipType: groupdomain.GroupMembershipManual, CreatedAt: groupRulesNow, UpdatedAt: groupRulesNow}); err != nil {
		t.Fatal(err)
	}
	if _, err := groupusecases.UpdateDynamicGroupRule(ctx, deps, "admin", "manual", `user.department == "Engineering"`, groupRulesNow); !errors.Is(err, groupusecases.ErrDynamicMembershipManaged) {
		t.Fatalf("err=%v, want ErrDynamicMembershipManaged", err)
	}
	if _, err := groupusecases.SetDynamicGroupRuleEnabled(ctx, deps, "admin", dynamicFixtureGroupID, true, groupRulesNow); !errors.Is(err, groupusecases.ErrInvalidDynamicGroupRule) {
		t.Fatalf("err=%v, want ErrInvalidDynamicGroupRule", err)
	}
}

//spec:covers EX-IDMANAGEMENT-067-01: Active でない User が式の値によらず規則に一致しないこと。
func TestDynamicRuleMatchesOnlyActiveUsers(t *testing.T) {
	compiled, err := groupdomain.CompileDynamicGroupRule(`user.department == "Engineering"`, []userdomain.UserAttributeDef{{Key: "department", Type: idmdomain.AttributeTypeString}})
	if err != nil {
		t.Fatal(err)
	}
	engineering := "Engineering"
	for status, want := range map[idmdomain.UserStatus]bool{
		idmdomain.UserStatusActive: true, idmdomain.UserStatusDisabled: false, idmdomain.UserStatusPendingDeletion: false,
	} {
		matched, err := compiled.Evaluate(userdomain.User{
			ID: "u", Lifecycle: userdomain.UserLifecycle{Status: status},
			Attributes: map[string]userdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &engineering}},
		})
		if err != nil || matched != want {
			t.Fatalf("%s: matched=%v err=%v, want %v", status, matched, err, want)
		}
	}
}

//spec:covers EX-IDMANAGEMENT-068-01: 規則の無効化が再評価を予約せず、その場で動的グループのメンバーシップをすべて外すこと。
func TestDisablingADynamicRuleRemovesEveryMembershipImmediately(t *testing.T) {
	ctx, deps, groups, _ := dynamicRuleFixture(t)
	if _, err := groupusecases.UpdateDynamicGroupRule(ctx, deps, "admin", dynamicFixtureGroupID, `user.department != ""`, groupRulesNow); err != nil {
		t.Fatal(err)
	}
	if _, err := groupusecases.SetDynamicGroupRuleEnabled(ctx, deps, "admin", dynamicFixtureGroupID, true, groupRulesNow); err != nil {
		t.Fatal(err)
	}
	if got := dynamicMemberIDs(ctx, t, groups); len(got) != 2 {
		t.Fatalf("前提が壊れている: members=%v", got)
	}
	jobs := jobsmemory.NewJobRepository()
	deps.JobRepo = jobs
	if _, err := groupusecases.SetDynamicGroupRuleEnabled(ctx, deps, "admin", dynamicFixtureGroupID, false, groupRulesNow); err != nil {
		t.Fatal(err)
	}
	if got := dynamicMemberIDs(ctx, t, groups); len(got) != 0 {
		t.Fatalf("無効化の後も所属が残る: %v", got)
	}
	if got := reconcileJobs(t, jobs); len(got) != 0 {
		t.Fatalf("無効化が再評価を予約した: %+v", got)
	}
}

//spec:covers EX-IDMANAGEMENT-069-01: 規則の版が進んだ後に実行された古い版の再評価のジョブが、メンバーシップを変えずに成功すること。
func TestStaleReconcileJobLeavesMembershipUntouched(t *testing.T) {
	ctx, deps, groups, users := dynamicRuleFixture(t)
	if _, err := groupusecases.UpdateDynamicGroupRule(ctx, deps, "admin", dynamicFixtureGroupID, `user.department == "Engineering"`, groupRulesNow); err != nil {
		t.Fatal(err)
	}
	if _, err := groupusecases.SetDynamicGroupRuleEnabled(ctx, deps, "admin", dynamicFixtureGroupID, true, groupRulesNow); err != nil {
		t.Fatal(err)
	}
	// 再評価すれば所属が変わる状態を作る。所属が変わらない母集団では、古いジョブを
	// 実行してしまう実装も同じ結果になる。
	sales, err := users.FindBySub(ctx, "u_sales")
	if err != nil || sales == nil {
		t.Fatalf("FindBySub=(%v,%v)", sales, err)
	}
	moved := *sales
	moved.Attributes = map[string]userdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: new("Engineering")}}
	if err := users.Save(ctx, &moved); err != nil {
		t.Fatal(err)
	}
	run := func(version int64) {
		t.Helper()
		params, err := json.Marshal(groupusecases.DynamicGroupReconcileParams{GroupID: dynamicFixtureGroupID, RuleVersion: version})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := groupusecases.DynamicGroupReconcileHandler(deps)(ctx, &jobsdomain.Job{TenantID: "acme", Params: params}); err != nil {
			t.Fatalf("版 %d のジョブ: err=%v, want success", version, err)
		}
	}
	run(1)
	if after := dynamicMemberIDs(ctx, t, groups); !slices.Equal(after, []string{"u_eng"}) {
		t.Fatalf("古いジョブが所属を変えた: %v", after)
	}
	// 対照: 現在の版のジョブは同じ状態から所属を変える。
	run(2)
	if after := dynamicMemberIDs(ctx, t, groups); !slices.Equal(after, []string{"u_eng", "u_sales"}) {
		t.Fatalf("現在の版のジョブ: members=%v, want [u_eng u_sales]", after)
	}
}

//spec:covers EX-IDMANAGEMENT-070-01: 全件の再評価による追加が GroupMemberAdded を発行せず、追加の件数を載せた DynamicMembershipEvaluated を一つ発行すること。
func TestDynamicMembershipChangesEmitOnlyTheEvaluationSummary(t *testing.T) {
	ctx, deps, groups, _ := dynamicRuleFixture(t)
	var events []spec.DomainEvent
	deps.Emit = func(event spec.DomainEvent) error { events = append(events, event); return nil }
	if _, err := groupusecases.UpdateDynamicGroupRule(ctx, deps, "admin", dynamicFixtureGroupID, `user.department == "Engineering"`, groupRulesNow); err != nil {
		t.Fatal(err)
	}
	if _, err := groupusecases.SetDynamicGroupRuleEnabled(ctx, deps, "admin", dynamicFixtureGroupID, true, groupRulesNow); err != nil {
		t.Fatal(err)
	}
	if got := dynamicMemberIDs(ctx, t, groups); !slices.Equal(got, []string{"u_eng"}) {
		t.Fatalf("members=%v, want [u_eng]", got)
	}
	var evaluated []*idmdomain.DynamicMembershipEvaluated
	for _, event := range events {
		switch e := event.(type) {
		case *idmdomain.GroupMemberAdded, *idmdomain.GroupMemberRemoved:
			t.Fatalf("動的な所属の変化が %s を発行した", event.EventType())
		case *idmdomain.DynamicMembershipEvaluated:
			evaluated = append(evaluated, e)
		}
	}
	if len(evaluated) != 1 || evaluated[0].AddedCount != 1 || evaluated[0].RemovedCount != 0 {
		t.Fatalf("DynamicMembershipEvaluated=%+v, want one with added=1 removed=0", evaluated)
	}
	// 二度目の再評価は、在籍を外して入れ直さず変化なしとして数える。
	rule, err := groups.FindDynamicRule(ctx, "acme", dynamicFixtureGroupID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := groupusecases.ReconcileDynamicGroup(ctx, deps, rule, groupRulesNow)
	if err != nil {
		t.Fatal(err)
	}
	if result != (groupusecases.DynamicReconcileResult{Unchanged: 1}) {
		t.Fatalf("result=%+v, want unchanged=1 only", result)
	}
}

//spec:covers EX-IDMANAGEMENT-071-01, EX-IDMANAGEMENT-071-02: 101 件の User のプレビューを ErrInvalidDynamicGroupRule で、存在しない User を含むプレビューを結果なしの ErrUserNotFound で拒否すること。
func TestDynamicRulePreviewLimitsAndUnknownUsers(t *testing.T) {
	ctx, deps, _, _ := dynamicRuleFixture(t)
	many := make([]string, 101)
	for i := range many {
		many[i] = "u_eng"
	}
	if _, err := groupusecases.PreviewDynamicGroupRule(ctx, deps, dynamicFixtureGroupID, `user.department == "Engineering"`, many); !errors.Is(err, groupusecases.ErrInvalidDynamicGroupRule) {
		t.Fatalf("101 件: err=%v, want ErrInvalidDynamicGroupRule", err)
	}
	if _, err := groupusecases.PreviewDynamicGroupRule(ctx, deps, dynamicFixtureGroupID, `user.department == "Engineering"`, many[:100]); err != nil {
		t.Fatalf("100 件: err=%v, want accepted", err)
	}
	results, err := groupusecases.PreviewDynamicGroupRule(ctx, deps, dynamicFixtureGroupID, `user.department == "Engineering"`, []string{"u_eng", "nobody"})
	if !errors.Is(err, idmusecases.ErrUserNotFound) || results != nil {
		t.Fatalf("results=%+v err=%v, want no results and ErrUserNotFound", results, err)
	}
}
