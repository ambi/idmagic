package usecases_test

// メンバーシップ CSV の計画と適用。ここでの主張は 2 本立てである — 呼び出し元が
// 観測する行操作と、リポジトリに実際に起きた (あるいは起きなかった) こと。
// 拒否は必ず「触れなかったもの」まで読み戻す。

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	idmmemory "github.com/ambi/idmagic/backend/idmanagement/db_memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	membershipTenant = tenancydomain.DefaultTenantID
	engineeringID    = "group-engineering"
	salesID          = "group-sales"
)

// membershipOwnership は所有権ガードの答え方を 1 つの型で表す。判定不能は Group と
// User で別々に起こせる。両方をまとめて落とすと Group 側が先に拒否してしまい、
// User 側の fail-closed がテストに触れられないまま残る。
type membershipOwnership struct {
	managedGroups    map[string]bool
	managedUsers     map[string]bool
	groupUnavailable bool
	userUnavailable  bool
}

func (g membershipOwnership) SourceManagedGroupIDs(_ context.Context, _ string, ids []string) (map[string]bool, error) {
	if g.groupUnavailable {
		return nil, errors.New("the group ownership record is unreachable")
	}
	return membershipOwnershipAnswer(ids, g.managedGroups), nil
}

func (g membershipOwnership) SourceManagedUserIDs(_ context.Context, _ string, ids []string) (map[string]bool, error) {
	if g.userUnavailable {
		return nil, errors.New("the user ownership record is unreachable")
	}
	return membershipOwnershipAnswer(ids, g.managedUsers), nil
}

func membershipOwnershipAnswer(ids []string, managed map[string]bool) map[string]bool {
	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		out[id] = managed[id]
	}
	return out
}

// recordingMembershipCommitter は確定ポートへ渡った書き込み集合そのものを記録する。
// リポジトリの読み戻しだけでは、行が監査種別や解除の向きを運んでいるかを観測できない。
type recordingMembershipCommitter struct {
	delegate  groupports.GroupMembershipImportRowCommitter
	mutations []groupports.GroupMembershipImportRowMutation
	failFor   map[string]bool
}

func (c *recordingMembershipCommitter) CommitGroupMembershipImportRow(
	ctx context.Context, mutation groupports.GroupMembershipImportRowMutation,
) error {
	c.mutations = append(c.mutations, mutation)
	if c.failFor[mutation.UserID] {
		return errors.New("the row could not be committed")
	}
	return c.delegate.CommitGroupMembershipImportRow(ctx, mutation)
}

type membershipFixture struct {
	t          *testing.T
	ctx        context.Context
	groups     *groupmemory.GroupRepository
	users      *usermemory.UserRepository
	committer  *recordingMembershipCommitter
	planDeps   groupusecases.GroupMembershipImportPlanDeps
	applyDeps  groupusecases.GroupMembershipImportApplyDeps
	ownership  membershipOwnership
	memberSeed []string
}

