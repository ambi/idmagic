package usecases

// Group CSV が宣言する具体例のうち、group_import_test.go と
// group_import_planner_test.go が押さえていなかったものを引き取る。
//
// どれも「拒否の型」ではなく「拒否が触れなかったもの」を読む。CSV の経路は
// preview と apply が別の入口なので、preview が保存層を動かさないこと自体が
// 具体例の `Then` に並んでいる。

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	idmmemory "github.com/ambi/idmagic/backend/idmanagement/db_memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// groupSnapshot はテナントの Group を (name, roles) の並びで写し取る。
// 「`Group` は変更されない」を、特定の 1 件ではなく集合として読む。
func (f *groupImportFixture) groupSnapshot(t *testing.T) []string {
	t.Helper()
	groups, err := f.groupRepo.ListAll(f.ctx, "acme")
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	out := make([]string, 0, len(groups))
	for _, group := range groups {
		out = append(out, group.Name+"="+strings.Join(group.Roles, "|"))
	}
	return out
}

// 具体例は preview の判定 5 種、行番号、安定したエラーコード、そして
// 「`Group` は変更されない」を並べ、そのあと apply の側で有効な行だけが
// 反映されることを言う。**判定の種類ごとに 1 行ずつ置く。** 1 種類だけの
// ファイルでは、判定を 1 つに固定する実装が通ってしまう。
//
//spec:covers EX-IDMANAGEMENT-026-01: preview が created/updated/unchanged/deleted/rejected と行番号と安定コードを返して Group を変えないこと、apply が有効な行だけを反映すること。
func TestGroupImportPreviewReportsEveryVerdictThenApplyWritesOnlyTheValidRows(t *testing.T) {
	f := newGroupImportFixture(t)
	f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual, "catalog:read")
	f.seedGroup(t, "group-2", "sales", groupdomain.GroupMembershipManual, "invoice:read")
	f.seedGroup(t, "group-3", "ops", groupdomain.GroupMembershipManual, "ops:read")

	document := "id,name,roles,membership_type,lifecycle_action\n" +
		",marketing,catalog:read,,\n" + // 2: created
		"group-1,engineering,catalog:read|invoice:read,,\n" + // 3: updated
		"group-2,sales,invoice:read,,\n" + // 4: unchanged
		"group-3,ops,ops:read,,delete\n" + // 5: deleted
		"unknown-id,ghost,catalog:read,,\n" // 6: rejected (target_not_found)

	before := f.groupSnapshot(t)
	summary, rows := f.preview(t, document)
	if summary.CreatedRows != 1 || summary.UpdatedRows != 1 || summary.UnchangedRows != 1 ||
		summary.DeletedRows != 1 || summary.RejectedRows != 1 || summary.TotalRows != 5 {
		t.Fatalf("preview summary = %+v, want one row of each verdict", summary)
	}
	for number, want := range map[int]groupdomain.GroupImportAction{
		2: groupdomain.GroupImportCreate,
		3: groupdomain.GroupImportUpdate,
		4: groupdomain.GroupImportUnchanged,
		5: groupdomain.GroupImportDeleted,
		6: groupdomain.GroupImportRejected,
	} {
		if got := rowByNumber(t, rows, number).Action; got != want {
			t.Fatalf("row %d action = %q, want %q", number, got, want)
		}
	}
	// 安定したエラーコードは、拒否行が運ぶ `Error.Code` である。
	if rejected := rowByNumber(t, rows, 6); rejected.Error == nil || rejected.Error.Code != "target_not_found" {
		t.Fatalf("row 6 error = %+v, want target_not_found", rejected.Error)
	}
	// 「`User` は変更されない」の Group 版。preview は保存層を 1 件も動かさない。
	if after := f.groupSnapshot(t); !equalGroupSnapshots(before, after) {
		t.Fatalf("preview changed the repository: before=%v after=%v", before, after)
	}

	applied := f.applyCSV(t, document)
	if applied.CreatedRows != 1 || applied.UpdatedRows != 1 || applied.UnchangedRows != 1 ||
		applied.DeletedRows != 1 || applied.RejectedRows != 1 {
		t.Fatalf("apply summary = %+v", applied)
	}
	if got := f.groupSnapshot(t); !equalGroupSnapshots(
		got, []string{"engineering=catalog:read|invoice:read", "marketing=catalog:read", "sales=invoice:read"},
	) {
		t.Fatalf("groups after apply = %v", got)
	}
}

