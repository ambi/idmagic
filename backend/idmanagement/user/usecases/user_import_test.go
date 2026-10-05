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
func TestCharacterizeStartUserImportPreviewByteLimit(t *testing.T) {
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