func newMembershipFixture(t *testing.T, membershipType groupdomain.GroupMembershipType, ownership membershipOwnership) *membershipFixture {
	t.Helper()
	now := time.Date(2026, 9, 6, 9, 0, 0, 0, time.UTC)
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: membershipTenant}, "", "")
	users := usermemory.NewUserRepository()
	for _, username := range []string{"alice", "bob", "carol", "dave"} {
		users.Seed(&userdomain.User{
			ID: "user-" + username, TenantID: membershipTenant, PreferredUsername: username,
			PasswordHash: "unused", CreatedAt: now, UpdatedAt: now,
		})
	}
	groups := groupmemory.NewGroupRepository()
	for _, group := range []*groupdomain.Group{
		{
			ID: engineeringID, TenantID: membershipTenant, Name: "engineering", Roles: []string{"catalog:read"},
			MembershipType: membershipType, CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: salesID, TenantID: membershipTenant, Name: "sales", Roles: []string{"invoice:read"},
			MembershipType: groupdomain.GroupMembershipManual, CreatedAt: now, UpdatedAt: now,
		},
	} {
		if err := groups.Save(ctx, group); err != nil {
			t.Fatal(err)
		}
	}
	seed := []string{"user-alice", "user-carol"}
	for _, userID := range seed {
		if _, err := groups.AddMember(ctx, &groupdomain.GroupMember{
			GroupID: engineeringID, UserID: userID, Source: groupdomain.MembershipSourceManual, CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	committer := &recordingMembershipCommitter{delegate: groupmemory.NewGroupMembershipImportRowCommitter(groups)}
	planDeps := groupusecases.GroupMembershipImportPlanDeps{
		GroupRepo: groups, UserRepo: users,
		GroupOwnershipGuard: ownership, UserOwnershipGuard: ownership,
	}
	return &membershipFixture{
		t: t, ctx: ctx, groups: groups, users: users, committer: committer,
		planDeps:  planDeps,
		applyDeps: groupusecases.GroupMembershipImportApplyDeps{Plan: planDeps, Committer: committer},
		ownership: ownership, memberSeed: seed,
	}
}

func (f *membershipFixture) plan(document string) (groupusecases.GroupMembershipImportPlanSummary, []groupdomain.GroupMembershipImportRowPlan, error) {
	f.t.Helper()
	var rows []groupdomain.GroupMembershipImportRowPlan
	summary, err := groupusecases.PlanGroupMembershipImport(
		f.ctx, f.planDeps, engineeringID, strings.NewReader(document), idmdomain.DefaultCSVTransferPolicy(),
		func(row groupdomain.GroupMembershipImportRowPlan) error {
			rows = append(rows, row)
			return nil
		})
	return summary, rows, err
}

func (f *membershipFixture) apply(document string) (groupusecases.GroupMembershipImportPlanSummary, error) {
	f.t.Helper()
	return groupusecases.ApplyGroupMembershipImport(
		f.ctx, f.applyDeps, engineeringID, strings.NewReader(document), idmdomain.DefaultCSVTransferPolicy(),
		"user-admin", time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC), nil)
}

func (f *membershipFixture) memberIDs(groupID string) []string {
	f.t.Helper()
	members, err := f.groups.ListMembersByGroup(f.ctx, membershipTenant, groupID)
	if err != nil {
		f.t.Fatal(err)
	}
	ids := make([]string, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.UserID)
	}
	return ids
}

func (f *membershipFixture) assertMembersUnchanged() {
	f.t.Helper()
	if got := f.memberIDs(engineeringID); !sameStrings(got, f.memberSeed) {
		f.t.Fatalf("membership changed to %v, want it left at %v", got, f.memberSeed)
	}
	if got := f.memberIDs(salesID); len(got) != 0 {
		f.t.Fatalf("the other group gained members: %v", got)
	}
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func rowCodes(rows []groupdomain.GroupMembershipImportRowPlan) []string {
	codes := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Error != nil {
			codes = append(codes, string(row.Error.Code))
			continue
		}
		codes = append(codes, string(row.Action))
	}
	return codes
}

