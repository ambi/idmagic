package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	memoryjobs "github.com/ambi/idmagic/backend/jobs/db_memory"
	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// seedAdminJobs は tenant-a に 3 件、tenant-b に 1 件の Job を作る。作成時刻を
// ずらして並び順とキーセットの継続を確かめられるようにする。
func seedAdminJobs(t *testing.T) (*memoryjobs.JobRepository, time.Time) {
	t.Helper()
	repo := memoryjobs.NewJobRepository()
	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	seed := []struct {
		tenant string
		kind   domain.JobKind
		offset time.Duration
	}{
		{"tenant-a", domain.KindNoopEcho, 0},
		{"tenant-a", domain.KindUserImportApply, time.Minute},
		{"tenant-a", domain.KindNoopEcho, 2 * time.Minute},
		{"tenant-b", domain.KindNoopEcho, 3 * time.Minute},
	}
	for _, s := range seed {
		if _, _, err := repo.Enqueue(context.Background(), ports.EnqueueInput{
			TenantID: s.tenant, Kind: s.kind, Lane: mustLane(t, s.kind),
			Params: json.RawMessage(`{"secret":"do-not-surface"}`), MaxAttempts: 3,
			RunAt: base.Add(s.offset), Now: base.Add(s.offset),
		}); err != nil {
			t.Fatalf("seed enqueue: %v", err)
		}
	}
	return repo, base
}

func mustLane(t *testing.T, kind domain.JobKind) domain.ExecutionLane {
	t.Helper()
	lane, ok := domain.LaneFor(kind)
	if !ok {
		t.Fatalf("no lane registered for %q", kind)
	}
	return lane
}

// REQ-JOBS-012: 一覧は自テナントに閉じ、他テナントの Job は結果にも件数にも現れない。
func TestListJobsForAdmin_StaysInsideTheTenant(t *testing.T) {
	repo, _ := seedAdminJobs(t)

	page, err := usecases.ListJobsForAdmin(context.Background(), usecases.AdminJobDeps{Repo: repo},
		usecases.ListJobsInput{Scope: usecases.TenantScope{TenantID: "tenant-a"}})
	if err != nil {
		t.Fatalf("ListJobsForAdmin() error = %v", err)
	}
	if len(page.Jobs) != 3 {
		t.Fatalf("got %d jobs, want 3", len(page.Jobs))
	}
	for _, j := range page.Jobs {
		if j.TenantID != "tenant-a" {
			t.Fatalf("job %s belongs to %q, want tenant-a only", j.ID, j.TenantID)
		}
	}
	// 新しい順。
	for i := 1; i < len(page.Jobs); i++ {
		if page.Jobs[i-1].CreatedAt.Before(page.Jobs[i].CreatedAt) {
			t.Fatalf("jobs are not newest-first: %v then %v", page.Jobs[i-1].CreatedAt, page.Jobs[i].CreatedAt)
		}
	}
}

// REQ-JOBS-012: system_admin が制御面テナントから明示したときだけ全テナントを見る。
func TestListJobsForAdmin_AllTenantsRequiresAnAuthorizedScope(t *testing.T) {
	repo, _ := seedAdminJobs(t)

	page, err := usecases.ListJobsForAdmin(context.Background(), usecases.AdminJobDeps{Repo: repo},
		usecases.ListJobsInput{Scope: usecases.TenantScope{AllTenants: true}})
	if err != nil {
		t.Fatalf("ListJobsForAdmin() error = %v", err)
	}
	if len(page.Jobs) != 4 {
		t.Fatalf("got %d jobs across tenants, want 4", len(page.Jobs))
	}

	// テナントも横断も指定しない範囲は、既定で全件を返さずに拒否する。
	if _, err := usecases.ListJobsForAdmin(context.Background(), usecases.AdminJobDeps{Repo: repo},
		usecases.ListJobsInput{}); err == nil {
		t.Fatal("an unscoped listing must be refused, not defaulted to every tenant")
	}
}

