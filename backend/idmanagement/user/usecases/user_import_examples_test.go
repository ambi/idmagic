package usecases

// User CSV が宣言する具体例のうち、planner / apply / export の既存テストが
// 押さえていなかったものを引き取る。
//
// どれも「拒否の型」ではなく「拒否が触れなかったもの」を読む。preview と apply は
// 別の入口なので、preview が保存層を動かさないこと自体が具体例の `Then` である。

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	idmmemory "github.com/ambi/idmagic/backend/idmanagement/db_memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// userSnapshot はテナントの User を (username, email, roles) の並びで写し取る。
// 「`User` は変更されない」を、特定の 1 件ではなく集合として読む。
func userSnapshot(t *testing.T, repo *usermemory.UserRepository) []string {
	t.Helper()
	users, err := repo.FindAll(importPlannerContext(), "acme")
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	out := make([]string, 0, len(users))
	for _, user := range users {
		email := ""
		if user.Email != nil {
			email = *user.Email
		}
		out = append(out, user.PreferredUsername+"/"+email+"/"+strings.Join(user.Roles, "|"))
	}
	return out
}

func sameUserSnapshots(left, right []string) bool {
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

// 上限は読みながら効かなければならない。ファイル全体を読み込んでから文句を言う
// 実装は、上限を超える数の行を先に計画してしまう。
//
//spec:covers EX-IDMANAGEMENT-004-02: 実効 CsvTransferPolicy の max_bytes / max_rows / max_field_bytes を超えた User CSV の投入が、それぞれの安定コードでファイルごと拒否されること。
func TestPlanUserImportRefusesFilesBeyondTheTransferPolicy(t *testing.T) {
	document := "preferred_username,email\nalice,alice@example.com\nbob,bob@example.com\n"
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
			policy:   idmdomain.CSVTransferPolicy{MaxRows: 100, MaxBytes: 32, MaxFieldBytes: 1 << 10},
			document: document,
			want:     idmdomain.CSVErrorCSVTooLarge,
		},
		{
			name:     "項目長の上限",
			policy:   idmdomain.CSVTransferPolicy{MaxRows: 100, MaxBytes: 1 << 20, MaxFieldBytes: 8},
			document: "preferred_username,email\n" + strings.Repeat("x", 64) + ",alice@example.com\n",
			want:     idmdomain.CSVErrorFieldTooLarge,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := usermemory.NewUserRepository()
			repo.Seed(importPlannerUser("user-alice", "alice"))
			before := userSnapshot(t, repo)

			var rows []userdomain.UserImportRowPlan
			_, err := PlanUserImport(importPlannerContext(), importPlannerDeps(repo, perUserImportOwnershipGuard{}),
				strings.NewReader(tc.document), tc.policy,
				func(row userdomain.UserImportRowPlan) error { rows = append(rows, row); return nil })
			csvErr, ok := errors.AsType[*idmdomain.CSVError](err)
			if !ok || csvErr.Code != tc.want {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
			if len(rows) > tc.policy.MaxRows {
				t.Fatalf("the planner emitted %d rows past a limit of %d", len(rows), tc.policy.MaxRows)
			}
			if after := userSnapshot(t, repo); !sameUserSnapshots(before, after) {
				t.Fatalf("the refusal changed the repository: before=%v after=%v", before, after)
			}
		})
	}
}