// 解除される。件数だけでなく、確定ポートへ渡った書き込み集合の向きと監査種別も見る。
// 「拒否コードを返しつつ処理は続ける」実装は前者を通っても後者で落ちる。
//
//spec:covers REQ-IDMANAGEMENT-029 / EX-IDMANAGEMENT-029-01: `present` の行だけが追加され、`absent` の行だけが
func TestGroupMembershipImportPreviewThenApplyAddsAndReleasesOnlyTheDeclaredRows(t *testing.T) {
	f := newMembershipFixture(t, groupdomain.GroupMembershipManual, membershipOwnership{})
	document := "user_id,preferred_username,membership_state\n" +
		"user-bob,bob,present\n" +
		"user-alice,alice,absent\n" +
		"user-carol,carol,present\n"

	summary, rows, err := f.plan(document)
	if err != nil {
		t.Fatal(err)
	}
	if got := rowCodes(rows); !sameStrings(got, []string{"added", "removed", "unchanged"}) {
		t.Fatalf("plan = %v, want added / removed / unchanged", got)
	}
	if summary.AddedRows != 1 || summary.RemovedRows != 1 || summary.UnchangedRows != 1 || summary.RejectedRows != 0 {
		t.Fatalf("summary = %+v", summary)
	}
	f.assertMembersUnchanged()

	applied, err := f.apply(document)
	if err != nil {
		t.Fatal(err)
	}
	if applied.AddedRows != 1 || applied.RemovedRows != 1 || applied.UnchangedRows != 1 {
		t.Fatalf("applied = %+v", applied)
	}
	if got := f.memberIDs(engineeringID); !sameStrings(got, []string{"user-bob", "user-carol"}) {
		t.Fatalf("membership = %v, want bob added and alice released", got)
	}
	if len(f.committer.mutations) != 2 {
		t.Fatalf("the committer saw %d mutations, want exactly the two rows that change something",
			len(f.committer.mutations))
	}
	add, release := f.committer.mutations[0], f.committer.mutations[1]
	if add.Release || add.UserID != "user-bob" || add.AuditEventType != "GroupMemberAdded" ||
		add.Member == nil || add.Member.Source != groupdomain.MembershipSourceManual {
		t.Fatalf("the add mutation is %+v", add)
	}
	if !release.Release || release.UserID != "user-alice" || release.AuditEventType != "GroupMemberRemoved" ||
		release.Member != nil {
		t.Fatalf("the release mutation is %+v", release)
	}
	for _, mutation := range f.committer.mutations {
		if mutation.GroupID != engineeringID || mutation.TenantID != membershipTenant || mutation.ActorUserID != "user-admin" {
			t.Fatalf("a mutation escaped its group, tenant, or actor: %+v", mutation)
		}
	}
}

// authoritative full-sync を採らなかったことの観測点であり、分割ファイルの安全性
// そのものである。
//
//spec:covers REQ-IDMANAGEMENT-030 / EX-IDMANAGEMENT-030-01: ファイルが名指ししない行は変更しない。ここが
func TestGroupMembershipImportLeavesRowsTheFileDoesNotName(t *testing.T) {
	f := newMembershipFixture(t, groupdomain.GroupMembershipManual, membershipOwnership{})
	// alice の行だけを含むファイル。carol はファイルに現れない。
	applied, err := f.apply("user_id,membership_state\nuser-alice,present\n")
	if err != nil {
		t.Fatal(err)
	}
	if applied.TotalRows != 1 || applied.RemovedRows != 0 || applied.UnchangedRows != 1 {
		t.Fatalf("applied = %+v, want the single named row planned as unchanged", applied)
	}
	if got := f.memberIDs(engineeringID); !sameStrings(got, []string{"user-alice", "user-carol"}) {
		t.Fatalf("membership = %v, want the unnamed member kept", got)
	}
	if len(f.committer.mutations) != 0 {
		t.Fatalf("an unchanged file reached the committer: %+v", f.committer.mutations)
	}
}

