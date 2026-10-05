package usecases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	idmmemory "github.com/ambi/idmagic/backend/idmanagement/db_memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
)

func TestStartUserImportPreviewStoresPayloadOutsideJobParams(t *testing.T) {
	ctx := importPlannerContext()
	artifacts := idmmemory.NewCSVArtifactStore()
	jobs := jobsmemory.NewJobRepository()
	job, err := StartUserImportPreview(ctx, UserImportStartDeps{Artifacts: artifacts, Jobs: jobs}, "admin", strings.NewReader("preferred_username\nalice\n"), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(job.Params), "alice") || strings.Contains(string(job.Params), "csv\"") {
		t.Fatalf("job params leaked CSV payload: %s", job.Params)
	}
	var params UserImportParams
	if err := json.Unmarshal(job.Params, &params); err != nil || params.ArtifactRef == "" || params.SourceSHA256 == "" || params.ByteSize == 0 {
		t.Fatalf("params=%+v err=%v", params, err)
	}
	reader, metadata, err := artifacts.OpenCSVArtifact(ctx, "acme", params.ArtifactRef)
	if err != nil {
		t.Fatal(err)
	}
	_ = reader.Close()
	if metadata.SHA256 != params.SourceSHA256 {
		t.Fatalf("metadata=%+v params=%+v", metadata, params)
	}
}

func TestUserImportPreviewHandlerStoresSafeErrorsOutsideJobResult(t *testing.T) {
	ctx := importPlannerContext()
	artifacts := idmmemory.NewCSVArtifactStore()
	jobs := jobsmemory.NewJobRepository()
	job, err := StartUserImportPreview(ctx, UserImportStartDeps{Artifacts: artifacts, Jobs: jobs}, "admin", strings.NewReader("preferred_username,email\nalice,not-an-email\n"), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := UserImportJobHandler(UserImportJobDeps{
		Artifacts: artifacts, Jobs: jobs,
		Plan: UserImportPlanDeps{UserRepo: usermemory.NewUserRepository(), SchemaReader: importSchemaReader{}, OwnershipGuard: perUserImportOwnershipGuard{}},
	}, UserImportModePreview)(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "not-an-email") || strings.Contains(string(raw), "errors") {
		t.Fatalf("job result leaked payload or embedded errors: %s", raw)
	}
	var result UserImportResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result.RejectedRows != 1 || result.ErrorTotal != 1 {
		t.Fatalf("result=%+v", result)
	}
	page, err := ReadUserImportErrorRange(ctx, artifacts, "acme", result, 1, 2)
	if err != nil || len(page) != 1 || page[0].Code != "invalid_email" || page[0].Column != "email" {
		t.Fatalf("page=%+v err=%v", page, err)
	}
}

// 上限に触れないファイルの誤りは、投入では問わずにそのまま保存し、プレビューのジョブが
// 一件のエラーとして記録して成功する。
func TestCharacterizeUserImportPreviewOfMalformedFiles(t *testing.T) {
	for _, tc := range []struct {
		name        string
		document    string
		wantError   UserImportRowError
		wantCreated int
	}{
		{
			name:      "禁止した見出し",
			document:  "preferred_username,password\nalice,secret\n",
			wantError: UserImportRowError{Row: 1, Column: "password", Code: "invalid_header"},
		},
		{
			name:        "途中の行の構文の誤り",
			document:    "preferred_username\nalice\n\"bob\nunterminated",
			wantError:   UserImportRowError{Row: 3, Code: "invalid_csv"},
			wantCreated: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := importPlannerContext()
			artifacts := idmmemory.NewCSVArtifactStore()
			jobs := jobsmemory.NewJobRepository()
			job, err := StartUserImportPreview(ctx, UserImportStartDeps{Artifacts: artifacts, Jobs: jobs}, "admin", strings.NewReader(tc.document), time.Now().UTC())
			if err != nil {
				t.Fatalf("StartUserImportPreview: %v", err)
			}
			var params UserImportParams
			if err := json.Unmarshal(job.Params, &params); err != nil {
				t.Fatal(err)
			}
			if want := sha256.Sum256([]byte(tc.document)); params.SourceSHA256 != hex.EncodeToString(want[:]) || params.ByteSize != int64(len(tc.document)) {
				t.Fatalf("stored sha=%s size=%d, want the submitted bytes unchanged", params.SourceSHA256, params.ByteSize)
			}
			raw, err := UserImportJobHandler(UserImportJobDeps{
				Artifacts: artifacts, Jobs: jobs,
				Plan: UserImportPlanDeps{UserRepo: usermemory.NewUserRepository(), SchemaReader: importSchemaReader{}, OwnershipGuard: perUserImportOwnershipGuard{}},
			}, UserImportModePreview)(context.Background(), job)
			if err != nil {
				t.Fatalf("preview job failed: %v", err)
			}
			var result UserImportResult
			if err := json.Unmarshal(raw, &result); err != nil {
				t.Fatal(err)
			}
			page, err := ReadUserImportErrorRange(ctx, artifacts, "acme", result, 1, 10)
			if err != nil {
				t.Fatal(err)
			}
			if result.CreatedRows != tc.wantCreated || result.RejectedRows != 1 || result.ErrorTotal != 1 || len(page) != 1 || page[0] != tc.wantError {
				t.Fatalf("result=%+v errors=%+v, want created=%d and the single error %+v", result, page, tc.wantCreated, tc.wantError)
			}
		})
	}
}