// 004-04 が並べる 4 つの識別子の誤りのうち、重複の 2 つをここが持つ。食い違いと
// 欠落は TestPlanUserImportRejectsIdentifierMismatchInvalidTypesAndMissingIdentifier
// が持つ。**重複はリポジトリを引かずにファイル内だけで判定できる衝突である。**
//
//spec:covers EX-IDMANAGEMENT-004-04: 同じ対象または同じ最終ユーザー名を複数行が指すファイルが、行ごとに安定コードで rejected になること。
func TestPlanUserImportRefusesDuplicateTargetsAndFinalUsernames(t *testing.T) {
	repo := usermemory.NewUserRepository()
	repo.Seed(importPlannerUser("user-alice", "alice"))
	before := userSnapshot(t, repo)

	plan, err := planUserImportForTest(importPlannerContext(),
		importPlannerDeps(repo, perUserImportOwnershipGuard{}),
		"id,preferred_username,email\n"+
			"user-alice,alice,first@example.com\n"+
			"user-alice,alice,second@example.com\n"+
			",fresh,fresh-first@example.com\n"+
			",fresh,fresh-second@example.com\n")
	if err != nil {
		t.Fatal(err)
	}
	if plan.RejectedRows() != 2 {
		t.Fatalf("plan = %+v, want the two later rows rejected", plan)
	}
	for i, want := range map[int]idmdomain.CSVErrorCode{1: "duplicate_target", 3: "duplicate_username"} {
		if plan.Rows[i].Error == nil || plan.Rows[i].Error.Code != want {
			t.Fatalf("row[%d] = %+v, want %q", i, plan.Rows[i], want)
		}
	}
	if after := userSnapshot(t, repo); !sameUserSnapshots(before, after) {
		t.Fatalf("planning changed the repository: before=%v after=%v", before, after)
	}
}

// 具体例は「保存済みのペイロードとダイジェストが一致しない」を含めて 5 つの入口を
// 並べる。**どれも `User` を変更しないことまで言う。** 適用のジョブが 1 つも
// 投入されなければ、変更は起こりようがない。ジョブが作られていないことで読む。
//
//spec:covers EX-IDMANAGEMENT-004-05: 存在しない / queued / failed / 別テナント / ダイジェスト不一致のプレビューを指す適用が拒否され、適用のジョブも User の変更も残さないこと。
func TestStartUserImportApplyRefusesEveryUnboundPreview(t *testing.T) {
	ctx := importPlannerContext()
	otherCtx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "other"}, "", "")

	for _, tc := range []struct {
		name string
		// arrange は preview job を望んだ状態にし、apply を呼ぶ文脈と対象 id を返す。
		arrange func(t *testing.T, artifacts *idmmemory.CSVArtifactStore, jobs *jobsmemory.JobRepository) (context.Context, string)
		wantErr error
	}{
		{
			name: "存在しない",
			arrange: func(t *testing.T, _ *idmmemory.CSVArtifactStore, _ *jobsmemory.JobRepository) (context.Context, string) {
				t.Helper()
				return ctx, "no-such-preview"
			},
			wantErr: ErrUserImportNotFound,
		},
		{
			name: "queued のまま",
			arrange: func(t *testing.T, artifacts *idmmemory.CSVArtifactStore, jobs *jobsmemory.JobRepository) (context.Context, string) {
				t.Helper()
				return ctx, startUserImportPreviewForTest(t, artifacts, jobs).ID
			},
			wantErr: ErrUserImportPreviewNotReady,
		},
		{
			name: "failed で終わっている",
			arrange: func(t *testing.T, artifacts *idmmemory.CSVArtifactStore, jobs *jobsmemory.JobRepository) (context.Context, string) {
				t.Helper()
				preview := startUserImportPreviewForTest(t, artifacts, jobs)
				claimUserImportJob(t, jobs)
				if _, err := jobs.Fail(ctx, preview.ID, "worker", jobsports.FailOutcome{
					NextStatus: jobsdomain.StatusFailed, Error: "planner crashed",
				}, time.Now().UTC()); err != nil {
					t.Fatalf("Fail: %v", err)
				}
				return ctx, preview.ID
			},
			wantErr: ErrUserImportPreviewNotReady,
		},
		{
			name: "別テナントに属する",
			arrange: func(t *testing.T, artifacts *idmmemory.CSVArtifactStore, jobs *jobsmemory.JobRepository) (context.Context, string) {
				t.Helper()
				preview := succeededUserImportPreview(t, artifacts, jobs, true)
				return otherCtx, preview
			},
			wantErr: ErrUserImportNotFound,
		},
		{
			name: "ダイジェストが一致しない",
			arrange: func(t *testing.T, artifacts *idmmemory.CSVArtifactStore, jobs *jobsmemory.JobRepository) (context.Context, string) {
				t.Helper()
				return ctx, succeededUserImportPreview(t, artifacts, jobs, false)
			},
			wantErr: ErrUserImportDigestMismatch,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			artifacts := idmmemory.NewCSVArtifactStore()
			jobs := jobsmemory.NewJobRepository()
			applyCtx, previewID := tc.arrange(t, artifacts, jobs)
			before := userImportApplyJobIDs(t, jobs)

			_, err := StartUserImportApply(
				applyCtx, UserImportStartDeps{Artifacts: artifacts, Jobs: jobs}, "admin", previewID, time.Now().UTC(),
			)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			// 拒否が適用のジョブを作っていないこと。作ってから失敗させる実装と区別する。
			if after := userImportApplyJobIDs(t, jobs); len(after) != len(before) {
				t.Fatalf("拒否が適用のジョブを %d 件残した", len(after)-len(before))
			}
		})
	}
}