func equalGroupSnapshots(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// 上限は読みながら効かなければならない。ファイル全体を読んでから文句を言う実装は、
// 上限を超える数の行を先に計画してしまう。行が 1 つも計画へ出ていないことで読む。
//
//spec:covers EX-IDMANAGEMENT-026-02: 実効 CsvTransferPolicy の max_bytes / max_rows / max_field_bytes を超えた Group CSV の投入が、それぞれの安定コードでファイルごと拒否されること。
func TestGroupImportRefusesFilesBeyondTheTransferPolicy(t *testing.T) {
	document := "id,name,roles\ngroup-1,engineering,catalog:read\ngroup-2,sales,invoice:read\n"
	for _, tc := range []struct {
		name     string
		policy   idmdomain.CSVTransferPolicy
		document string
		want     idmdomain.CSVErrorCode
	}{
		{
			name:     "行数の上限",
			policy:   idmdomain.CSVTransferPolicy{MaxRows: 1, MaxBytes: 1 << 20, MaxFieldBytes: 1 << 10},
			document: document,
			want:     idmdomain.CSVErrorTooManyRows,
		},
		{
			name:     "byte 数の上限",
			policy:   idmdomain.CSVTransferPolicy{MaxRows: 100, MaxBytes: 24, MaxFieldBytes: 1 << 10},
			document: document,
			want:     idmdomain.CSVErrorCSVTooLarge,
		},
		{
			name:     "項目長の上限",
			policy:   idmdomain.CSVTransferPolicy{MaxRows: 100, MaxBytes: 1 << 20, MaxFieldBytes: 8},
			document: "id,name,roles\ngroup-1," + strings.Repeat("x", 64) + ",catalog:read\n",
			want:     idmdomain.CSVErrorFieldTooLarge,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newGroupImportFixture(t)
			f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual, "catalog:read")
			before := f.groupSnapshot(t)

			var rows []groupdomain.GroupImportRowPlan
			_, err := PlanGroupImport(f.ctx, f.plan, strings.NewReader(tc.document), tc.policy,
				func(row groupdomain.GroupImportRowPlan) error { rows = append(rows, row); return nil })
			csvErr, ok := errors.AsType[*idmdomain.CSVError](err)
			if !ok || csvErr.Code != tc.want {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
			if len(rows) > tc.policy.MaxRows {
				t.Fatalf("the planner emitted %d rows past a limit of %d", len(rows), tc.policy.MaxRows)
			}
			if after := f.groupSnapshot(t); !equalGroupSnapshots(before, after) {
				t.Fatalf("the refusal changed the repository: before=%v after=%v", before, after)
			}
		})
	}
}

// 具体例は「行に `id` も `name` も無い」を、識別子の食い違いと重複と並べて言う。
// 残りの 2 つは planner のテストが持つので、ここは欠落だけを名指す。
//
//spec:covers EX-IDMANAGEMENT-026-04: 識別子を 1 つも持たない行が安定コードで rejected になり、作成に落ちないこと。
func TestGroupImportPlannerRefusesRowsWithNoIdentifier(t *testing.T) {
	f := newGroupImportFixture(t)
	before := f.groupSnapshot(t)

	rows := planRows(t, f, "id,name,roles\n,,catalog:read\n")
	row := rowByNumber(t, rows, 2)
	if row.Action != groupdomain.GroupImportRejected || row.Error == nil {
		t.Fatalf("row = %+v, want rejected with a stable code", row)
	}
	if row.Error.Code != "missing_identifier" {
		t.Fatalf("error code = %q, want missing_identifier", row.Error.Code)
	}
	f.applyCSV(t, "id,name,roles\n,,catalog:read\n")
	if after := f.groupSnapshot(t); !equalGroupSnapshots(before, after) {
		t.Fatalf("識別子の無い行が Group を作った: %v", after)
	}
}

