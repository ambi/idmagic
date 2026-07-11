package memory

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
)

func newTestJob(t *testing.T, r *JobRepository, now time.Time) *domain.Job {
	t.Helper()
	job, _, err := r.Enqueue(context.Background(), ports.EnqueueInput{
		TenantID:    "tenant-a",
		Kind:        domain.KindNoopEcho,
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
	r := NewJobRepository()
	now := time.Now().UTC()
	dedup := "import-2026-01"
	input := ports.EnqueueInput{
		TenantID: "tenant-a", Kind: domain.KindNoopEcho, Params: json.RawMessage(`{}`),
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
	r := NewJobRepository()
	now := time.Now().UTC()
	dedup := "import-2026-01"
	input := ports.EnqueueInput{
		TenantID: "tenant-a", Kind: domain.KindNoopEcho, Params: json.RawMessage(`{}`),
		DedupKey: &dedup, MaxAttempts: domain.DefaultMaxAttempts, Now: now, RunAt: now,
	}
	first, _, err := r.Enqueue(context.Background(), input)
	if err != nil {
		t.Fatalf("first Enqueue() error = %v", err)
	}
	claimed, err := r.ClaimBatch(context.Background(), "worker-1", 10, time.Minute, now)
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

func TestClaimBatch_ExcludesFutureRunAt(t *testing.T) {
	r := NewJobRepository()
	now := time.Now().UTC()
	_, _, err := r.Enqueue(context.Background(), ports.EnqueueInput{
		TenantID: "tenant-a", Kind: domain.KindNoopEcho, Params: json.RawMessage(`{}`),
		MaxAttempts: domain.DefaultMaxAttempts, Now: now, RunAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	claimed, err := r.ClaimBatch(context.Background(), "worker-1", 10, time.Minute, now)
	if err != nil {
		t.Fatalf("ClaimBatch() error = %v", err)
	}
	if len(claimed) != 0 {
		t.Errorf("ClaimBatch() claimed %d jobs, want 0 (run_at in the future)", len(claimed))
	}
}

func TestClaimBatch_IncrementsAttemptsAndSetsLease(t *testing.T) {
	r := NewJobRepository()
	now := time.Now().UTC()
	job := newTestJob(t, r, now)

	claimed, err := r.ClaimBatch(context.Background(), "worker-1", 10, time.Minute, now)
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
	r := NewJobRepository()
	now := time.Now().UTC()
	newTestJob(t, r, now)

	first, err := r.ClaimBatch(context.Background(), "worker-1", 10, time.Minute, now)
	if err != nil || len(first) != 1 {
		t.Fatalf("first ClaimBatch() = %v, %v", first, err)
	}

	// worker-1 crashes without heartbeating; simulate lease expiry by polling
	// after the lease's expiry time.
	afterExpiry := now.Add(2 * time.Minute)
	stillLocked, err := r.ClaimBatch(context.Background(), "worker-2", 10, time.Minute, now.Add(30*time.Second))
	if err != nil {
		t.Fatalf("ClaimBatch() before expiry error = %v", err)
	}
	if len(stillLocked) != 0 {
		t.Fatalf("ClaimBatch() before lease expiry claimed %d jobs, want 0 (JobLeaseExclusivity)", len(stillLocked))
	}

	reclaimed, err := r.ClaimBatch(context.Background(), "worker-2", 10, time.Minute, afterExpiry)
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

// TestClaimBatch_ConcurrentClaimIsExclusive is the JobLeaseExclusivity
// invariant under concurrency: N jobs claimed by many concurrent callers must
// each be claimed by exactly one worker. Run with -race.
func TestClaimBatch_ConcurrentClaimIsExclusive(t *testing.T) {
	r := NewJobRepository()
	now := time.Now().UTC()
	const numJobs = 50
	for range numJobs {
		newTestJob(t, r, now)
	}

	const numWorkers = 10
	var mu sync.Mutex
	claimedBy := map[string]string{} // jobID -> workerID
	var wg sync.WaitGroup
	for w := range numWorkers {
		wg.Add(1)
		workerID := "worker-" + string(rune('A'+w))
		go func() {
			defer wg.Done()
			for range numJobs {
				claimed, err := r.ClaimBatch(context.Background(), workerID, 1, time.Minute, now)
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
	r := NewJobRepository()
	now := time.Now().UTC()
	newTestJob(t, r, now)
	claimed, _ := r.ClaimBatch(context.Background(), "worker-1", 10, time.Minute, now)
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
	r := NewJobRepository()
	now := time.Now().UTC()
	newTestJob(t, r, now)
	claimed, _ := r.ClaimBatch(context.Background(), "worker-1", 10, time.Minute, now)
	job := claimed[0]

	_, err := r.Heartbeat(context.Background(), job.ID, "worker-2", time.Minute, now)
	if !errors.Is(err, ports.ErrJobLeaseLost) {
		t.Errorf("Heartbeat() error = %v, want ErrJobLeaseLost", err)
	}
}

func TestComplete_Success(t *testing.T) {
	r := NewJobRepository()
	now := time.Now().UTC()
	newTestJob(t, r, now)
	claimed, _ := r.ClaimBatch(context.Background(), "worker-1", 10, time.Minute, now)
	job := claimed[0]

	result := json.RawMessage(`{"ok":true}`)
	got, err := r.Complete(context.Background(), job.ID, "worker-1", result, now)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if got.Status != domain.StatusSucceeded {
		t.Errorf("Status = %q, want %q", got.Status, domain.StatusSucceeded)
	}
	if string(got.Result) != string(result) {
		t.Errorf("Result = %s, want %s", got.Result, result)
	}
}

func TestComplete_WrongWorkerReturnsErrJobLeaseLost(t *testing.T) {
	r := NewJobRepository()
	now := time.Now().UTC()
	newTestJob(t, r, now)
	claimed, _ := r.ClaimBatch(context.Background(), "worker-1", 10, time.Minute, now)
	job := claimed[0]

	_, err := r.Complete(context.Background(), job.ID, "worker-2", nil, now)
	if !errors.Is(err, ports.ErrJobLeaseLost) {
		t.Errorf("Complete() error = %v, want ErrJobLeaseLost", err)
	}
}

func TestFail_RetrySetsQueuedAndRunAt(t *testing.T) {
	r := NewJobRepository()
	now := time.Now().UTC()
	newTestJob(t, r, now)
	claimed, _ := r.ClaimBatch(context.Background(), "worker-1", 10, time.Minute, now)
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
	r := NewJobRepository()
	now := time.Now().UTC()
	newTestJob(t, r, now)
	claimed, _ := r.ClaimBatch(context.Background(), "worker-1", 10, time.Minute, now)
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

	// JobTerminalStateIsImmutable: further Fail/Complete calls must not
	// succeed since the lease was released on the terminal transition.
	if _, err := r.Complete(context.Background(), job.ID, "worker-1", nil, now); !errors.Is(err, ports.ErrJobLeaseLost) {
		t.Errorf("Complete() on terminal Job error = %v, want ErrJobLeaseLost", err)
	}
}

func TestCancel_TerminalReturnsErrJobAlreadyTerminal(t *testing.T) {
	r := NewJobRepository()
	now := time.Now().UTC()
	job := newTestJob(t, r, now)
	claimed, _ := r.ClaimBatch(context.Background(), "worker-1", 10, time.Minute, now)
	if _, err := r.Complete(context.Background(), claimed[0].ID, "worker-1", nil, now); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	_, err := r.Cancel(context.Background(), job.ID, now)
	if !errors.Is(err, ports.ErrJobAlreadyTerminal) {
		t.Errorf("Cancel() error = %v, want ErrJobAlreadyTerminal", err)
	}
}

func TestCancel_FromQueuedSucceeds(t *testing.T) {
	r := NewJobRepository()
	now := time.Now().UTC()
	job := newTestJob(t, r, now)

	got, err := r.Cancel(context.Background(), job.ID, now)
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if got.Status != domain.StatusCanceled {
		t.Errorf("Status = %q, want %q", got.Status, domain.StatusCanceled)
	}
}

func TestGet_NotFound(t *testing.T) {
	r := NewJobRepository()
	if _, err := r.Get(context.Background(), "does-not-exist"); !errors.Is(err, ports.ErrJobNotFound) {
		t.Errorf("Get() error = %v, want ErrJobNotFound", err)
	}
}