// 投入は max_bytes ちょうどのファイルをそのまま保存し、1 byte 超えるファイルを
// csv_too_large で拒否してジョブを作らない。
//
//spec:covers REQ-IDMANAGEMENT-004, EX-IDMANAGEMENT-004-02: max_bytes ちょうどのUser CSV をすべて保存し、1 byte 超えるファイルの投入を csv_too_large で拒否してジョブを作らないこと。
func TestStartUserImportPreviewRefusesFilesBeyondTheByteLimit(t *testing.T) {
	document := "preferred_username\nalice\n"
	for _, tc := range []struct {
		name     string
		maxBytes int
		wantErr  bool
	}{
		{name: "上限ちょうど", maxBytes: len(document)},
		{name: "上限を 1 byte 超える", maxBytes: len(document) - 1, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := importPlannerContext()
			jobs := jobsmemory.NewJobRepository()
			policy := idmdomain.CSVTransferPolicy{MaxRows: 100, MaxBytes: tc.maxBytes, MaxFieldBytes: 1 << 10}
			job, err := StartUserImportPreview(ctx, UserImportStartDeps{Artifacts: idmmemory.NewCSVArtifactStore(), Jobs: jobs, Policy: policy}, "admin", strings.NewReader(document), time.Now().UTC())
			if tc.wantErr {
				csvErr, ok := errors.AsType[*idmdomain.CSVError](err)
				queued, listErr := jobs.ListByTenantAndKinds(ctx, "acme", []jobsdomain.JobKind{jobsdomain.KindUserImportPreview}, 10)
				if !ok || csvErr.Code != idmdomain.CSVErrorCSVTooLarge || job != nil || listErr != nil || len(queued) != 0 {
					t.Fatalf("job=%v err=%v queued=%d, want csv_too_large and no job", job, err, len(queued))
				}
				return
			}
			if err != nil {
				t.Fatalf("StartUserImportPreview: %v", err)
			}
			var params UserImportParams
			if err := json.Unmarshal(job.Params, &params); err != nil || params.ByteSize != int64(len(document)) {
				t.Fatalf("params=%+v err=%v, want the whole %d bytes stored", params, err, len(document))
			}
		})
	}
}

