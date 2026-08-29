package usecases

// Group CSV の受入境界。export → preview → apply を 1 本の経路として通し、
// 往復不変条件 (REQ-IDMANAGEMENT-027)、不変な membership_type と source guard
// (REQ-IDMANAGEMENT-026)、明示的な行操作による削除 (REQ-IDMANAGEMENT-028) を
// 「呼び出し元が観測するもの」と「拒否が触れなかったもの」の両方で確かめる。

import (
	"bytes"
	"context"
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
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func groupImportContext() context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "acme"}, "", "")
}

type groupImportFixture struct {
	ctx       context.Context
	groupRepo *groupmemory.GroupRepository
	userRepo  *usermemory.UserRepository
	artifacts *idmmemory.CSVArtifactStore
	committer *recordingGroupImportCommitter
	plan      GroupImportPlanDeps
	apply     GroupImportApplyDeps
	now       time.Time
}

func newGroupImportFixture(t *testing.T) *groupImportFixture {
	t.Helper()
	f := &groupImportFixture{
		ctx:       groupImportContext(),
		groupRepo: groupmemory.NewGroupRepository(),
		userRepo:  usermemory.NewUserRepository(),
		artifacts: idmmemory.NewCSVArtifactStore(),
		now:       time.Date(2026, 8, 29, 9, 0, 0, 0, time.UTC),
	}
	f.committer = &recordingGroupImportCommitter{inner: groupmemory.NewGroupImportRowCommitter(f.groupRepo)}
	// 所有権ガードを省くと計画器は fail-closed に全行を拒否する。既定では
	// 「どの Group も外部管理でない」ガードを配線し、拒否を見たいテストだけが
	// 差し替える。
	f.plan = GroupImportPlanDeps{
		GroupRepo: f.groupRepo, OwnershipGuard: groupSourceGuardStub{},
		GroupSchemaReader: groupAttributeSchemaStub{},
	}
	f.apply = GroupImportApplyDeps{Plan: f.plan, Committer: f.committer}
	return f
}

// groupAttributeSchemaStub はテナントが 1 個のカスタム属性を定義している状態を表す。
type groupAttributeSchemaStub struct{}

func (groupAttributeSchemaStub) EffectiveGroupAttributeDefs(context.Context, string) ([]groupdomain.GroupAttributeDef, error) {
	return []groupdomain.GroupAttributeDef{{Key: "cost_center", Type: idmdomain.AttributeTypeString}}, nil
}

// recordingGroupImportCommitter は確定ポートへ渡った変更集合を記録する。メモリ
// リポジトリの Delete は membership を自前で落とすため、cascade を「リポジトリを
// 読み戻す」だけでは観測できない。行が運ぶ完全な書き込み集合そのものを見る。
type recordingGroupImportCommitter struct {
	inner     groupports.GroupImportRowCommitter
	mutations []groupports.GroupImportRowMutation
}

func (c *recordingGroupImportCommitter) CommitGroupImportRow(ctx context.Context, mutation groupports.GroupImportRowMutation) error {
	c.mutations = append(c.mutations, mutation)
	return c.inner.CommitGroupImportRow(ctx, mutation)
}

func (f *groupImportFixture) seedGroup(t *testing.T, id, name string, membership groupdomain.GroupMembershipType, roles ...string) *groupdomain.Group {
	t.Helper()
	group := &groupdomain.Group{
		ID: id, TenantID: "acme", Name: name, Roles: roles,
		MembershipType: membership, CreatedAt: f.now, UpdatedAt: f.now,
	}
	if err := f.groupRepo.Save(f.ctx, group); err != nil {
		t.Fatalf("seed group %q: %v", name, err)
	}
	return group
}

func (f *groupImportFixture) seedMember(t *testing.T, groupID, userID string) {
	t.Helper()
	user := &userdomain.User{
		ID: userID, TenantID: "acme", PreferredUsername: userID, PasswordHash: "hash",
		Roles: []string{}, CreatedAt: f.now, UpdatedAt: f.now,
	}
	if err := f.userRepo.Save(f.ctx, user); err != nil {
		t.Fatalf("seed user %q: %v", userID, err)
	}
	if _, err := f.groupRepo.AddMember(f.ctx, &groupdomain.GroupMember{
		GroupID: groupID, UserID: userID, Source: groupdomain.MembershipSourceManual, CreatedAt: f.now,
	}); err != nil {
		t.Fatalf("seed member %q: %v", userID, err)
	}
}