// CSV から書き換えない。判定不能も所有と同じに扱う。
//
//spec:covers REQ-IDMANAGEMENT-031 / EX-IDMANAGEMENT-031-01, EX-IDMANAGEMENT-031-02, EX-IDMANAGEMENT-031-03, EX-IDMANAGEMENT-031-04, EX-IDMANAGEMENT-031-05: 上位の権威が所有する所属は、
func TestGroupMembershipImportFailsClosedForDynamicAndSourceManagedAuthorities(t *testing.T) {
	document := "user_id,membership_state\nuser-bob,present\nuser-alice,absent\n"

	t.Run("動的グループはファイル全体を拒否する", func(t *testing.T) {
		f := newMembershipFixture(t, groupdomain.GroupMembershipDynamic, membershipOwnership{})
		_, _, err := f.plan(document)
		assertMembershipFileRefusal(t, err, "dynamic_group")
		if _, err := f.apply(document); err == nil {
			t.Fatal("apply accepted a dynamic group")
		}
		f.assertMembersUnchanged()
	})

	t.Run("外部所有のグループはファイル全体を拒否する", func(t *testing.T) {
		f := newMembershipFixture(t, groupdomain.GroupMembershipManual,
			membershipOwnership{managedGroups: map[string]bool{engineeringID: true}})
		_, _, err := f.plan(document)
		assertMembershipFileRefusal(t, err, "source_managed")
		if _, err := f.apply(document); err == nil {
			t.Fatal("apply accepted a source-managed group")
		}
		f.assertMembersUnchanged()
	})

	t.Run("所有権を判定できないグループはファイル全体を拒否する", func(t *testing.T) {
		f := newMembershipFixture(t, groupdomain.GroupMembershipManual,
			membershipOwnership{groupUnavailable: true})
		_, _, err := f.plan(document)
		assertMembershipFileRefusal(t, err, "source_managed")
		f.assertMembersUnchanged()
	})

	// User 側の判定不能は Group 側とは別に観測する。両方をまとめて落とすと Group の
	// 拒否が先に効いてしまい、この分岐は一度も通らない。
	t.Run("所有権を判定できない User はその行を拒否する", func(t *testing.T) {
		f := newMembershipFixture(t, groupdomain.GroupMembershipManual,
			membershipOwnership{userUnavailable: true})
		_, rows, err := f.plan(document)
		if err != nil {
			t.Fatal(err)
		}
		if got := rowCodes(rows); !sameStrings(got, []string{"source_managed", "source_managed"}) {
			t.Fatalf("plan = %v, want every row refused fail-closed", got)
		}
		applied, err := f.apply(document)
		if err != nil {
			t.Fatal(err)
		}
		if applied.RejectedRows != 2 || applied.AddedRows != 0 || applied.RemovedRows != 0 {
			t.Fatalf("applied = %+v, want nothing written", applied)
		}
		f.assertMembersUnchanged()
	})

	t.Run("対象グループが存在しなければファイル全体を拒否する", func(t *testing.T) {
		f := newMembershipFixture(t, groupdomain.GroupMembershipManual, membershipOwnership{})
		if err := f.groups.Delete(f.ctx, membershipTenant, engineeringID); err != nil {
			t.Fatal(err)
		}
		_, _, err := f.plan(document)
		assertMembershipFileRefusal(t, err, "target_not_found")
	})

	t.Run("外部所有の User はその行だけを拒否する", func(t *testing.T) {
		f := newMembershipFixture(t, groupdomain.GroupMembershipManual,
			membershipOwnership{managedUsers: map[string]bool{"user-bob": true}})
		applied, err := f.apply(document)
		if err != nil {
			t.Fatal(err)
		}
		if applied.RejectedRows != 1 || applied.RemovedRows != 1 {
			t.Fatalf("applied = %+v, want the owned row refused and the local row applied", applied)
		}
		if got := f.memberIDs(engineeringID); !sameStrings(got, []string{"user-carol"}) {
			t.Fatalf("membership = %v, want bob refused and alice released", got)
		}
	})

	t.Run("動的規則が作った所属は present でも absent でも拒否する", func(t *testing.T) {
		f := newMembershipFixture(t, groupdomain.GroupMembershipManual, membershipOwnership{})
		ruleVersion := int64(1)
		if _, err := f.groups.AddMember(f.ctx, &groupdomain.GroupMember{
			GroupID: engineeringID, UserID: "user-dave", Source: groupdomain.MembershipSourceDynamicRule,
			RuleVersion: &ruleVersion, CreatedAt: time.Now().UTC(),
		}); err != nil {
			t.Fatal(err)
		}
		_, rows, err := f.plan("user_id,membership_state\nuser-dave,present\n")
		if err != nil {
			t.Fatal(err)
		}
		if got := rowCodes(rows); !sameStrings(got, []string{"dynamic_membership"}) {
			t.Fatalf("plan for a rule-owned membership = %v", got)
		}
		_, rows, err = f.plan("user_id,membership_state\nuser-dave,absent\n")
		if err != nil {
			t.Fatal(err)
		}
		if got := rowCodes(rows); !sameStrings(got, []string{"dynamic_membership"}) {
			t.Fatalf("plan for releasing a rule-owned membership = %v", got)
		}
		if got := f.memberIDs(engineeringID); len(got) != 3 {
			t.Fatalf("the refusal changed membership: %v", got)
		}
	})
}