// 失敗した行だけが rejected になり、その行のどの列も保存されず、他の有効な行は
// 適用を続ける。**行の一部だけが残る形を落とすのがこの具体例である。**
//
//spec:covers EX-IDMANAGEMENT-026-11: 1 行の確定が途中で失敗したとき、その行は一部も保存されず、他の有効な行は適用され続けること。
func TestGroupImportOneFailingRowIsAtomicAndDoesNotStopTheOthers(t *testing.T) {
	f := newGroupImportFixture(t)
	f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual, "catalog:read")
	f.seedGroup(t, "group-2", "sales", groupdomain.GroupMembershipManual, "invoice:read")
	// engineering の行だけ確定に失敗させる。
	f.apply.Committer = &failingGroupImportCommitter{
		inner: f.committer, failFor: "engineering", err: errors.New("commit boundary is down"),
	}

	document := "id,name,roles,email\n" +
		"group-1,engineering,catalog:read|invoice:read,eng@example.test\n" +
		"group-2,sales,invoice:read|catalog:read,sales@example.test\n"
	applied := f.applyCSV(t, document)
	if applied.RejectedRows != 1 || applied.UpdatedRows != 1 {
		t.Fatalf("apply summary = %+v, want 1 rejected and 1 updated", applied)
	}

	failed, err := f.groupRepo.FindByID(f.ctx, "acme", "group-1")
	if err != nil || failed == nil {
		t.Fatalf("read back: %v", err)
	}
	// 行は不可分である。ロールも連絡先も、片方だけ書かれてはいけない。
	if len(failed.Roles) != 1 || failed.Roles[0] != "catalog:read" || failed.Email != nil {
		t.Fatalf("失敗した行の一部が保存された: roles=%v email=%v", failed.Roles, failed.Email)
	}
	survived, err := f.groupRepo.FindByID(f.ctx, "acme", "group-2")
	if err != nil || survived == nil {
		t.Fatalf("read back: %v", err)
	}
	if len(survived.Roles) != 2 || survived.Email == nil || *survived.Email != "sales@example.test" {
		t.Fatalf("他の有効な行が適用されていない: roles=%v email=%v", survived.Roles, survived.Email)
	}
}

// failingGroupImportCommitter は名前で選んだ 1 行だけ確定を失敗させる。
type failingGroupImportCommitter struct {
	inner   groupports.GroupImportRowCommitter
	failFor string
	err     error
}

func (c *failingGroupImportCommitter) CommitGroupImportRow(
	ctx context.Context, mutation groupports.GroupImportRowMutation,
) error {
	if mutation.After != nil && mutation.After.Name == c.failFor {
		return c.err
	}
	return c.inner.CommitGroupImportRow(ctx, mutation)
}