func (f *groupImportFixture) exportCSV(t *testing.T) string {
	t.Helper()
	deps := GroupCSVExportDeps{
		GroupRepo: f.groupRepo, SchemaReader: groupAttributeSchemaStub{}, Artifacts: f.artifacts,
	}
	schema, err := groupdomain.NewGroupCSVSchema([]groupdomain.GroupAttributeDef{
		{Key: "cost_center", Type: idmdomain.AttributeTypeString},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ExportGroupCSV(f.ctx, deps, schema.ColumnKeys(), idmdomain.DefaultCSVTransferPolicy())
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	reader, _, err := f.artifacts.OpenCSVArtifact(f.ctx, "acme", result.Artifact.Ref)
	if err != nil {
		t.Fatalf("open artifact: %v", err)
	}
	defer func() { _ = reader.Close() }()
	var out bytes.Buffer
	if _, err := io.Copy(&out, reader); err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	return out.String()
}

func (f *groupImportFixture) preview(t *testing.T, document string) (GroupImportPlanSummary, []groupdomain.GroupImportRowPlan) {
	t.Helper()
	var rows []groupdomain.GroupImportRowPlan
	summary, err := PlanGroupImport(f.ctx, f.plan, strings.NewReader(document), idmdomain.DefaultCSVTransferPolicy(),
		func(row groupdomain.GroupImportRowPlan) error { rows = append(rows, row); return nil })
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	return summary, rows
}

func (f *groupImportFixture) applyCSV(t *testing.T, document string) GroupImportPlanSummary {
	t.Helper()
	summary, err := ApplyGroupImport(f.ctx, f.apply, strings.NewReader(document), idmdomain.DefaultCSVTransferPolicy(),
		"operator", f.now.Add(time.Hour), nil)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	return summary
}

func rowByNumber(t *testing.T, rows []groupdomain.GroupImportRowPlan, number int) groupdomain.GroupImportRowPlan {
	t.Helper()
	for _, row := range rows {
		if row.Row == number {
			return row
		}
	}
	t.Fatalf("row %d is absent from the plan", number)
	return groupdomain.GroupImportRowPlan{}
}

// scenario REQ-IDMANAGEMENT-027: 無編集の export をそのまま preview すると全行
// unchanged になり、`lifecycle_action` は全行で空として出力される。
func TestGroupImportUneditedExportRoundTripsAsUnchanged(t *testing.T) {
	f := newGroupImportFixture(t)
	// 連絡先とカスタム属性を持つ Group も往復の対象である。値には数式の引き金と
	// 引用符を含め、可逆な変換を経ても差分にならないことを確かめる。
	first := f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual, "catalog:read")
	email := "eng@example.test"
	center := `=SUM(A1),"quoted"`
	first.Email = &email
	first.Attributes = map[string]userdomain.AttributeValue{
		"cost_center": {Type: idmdomain.AttributeTypeString, String: &center},
	}
	if err := f.groupRepo.Save(f.ctx, first); err != nil {
		t.Fatal(err)
	}
	f.seedGroup(t, "group-2", "sales", groupdomain.GroupMembershipManual, "invoice:read")

	document := f.exportCSV(t)
	if !strings.Contains(document, "custom:cost_center") || !strings.Contains(document, "email") {
		t.Fatalf("export header must carry email and the custom column: %q", document)
	}
	header, _, _ := strings.Cut(document, "\n")
	if !strings.Contains(header, "lifecycle_action") {
		t.Fatalf("export header %q must carry lifecycle_action so the round trip is complete", header)
	}
	for _, line := range strings.Split(strings.TrimSpace(document), "\n")[1:] {
		if !strings.HasSuffix(line, ",,,") && strings.Contains(line, ",delete,") {
			t.Fatalf("export row %q must leave lifecycle_action empty", line)
		}
	}

	summary, rows := f.preview(t, document)
	if summary.TotalRows != 2 || summary.UnchangedRows != 2 {
		t.Fatalf("summary = %+v, want 2 rows all unchanged", summary)
	}
	for _, row := range rows {
		if row.Action != groupdomain.GroupImportUnchanged {
			t.Fatalf("row %d action = %q, want unchanged", row.Row, row.Action)
		}
	}
}

// scenario REQ-IDMANAGEMENT-026: 既存 Group の membership_type を変える行は
// immutable_membership_type で拒否され、その Group は何も変わらない。
func TestGroupImportRefusesMembershipTypeChangeAndLeavesTheGroupUntouched(t *testing.T) {
	f := newGroupImportFixture(t)
	f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual, "catalog:read")

	document := "id,name,roles,membership_type\ngroup-1,engineering,catalog:read|invoice:read,dynamic\n"
	summary, rows := f.preview(t, document)
	if summary.RejectedRows != 1 {
		t.Fatalf("summary = %+v, want the row rejected", summary)
	}
	row := rowByNumber(t, rows, 2)
	if row.Error == nil || row.Error.Code != "immutable_membership_type" {
		t.Fatalf("row error = %+v, want immutable_membership_type", row.Error)
	}

	applied := f.applyCSV(t, document)
	if applied.RejectedRows != 1 || applied.UpdatedRows != 0 {
		t.Fatalf("apply summary = %+v, want the row rejected and nothing updated", applied)
	}
	after, err := f.groupRepo.FindByID(f.ctx, "acme", "group-1")
	if err != nil || after == nil {
		t.Fatalf("read back: %v", err)
	}
	if after.MembershipType.Effective() != groupdomain.GroupMembershipManual {
		t.Fatalf("membership_type = %q, want manual: the refusal must leave it untouched", after.MembershipType)
	}
	if len(after.Roles) != 1 || after.Roles[0] != "catalog:read" {
		t.Fatalf("roles = %v, want the original roles: the refusal must not apply the rest of the row", after.Roles)
	}
}

