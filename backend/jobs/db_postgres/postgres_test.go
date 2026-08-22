package db_postgres_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/ambi/idmagic/backend/jobs/db_postgres"
	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

// resetJobsTable truncates jobs before a test runs. embedded-postgres is a
// single shared instance for the whole test binary (pgtest.Main), not a
// per-test transaction, and ClaimBatch deliberately scans across tenants
// (a worker pool serves every tenant), so a job left StatusQueued by an
// earlier test would otherwise be swept up by a later test's ClaimBatch call.
func resetJobsTable(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), "TRUNCATE jobs"); err != nil {
		t.Fatalf("truncate jobs: %v", err)
	}
}

func newTestJob(t *testing.T, r *postgres.JobRepository, tenantID string, now time.Time) *domain.Job {
	t.Helper()
	job, _, err := r.Enqueue(context.Background(), ports.EnqueueInput{
		TenantID:    tenantID,
		Kind:        domain.KindNoopEcho,
		Lane:        domain.LaneDefault,
		Params:      json.RawMessage(`{}`),
		MaxAttempts: domain.DefaultMaxAttempts,
		Now:         now,
		RunAt:       now,
	})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	return job
}

func TestEnqueue_DedupReturnsExistingNonTerminalJob(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	dedup := "import-2026-01"
	input := ports.EnqueueInput{
		TenantID: tenant.ID, Kind: domain.KindNoopEcho, Lane: domain.LaneDefault, Params: json.RawMessage(`{}`),
		DedupKey: &dedup, MaxAttempts: domain.DefaultMaxAttempts, Now: now, RunAt: now,
	}
	first, created, err := r.Enqueue(context.Background(), input)
	if err != nil {
		t.Fatalf("first Enqueue() error = %v", err)
	}
	if !created {
		t.Error("first Enqueue() created = false, want true")
	}
	second, created, err := r.Enqueue(context.Background(), input)
	if err != nil {
		t.Fatalf("second Enqueue() error = %v", err)
	}
	if created {
		t.Error("second Enqueue() created = true, want false (dedup hit)")
	}
	if second.ID != first.ID {
		t.Errorf("second Enqueue() returned a new Job %q, want existing %q", second.ID, first.ID)
	}
}

func TestEnqueue_DedupIgnoresTerminalJobs(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	dedup := "import-2026-01"
	input := ports.EnqueueInput{
		TenantID: tenant.ID, Kind: domain.KindNoopEcho, Lane: domain.LaneDefault, Params: json.RawMessage(`{}`),
		DedupKey: &dedup, MaxAttempts: domain.DefaultMaxAttempts, Now: now, RunAt: now,
	}
	first, _, err := r.Enqueue(context.Background(), input)
	if err != nil {
		t.Fatalf("first Enqueue() error = %v", err)
	}
	claimed, err := r.ClaimBatch(context.Background(), "worker-1", domain.LaneDefault, 10, time.Minute, now)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ClaimBatch() = %v, %v", claimed, err)
	}
	if _, err := r.Complete(context.Background(), first.ID, "worker-1", nil, now); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	second, created, err := r.Enqueue(context.Background(), input)
	if err != nil {
		t.Fatalf("second Enqueue() error = %v", err)
	}
	if !created {
		t.Error("second Enqueue() created = false, want true (previous dedup match is terminal)")
	}
	if second.ID == first.ID {
		t.Error("second Enqueue() reused a terminal Job's dedup key, want a new Job")
	}
}

// TestClaimBatch_ExcludesOtherLanes: RED for lane isolation against
// the real ClaimJobs SQL (jobs_claimable_idx is lane-prefixed) — a due,
// queued Job in one lane must never be returned when claiming a different
// lane.
func TestClaimBatch_ExcludesOtherLanes(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	if _, _, err := r.Enqueue(context.Background(), ports.EnqueueInput{
		TenantID: tenant.ID, Kind: domain.KindNoopEcho, Lane: domain.LaneDefault, Params: json.RawMessage(`{}`),
		MaxAttempts: domain.DefaultMaxAttempts, Now: now, RunAt: now,
	}); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	claimed, err := r.ClaimBatch(context.Background(), "worker-1", domain.LaneBulk, 10, time.Minute, now)
	if err != nil {
		t.Fatalf("ClaimBatch() error = %v", err)
	}
	if len(claimed) != 0 {
		t.Errorf("ClaimBatch(lane=bulk) claimed %d jobs, want 0 (job is lane=default)", len(claimed))
	}
}