// 具体例は「再インポートできない成功済み成果物を作らない」まで言う。成果物ストアに
// 何も残らないことを読むのは、書き終えてから失敗する実装と区別するためである。
//
//spec:covers EX-IDMANAGEMENT-027-03: 生成結果が CsvTransferPolicy の上限を超える Group エクスポートが失敗し、成果物を残さないこと。
func TestGroupCSVExportFailsRatherThanWriteAnUnimportableArtifact(t *testing.T) {
	f := newGroupImportFixture(t)
	for i := range 5 {
		f.seedGroup(t, "group-"+string(rune('a'+i)), "team-"+string(rune('a'+i)),
			groupdomain.GroupMembershipManual, "catalog:read")
	}
	artifacts := &countingCSVArtifactStore{CSVArtifactStore: idmmemory.NewCSVArtifactStore()}
	deps := GroupCSVExportDeps{
		GroupRepo: f.groupRepo, SchemaReader: groupAttributeSchemaStub{}, Artifacts: artifacts,
	}
	schema, err := groupdomain.NewGroupCSVSchema([]groupdomain.GroupAttributeDef{
		{Key: "cost_center", Type: idmdomain.AttributeTypeString},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ExportGroupCSV(f.ctx, deps, schema.ColumnKeys(),
		idmdomain.CSVTransferPolicy{MaxRows: 2, MaxBytes: 1 << 20, MaxFieldBytes: 1 << 10})
	csvErr, ok := errors.AsType[*idmdomain.CSVError](err)
	if !ok || csvErr.Code != idmdomain.CSVErrorTooManyRows {
		t.Fatalf("error = %v, want too_many_rows", err)
	}
	if artifacts.stored != 0 {
		t.Fatalf("上限を超えたエクスポートが %d 件の成果物を残した", artifacts.stored)
	}
}

// countingCSVArtifactStore は成功して保存された成果物の数を数える。
type countingCSVArtifactStore struct {
	*idmmemory.CSVArtifactStore
	stored int
}

func (s *countingCSVArtifactStore) PutCSVArtifact(
	ctx context.Context, tenantID string, write func(io.Writer) error,
) (idmports.CSVArtifact, error) {
	artifact, err := s.CSVArtifactStore.PutCSVArtifact(ctx, tenantID, write)
	if err == nil {
		s.stored++
	}
	return artifact, err
}

// 具体例は 4 種類の不正値を並べる。**種類ごとに 1 行ずつ置く。** 1 つだけを見る
// テストは、他の列の検証が抜けた実装を通す。どの行も拒否のあとで保存済みの値へ
// 触れていないことまで読む。
//
//spec:covers EX-IDMANAGEMENT-027-05: roles / membership_type / email / custom:<key> の不正値がそれぞれ安定コードで rejected になり、保存済みの値を変えないこと。
func TestGroupImportRejectsEachKindOfInvalidValueWithoutTouchingTheGroup(t *testing.T) {
	for _, tc := range []struct {
		name     string
		document string
		want     idmdomain.CSVErrorCode
	}{
		{
			name:     "roles",
			document: "id,roles\ngroup-1,catalog:read||invoice:read\n",
			want:     "invalid_roles",
		},
		{
			name:     "membership_type",
			document: "name,membership_type\nfresh,static\n",
			want:     "invalid_membership_type",
		},
		{
			name:     "email",
			document: "id,email\ngroup-1,not-an-address\n",
			want:     "invalid_email",
		},
		{
			name:     "custom:<key>",
			document: "id,custom:headcount\ngroup-1,not-a-number\n",
			want:     "invalid_attribute",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newGroupImportFixture(t)
			f.plan.GroupSchemaReader = typedGroupAttributeSchemaStub{}
			f.apply.Plan = f.plan
			email := "eng@example.test"
			group := f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual, "catalog:read")
			group.Email = &email
			if err := f.groupRepo.Save(f.ctx, group); err != nil {
				t.Fatal(err)
			}
			before := f.groupSnapshot(t)

			rows := planRows(t, f, tc.document)
			row := rowByNumber(t, rows, 2)
			if row.Action != groupdomain.GroupImportRejected || row.Error == nil || row.Error.Code != tc.want {
				t.Fatalf("row = %+v, want rejected with %q", row, tc.want)
			}
			f.applyCSV(t, tc.document)
			if after := f.groupSnapshot(t); !equalGroupSnapshots(before, after) {
				t.Fatalf("拒否が Group を変えた: before=%v after=%v", before, after)
			}
			stored, err := f.groupRepo.FindByID(f.ctx, "acme", "group-1")
			if err != nil || stored == nil {
				t.Fatal(err)
			}
			if stored.Email == nil || *stored.Email != email {
				t.Fatalf("拒否が email を変えた: %v", stored.Email)
			}
		})
	}
}

// typedGroupAttributeSchemaStub は文字列でない型のカスタム属性も宣言する。
// 「宣言された型の字句形に合わない」を観測するには、字句形が値と区別できる型が要る。
type typedGroupAttributeSchemaStub struct{}