func assertMembershipFileRefusal(t *testing.T, err error, want idmdomain.CSVErrorCode) {
	t.Helper()
	var csvErr *idmdomain.CSVError
	if !errors.As(err, &csvErr) {
		t.Fatalf("err = %v, want a CSV error refusing the whole file", err)
	}
	if csvErr.Code != want {
		t.Fatalf("code = %q, want %q", csvErr.Code, want)
	}
}

// 識別子、別グループを指す照合列。どれもメンバーシップを変えない。
//
//spec:covers EX-IDMANAGEMENT-029-04, EX-IDMANAGEMENT-029-05, EX-IDMANAGEMENT-029-06, EX-IDMANAGEMENT-029-07: 意図の列を欠いたファイル、丸められない値、解決できない
func TestGroupMembershipImportRefusesUnusableRowsWithoutTouchingMembership(t *testing.T) {
	t.Run("membership_state 列が無いファイルは受理しない", func(t *testing.T) {
		f := newMembershipFixture(t, groupdomain.GroupMembershipManual, membershipOwnership{})
		_, _, err := f.plan("user_id,preferred_username\nuser-bob,bob\n")
		assertMembershipFileRefusal(t, err, idmdomain.CSVErrorInvalidHeader)
		var csvErr *idmdomain.CSVError
		_ = errors.As(err, &csvErr)
		if csvErr.Column != "membership_state" {
			t.Fatalf("column = %q, want the missing intent column named", csvErr.Column)
		}
		f.assertMembersUnchanged()
	})

	t.Run("行ごとの拒否", func(t *testing.T) {
		f := newMembershipFixture(t, groupdomain.GroupMembershipManual, membershipOwnership{})
		document := "group_id,group_name,user_id,preferred_username,membership_state\n" +
			",,user-bob,bob,\n" + // 空セルは丸めない
			",,user-bob,bob,maybe\n" + // 未知の値も丸めない
			",,,,present\n" + // 識別子が無い
			",,user-bob,carol,present\n" + // id と username が食い違う
			",,user-nobody,,present\n" + // 解決できない
			"group-sales,,user-bob,bob,present\n" + // 別グループの id
			",Sales,user-bob,bob,absent\n" + // 別グループの name
			",,user-alice,alice,absent\n" + // ここだけ通る
			",,user-alice,alice,present\n" // 同じ User の 2 度目
		_, rows, err := f.plan(document)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{
			"invalid_membership_state", "invalid_membership_state", "missing_identifier",
			"identifier_mismatch", "target_not_found", "group_mismatch", "group_mismatch",
			"removed", "duplicate_target",
		}
		if got := rowCodes(rows); !sameStrings(got, want) {
			t.Fatalf("plan = %v,\nwant %v", got, want)
		}
		f.assertMembersUnchanged()
	})

	//spec:covers EX-IDMANAGEMENT-030-04: 読み取り専用の列を編集しても行操作は変わらない。
	// `group_name` だけは照合列なので、別のグループを指したときにだけ行を拒否する。
	t.Run("読み取り専用の列は受理して無視する", func(t *testing.T) {
		f := newMembershipFixture(t, groupdomain.GroupMembershipManual, membershipOwnership{})
		// `source` と `created_at` に何を書いても、行操作は変わらない。
		_, rows, err := f.plan(
			"user_id,membership_state,source,created_at\nuser-alice,present,dynamic_rule,2001-01-01T00:00:00Z\n")
		if err != nil {
			t.Fatal(err)
		}
		if got := rowCodes(rows); !sameStrings(got, []string{"unchanged"}) {
			t.Fatalf("plan = %v, want the read-only cells ignored", got)
		}
		// 同じ `group_name` を大小まちまちに書いても、対象は同じ Group である。
		_, rows, err = f.plan("group_name,user_id,membership_state\nENGINEERING,user-alice,present\n")
		if err != nil {
			t.Fatal(err)
		}
		if got := rowCodes(rows); !sameStrings(got, []string{"unchanged"}) {
			t.Fatalf("plan = %v, want the group name matched case-insensitively", got)
		}
	})
}