// REQ-JOBS-012: 状態・種別・レーンで絞り込める。
func TestListJobsForAdmin_Filters(t *testing.T) {
	repo, _ := seedAdminJobs(t)
	ctx := context.Background()
	deps := usecases.AdminJobDeps{Repo: repo}

	byKind, err := usecases.ListJobsForAdmin(ctx, deps, usecases.ListJobsInput{
		Scope: usecases.TenantScope{TenantID: "tenant-a"},
		Kinds: []domain.JobKind{domain.KindUserImportApply},
	})
	if err != nil {
		t.Fatalf("ListJobsForAdmin() error = %v", err)
	}
	if len(byKind.Jobs) != 1 || byKind.Jobs[0].Kind != domain.KindUserImportApply {
		t.Fatalf("kind filter returned %#v", byKind.Jobs)
	}

	byStatus, err := usecases.ListJobsForAdmin(ctx, deps, usecases.ListJobsInput{
		Scope:    usecases.TenantScope{TenantID: "tenant-a"},
		Statuses: []domain.JobStatus{domain.StatusSucceeded},
	})
	if err != nil {
		t.Fatalf("ListJobsForAdmin() error = %v", err)
	}
	if len(byStatus.Jobs) != 0 {
		t.Fatalf("status filter returned %d jobs, want 0", len(byStatus.Jobs))
	}

	byLane, err := usecases.ListJobsForAdmin(ctx, deps, usecases.ListJobsInput{
		Scope: usecases.TenantScope{TenantID: "tenant-a"},
		Lane:  domain.LaneBulk,
	})
	if err != nil {
		t.Fatalf("ListJobsForAdmin() error = %v", err)
	}
	if len(byLane.Jobs) != 1 || byLane.Jobs[0].Lane != domain.LaneBulk {
		t.Fatalf("lane filter returned %#v", byLane.Jobs)
	}
}