func (typedGroupAttributeSchemaStub) EffectiveGroupAttributeDefs(
	context.Context, string,
) ([]groupdomain.GroupAttributeDef, error) {
	return []groupdomain.GroupAttributeDef{
		{Key: "cost_center", Type: idmdomain.AttributeTypeString},
		{Key: "headcount", Type: idmdomain.AttributeTypeNumber},
	}, nil
}

// 削除の行にも所有権のガードが効く。**削除は取り返しが付かないので、
// 更新と同じガードが同じ強さで効いていることを別に読む。**
//
//spec:covers EX-IDMANAGEMENT-028-05: 外部の取り込み元が管理する Group、および所有権を判定できない Group への削除行が fail-closed に拒否され、Group が残ること。
func TestGroupImportRefusesDeletingSourceManagedGroups(t *testing.T) {
	for name, guard := range map[string]groupSourceGuardStub{
		"source managed":       {managed: map[string]bool{"group-1": true}},
		"ownership unknowable": {err: errors.New("the source registry is unreachable")},
	} {
		t.Run(name, func(t *testing.T) {
			f := newGroupImportFixture(t)
			f.seedGroup(t, "group-1", "engineering", groupdomain.GroupMembershipManual, "catalog:read")
			f.seedMember(t, "group-1", "alice")
			f.plan.OwnershipGuard = guard
			f.apply.Plan = f.plan

			document := "id,name,lifecycle_action\ngroup-1,engineering,delete\n"
			applied := f.applyCSV(t, document)
			if applied.RejectedRows != 1 || applied.DeletedRows != 0 {
				t.Fatalf("apply summary = %+v, want the deletion refused", applied)
			}
			survivor, err := f.groupRepo.FindByID(f.ctx, "acme", "group-1")
			if err != nil || survivor == nil {
				t.Fatalf("拒否された削除が Group を消した: %+v, %v", survivor, err)
			}
			members, err := f.groupRepo.ListMembersByGroup(f.ctx, "acme", "group-1")
			if err != nil || len(members) != 1 {
				t.Fatalf("拒否された削除が所属を外した: %+v, %v", members, err)
			}
		})
	}
}

