package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmmemory "github.com/ambi/idmagic/backend/idmanagement/internal/db_memory"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/internal/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/internal/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/jobs"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
)

//spec:covers REQ-IDMANAGEMENT-080: 入力の期限ちょうどでは適用を受け付け、期限超過後は再投入せず、すでに投入した適用も行を確定しない。
func TestUserImportInputExpiryDoesNotExtendWithApply(t *testing.T) {
	ctx := importPlannerContext()
	created := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	boundary := created.Add(30 * 24 * time.Hour)
	artifacts := idmmemory.NewCSVArtifactStore()
	artifacts.Now = func() time.Time { return created }
	repo := jobs.NewJobRepository()
	deps := UserImportStartDeps{SchemaReader: importSchemaReader{}, Artifacts: artifacts, Jobs: repo, Enqueuer: testJobEnqueuer(repo)}
	preview, err := StartUserImportPreview(ctx, deps, "admin", strings.NewReader("preferred_username\nalice\n"), created)
	if err != nil {
		t.Fatal(err)
	}
	var params UserImportParams
	if err := json.Unmarshal(preview.Params, &params); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ClaimBatch(ctx, "worker", jobsdomain.LaneBulk, 1, time.Minute, created); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Complete(ctx, preview.ID, "worker", mustJSON(t, UserImportResult{SourceSHA256: params.SourceSHA256}), created); err != nil {
		t.Fatal(err)
	}
	apply, err := StartUserImportApply(ctx, deps, "admin", preview.ID, boundary)
	if err != nil {
		t.Fatalf("期限ちょうどを拒否した: %v", err)
	}
	if _, err := StartUserImportApply(ctx, deps, "admin", preview.ID, boundary.Add(time.Microsecond)); !errors.Is(err, ErrUserImportDigestMismatch) {
		t.Errorf("期限超過の適用を受け付けた: %v", err)
	}
	queued, err := repo.ListByTenantAndKinds(ctx, "acme", []jobsdomain.JobKind{jobsdomain.KindUserImportApply}, 0)
	if err != nil || len(queued) != 1 {
		t.Fatalf("適用ジョブ=%v err=%v", queued, err)
	}
	users := usermemory.NewUserRepository()
	committer := &importRowCommitter{}
	_, err = UserImportJobHandler(UserImportJobDeps{
		Artifacts: artifacts, Jobs: repo, Now: func() time.Time { return boundary.Add(time.Second) },
		Apply: UserImportApplyDeps{Plan: importPlannerDeps(users, perUserImportOwnershipGuard{}), Committer: committer, PasswordHasher: importTestHasher{}, DynamicGroups: importDynamicGroups(users), Emit: discardImportEvents},
	}, UserImportModeApply)(ctx, apply)
	if err == nil || len(committer.mutations) != 0 {
		t.Fatalf("期限切れの適用: err=%v mutations=%d", err, len(committer.mutations))
	}
}

//spec:covers REQ-IDMANAGEMENT-080: 中断された計画は、先読み済みの CSV の次の行を処理しない。
func TestUserImportStopsPlanningAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(importPlannerContext())
	defer cancel()
	rows := 0
	_, err := PlanUserImport(ctx, importPlannerDeps(usermemory.NewUserRepository(), perUserImportOwnershipGuard{}), strings.NewReader("preferred_username\nalice\nbob\n"), idmdomain.DefaultCSVTransferPolicy(), func(_ userdomain.UserImportRowPlan) error {
		rows++
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) || rows != 1 {
		t.Fatalf("rows=%d err=%v", rows, err)
	}
}

//spec:covers REQ-IDMANAGEMENT-080: 期限切れのプレビューは行を計画せず、結果の成果物も作らない。
func TestUserImportExpiredPreviewCreatesNoResult(t *testing.T) {
	ctx := importPlannerContext()
	created := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	store := idmmemory.NewCSVArtifactStore()
	store.Now = func() time.Time { return created }
	repo := jobs.NewJobRepository()
	job, err := StartUserImportPreview(ctx, UserImportStartDeps{SchemaReader: importSchemaReader{}, Artifacts: store, Jobs: repo, Enqueuer: testJobEnqueuer(repo)}, "admin", strings.NewReader("preferred_username\nalice\n"), created)
	if err != nil {
		t.Fatal(err)
	}
	before, err := store.CSVArtifactUsage(ctx, "acme")
	if err != nil {
		t.Fatal(err)
	}
	_, err = UserImportJobHandler(UserImportJobDeps{Artifacts: store, Jobs: repo, Plan: importPlannerDeps(usermemory.NewUserRepository(), perUserImportOwnershipGuard{}), Now: func() time.Time { return created.Add(idmusecases.CSVArtifactRetention + time.Second) }}, UserImportModePreview)(ctx, job)
	if err == nil {
		t.Fatal("期限切れのプレビューが成功した")
	}
	after, err := store.CSVArtifactUsage(ctx, "acme")
	if err != nil || before != after {
		t.Fatalf("usage=%d -> %d err=%v", before, after, err)
	}
}