// scenario REQ-IDMANAGEMENT-026: 外部の取り込み元が管理する Group と、所有権を
// 判定できない Group は fail-closed に拒否され、値は 1 つも書き換わらない。
func TestGroupImportRefusesSourceManagedGroupsFailClosed(t *testing.T) {
	for name, guard := range map[string]groupSourceGuardStub{
		"source managed":       {managed: map[string]bool{"group-1": true}},
		"ownership unknowable": {err: io.ErrUnexpectedEOF},
	} {
		t.Run(name, func(t *testing.T) {
			f := newGroupImportFixture(t)
			f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual, "catalog:read")
			f.plan.OwnershipGuard = guard
			f.apply.Plan = f.plan

			document := "id,name,roles\ngroup-1,engineering,catalog:read|invoice:read\n"
			applied := f.applyCSV(t, document)
			if applied.RejectedRows != 1 {
				t.Fatalf("summary = %+v, want the row rejected", applied)
			}
			after, err := f.groupRepo.FindByID(f.ctx, "acme", "group-1")
			if err != nil || after == nil {
				t.Fatalf("read back: %v", err)
			}
			if len(after.Roles) != 1 || after.Roles[0] != "catalog:read" {
				t.Fatalf("roles = %v, want the original roles untouched", after.Roles)
			}
		})
	}
}

// scenario REQ-IDMANAGEMENT-028: `lifecycle_action=delete` の行だけが Group と
// その membership を消し、同じファイルの create/update は影響を受けない。
func TestGroupImportDeletesOnlyTheRowsThatAskForIt(t *testing.T) {
	f := newGroupImportFixture(t)
	f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual, "catalog:read")
	f.seedGroup(t, "group-2", "sales", groupdomain.GroupMembershipManual, "invoice:read")
	f.seedMember(t, "group-1", "alice")

	document := "id,name,roles,lifecycle_action\n" +
		"group-1,engineering,catalog:read,delete\n" +
		"group-2,sales,invoice:read|catalog:read,\n" +
		",marketing,catalog:read,\n"

	summary, rows := f.preview(t, document)
	if summary.DeletedRows != 1 || summary.DeletedMemberships != 1 {
		t.Fatalf("preview summary = %+v, want 1 deletion carrying 1 membership", summary)
	}
	if summary.UpdatedRows != 1 || summary.CreatedRows != 1 {
		t.Fatalf("preview summary = %+v, want 1 update and 1 create beside the deletion", summary)
	}
	if action := rowByNumber(t, rows, 2).Action; action != groupdomain.GroupImportDeleted {
		t.Fatalf("row 2 action = %q, want deleted", action)
	}
	if _, err := f.groupRepo.FindByID(f.ctx, "acme", "group-1"); err != nil {
		t.Fatalf("preview must not touch the repository: %v", err)
	}
	if before, _ := f.groupRepo.FindByID(f.ctx, "acme", "group-1"); before == nil {
		t.Fatal("preview deleted a group; a preview causes no side effects")
	}

	applied := f.applyCSV(t, document)
	if applied.DeletedRows != 1 || applied.UpdatedRows != 1 || applied.CreatedRows != 1 {
		t.Fatalf("apply summary = %+v", applied)
	}
	if gone, err := f.groupRepo.FindByID(f.ctx, "acme", "group-1"); err != nil || gone != nil {
		t.Fatalf("group-1 = %+v, %v; want deleted", gone, err)
	}
	members, err := f.groupRepo.ListMembersByGroup(f.ctx, "acme", "group-1")
	if err != nil || len(members) != 0 {
		t.Fatalf("memberships = %+v, %v; want released by cascade", members, err)
	}
	// 削除は membership の解放と同じ 1 つの書き込み集合で確定する。リポジトリを
	// 読み戻すだけでは、Delete 自身の後始末と cascade を区別できない。
	var deletion *groupports.GroupImportRowMutation
	for i, mutation := range f.committer.mutations {
		if mutation.Delete {
			deletion = &f.committer.mutations[i]
		}
	}
	if deletion == nil {
		t.Fatal("no delete mutation reached the commit boundary")
	}
	if len(deletion.RemovedMemberships) != 1 || deletion.RemovedMemberships[0] != "alice" {
		t.Fatalf("removed memberships = %v, want [alice] released in the same transaction", deletion.RemovedMemberships)
	}
	if !deletion.ReleasesGroupQuota || deletion.AuditEventType != "GroupDeleted" {
		t.Fatalf("delete mutation = %+v, want the quota release and the audit record in the same write set", deletion)
	}
	kept, err := f.groupRepo.FindByID(f.ctx, "acme", "group-2")
	if err != nil || kept == nil || len(kept.Roles) != 2 {
		t.Fatalf("group-2 = %+v, %v; want the update applied beside the deletion", kept, err)
	}
	all, err := f.groupRepo.ListAll(f.ctx, "acme")
	if err != nil || len(all) != 2 {
		t.Fatalf("groups = %+v, %v; want sales and marketing", all, err)
	}
}