// 1 行も計画せずファイルごと拒否される。上限は行数、byte 数、項目長のそれぞれに効く。
//
//spec:covers EX-IDMANAGEMENT-029-02: 実効転送ポリシーの上限を超えたインポートは、
func TestGroupMembershipImportRefusesFilesBeyondTheTransferPolicy(t *testing.T) {
	cases := []struct {
		name     string
		policy   idmdomain.CSVTransferPolicy
		document string
		want     idmdomain.CSVErrorCode
	}{
		{
			name:     "行数の上限",
			policy:   idmdomain.CSVTransferPolicy{MaxRows: 1, MaxBytes: 1 << 20, MaxFieldBytes: 1 << 10},
			document: "user_id,membership_state\nuser-alice,present\nuser-bob,present\n",
			want:     idmdomain.CSVErrorTooManyRows,
		},
		{
			name:     "byte 数の上限",
			policy:   idmdomain.CSVTransferPolicy{MaxRows: 100, MaxBytes: 24, MaxFieldBytes: 1 << 10},
			document: "user_id,membership_state\nuser-alice,present\nuser-bob,present\n",
			want:     idmdomain.CSVErrorCSVTooLarge,
		},
		{
			name:     "項目長の上限",
			policy:   idmdomain.CSVTransferPolicy{MaxRows: 100, MaxBytes: 1 << 20, MaxFieldBytes: 8},
			document: "user_id,membership_state\n" + strings.Repeat("x", 64) + ",present\n",
			want:     idmdomain.CSVErrorFieldTooLarge,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newMembershipFixture(t, groupdomain.GroupMembershipManual, membershipOwnership{})
			var rows []groupdomain.GroupMembershipImportRowPlan
			_, err := groupusecases.PlanGroupMembershipImport(
				f.ctx, f.planDeps, engineeringID, strings.NewReader(tc.document), tc.policy,
				func(row groupdomain.GroupMembershipImportRowPlan) error {
					rows = append(rows, row)
					return nil
				})
			assertMembershipFileRefusal(t, err, tc.want)
			// 上限は読みながら効く。ファイル全体を読み込んでから文句を言う実装は、
			// 上限を超える数の行を計画してしまうのでここで落ちる。
			if len(rows) > tc.policy.MaxRows {
				t.Fatalf("the planner emitted %d rows past a limit of %d", len(rows), tc.policy.MaxRows)
			}
			f.assertMembersUnchanged()
		})
	}
}

// 失敗し、再インポートできない成功済み成果物を作らない。成果物ストアに何も残らない
// ことまで読むのは、書き終えてから失敗する実装と区別するためである。
//
//spec:covers EX-IDMANAGEMENT-030-03: 生成結果が転送ポリシーを超えるエクスポートは
func TestGroupMembershipExportFailsRatherThanWriteAnUnimportableArtifact(t *testing.T) {
	f := newMembershipFixture(t, groupdomain.GroupMembershipManual, membershipOwnership{})
	artifacts := newMembershipArtifactStore()
	_, err := groupusecases.ExportGroupMembershipCSV(f.ctx, groupusecases.GroupMembershipCSVExportDeps{
		GroupRepo: f.groups, UserRepo: f.users, Artifacts: artifacts,
	}, engineeringID, groupdomain.NewGroupMembershipCSVSchema().ColumnKeys(),
		idmdomain.CSVTransferPolicy{MaxRows: 100, MaxBytes: 32, MaxFieldBytes: 1 << 10})
	var csvErr *idmdomain.CSVError
	if !errors.As(err, &csvErr) || csvErr.Code != idmdomain.CSVErrorCSVTooLarge {
		t.Fatalf("err = %v, want the export refused by the transfer policy", err)
	}

	// 対照: 上限内なら同じエクスポートが成功し、その成果物はプレビューできる。
	export, err := groupusecases.ExportGroupMembershipCSV(f.ctx, groupusecases.GroupMembershipCSVExportDeps{
		GroupRepo: f.groups, UserRepo: f.users, Artifacts: artifacts,
	}, engineeringID, groupdomain.NewGroupMembershipCSVSchema().ColumnKeys(), idmdomain.DefaultCSVTransferPolicy())
	if err != nil {
		t.Fatal(err)
	}
	summary, _, err := f.plan(membershipArtifactContent(t, artifacts, export.Artifact.Ref))
	if err != nil || summary.UnchangedRows != 2 {
		t.Fatalf("the successful export could not be previewed: %+v %v", summary, err)
	}
}

