package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	idmports "github.com/ambi/idmagic/backend/idmanagement/internal/ports"
	"github.com/ambi/idmagic/backend/jobs"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
)

//spec:covers REQ-IDMANAGEMENT-080: Group とメンバーシップの適用を再投入しても入力の期限を延長せず、期限切れのプレビューと適用を実行しない。
func TestGroupCSVImportInputExpiryDoesNotExtendWithApply(t *testing.T) {
	for _, membership := range []bool{false, true} {
		name := "group"
		if membership {
			name = "membership"
		}
		t.Run(name, func(t *testing.T) {
			f := newGroupImportFixture(t)
			created := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
			boundary := created.Add(30 * 24 * time.Hour)
			repo := jobs.NewJobRepository()
			groupDeps := GroupImportStartDeps{SchemaReader: groupAttributeSchemaStub{}, Artifacts: f.artifacts, Jobs: repo, Enqueuer: testJobEnqueuer(repo)}
			memberDeps := GroupMembershipImportStartDeps{Artifacts: f.artifacts, Jobs: repo, Enqueuer: testJobEnqueuer(repo)}
			startPreview := func() (*jobsdomain.Job, error) {
				if membership {
					return StartGroupMembershipImportPreview(f.ctx, memberDeps, "admin", "group-a", strings.NewReader("user_id,membership_state\nuser-a,present\n"), created)
				}
				return StartGroupImportPreview(f.ctx, groupDeps, "admin", strings.NewReader("name\nsales\n"), created)
			}
			startApply := func(id string, now time.Time) (*jobsdomain.Job, error) {
				if membership {
					return StartGroupMembershipImportApply(f.ctx, memberDeps, "admin", "group-a", id, now)
				}
				return StartGroupImportApply(f.ctx, groupDeps, "admin", id, now)
			}
			preview, err := startPreview()
			if err != nil {
				t.Fatal(err)
			}
			var params struct {
				SourceSHA256 string `json:"source_sha256"`
			}
			if err := json.Unmarshal(preview.Params, &params); err != nil {
				t.Fatal(err)
			}
			if _, err := repo.ClaimBatch(f.ctx, "worker", jobsdomain.LaneBulk, 1, time.Minute, created); err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(map[string]string{"source_sha256": params.SourceSHA256, "group_id": "group-a"})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := repo.Complete(f.ctx, preview.ID, "worker", raw, created); err != nil {
				t.Fatal(err)
			}
			apply, err := startApply(preview.ID, boundary)
			if err != nil {
				t.Fatalf("期限ちょうどを拒否した: %v", err)
			}
			wantErr := ErrGroupImportDigestMismatch
			if membership {
				wantErr = ErrGroupMembershipImportDigestMismatch
			}
			if _, err := startApply(preview.ID, boundary.Add(time.Microsecond)); !errors.Is(err, wantErr) {
				t.Fatalf("期限切れの再投入: %v", err)
			}
			applyKind := jobsdomain.KindGroupImportApply
			if membership {
				applyKind = jobsdomain.KindGroupMembershipImportApply
			}
			queued, err := repo.ListByTenantAndKinds(f.ctx, "acme", []jobsdomain.JobKind{applyKind}, 0)
			if err != nil || len(queued) != 1 {
				t.Fatalf("適用ジョブ=%v err=%v", queued, err)
			}
			clock := func() time.Time { return boundary.Add(time.Second) }
			if membership {
				_, err = GroupMembershipImportJobHandler(GroupMembershipImportJobDeps{Artifacts: f.artifacts, Jobs: repo, Now: clock}, GroupMembershipImportModeApply)(f.ctx, apply)
			} else {
				_, err = GroupImportJobHandler(GroupImportJobDeps{Artifacts: f.artifacts, Jobs: repo, Apply: f.apply, Now: clock}, GroupImportModeApply)(f.ctx, apply)
			}
			if !errors.Is(err, wantErr) || len(f.committer.mutations) != 0 {
				t.Fatalf("期限切れの適用: err=%v mutations=%d", err, len(f.committer.mutations))
			}
			if membership {
				_, err = GroupMembershipImportJobHandler(GroupMembershipImportJobDeps{Artifacts: f.artifacts, Jobs: repo, Now: clock}, GroupMembershipImportModePreview)(f.ctx, preview)
			} else {
				_, err = GroupImportJobHandler(GroupImportJobDeps{Artifacts: f.artifacts, Jobs: repo, Plan: f.plan, Now: clock}, GroupImportModePreview)(f.ctx, preview)
			}
			if !errors.Is(err, idmports.ErrCSVArtifactNotFound) {
				t.Fatalf("期限切れのプレビュー: %v", err)
			}
		})
	}
}

//spec:covers REQ-IDMANAGEMENT-080: 中断後は Group CSV の次の行を計画しない。
func TestGroupImportStopsPlanningAfterCancellation(t *testing.T) {
	f := newGroupImportFixture(t)
	f.seedGroup(t, "group-a", "sales", groupdomain.GroupMembershipManual)
	ctx, cancel := context.WithCancel(f.ctx)
	defer cancel()
	rows := 0
	_, err := PlanGroupImport(ctx, f.plan, strings.NewReader("name\nsales\nops\n"), idmdomain.DefaultCSVTransferPolicy(), func(groupdomain.GroupImportRowPlan) error {
		rows++
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) || rows != 1 {
		t.Fatalf("rows=%d err=%v", rows, err)
	}
}