func TestClaimBatch_ExcludesFutureRunAt(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	if _, _, err := r.Enqueue(context.Background(), ports.EnqueueInput{
		TenantID: tenant.ID, Kind: domain.KindNoopEcho, Lane: domain.LaneDefault, Params: json.RawMessage(`{}`),
		MaxAttempts: domain.DefaultMaxAttempts, Now: now, RunAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	claimed, err := r.ClaimBatch(context.Background(), "worker-1", domain.LaneDefault, 10, time.Minute, now)
	if err != nil {
		t.Fatalf("ClaimBatch() error = %v", err)
	}
	if len(claimed) != 0 {
		t.Errorf("ClaimBatch() claimed %d jobs, want 0 (run_at in the future)", len(claimed))
	}
}

func TestClaimBatch_IncrementsAttemptsAndSetsLease(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	job := newTestJob(t, r, tenant.ID, now)

	claimed, err := r.ClaimBatch(context.Background(), "worker-1", domain.LaneDefault, 10, time.Minute, now)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ClaimBatch() = %v, %v", claimed, err)
	}
	got := claimed[0]
	if got.ID != job.ID {
		t.Fatalf("claimed job ID = %q, want %q", got.ID, job.ID)
	}
	if got.Status != domain.StatusRunning {
		t.Errorf("Status = %q, want %q", got.Status, domain.StatusRunning)
	}
	if got.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", got.Attempts)
	}
	if got.LeaseOwner == nil || *got.LeaseOwner != "worker-1" {
		t.Errorf("LeaseOwner = %v, want worker-1", got.LeaseOwner)
	}
	if got.LeaseExpiresAt == nil || !got.LeaseExpiresAt.Equal(now.Add(time.Minute)) {
		t.Errorf("LeaseExpiresAt = %v, want %v", got.LeaseExpiresAt, now.Add(time.Minute))
	}
}

func TestClaimBatch_ReclaimsExpiredLease(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	newTestJob(t, r, tenant.ID, now)

	first, err := r.ClaimBatch(context.Background(), "worker-1", domain.LaneDefault, 10, time.Minute, now)
	if err != nil || len(first) != 1 {
		t.Fatalf("first ClaimBatch() = %v, %v", first, err)
	}

	stillLocked, err := r.ClaimBatch(context.Background(), "worker-2", domain.LaneDefault, 10, time.Minute, now.Add(30*time.Second))
	if err != nil {
		t.Fatalf("ClaimBatch() before expiry error = %v", err)
	}
	if len(stillLocked) != 0 {
		t.Fatalf("ClaimBatch() before lease expiry claimed %d jobs, want 0 (JobLeaseExclusivity)", len(stillLocked))
	}

	afterExpiry := now.Add(2 * time.Minute)
	reclaimed, err := r.ClaimBatch(context.Background(), "worker-2", domain.LaneDefault, 10, time.Minute, afterExpiry)
	if err != nil || len(reclaimed) != 1 {
		t.Fatalf("ClaimBatch() after expiry = %v, %v", reclaimed, err)
	}
	got := reclaimed[0]
	if got.Status != domain.StatusRunning {
		t.Errorf("Status = %q, want %q (unchanged across reclaim)", got.Status, domain.StatusRunning)
	}
	if got.Attempts != 2 {
		t.Errorf("Attempts = %d, want 2 (second worker's attempt)", got.Attempts)
	}
	if got.LeaseOwner == nil || *got.LeaseOwner != "worker-2" {
		t.Errorf("LeaseOwner = %v, want worker-2", got.LeaseOwner)
	}
}