// REQ-JOBS-012: カーソルはページを継いで重複も欠落も生まず、絞り込みが変われば拒否される。
func TestListJobsForAdmin_CursorContinuesAndBindsTheFilter(t *testing.T) {
	repo, _ := seedAdminJobs(t)
	ctx := context.Background()
	deps := usecases.AdminJobDeps{Repo: repo}
	scope := usecases.TenantScope{TenantID: "tenant-a"}

	first, err := usecases.ListJobsForAdmin(ctx, deps, usecases.ListJobsInput{Scope: scope, Limit: 2})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first.Jobs) != 2 || first.NextCursor == "" {
		t.Fatalf("first page = %d jobs, cursor %q", len(first.Jobs), first.NextCursor)
	}

	second, err := usecases.ListJobsForAdmin(ctx, deps, usecases.ListJobsInput{Scope: scope, Limit: 2, Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(second.Jobs) != 1 {
		t.Fatalf("second page = %d jobs, want 1", len(second.Jobs))
	}
	if second.NextCursor != "" {
		t.Fatalf("last page must not offer a cursor, got %q", second.NextCursor)
	}
	seen := map[string]bool{}
	for _, j := range append(append([]*domain.Job{}, first.Jobs...), second.Jobs...) {
		if seen[j.ID] {
			t.Fatalf("job %s appeared on both pages", j.ID)
		}
		seen[j.ID] = true
	}

	// 絞り込みを変えたカーソルは、黙って別の位置に読み替えず拒否する。
	if _, err := usecases.ListJobsForAdmin(ctx, deps, usecases.ListJobsInput{
		Scope: scope, Limit: 2, Cursor: first.NextCursor, Lane: domain.LaneBulk,
	}); !errors.Is(err, usecases.ErrJobCursorMismatch) {
		t.Fatalf("changed filter with an old cursor: err = %v, want ErrJobCursorMismatch", err)
	}

	// 別テナントで発行されたカーソルも同じ理由で拒否する。
	if _, err := usecases.ListJobsForAdmin(ctx, deps, usecases.ListJobsInput{
		Scope: usecases.TenantScope{TenantID: "tenant-b"}, Limit: 2, Cursor: first.NextCursor,
	}); !errors.Is(err, usecases.ErrJobCursorMismatch) {
		t.Fatalf("cursor from another tenant: err = %v, want ErrJobCursorMismatch", err)
	}
}

// REQ-JOBS-012: 他テナントの Job は id を知っていても存在しないものとして扱う。
func TestGetJobForAdmin_HidesOtherTenants(t *testing.T) {
	repo, _ := seedAdminJobs(t)
	ctx := context.Background()
	deps := usecases.AdminJobDeps{Repo: repo}

	all, err := usecases.ListJobsForAdmin(ctx, deps, usecases.ListJobsInput{Scope: usecases.TenantScope{AllTenants: true}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var foreign *domain.Job
	for _, j := range all.Jobs {
		if j.TenantID == "tenant-b" {
			foreign = j
		}
	}
	if foreign == nil {
		t.Fatal("seed did not create a tenant-b job")
	}

	if _, err := usecases.GetJobForAdmin(ctx, deps, foreign.ID, usecases.TenantScope{TenantID: "tenant-a"}); !errors.Is(err, ports.ErrJobNotFound) {
		t.Fatalf("cross-tenant read: err = %v, want ErrJobNotFound", err)
	}
	if _, err := usecases.GetJobForAdmin(ctx, deps, foreign.ID, usecases.TenantScope{AllTenants: true}); err != nil {
		t.Fatalf("cross-tenant read as system_admin: %v", err)
	}
}

// REQ-JOBS-013: 終端に達していない Job だけを取り消せ、JobCanceled が発行される。
func TestCancelJobForAdmin(t *testing.T) {
	repo, base := seedAdminJobs(t)
	ctx := context.Background()
	var emitted []spec.DomainEvent
	deps := usecases.AdminJobDeps{Repo: repo, Emit: func(e spec.DomainEvent) { emitted = append(emitted, e) }}

	page, err := usecases.ListJobsForAdmin(ctx, deps, usecases.ListJobsInput{Scope: usecases.TenantScope{TenantID: "tenant-a"}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	target := page.Jobs[0]

	canceled, err := usecases.CancelJobForAdmin(ctx, deps, target.ID, usecases.TenantScope{TenantID: "tenant-a"}, base)
	if err != nil {
		t.Fatalf("CancelJobForAdmin() error = %v", err)
	}
	if canceled.Status != domain.StatusCanceled {
		t.Fatalf("status = %q, want canceled", canceled.Status)
	}
	if len(emitted) != 1 {
		t.Fatalf("emitted %d events, want 1", len(emitted))
	}
	if _, ok := emitted[0].(*domain.JobCanceled); !ok {
		t.Fatalf("emitted %T, want *domain.JobCanceled", emitted[0])
	}

	// 終端に達した Job の取り消しは、成功として黙認せず拒否する。
	if _, err := usecases.CancelJobForAdmin(ctx, deps, target.ID, usecases.TenantScope{TenantID: "tenant-a"}, base); !errors.Is(err, ports.ErrJobAlreadyTerminal) {
		t.Fatalf("cancelling a terminal job: err = %v, want ErrJobAlreadyTerminal", err)
	}
}

// REQ-JOBS-013: 他テナントの Job は取り消せず、存在しないものとして扱う。
func TestCancelJobForAdmin_HidesOtherTenants(t *testing.T) {
	repo, base := seedAdminJobs(t)
	ctx := context.Background()
	deps := usecases.AdminJobDeps{Repo: repo}

	all, err := usecases.ListJobsForAdmin(ctx, deps, usecases.ListJobsInput{Scope: usecases.TenantScope{AllTenants: true}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var foreign *domain.Job
	for _, j := range all.Jobs {
		if j.TenantID == "tenant-b" {
			foreign = j
		}
	}

	if _, err := usecases.CancelJobForAdmin(ctx, deps, foreign.ID, usecases.TenantScope{TenantID: "tenant-a"}, base); !errors.Is(err, ports.ErrJobNotFound) {
		t.Fatalf("cross-tenant cancel: err = %v, want ErrJobNotFound", err)
	}
	got, err := repo.Get(ctx, foreign.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != domain.StatusQueued {
		t.Fatalf("a refused cross-tenant cancel changed the job to %q", got.Status)
	}
}
