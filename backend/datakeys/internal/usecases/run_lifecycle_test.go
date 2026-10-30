package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/internal/domain"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
)

const lifecycleCadence = 90 * 24 * time.Hour

type lifecycleMigrator struct {
	pending    int
	pendingErr error
}

func (m lifecycleMigrator) ReencryptBatch(context.Context, string, int, int) (int, error) {
	return 0, nil
}

func (m lifecycleMigrator) PendingCount(context.Context, string, int) (int, error) {
	return m.pending, m.pendingErr
}

type lifecycleJobs struct {
	enqueued []string
	err      error
}

func (j *lifecycleJobs) Enqueue(_ context.Context, input jobsports.EnqueueInput, _ time.Time) (*jobsdomain.Job, error) {
	if j.err != nil {
		return nil, j.err
	}
	j.enqueued = append(j.enqueued, *input.DedupKey)
	return &jobsdomain.Job{}, nil
}

//spec:covers REQ-DATAKEYS-007: 経過が周期の直前なら対象にせず、ちょうど周期とその直後なら対象にする。
func TestRotationDueBoundary(t *testing.T) {
	activated := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	key := &domain.TenantDataEncryptionKey{CreatedAt: activated.Add(-time.Hour), ActivatedAt: &activated}
	for _, tc := range []struct {
		name string
		now  time.Time
		want bool
	}{
		{"just before", activated.Add(lifecycleCadence - time.Nanosecond), false},
		{"exactly", activated.Add(lifecycleCadence), true},
		{"just after", activated.Add(lifecycleCadence + time.Nanosecond), true},
	} {
		if got := domain.RotationDue(key, tc.now, lifecycleCadence); got != tc.want {
			t.Errorf("%s: RotationDue = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestRotationDueFallsBackToCreatedAtWithoutActivation(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	key := &domain.TenantDataEncryptionKey{CreatedAt: created}
	if !domain.RotationDue(key, created.Add(lifecycleCadence), lifecycleCadence) {
		t.Fatal("key without activated_at should age from created_at")
	}
}

func lifecycleActiveVersion(t *testing.T, deps Deps, tenantID string) int {
	t.Helper()
	key, err := deps.Repository.FindActive(t.Context(), tenantID)
	if err != nil {
		t.Fatal(err)
	}
	return key.Version
}

//spec:covers REQ-DATAKEYS-007: 周期前のテナントはローテーションせず、未移行を報告する FieldMigrator の再暗号化のジョブだけを予約する。
func TestRunTenantDataKeyLifecycleBeforeCadenceRecoversOnlyPendingMigrators(t *testing.T) {
	deps := newTestDeps(t)
	jobs := &lifecycleJobs{}
	deps.Jobs = jobs
	deps.Migrators = NewMigratorRegistry()
	deps.Migrators.Register("pending", lifecycleMigrator{pending: 1})
	deps.Migrators.Register("clean", lifecycleMigrator{})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := BootstrapTenantDataKey(t.Context(), deps, "tenant-a", now); err != nil {
		t.Fatal(err)
	}

	if err := RunTenantDataKeyLifecycle(t.Context(), deps, "tenant-a", now.Add(lifecycleCadence-time.Second), lifecycleCadence); err != nil {
		t.Fatal(err)
	}

	if got := lifecycleActiveVersion(t, deps, "tenant-a"); got != 1 {
		t.Fatalf("active version = %d, want 1", got)
	}
	want := ReencryptionDedupKey("tenant-a", "pending")
	if len(jobs.enqueued) != 1 || jobs.enqueued[0] != want {
		t.Fatalf("enqueued = %v, want only %q", jobs.enqueued, want)
	}
}

//spec:covers REQ-DATAKEYS-007: 周期を過ぎたテナントはローテーションする。
func TestRunTenantDataKeyLifecycleRotatesAtCadence(t *testing.T) {
	deps := newTestDeps(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := BootstrapTenantDataKey(t.Context(), deps, "tenant-a", now); err != nil {
		t.Fatal(err)
	}
	if err := RunTenantDataKeyLifecycle(t.Context(), deps, "tenant-a", now.Add(lifecycleCadence), lifecycleCadence); err != nil {
		t.Fatal(err)
	}
	if got := lifecycleActiveVersion(t, deps, "tenant-a"); got != 2 {
		t.Fatalf("active version = %d, want 2", got)
	}
}

//spec:covers REQ-DATAKEYS-007: DEK を持たないテナントには DEK を作らず、失敗にもしない。
func TestRunTenantDataKeyLifecycleSkipsTenantWithoutDataKey(t *testing.T) {
	deps := newTestDeps(t)
	if err := RunTenantDataKeyLifecycle(t.Context(), deps, "tenant-a", time.Now().UTC(), lifecycleCadence); err != nil {
		t.Fatal(err)
	}
	if _, err := deps.Repository.FindActive(t.Context(), "tenant-a"); !errors.Is(err, domain.ErrNoActiveDataKey) {
		t.Fatalf("FindActive error = %v, want ErrNoActiveDataKey", err)
	}
}

func TestRunTenantDataKeyLifecycleReportsPendingCountFailure(t *testing.T) {
	deps := newTestDeps(t)
	pendingErr := errors.New("count failed")
	deps.Migrators = NewMigratorRegistry()
	deps.Migrators.Register("broken", lifecycleMigrator{pendingErr: pendingErr})
	deps.Jobs = &lifecycleJobs{}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := BootstrapTenantDataKey(t.Context(), deps, "tenant-a", now); err != nil {
		t.Fatal(err)
	}
	if err := RunTenantDataKeyLifecycle(t.Context(), deps, "tenant-a", now, lifecycleCadence); !errors.Is(err, pendingErr) {
		t.Fatalf("error = %v, want %v", err, pendingErr)
	}
}

// 投入の失敗は次回の実行で回収するので、そのテナントの失敗にしない。
func TestRunTenantDataKeyLifecycleToleratesEnqueueFailure(t *testing.T) {
	deps := newTestDeps(t)
	deps.Migrators = NewMigratorRegistry()
	deps.Migrators.Register("pending", lifecycleMigrator{pending: 1})
	deps.Jobs = &lifecycleJobs{err: errors.New("over quota")}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := BootstrapTenantDataKey(t.Context(), deps, "tenant-a", now); err != nil {
		t.Fatal(err)
	}
	if err := RunTenantDataKeyLifecycle(t.Context(), deps, "tenant-a", now, lifecycleCadence); err != nil {
		t.Fatal(err)
	}
}
