package idmanagement_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	idmmemory "github.com/ambi/idmagic/backend/idmanagement/internal/db_memory"
	idmports "github.com/ambi/idmagic/backend/idmanagement/internal/ports"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/internal/usecases"
	"github.com/ambi/idmagic/backend/jobs"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

// 成果物の保存時刻はメモリの保存先の時計で決まるので、具体型に触れられるこのディレクトリで確かめる。
// 保持期限の削除のバッチがこの操作を呼ぶことは bootstrap のテストが確かめる。
//
//spec:covers EX-IDMANAGEMENT-080-01: 保持期限の削除のバッチが、IdManagement の成果物ストアから保持期限を過ぎた成果物を消し、新しい成果物を残すこと。
func TestPurgeExpiredCSVArtifactsKeepsArtifactsWithinRetention(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := idmmemory.NewCSVArtifactStore()
	ctx := context.Background()
	put := func(savedAt time.Time) string {
		t.Helper()
		store.Now = func() time.Time { return savedAt }
		metadata, err := store.PutCSVArtifact(ctx, "tenant-a", func(output io.Writer) error {
			_, err := io.WriteString(output, "preferred_username\nalice\n")
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return metadata.Ref
	}
	expired := put(now.Add(-idmusecases.CSVArtifactRetention - time.Hour))
	fresh := put(now.Add(-time.Hour))

	repo := jobs.NewJobRepository()
	canceler := jobs.NewCanceler(repo, tenancy.NewMemoryModule().QuotaRepo, func(spec.DomainEvent) {})
	if _, purged, err := (idmanagement.Module{CSVArtifacts: store}).PurgeExpiredCSVArtifacts(ctx, repo, canceler, now); err != nil || !purged {
		t.Fatalf("purged=%v err=%v", purged, err)
	}
	if _, _, err := store.OpenCSVArtifact(ctx, "tenant-a", expired); !errors.Is(err, idmports.ErrCSVArtifactNotFound) {
		t.Fatalf("保持期限を過ぎた成果物が残った: err=%v", err)
	}
	if _, _, err := store.OpenCSVArtifact(ctx, "tenant-a", fresh); err != nil {
		t.Fatalf("新しい成果物が消えた: %v", err)
	}
}

type observedRetentionStore struct {
	*idmmemory.CSVArtifactStore
	beforeDelete func()
}

func (s observedRetentionStore) DeleteCSVArtifactsCreatedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	s.beforeDelete()
	return s.CSVArtifactStore.DeleteCSVArtifactsCreatedBefore(ctx, cutoff)
}

//spec:covers REQ-IDMANAGEMENT-080: 三種類のプレビューと適用の待機中と実行中のジョブを、期限切れの入力の削除より先に取り消し、期限ちょうどの入力とジョブを残す。
func TestCSVRetentionCancelsExpiredImportsBeforeDeletingInputs(t *testing.T) {
	for _, pair := range [][2]jobsdomain.JobKind{
		{jobsdomain.KindUserImportPreview, jobsdomain.KindUserImportApply},
		{jobsdomain.KindGroupImportPreview, jobsdomain.KindGroupImportApply},
		{jobsdomain.KindGroupMembershipImportPreview, jobsdomain.KindGroupMembershipImportApply},
	} {
		for _, kind := range pair {
			for _, status := range []jobsdomain.JobStatus{jobsdomain.StatusQueued, jobsdomain.StatusRunning} {
				t.Run(string(kind)+"/"+string(status), func(t *testing.T) {
					ctx := context.Background()
					now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
					store := idmmemory.NewCSVArtifactStore()
					repo := jobs.NewJobRepository()
					enqueue := func(tenant string, jobKind jobsdomain.JobKind, params map[string]string) *jobsdomain.Job {
						t.Helper()
						raw, err := json.Marshal(params)
						if err != nil {
							t.Fatal(err)
						}
						job, _, err := repo.Enqueue(ctx, jobsports.EnqueueInput{TenantID: tenant, Kind: jobKind, Lane: jobsdomain.LaneBulk, Params: raw, Now: now, RunAt: now, MaxAttempts: 1})
						if err != nil {
							t.Fatal(err)
						}
						return job
					}
					claim := func() {
						t.Helper()
						if _, err := repo.ClaimBatch(ctx, "worker", jobsdomain.LaneBulk, 100, time.Minute, now); err != nil {
							t.Fatal(err)
						}
					}
					var expiredID, expiredRef, boundaryID, boundaryRef string
					for _, expired := range []bool{true, false} {
						savedAt := now.Add(-30 * 24 * time.Hour)
						tenant := "boundary-tenant"
						if expired {
							savedAt = savedAt.Add(-time.Second)
							tenant = "expired-tenant"
						}
						store.Now = func() time.Time { return savedAt }
						artifact, err := store.PutCSVArtifact(ctx, tenant, func(w io.Writer) error {
							_, err := io.WriteString(w, "preferred_username,email\nalice,alice@example.com\n")
							return err
						})
						if err != nil {
							t.Fatal(err)
						}
						preview := enqueue(tenant, pair[0], map[string]string{"artifact_ref": artifact.Ref})
						job := preview
						if kind == pair[1] {
							claim()
							if _, err := repo.Complete(ctx, preview.ID, "worker", json.RawMessage(`{}`), now); err != nil {
								t.Fatal(err)
							}
							job = enqueue(tenant, kind, map[string]string{"preview_job_id": preview.ID})
						}
						if status == jobsdomain.StatusRunning {
							claim()
						}
						if expired {
							expiredID, expiredRef = job.ID, artifact.Ref
						} else {
							boundaryID, boundaryRef = job.ID, artifact.Ref
						}
					}
					observed := observedRetentionStore{store, func() {
						job, err := repo.Get(ctx, expiredID)
						if err != nil || job.Status != jobsdomain.StatusCanceled {
							t.Errorf("削除前の期限切れジョブ=%+v err=%v", job, err)
						}
					}}
					canceler := jobs.NewCanceler(repo, tenancy.NewMemoryModule().QuotaRepo, func(spec.DomainEvent) {})
					deleted, _, err := (idmanagement.Module{CSVArtifacts: observed}).PurgeExpiredCSVArtifacts(ctx, repo, canceler, now)
					if err != nil || deleted != 1 {
						t.Fatalf("deleted=%d err=%v", deleted, err)
					}
					if _, _, err := store.OpenCSVArtifact(ctx, "expired-tenant", expiredRef); !errors.Is(err, idmports.ErrCSVArtifactNotFound) {
						t.Fatalf("期限切れの入力が残った: %v", err)
					}
					if _, _, err := store.OpenCSVArtifact(ctx, "boundary-tenant", boundaryRef); err != nil {
						t.Fatal(err)
					}
					if job, err := repo.Get(ctx, boundaryID); err != nil || job.Status != status {
						t.Fatalf("期限内のジョブ=%+v err=%v", job, err)
					}
				})
			}
		}
	}
}

type failingRetentionCanceler struct{ err error }

func (c failingRetentionCanceler) Cancel(context.Context, string, time.Time) (*jobsdomain.Job, error) {
	return nil, c.err
}

//spec:covers REQ-IDMANAGEMENT-080: ジョブを終端にできなければ入力を削除せず、再実行で回収する。
func TestCSVRetentionKeepsInputsWhenCancellationFails(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	store := idmmemory.NewCSVArtifactStore()
	store.Now = func() time.Time { return now.Add(-31 * 24 * time.Hour) }
	artifact, err := store.PutCSVArtifact(ctx, "tenant", func(w io.Writer) error { _, err := io.WriteString(w, "alice@example.com"); return err })
	if err != nil {
		t.Fatal(err)
	}
	repo := jobs.NewJobRepository()
	params, err := json.Marshal(map[string]string{"artifact_ref": artifact.Ref})
	if err != nil {
		t.Fatal(err)
	}
	job, _, err := repo.Enqueue(ctx, jobsports.EnqueueInput{TenantID: "tenant", Kind: jobsdomain.KindUserImportPreview, Lane: jobsdomain.LaneBulk, Params: params, Now: now, RunAt: now})
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("cancel unavailable")
	module := idmanagement.Module{CSVArtifacts: store}
	if _, _, err := module.PurgeExpiredCSVArtifacts(ctx, repo, failingRetentionCanceler{failure}, now); !errors.Is(err, failure) {
		t.Fatalf("err=%v", err)
	}
	reader, _, err := store.OpenCSVArtifact(ctx, "tenant", artifact.Ref)
	if err != nil {
		t.Fatalf("取り消しに失敗した入力が消えた: %v", err)
	}
	_ = reader.Close()
	stored, err := repo.Get(ctx, job.ID)
	if err != nil || stored.Status != jobsdomain.StatusQueued {
		t.Fatalf("job=%+v err=%v", stored, err)
	}
	canceler := jobs.NewCanceler(repo, tenancy.NewMemoryModule().QuotaRepo, func(spec.DomainEvent) {})
	if deleted, _, err := module.PurgeExpiredCSVArtifacts(ctx, repo, canceler, now); err != nil || deleted != 1 {
		t.Fatalf("再実行: deleted=%d err=%v", deleted, err)
	}
}

//spec:covers REQ-IDMANAGEMENT-080: 一覧のページをまたぐ全テナントの期限切れ入力を取り消し、別種のジョブを取り消さない。
func TestCSVRetentionVisitsEveryPageOfImports(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	store := idmmemory.NewCSVArtifactStore()
	store.Now = func() time.Time { return now.Add(-31 * 24 * time.Hour) }
	artifact, err := store.PutCSVArtifact(ctx, "tenant", func(w io.Writer) error { _, err := io.WriteString(w, "alice@example.com"); return err })
	if err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(map[string]string{"artifact_ref": artifact.Ref})
	if err != nil {
		t.Fatal(err)
	}
	repo := jobs.NewJobRepository()
	for range 201 {
		if _, _, err := repo.Enqueue(ctx, jobsports.EnqueueInput{TenantID: "tenant", Kind: jobsdomain.KindUserImportPreview, Lane: jobsdomain.LaneBulk, Params: params, Now: now, RunAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	other, _, err := repo.Enqueue(ctx, jobsports.EnqueueInput{TenantID: "tenant", Kind: jobsdomain.KindNoopEcho, Lane: jobsdomain.LaneDefault, Params: json.RawMessage(`{}`), Now: now, RunAt: now})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	canceler := jobs.NewCanceler(repo, tenancy.NewMemoryModule().QuotaRepo, func(event spec.DomainEvent) {
		if _, ok := event.(*jobsdomain.JobCanceled); ok {
			count++
		}
	})
	if deleted, _, err := (idmanagement.Module{CSVArtifacts: store}).PurgeExpiredCSVArtifacts(ctx, repo, canceler, now); err != nil || deleted != 1 {
		t.Fatalf("deleted=%d err=%v", deleted, err)
	}
	all, err := repo.ListByTenantAndKinds(ctx, "tenant", []jobsdomain.JobKind{jobsdomain.KindUserImportPreview}, 0)
	if err != nil || len(all) != 201 {
		t.Fatalf("jobs=%d err=%v", len(all), err)
	}
	for _, job := range all {
		if job.Status != jobsdomain.StatusCanceled {
			t.Fatalf("取り残したジョブ: %+v", job)
		}
	}
	if count != 201 {
		t.Fatalf("JobCanceled=%d", count)
	}
	if job, err := repo.Get(ctx, other.ID); err != nil || job.Status != jobsdomain.StatusQueued {
		t.Fatalf("別種のジョブ=%+v err=%v", job, err)
	}
}