// 行数と項目長の超過も、byte 数と同じく投入の時点で拒否する。プレビューのジョブを
// 作ってから中で気付く実装は、上限の手前までの行を計画し、その適用が先頭の行を取り込む。
//
//spec:covers REQ-IDMANAGEMENT-004, EX-IDMANAGEMENT-004-02: max_rows または max_field_bytes を超える User CSV の投入が、安定コードで拒否され、プレビューのジョブも成果物も作らないこと。
func TestStartUserImportPreviewRefusesFilesBeyondTheRowAndFieldLimits(t *testing.T) {
	for _, tc := range []struct {
		name     string
		policy   idmdomain.CSVTransferPolicy
		document string
		want     idmdomain.CSVErrorCode
	}{
		{
			name:     "行数の上限",
			policy:   idmdomain.CSVTransferPolicy{MaxRows: 1, MaxBytes: 1 << 20, MaxFieldBytes: 1 << 10},
			document: "preferred_username\nalice\nbob\n",
			want:     idmdomain.CSVErrorTooManyRows,
		},
		{
			name:     "項目長の上限",
			policy:   idmdomain.CSVTransferPolicy{MaxRows: 100, MaxBytes: 1 << 20, MaxFieldBytes: 8},
			document: "preferred_username\nalice\n" + strings.Repeat("x", 64) + "\n",
			want:     idmdomain.CSVErrorFieldTooLarge,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := importPlannerContext()
			artifacts := idmmemory.NewCSVArtifactStore()
			jobs := jobsmemory.NewJobRepository()
			job, err := StartUserImportPreview(ctx, UserImportStartDeps{Artifacts: artifacts, Jobs: jobs, Policy: tc.policy}, "admin", strings.NewReader(tc.document), time.Now().UTC())
			if csvErr, ok := errors.AsType[*idmdomain.CSVError](err); !ok || csvErr.Code != tc.want || job != nil {
				t.Fatalf("job=%v err=%v, want the submission refused with %q", job, err, tc.want)
			}
			queued, err := jobs.ListByTenantAndKinds(ctx, "acme", []jobsdomain.JobKind{jobsdomain.KindUserImportPreview}, 10)
			if err != nil || len(queued) != 0 {
				t.Fatalf("queued=%d err=%v, want no preview job", len(queued), err)
			}
			if stored, _ := artifacts.DeleteCSVArtifactsCreatedBefore(ctx, time.Now().Add(time.Hour)); stored != 0 {
				t.Fatalf("the refusal left %d artifacts", stored)
			}
		})
	}
}