// 判定し直す。プレビューが暗黙の楽観的ロックの迂回路にならないことの観測点である。
//
//spec:covers EX-IDMANAGEMENT-029-09: 適用は古い計画を実行せず、現在の所属から
func TestGroupMembershipImportApplyReplansAgainstCurrentState(t *testing.T) {
	f := newMembershipFixture(t, groupdomain.GroupMembershipManual, membershipOwnership{})
	document := "user_id,membership_state\nuser-bob,present\nuser-alice,absent\n"

	summary, _, err := f.plan(document)
	if err != nil {
		t.Fatal(err)
	}
	if summary.AddedRows != 1 || summary.RemovedRows != 1 {
		t.Fatalf("preview = %+v", summary)
	}
	// プレビューのあとで、別経路が同じ 2 件を先に反映してしまう。
	if _, err := f.groups.AddMember(f.ctx, &groupdomain.GroupMember{
		GroupID: engineeringID, UserID: "user-bob", Source: groupdomain.MembershipSourceManual, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.groups.RemoveMember(f.ctx, membershipTenant, engineeringID, "user-alice"); err != nil {
		t.Fatal(err)
	}

	applied, err := f.apply(document)
	if err != nil {
		t.Fatal(err)
	}
	if applied.UnchangedRows != 2 || applied.AddedRows != 0 || applied.RemovedRows != 0 {
		t.Fatalf("applied = %+v, want both rows re-decided as unchanged", applied)
	}
	if len(f.committer.mutations) != 0 {
		t.Fatalf("the stale plan reached the committer: %+v", f.committer.mutations)
	}
}

// 先に受理した行は巻き戻らない。
//
//spec:covers EX-IDMANAGEMENT-029-10: 1 行の確定が失敗しても、その行だけが拒否になり、
func TestGroupMembershipImportKeepsRowsAppliedWhenALaterRowFails(t *testing.T) {
	f := newMembershipFixture(t, groupdomain.GroupMembershipManual, membershipOwnership{})
	f.committer.failFor = map[string]bool{"user-dave": true}

	applied, err := f.apply("user_id,membership_state\nuser-bob,present\nuser-dave,present\n")
	if err != nil {
		t.Fatal(err)
	}
	if applied.AddedRows != 1 || applied.RejectedRows != 1 {
		t.Fatalf("applied = %+v, want the first row kept and the second refused", applied)
	}
	if got := f.memberIDs(engineeringID); !sameStrings(got, []string{"user-alice", "user-bob", "user-carol"}) {
		t.Fatalf("membership = %v, want bob kept and dave absent", got)
	}
}

// 互換列でエクスポートし、無編集のプレビューが全行 `unchanged` になる。利用者名に数式の
// 引き金・アポストロフィー・カンマ・引用符・改行を含めることで、可逆な数式安全変換と
// RFC 4180 の引用を通しても `decode(encode(value))` が元の値と一致することを、往復
// そのもので観測する。容量の契約でもある。
//
//spec:covers EX-IDMANAGEMENT-030-01, EX-IDMANAGEMENT-030-02: 10,000 件の所属を全 import
func TestGroupMembershipImportTenThousandMembershipsRoundTripAsUnchanged(t *testing.T) {
	f := newMembershipFixture(t, groupdomain.GroupMembershipManual, membershipOwnership{})
	now := time.Date(2026, 9, 6, 9, 0, 0, 0, time.UTC)
	// 数式の引き金、既存のアポストロフィー、カンマ、引用符、改行を含む利用者名も、
	// 可逆な変換で往復する。
	usernames := map[string]string{}
	for i := range 10_000 {
		id := membershipUserID(i)
		usernames[id] = membershipHostileUsername(i, id)
		f.users.Seed(&userdomain.User{
			ID: id, TenantID: membershipTenant, PreferredUsername: usernames[id],
			PasswordHash: "unused", CreatedAt: now, UpdatedAt: now,
		})
		if _, err := f.groups.AddMember(f.ctx, &groupdomain.GroupMember{
			GroupID: engineeringID, UserID: id, Source: groupdomain.MembershipSourceManual, CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	artifacts := newMembershipArtifactStore()
	schema := groupdomain.NewGroupMembershipCSVSchema()
	export, err := groupusecases.ExportGroupMembershipCSV(f.ctx, groupusecases.GroupMembershipCSVExportDeps{
		GroupRepo: f.groups, UserRepo: f.users, Artifacts: artifacts,
	}, engineeringID, schema.ColumnKeys(), idmdomain.DefaultCSVTransferPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if export.TotalRows != 10_002 {
		t.Fatalf("the export wrote %d rows, want every membership", export.TotalRows)
	}
	document := membershipArtifactContent(t, artifacts, export.Artifact.Ref)
	if strings.Count(document, "present") != 10_002 {
		t.Fatalf("the export did not write `present` on every row")
	}

	// 往復した各セルが元の値に一致すること。行操作だけを見るテストは、識別子が
	// `user_id` で解決される以上、利用者名が壊れても `unchanged` のまま通ってしまう。
	assertMembershipUsernamesSurviveTheRoundTrip(t, document, usernames)

	summary, _, err := f.plan(document)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalRows != 10_002 || summary.UnchangedRows != 10_002 {
		t.Fatalf("round trip = %+v, want every row unchanged", summary)
	}
}

// membershipHostileUsername は表計算と RFC 4180 の双方に対して意地の悪い値を作る。
// 先頭文字は数式の引き金とアポストロフィーを順に取り、本体はカンマ・引用符・改行を含む。
// 空白そのものを先頭に置く引き金 (TAB / CR / LF) は使わない。識別子のセルは前後の
// 空白を意図的に落とすため、そこだけは往復ではなく正規化が働く境界である。可逆な
// 変換そのものは idmdomain の FuzzCSVFormulaSafeCodec が全引き金を通して見ている。
func membershipHostileUsername(i int, id string) string {
	leading := []string{"=", "+", "-", "@", "'"}[i%5]
	return leading + "user," + "\"" + id + "\"\nsecond"
}

func assertMembershipUsernamesSurviveTheRoundTrip(t *testing.T, document string, want map[string]string) {
	t.Helper()
	schema := groupdomain.NewGroupMembershipCSVSchema()
	reader, err := idmdomain.NewCSVReader(strings.NewReader(document), schema.Accepts, idmdomain.DefaultCSVTransferPolicy())
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for {
		record, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || record.Row == nil {
			t.Fatalf("re-reading the export failed: %v", err)
		}
		userID := record.Row.TrimmedCell("user_id")
		expected, tracked := want[userID]
		if !tracked {
			continue
		}
		cell, _ := record.Row.Cell("preferred_username")
		if cell.Raw != expected {
			t.Fatalf("decode(encode(%q)) = %q", expected, cell.Raw)
		}
		seen++
	}
	if seen != len(want) {
		t.Fatalf("the round trip carried %d of %d hostile usernames", seen, len(want))
	}
}

func membershipUserID(i int) string {
	return fmt.Sprintf("user-bulk-%05d", i)
}

func newMembershipArtifactStore() *idmmemory.CSVArtifactStore {
	return idmmemory.NewCSVArtifactStore()
}

func membershipArtifactContent(t *testing.T, artifacts *idmmemory.CSVArtifactStore, ref string) string {
	t.Helper()
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: membershipTenant}, "", "")
	reader, _, err := artifacts.OpenCSVArtifact(ctx, membershipTenant, ref)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