// 具体例は「保存済みのペイロードとダイジェストが一致しない」を含めて 5 つの入口を
// 並べ、どれも `Group` を変更しないと言う。適用のジョブが 1 つも投入されなければ
// 変更は起こりようがないので、ジョブが作られていないことで読む。
//
//spec:covers EX-IDMANAGEMENT-026-05: 存在しない / queued / failed / 別テナント / ダイジェスト不一致のプレビューを指す適用が拒否され、適用のジョブも Group の変更も残さないこと。
func TestStartGroupImportApplyRefusesEveryUnboundPreview(t *testing.T) {
	ctx := groupImportContext()
	otherCtx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "other"}, "", "")

	for _, tc := range []struct {
		name    string
		arrange func(t *testing.T, artifacts *idmmemory.CSVArtifactStore, jobs *jobsmemory.JobRepository) (context.Context, string)
		wantErr error
	}{
		{
			name: "存在しない",
			arrange: func(t *testing.T, _ *idmmemory.CSVArtifactStore, _ *jobsmemory.JobRepository) (context.Context, string) {
				t.Helper()
				return ctx, "no-such-preview"
			},
			wantErr: ErrGroupImportNotFound,
		},
		{
			name: "queued のまま",
			arrange: func(t *testing.T, artifacts *idmmemory.CSVArtifactStore, jobs *jobsmemory.JobRepository) (context.Context, string) {
				t.Helper()
				return ctx, startGroupImportPreviewForTest(t, artifacts, jobs).ID
			},
			wantErr: ErrGroupImportPreviewNotReady,
		},
		{
			name: "failed で終わっている",
			arrange: func(t *testing.T, artifacts *idmmemory.CSVArtifactStore, jobs *jobsmemory.JobRepository) (context.Context, string) {
				t.Helper()
				preview := startGroupImportPreviewForTest(t, artifacts, jobs)
				claimGroupImportJob(t, jobs)
				if _, err := jobs.Fail(ctx, preview.ID, "worker", jobsports.FailOutcome{
					NextStatus: jobsdomain.StatusFailed, Error: "planner crashed",
				}, time.Now().UTC()); err != nil {
					t.Fatalf("Fail: %v", err)
				}
				return ctx, preview.ID
			},
			wantErr: ErrGroupImportPreviewNotReady,
		},
		{
			name: "別テナントに属する",
			arrange: func(t *testing.T, artifacts *idmmemory.CSVArtifactStore, jobs *jobsmemory.JobRepository) (context.Context, string) {
				t.Helper()
				return otherCtx, succeededGroupImportPreview(t, artifacts, jobs, true)
			},
			wantErr: ErrGroupImportNotFound,
		},
		{
			name: "ダイジェストが一致しない",
			arrange: func(t *testing.T, artifacts *idmmemory.CSVArtifactStore, jobs *jobsmemory.JobRepository) (context.Context, string) {
				t.Helper()
				return ctx, succeededGroupImportPreview(t, artifacts, jobs, false)
			},
			wantErr: ErrGroupImportDigestMismatch,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			artifacts := idmmemory.NewCSVArtifactStore()
			jobs := jobsmemory.NewJobRepository()
			applyCtx, previewID := tc.arrange(t, artifacts, jobs)

			//nolint:contextcheck // arrange が返す文脈は、越境を起こすためにわざと別テナントである
			_, err := StartGroupImportApply(
				applyCtx, GroupImportStartDeps{Artifacts: artifacts, Jobs: jobs}, "operator", previewID, time.Now().UTC(),
			)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			// 拒否が適用のジョブを作っていないこと。作ってから失敗させる実装と区別する。
			applyJobs, err := jobs.ListByTenantAndKinds(
				ctx, "acme", []jobsdomain.JobKind{jobsdomain.KindGroupImportApply}, 100,
			)
			if err != nil {
				t.Fatalf("ListByTenantAndKinds: %v", err)
			}
			if len(applyJobs) != 0 {
				t.Fatalf("拒否が適用のジョブを %d 件残した", len(applyJobs))
			}
		})
	}
}

func startGroupImportPreviewForTest(
	t *testing.T, artifacts *idmmemory.CSVArtifactStore, jobs *jobsmemory.JobRepository,
) *jobsdomain.Job {
	t.Helper()
	preview, err := StartGroupImportPreview(
		groupImportContext(), GroupImportStartDeps{Artifacts: artifacts, Jobs: jobs}, "operator",
		strings.NewReader("name,roles\nsales,catalog:read\n"), time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("StartGroupImportPreview: %v", err)
	}
	return preview
}

func claimGroupImportJob(t *testing.T, jobs *jobsmemory.JobRepository) {
	t.Helper()
	claimed, err := jobs.ClaimBatch(
		groupImportContext(), "worker", jobsdomain.LaneBulk, 1, time.Minute, time.Now().UTC(),
	)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ClaimBatch = %+v, %v", claimed, err)
	}
}

// succeededGroupImportPreview は成功したプレビューを建てて id を返す。matchingDigest が
// false のとき、結果が運ぶ SHA-256 を投入時のものとずらす。
func succeededGroupImportPreview(
	t *testing.T, artifacts *idmmemory.CSVArtifactStore, jobs *jobsmemory.JobRepository, matchingDigest bool,
) string {
	t.Helper()
	preview := startGroupImportPreviewForTest(t, artifacts, jobs)
	var params GroupImportParams
	if err := json.Unmarshal(preview.Params, &params); err != nil {
		t.Fatal(err)
	}
	digest := params.SourceSHA256
	if !matchingDigest {
		digest = strings.Repeat("0", len(digest))
	}
	claimGroupImportJob(t, jobs)
	result, err := json.Marshal(GroupImportResult{SourceSHA256: digest})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jobs.Complete(
		groupImportContext(), preview.ID, "worker", result, time.Now().UTC(),
	); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	return preview.ID
}