// scenario REQ-IDMANAGEMENT-028: 削除の意図は更新と同居できず、対象を解決できない
// 削除は作成に落ちない。どちらの拒否も Group を作らず消さない。
func TestGroupImportRefusesConflictingAndUnresolvableDeletions(t *testing.T) {
	f := newGroupImportFixture(t)
	f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual, "catalog:read")
	f.seedGroup(t, "group-2", "sales", groupdomain.GroupMembershipManual, "invoice:read")

	document := "id,name,roles,lifecycle_action\n" +
		"group-1,engineering,catalog:read|invoice:read,delete\n" +
		",ghost,catalog:read,delete\n" +
		"group-2,sales,invoice:read,purge\n"

	_, rows := f.preview(t, document)
	for number, want := range map[int]idmdomain.CSVErrorCode{
		2: "conflicting_lifecycle_action",
		3: "target_not_found",
		4: "invalid_lifecycle_action",
	} {
		row := rowByNumber(t, rows, number)
		if row.Action != groupdomain.GroupImportRejected || row.Error == nil || row.Error.Code != want {
			t.Fatalf("row %d = %+v, want rejected with %q", number, row, want)
		}
	}

	f.applyCSV(t, document)
	survivor, err := f.groupRepo.FindByID(f.ctx, "acme", "group-1")
	if err != nil || survivor == nil {
		t.Fatalf("group-1 = %+v, %v; a refused deletion must leave the group", survivor, err)
	}
	if len(survivor.Roles) != 1 {
		t.Fatalf("roles = %v; a refused row must not apply its update either", survivor.Roles)
	}
	all, err := f.groupRepo.ListAll(f.ctx, "acme")
	if err != nil || len(all) != 2 {
		t.Fatalf("groups = %+v, %v; an unresolvable delete must not create one", all, err)
	}
}

type groupSourceGuardStub struct {
	managed map[string]bool
	err     error
}

func (g groupSourceGuardStub) SourceManagedGroupIDs(_ context.Context, _ string, groupIDs []string) (map[string]bool, error) {
	if g.err != nil {
		return nil, g.err
	}
	out := make(map[string]bool, len(groupIDs))
	for _, id := range groupIDs {
		out[id] = g.managed[id]
	}
	return out, nil
}

// scenario REQ-IDMANAGEMENT-027: 10,000 Group を全 import 互換列で export し、
// 無編集のまま preview すると全行 unchanged になる。往復の保証を、小さな移行
// 単位を超える規模で確かめる。
func TestGroupImportTenThousandGroupsRoundTripAsUnchanged(t *testing.T) {
	f := newGroupImportFixture(t)
	for index := range 10_000 {
		f.seedGroup(t, fmt.Sprintf("group-%05d", index), fmt.Sprintf("team-%05d", index),
			groupdomain.GroupMembershipManual, "catalog:read", "invoice:read")
	}

	document := f.exportCSV(t)
	policy := idmdomain.DefaultCSVTransferPolicy()
	if len(document) >= policy.MaxBytes {
		t.Fatalf("export bytes=%d exceeds the effective policy=%d", len(document), policy.MaxBytes)
	}

	summary, err := PlanGroupImport(f.ctx, f.plan, strings.NewReader(document), policy, nil)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalRows != 10_000 || summary.UnchangedRows != 10_000 {
		t.Fatalf("summary = %+v, want 10000 rows all unchanged", summary)
	}
	if summary.DeletedRows != 0 || summary.DeletedMemberships != 0 {
		t.Fatalf("summary = %+v, an unedited round trip must delete nothing", summary)
	}
}