func startUserImportPreviewForTest(
	t *testing.T, artifacts *idmmemory.CSVArtifactStore, jobs *jobsmemory.JobRepository,
) *jobsdomain.Job {
	t.Helper()
	preview, err := StartUserImportPreview(
		importPlannerContext(), UserImportStartDeps{Artifacts: artifacts, Jobs: jobs}, "admin",
		strings.NewReader("preferred_username\nalice\n"), time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("StartUserImportPreview: %v", err)
	}
	return preview
}

func claimUserImportJob(t *testing.T, jobs *jobsmemory.JobRepository) {
	t.Helper()
	claimed, err := jobs.ClaimBatch(importPlannerContext(), "worker", jobsdomain.LaneBulk, 1, time.Minute, time.Now().UTC())
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ClaimBatch = %+v, %v", claimed, err)
	}
}

// succeededUserImportPreview は成功したプレビューを建てて id を返す。matchingDigest が
// false のとき、結果が運ぶ SHA-256 を投入時のものとずらす。
func succeededUserImportPreview(
	t *testing.T, artifacts *idmmemory.CSVArtifactStore,
	jobs *jobsmemory.JobRepository, matchingDigest bool,
) string {
	t.Helper()
	preview := startUserImportPreviewForTest(t, artifacts, jobs)
	var params UserImportParams
	if err := json.Unmarshal(preview.Params, &params); err != nil {
		t.Fatal(err)
	}
	digest := params.SourceSHA256
	if !matchingDigest {
		digest = strings.Repeat("0", len(digest))
	}
	claimUserImportJob(t, jobs)
	if _, err := jobs.Complete(
		importPlannerContext(), preview.ID, "worker",
		mustJSON(t, UserImportResult{SourceSHA256: digest}), time.Now().UTC(),
	); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	return preview.ID
}

// userImportApplyJobIDs は投入済みの適用ジョブの id を返す。
func userImportApplyJobIDs(t *testing.T, jobs *jobsmemory.JobRepository) []string {
	t.Helper()
	page, err := jobs.ListByTenantAndKinds(
		importPlannerContext(), "acme", []jobsdomain.JobKind{jobsdomain.KindUserImportApply}, 100,
	)
	if err != nil {
		t.Fatalf("ListByTenantAndKinds: %v", err)
	}
	ids := make([]string, 0, len(page))
	for _, job := range page {
		ids = append(ids, job.ID)
	}
	return ids
}