// TestClaimBatch_ConcurrentClaimIsExclusive proves the actual `FOR UPDATE SKIP
// LOCKED` SQL, not just the memory adapter's mutex approximation:
// many concurrent callers claiming from the same real PostgreSQL table must
// never claim the same job twice (JobLeaseExclusivity). Run with -race.
func TestClaimBatch_ConcurrentClaimIsExclusive(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	const numJobs = 30
	for range numJobs {
		newTestJob(t, r, tenant.ID, now)
	}

	const numWorkers = 8
	var mu sync.Mutex
	claimedBy := map[string]string{}
	var wg sync.WaitGroup
	for w := range numWorkers {
		wg.Add(1)
		workerID := "worker-" + string(rune('A'+w))
		go func() {
			defer wg.Done()
			for range numJobs {
				claimed, err := r.ClaimBatch(context.Background(), workerID, domain.LaneDefault, 1, time.Minute, now)
				if err != nil {
					t.Errorf("ClaimBatch() error = %v", err)
					return
				}
				for _, j := range claimed {
					mu.Lock()
					if owner, ok := claimedBy[j.ID]; ok {
						t.Errorf("job %q claimed by both %q and %q (JobLeaseExclusivity violated)", j.ID, owner, workerID)
					}
					claimedBy[j.ID] = workerID
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()

	if len(claimedBy) != numJobs {
		t.Errorf("claimed %d distinct jobs, want %d", len(claimedBy), numJobs)
	}
}

func TestHeartbeat_ExtendsLease(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	newTestJob(t, r, tenant.ID, now)
	claimed, _ := r.ClaimBatch(context.Background(), "worker-1", domain.LaneDefault, 10, time.Minute, now)
	job := claimed[0]

	newExpiry, err := r.Heartbeat(context.Background(), job.ID, "worker-1", time.Minute, now.Add(30*time.Second))
	if err != nil {
		t.Fatalf("Heartbeat() error = %v", err)
	}
	want := now.Add(30 * time.Second).Add(time.Minute)
	if !newExpiry.Equal(want) {
		t.Errorf("Heartbeat() = %v, want %v", newExpiry, want)
	}
}

func TestHeartbeat_WrongWorkerReturnsErrJobLeaseLost(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	newTestJob(t, r, tenant.ID, now)
	claimed, _ := r.ClaimBatch(context.Background(), "worker-1", domain.LaneDefault, 10, time.Minute, now)
	job := claimed[0]

	_, err := r.Heartbeat(context.Background(), job.ID, "worker-2", time.Minute, now)
	if !errors.Is(err, ports.ErrJobLeaseLost) {
		t.Errorf("Heartbeat() error = %v, want ErrJobLeaseLost", err)
	}
}

func TestComplete_Success(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	newTestJob(t, r, tenant.ID, now)
	claimed, _ := r.ClaimBatch(context.Background(), "worker-1", domain.LaneDefault, 10, time.Minute, now)
	job := claimed[0]

	result := json.RawMessage(`{"ok":true}`)
	got, err := r.Complete(context.Background(), job.ID, "worker-1", result, now)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if got.Status != domain.StatusSucceeded {
		t.Errorf("Status = %q, want %q", got.Status, domain.StatusSucceeded)
	}
	// Compare decoded values, not raw bytes: JSONB re-serializes on storage
	// (e.g. `{"ok":true}` round-trips as `{"ok": true}`).
	var gotVal, wantVal any
	if err := json.Unmarshal(got.Result, &gotVal); err != nil {
		t.Fatalf("unmarshal got.Result: %v", err)
	}
	if err := json.Unmarshal(result, &wantVal); err != nil {
		t.Fatalf("unmarshal want result: %v", err)
	}
	if fmt.Sprint(gotVal) != fmt.Sprint(wantVal) {
		t.Errorf("Result = %s, want %s", got.Result, result)
	}
}

func TestComplete_WrongWorkerReturnsErrJobLeaseLost(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	newTestJob(t, r, tenant.ID, now)
	claimed, _ := r.ClaimBatch(context.Background(), "worker-1", domain.LaneDefault, 10, time.Minute, now)
	job := claimed[0]

	_, err := r.Complete(context.Background(), job.ID, "worker-2", nil, now)
	if !errors.Is(err, ports.ErrJobLeaseLost) {
		t.Errorf("Complete() error = %v, want ErrJobLeaseLost", err)
	}
}

func TestFail_RetrySetsQueuedAndRunAt(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	newTestJob(t, r, tenant.ID, now)
	claimed, _ := r.ClaimBatch(context.Background(), "worker-1", domain.LaneDefault, 10, time.Minute, now)
	job := claimed[0]

	nextRunAt := now.Add(30 * time.Second)
	got, err := r.Fail(context.Background(), job.ID, "worker-1", ports.FailOutcome{
		NextStatus: domain.StatusQueued, RunAt: nextRunAt, Error: "temporary failure",
	}, now)
	if err != nil {
		t.Fatalf("Fail() error = %v", err)
	}
	if got.Status != domain.StatusQueued {
		t.Errorf("Status = %q, want %q", got.Status, domain.StatusQueued)
	}
	if !got.RunAt.Equal(nextRunAt) {
		t.Errorf("RunAt = %v, want %v", got.RunAt, nextRunAt)
	}
	if got.Error == nil || *got.Error != "temporary failure" {
		t.Errorf("Error = %v, want %q", got.Error, "temporary failure")
	}
}

func TestFail_DeadLetterSetsFailedTerminal(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	newTestJob(t, r, tenant.ID, now)
	claimed, _ := r.ClaimBatch(context.Background(), "worker-1", domain.LaneDefault, 10, time.Minute, now)
	job := claimed[0]

	got, err := r.Fail(context.Background(), job.ID, "worker-1", ports.FailOutcome{
		NextStatus: domain.StatusFailed, Error: "permanent failure",
	}, now)
	if err != nil {
		t.Fatalf("Fail() error = %v", err)
	}
	if got.Status != domain.StatusFailed {
		t.Errorf("Status = %q, want %q", got.Status, domain.StatusFailed)
	}
	if !domain.IsJobLifecycleTerminal(got.Status) {
		t.Error("dead-lettered Job should be terminal")
	}

	if _, err := r.Complete(context.Background(), job.ID, "worker-1", nil, now); !errors.Is(err, ports.ErrJobLeaseLost) {
		t.Errorf("Complete() on terminal Job error = %v, want ErrJobLeaseLost", err)
	}
}

func TestCancel_TerminalReturnsErrJobAlreadyTerminal(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	job := newTestJob(t, r, tenant.ID, now)
	claimed, _ := r.ClaimBatch(context.Background(), "worker-1", domain.LaneDefault, 10, time.Minute, now)
	if _, err := r.Complete(context.Background(), claimed[0].ID, "worker-1", nil, now); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	_, err := r.Cancel(context.Background(), job.ID, now)
	if !errors.Is(err, ports.ErrJobAlreadyTerminal) {
		t.Errorf("Cancel() error = %v, want ErrJobAlreadyTerminal", err)
	}
}

func TestCancel_FromQueuedSucceeds(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	job := newTestJob(t, r, tenant.ID, now)

	got, err := r.Cancel(context.Background(), job.ID, now)
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if got.Status != domain.StatusCanceled {
		t.Errorf("Status = %q, want %q", got.Status, domain.StatusCanceled)
	}
}

// TestLaneDepths_CountsQueuedAndRunningPerLane: RED for wi-261 T006 against
// the real SQL (count(*) FILTER + GROUP BY lane).
func TestLaneDepths_CountsQueuedAndRunningPerLane(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenant := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	now := pgtest.Now()
	enqueueLane := func(lane domain.ExecutionLane) *domain.Job {
		job, _, err := r.Enqueue(context.Background(), ports.EnqueueInput{
			TenantID: tenant.ID, Kind: domain.KindNoopEcho, Lane: lane, Params: json.RawMessage(`{}`),
			MaxAttempts: domain.DefaultMaxAttempts, Now: now, RunAt: now,
		})
		if err != nil {
			t.Fatalf("Enqueue() error = %v", err)
		}
		return job
	}
	enqueueLane(domain.LaneDefault)
	enqueueLane(domain.LaneDefault)
	enqueueLane(domain.LaneBulk)
	if _, err := r.ClaimBatch(context.Background(), "worker-1", domain.LaneBulk, 10, time.Minute, now); err != nil {
		t.Fatalf("ClaimBatch() error = %v", err)
	}

	depths, err := r.LaneDepths(context.Background())
	if err != nil {
		t.Fatalf("LaneDepths() error = %v", err)
	}
	byLane := map[domain.ExecutionLane]ports.LaneDepth{}
	for _, d := range depths {
		byLane[d.Lane] = d
	}
	if got := byLane[domain.LaneDefault]; got.Queued != 2 || got.Running != 0 {
		t.Errorf("default lane depth = %+v, want Queued=2 Running=0", got)
	}
	if got := byLane[domain.LaneBulk]; got.Queued != 0 || got.Running != 1 {
		t.Errorf("bulk lane depth = %+v, want Queued=0 Running=1", got)
	}
	if _, ok := byLane[domain.LaneLatencySensitive]; ok {
		t.Error("latency_sensitive lane has no rows, want it absent from LaneDepths()")
	}
}

func TestGet_NotFound(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	if _, err := r.Get(context.Background(), "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ports.ErrJobNotFound) {
		t.Errorf("Get() error = %v, want ErrJobNotFound", err)
	}
}

func TestListByTenantAndKinds_ScopesAndOrders(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenantA := pgfixtures.SeedTenant(t, pool)
	tenantB := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	base := pgtest.Now()
	enqueue := func(tenantID string, kind domain.JobKind, lane domain.ExecutionLane, at time.Time) *domain.Job {
		job, _, err := r.Enqueue(context.Background(), ports.EnqueueInput{
			TenantID: tenantID, Kind: kind, Lane: lane, Params: json.RawMessage(`{}`),
			MaxAttempts: 1, Now: at, RunAt: at,
		})
		if err != nil {
			t.Fatalf("Enqueue: %v", err)
		}
		return job
	}
	older := enqueue(tenantA.ID, domain.KindNoopEcho, domain.LaneDefault, base.Add(-time.Minute))
	newer := enqueue(tenantA.ID, domain.KindNoopEcho, domain.LaneDefault, base)
	enqueue(tenantA.ID, domain.KindUserImportApply, domain.LaneBulk, base) // other kind, excluded
	enqueue(tenantB.ID, domain.KindNoopEcho, domain.LaneDefault, base)     // other tenant, excluded

	got, err := r.ListByTenantAndKinds(context.Background(), tenantA.ID, []domain.JobKind{domain.KindNoopEcho}, 0)
	if err != nil {
		t.Fatalf("ListByTenantAndKinds: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d jobs, want 2 (tenant+kind scoped)", len(got))
	}
	if got[0].ID != newer.ID || got[1].ID != older.ID {
		t.Errorf("order = [%s %s], want newest-first [%s %s]", got[0].ID, got[1].ID, newer.ID, older.ID)
	}

	limited, err := r.ListByTenantAndKinds(context.Background(), tenantA.ID, []domain.JobKind{domain.KindNoopEcho}, 1)
	if err != nil {
		t.Fatalf("ListByTenantAndKinds limit: %v", err)
	}
	if len(limited) != 1 || limited[0].ID != newer.ID {
		t.Errorf("limit=1 got %d jobs, want [%s]", len(limited), newer.ID)
	}

	none, err := r.ListByTenantAndKinds(context.Background(), tenantA.ID, nil, 0)
	if err != nil {
		t.Fatalf("ListByTenantAndKinds no kinds: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("no kinds got %d jobs, want 0", len(none))
	}
}

// REQ-JOBS-012: 管理 API の一覧が、テナント、状態、種別、レーンで絞り込まれ、
// キーセットでページを継ぎ、範囲を与えられなければ全件へ退避せず拒否すること。
// memory の実装と同じ意味論を、実際の SQL に対しても固定する。
func TestListForAdmin(t *testing.T) {
	pool := pgtest.Require(t)
	resetJobsTable(t, pool)
	tenantA := pgfixtures.SeedTenant(t, pool)
	tenantB := pgfixtures.SeedTenant(t, pool)
	r := &postgres.JobRepository{Pool: pool}
	base := pgtest.Now()

	enqueue := func(tenantID string, kind domain.JobKind, lane domain.ExecutionLane, offset time.Duration) *domain.Job {
		t.Helper()
		job, _, err := r.Enqueue(context.Background(), ports.EnqueueInput{
			TenantID: tenantID, Kind: kind, Lane: lane, Params: json.RawMessage(`{}`),
			MaxAttempts: domain.DefaultMaxAttempts, Now: base.Add(offset), RunAt: base.Add(offset),
		})
		if err != nil {
			t.Fatalf("Enqueue() error = %v", err)
		}
		return job
	}
	oldest := enqueue(tenantA.ID, domain.KindNoopEcho, domain.LaneDefault, 0)
	middle := enqueue(tenantA.ID, domain.KindUserImportApply, domain.LaneBulk, time.Minute)
	newest := enqueue(tenantA.ID, domain.KindNoopEcho, domain.LaneDefault, 2*time.Minute)
	foreign := enqueue(tenantB.ID, domain.KindNoopEcho, domain.LaneDefault, 3*time.Minute)

	ctx := context.Background()
	ids := func(jobs []*domain.Job) []string {
		out := make([]string, len(jobs))
		for i, j := range jobs {
			out[i] = j.ID
		}
		return out
	}

	t.Run("tenant scoped and newest first", func(t *testing.T) {
		jobs, err := r.ListForAdmin(ctx, ports.AdminJobFilter{TenantID: tenantA.ID})
		if err != nil {
			t.Fatalf("ListForAdmin() error = %v", err)
		}
		want := []string{newest.ID, middle.ID, oldest.ID}
		if got := ids(jobs); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("ids = %v, want %v", got, want)
		}
	})

	t.Run("another tenant is absent", func(t *testing.T) {
		jobs, err := r.ListForAdmin(ctx, ports.AdminJobFilter{TenantID: tenantA.ID})
		if err != nil {
			t.Fatalf("ListForAdmin() error = %v", err)
		}
		for _, j := range jobs {
			if j.ID == foreign.ID {
				t.Fatal("another tenant's job appeared in a tenant-scoped listing")
			}
		}
	})

	t.Run("all tenants", func(t *testing.T) {
		jobs, err := r.ListForAdmin(ctx, ports.AdminJobFilter{AllTenants: true})
		if err != nil {
			t.Fatalf("ListForAdmin() error = %v", err)
		}
		if len(jobs) != 4 {
			t.Fatalf("got %d jobs, want 4", len(jobs))
		}
	})

	t.Run("kind, lane, and status narrow the page", func(t *testing.T) {
		byKind, err := r.ListForAdmin(ctx, ports.AdminJobFilter{
			TenantID: tenantA.ID, Kinds: []domain.JobKind{domain.KindUserImportApply},
		})
		if err != nil {
			t.Fatalf("ListForAdmin() error = %v", err)
		}
		if len(byKind) != 1 || byKind[0].ID != middle.ID {
			t.Fatalf("kind filter = %v, want [%s]", ids(byKind), middle.ID)
		}
		byLane, err := r.ListForAdmin(ctx, ports.AdminJobFilter{TenantID: tenantA.ID, Lane: domain.LaneBulk})
		if err != nil {
			t.Fatalf("ListForAdmin() error = %v", err)
		}
		if len(byLane) != 1 || byLane[0].ID != middle.ID {
			t.Fatalf("lane filter = %v, want [%s]", ids(byLane), middle.ID)
		}
		byStatus, err := r.ListForAdmin(ctx, ports.AdminJobFilter{
			TenantID: tenantA.ID, Statuses: []domain.JobStatus{domain.StatusFailed},
		})
		if err != nil {
			t.Fatalf("ListForAdmin() error = %v", err)
		}
		if len(byStatus) != 0 {
			t.Fatalf("status filter = %v, want none", ids(byStatus))
		}
	})

	t.Run("keyset continues without a gap or a repeat", func(t *testing.T) {
		first, err := r.ListForAdmin(ctx, ports.AdminJobFilter{TenantID: tenantA.ID, Limit: 2})
		if err != nil {
			t.Fatalf("ListForAdmin() error = %v", err)
		}
		if len(first) != 2 {
			t.Fatalf("first page = %d jobs, want 2", len(first))
		}
		last := first[len(first)-1]
		second, err := r.ListForAdmin(ctx, ports.AdminJobFilter{
			TenantID: tenantA.ID, Limit: 2, BeforeCreatedAt: last.CreatedAt, BeforeID: last.ID,
		})
		if err != nil {
			t.Fatalf("ListForAdmin() error = %v", err)
		}
		if len(second) != 1 || second[0].ID != oldest.ID {
			t.Fatalf("second page = %v, want [%s]", ids(second), oldest.ID)
		}
	})

	t.Run("an unscoped filter is refused", func(t *testing.T) {
		if _, err := r.ListForAdmin(ctx, ports.AdminJobFilter{}); !errors.Is(err, ports.ErrAdminJobFilterUnscoped) {
			t.Fatalf("err = %v, want ErrAdminJobFilterUnscoped", err)
		}
	})
}