// 適用のジョブは、プレビューの後に上限が下がって保存したファイルが上限を超えたとき、
// 一行も確定せずに失敗する。解析の途中で気付く実装は、上限の手前の行を確定してしまう。
//
//spec:covers REQ-IDMANAGEMENT-004, EX-IDMANAGEMENT-004-02: 実効の上限を超えたファイルの適用のジョブが、行を一つも確定せずに失敗すること。
func TestUserImportJobFailsWithoutApplyingAFileBeyondTheLimits(t *testing.T) {
	ctx := importPlannerContext()
	artifacts := idmmemory.NewCSVArtifactStore()
	jobs := jobsmemory.NewJobRepository()
	preview, err := StartUserImportPreview(ctx, UserImportStartDeps{Artifacts: artifacts, Jobs: jobs}, "admin", strings.NewReader("preferred_username\nalice\nbob\n"), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	var params UserImportParams
	if err := json.Unmarshal(preview.Params, &params); err != nil {
		t.Fatal(err)
	}
	if claimed, err := jobs.ClaimBatch(ctx, "worker", jobsdomain.LaneBulk, 1, time.Minute, time.Now().UTC()); err != nil || len(claimed) != 1 {
		t.Fatalf("claim=%+v err=%v", claimed, err)
	}
	if _, err := jobs.Complete(ctx, preview.ID, "worker", mustJSON(t, UserImportResult{SourceSHA256: params.SourceSHA256}), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	apply, err := StartUserImportApply(ctx, UserImportStartDeps{Artifacts: artifacts, Jobs: jobs}, "admin", preview.ID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	repo := usermemory.NewUserRepository()
	committer := &importRowCommitter{}
	_, err = UserImportJobHandler(UserImportJobDeps{
		Artifacts: artifacts, Jobs: jobs,
		Apply: UserImportApplyDeps{
			Plan: importPlannerDeps(repo, perUserImportOwnershipGuard{}), Committer: committer,
			PasswordHasher: importTestHasher{}, DynamicGroups: importDynamicGroups(repo),
		},
		Policy: idmdomain.CSVTransferPolicy{MaxRows: 1, MaxBytes: 1 << 20, MaxFieldBytes: 1 << 10},
	}, UserImportModeApply)(context.Background(), apply)
	if csvErr, ok := errors.AsType[*idmdomain.CSVError](err); !ok || csvErr.Code != idmdomain.CSVErrorTooManyRows {
		t.Fatalf("err=%v, want the apply job failed with too_many_rows", err)
	}
	if len(committer.mutations) != 0 {
		t.Fatalf("the failed apply committed %d rows", len(committer.mutations))
	}
}

func TestStartUserImportApplyRequiresSucceededSameTenantPreviewAndDigest(t *testing.T) {
	ctx := importPlannerContext()
	artifacts := idmmemory.NewCSVArtifactStore()
	jobs := jobsmemory.NewJobRepository()
	preview, err := StartUserImportPreview(ctx, UserImportStartDeps{Artifacts: artifacts, Jobs: jobs}, "admin", strings.NewReader("preferred_username\nalice\n"), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := StartUserImportApply(ctx, UserImportStartDeps{Artifacts: artifacts, Jobs: jobs}, "admin", preview.ID, time.Now().UTC()); !errors.Is(err, ErrUserImportPreviewNotReady) {
		t.Fatalf("queued preview err=%v", err)
	}
	var params UserImportParams
	_ = json.Unmarshal(preview.Params, &params)
	claimed, err := jobs.ClaimBatch(ctx, "worker", jobsdomain.LaneBulk, 1, time.Minute, time.Now().UTC())
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim=%+v err=%v", claimed, err)
	}
	if _, err := jobs.Complete(ctx, preview.ID, "worker", mustJSON(t, UserImportResult{SourceSHA256: params.SourceSHA256}), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	apply, err := StartUserImportApply(ctx, UserImportStartDeps{Artifacts: artifacts, Jobs: jobs}, "admin", preview.ID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(apply.Params), params.ArtifactRef) || strings.Contains(string(apply.Params), "alice") {
		t.Fatalf("apply params leaked artifact or CSV: %s", apply.Params)
	}
	var applyParams UserImportParams
	_ = json.Unmarshal(apply.Params, &applyParams)
	if applyParams.PreviewJobID != preview.ID || applyParams.SourceSHA256 != params.SourceSHA256 {
		t.Fatalf("apply params=%+v", applyParams)
	}
}

func TestReadUserImportErrorRangeCrossesImmutableArtifactPages(t *testing.T) {
	ctx := importPlannerContext()
	artifacts := idmmemory.NewCSVArtifactStore()
	metadata, err := artifacts.PutCSVArtifactPages(ctx, "acme", func(emit func([]byte) error) error {
		for pageNumber := range 3 {
			page := make([]UserImportRowError, 0, UserImportErrorArtifactPageSize)
			for index := range UserImportErrorArtifactPageSize {
				ordinal := pageNumber*UserImportErrorArtifactPageSize + index + 1
				if ordinal > 450 {
					break
				}
				page = append(page, UserImportRowError{Row: ordinal + 1, Code: "invalid_email"})
			}
			payload, err := json.Marshal(page)
			if err != nil {
				return err
			}
			if err := emit(payload); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	result := UserImportResult{ErrorArtifactRef: metadata.Ref, ErrorArtifactSHA: metadata.SHA256, ErrorTotal: 450}
	page, err := ReadUserImportErrorRange(ctx, artifacts, "acme", result, 151, 200)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 200 || page[0].Row != 152 || page[199].Row != 351 {
		t.Fatalf("page first=%+v last=%+v len=%d", page[0], page[len(page)-1], len(page))
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