// 具体例は 4 種類の型の不正と、必須のカスタム属性、`required_actions` を並べる。
// **種類ごとに 1 行ずつ置く。** 1 つだけを見るテストは、他の型の検証が抜けた実装を
// 通す。加えて「値はジョブの表示にも監査イベントにも含めない」を、拒否行が運ぶ
// 情報が位置と安定コードだけであることで読む。
//
//spec:covers EX-IDMANAGEMENT-007-05: 真偽値・数値・日付・必須のカスタム属性・required_actions の不正がそれぞれ安定コードで rejected になり、拒否行がセル値を運ばないこと。
func TestPlanUserImportRejectsEachInvalidTypedCellWithoutCarryingItsValue(t *testing.T) {
	defs := []userdomain.UserAttributeDef{
		{Key: "department", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityPrivate},
		{Key: "headcount", Type: idmdomain.AttributeTypeNumber, Visibility: idmdomain.AttrVisibilityPrivate},
		{Key: "hired_on", Type: idmdomain.AttributeTypeDate, Visibility: idmdomain.AttrVisibilityPrivate},
		{Key: "cost_center", Type: idmdomain.AttributeTypeString, Required: true, Visibility: idmdomain.AttrVisibilityPrivate},
	}
	for _, tc := range []struct {
		name     string
		document string
		secret   string
		want     idmdomain.CSVErrorCode
	}{
		{
			name:     "真偽値",
			document: "id,email_verified\nuser-alice,yes-please\n",
			secret:   "yes-please",
			want:     "invalid_boolean",
		},
		{
			name:     "数値",
			document: "id,custom:headcount\nuser-alice,twelve\n",
			secret:   "twelve",
			want:     "invalid_custom_attribute",
		},
		{
			name:     "日付",
			document: "id,custom:hired_on\nuser-alice,2026-13-45\n",
			secret:   "2026-13-45",
			want:     "invalid_custom_attribute",
		},
		{
			name:     "必須のカスタム属性",
			document: "id,attr:cost_center\nuser-alice,\n",
			secret:   "",
			want:     "invalid_custom_attribute",
		},
		{
			name:     "required_actions",
			document: "id,required_actions\nuser-alice,dance_a_jig\n",
			secret:   "dance_a_jig",
			want:     "invalid_required_actions",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := usermemory.NewUserRepository()
			repo.Seed(importPlannerUser("user-alice", "alice"))
			deps := importPlannerDeps(repo, perUserImportOwnershipGuard{})
			deps.SchemaReader = importSchemaReader{defs: defs}
			before := userSnapshot(t, repo)

			plan, err := planUserImportForTest(importPlannerContext(), deps, tc.document)
			if err != nil {
				t.Fatal(err)
			}
			if plan.RejectedRows() != 1 || plan.Rows[0].Error == nil || plan.Rows[0].Error.Code != tc.want {
				t.Fatalf("plan = %+v, want one row rejected with %q", plan, tc.want)
			}
			// 拒否行が運ぶのは位置と安定コードだけである。セル値を載せる実装は、
			// ジョブの表示と監査イベントの両方へその値を漏らす。
			if tc.secret != "" {
				encoded, err := json.Marshal(plan.Rows[0].Error)
				if err != nil {
					t.Fatal(err)
				}
				if bytes.Contains(encoded, []byte(tc.secret)) {
					t.Fatalf("拒否行がセル値 %q を運んだ: %s", tc.secret, encoded)
				}
			}
			if after := userSnapshot(t, repo); !sameUserSnapshots(before, after) {
				t.Fatalf("拒否が User を変えた: before=%v after=%v", before, after)
			}
		})
	}
}

// 具体例は読み取り専用列を「受理したうえで無視し、書き込み可能な列に差分がなければ
// `unchanged` とする」と言う。**受理と無視の両方を読む。** ファイルごと拒否する
// 実装も、読み取り専用列を書き込んでしまう実装も、片方だけの観測では通る。
//
//spec:covers EX-IDMANAGEMENT-007-04: status / mfa_enrolled / created_at / updated_at / id だけを編集した行が受理されたうえで unchanged になり、読み取り専用の値が動かないこと。
func TestPlanUserImportAcceptsAndIgnoresReadOnlyColumns(t *testing.T) {
	repo := usermemory.NewUserRepository()
	repo.Seed(importPlannerUser("user-alice", "alice"))

	plan, err := planUserImportForTest(importPlannerContext(),
		importPlannerDeps(repo, perUserImportOwnershipGuard{}),
		"id,status,mfa_enrolled,created_at,updated_at\n"+
			"user-alice,disabled,true,2001-01-01T00:00:00Z,2001-01-01T00:00:00Z\n")
	if err != nil {
		t.Fatal(err)
	}
	if plan.UnchangedRows() != 1 || plan.RejectedRows() != 0 {
		t.Fatalf("plan = %+v, want the row accepted and unchanged", plan)
	}
	planned := plan.Rows[0].User
	if planned.Lifecycle.Status != idmdomain.UserStatusActive || planned.MfaEnrolled {
		t.Fatalf("読み取り専用列が計画へ入った: status=%s mfa=%v", planned.Lifecycle.Status, planned.MfaEnrolled)
	}
	if planned.CreatedAt.Year() == 2001 || planned.UpdatedAt.Year() == 2001 {
		t.Fatalf("時刻の列が計画へ入った: created=%s updated=%s", planned.CreatedAt, planned.UpdatedAt)
	}
}
