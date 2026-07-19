package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	memoryjobs "github.com/ambi/idmagic/backend/jobs/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// eventRecorder is a concurrency-safe spec.DomainEvent sink: Runner invokes
// Emit from worker goroutines, so tests must not append to a plain slice
// without synchronization.
type eventRecorder struct {
	mu     sync.Mutex
	events []spec.DomainEvent
}

func (r *eventRecorder) record(e spec.DomainEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}

func (r *eventRecorder) typeCounts() map[string]int {
	r.mu.Lock()
	defer r.mu.Unlock()
	counts := map[string]int{}
	for _, e := range r.events {
		counts[e.EventType()]++
	}
	return counts
}

func waitForStatus(t *testing.T, repo *memoryjobs.JobRepository, jobID string, want domain.JobStatus, timeout time.Duration) *domain.Job {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		job, err := repo.Get(context.Background(), jobID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if job.Status == want {
			return job
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("job %q did not reach status %q within %v", jobID, want, timeout)
	return nil
}

func enqueueTestJob(t *testing.T, repo *memoryjobs.JobRepository, maxAttempts int) *domain.Job {
	t.Helper()
	deps := usecases.EnqueueDeps{Repo: repo}
	job, err := usecases.Enqueue(context.Background(), deps, ports.EnqueueInput{
		TenantID:    "tenant-a",
		Kind:        domain.KindNoopEcho,
		Params:      json.RawMessage(`{}`),
		MaxAttempts: maxAttempts,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	return job
}

// TestRunner_OnlyClaimsConfiguredLane: RED for ADR-129 lane isolation
// (spec/contexts/jobs.yaml scenario "bulk laneのbacklogが滞留してもlatency_sensitiveジョブは専用実行枠でclaimされる"):
// a Runner configured for LaneLatencySensitive must never claim a Job whose
// JobKind resolves to a different lane, even when that Job is due and the
// Runner has free concurrency.
func TestRunner_OnlyClaimsConfiguredLane(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	handlers := usecases.NewHandlerRegistry()
	handlers.Register(domain.KindUserImportPreview, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		return json.RawMessage(`{}`), nil
	})

	bulkJob := enqueueTestJobWithKind(t, repo, domain.KindUserImportPreview, domain.DefaultMaxAttempts)

	runner := usecases.NewRunner(
		usecases.RunnerConfig{WorkerID: "worker-1", Lane: domain.LaneLatencySensitive, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute},
		usecases.RunnerDeps{Repo: repo, Handlers: handlers},
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()

	// The bulk-lane Job must stay queued: a latency_sensitive-only Runner
	// polls repeatedly but must never claim it.
	time.Sleep(50 * time.Millisecond)
	still, err := repo.Get(context.Background(), bulkJob.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if still.Status != domain.StatusQueued {
		t.Errorf("bulk-lane job Status = %q after latency_sensitive-only Runner polled, want %q (lane isolation violated)", still.Status, domain.StatusQueued)
	}

	cancel()
	<-done
}

// fakeJobsMetrics records every JobsMetrics call for assertions (wi-261 T006).
type fakeJobsMetrics struct {
	mu             sync.Mutex
	claimLatencies []time.Duration
	outcomes       []string
	retries        int
}

func (f *fakeJobsMetrics) RecordJobClaimLatency(_ domain.ExecutionLane, latency time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.claimLatencies = append(f.claimLatencies, latency)
}

func (f *fakeJobsMetrics) RecordJobOutcome(_ domain.ExecutionLane, outcome string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.outcomes = append(f.outcomes, outcome)
}

func (f *fakeJobsMetrics) RecordJobRetry(domain.ExecutionLane) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.retries++
}

func (f *fakeJobsMetrics) snapshot() (outcomes []string, retries int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.outcomes...), f.retries
}

// TestRunner_RecordsMetrics: RED for wi-261 T006 — a Runner with
// RunnerDeps.Metrics set must record claim latency for every claimed Job and
// the terminal outcome ("succeeded"/"failed"), plus a retry count for each
// non-terminal failure, matching the usecases.JobsMetrics contract.
func TestRunner_RecordsMetrics(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	handlers := usecases.NewHandlerRegistry()
	var mu sync.Mutex
	attempts := 0
	handlers.Register(domain.KindNoopEcho, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		mu.Lock()
		attempts++
		n := attempts
		mu.Unlock()
		if n == 1 {
			return nil, errors.New("transient failure")
		}
		return json.RawMessage(`{}`), nil
	})
	metrics := &fakeJobsMetrics{}
	runner := usecases.NewRunner(
		usecases.RunnerConfig{
			WorkerID: "worker-1", Lane: domain.LaneDefault, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute,
			BackoffBase: 5 * time.Millisecond, BackoffCap: 5 * time.Millisecond,
		},
		usecases.RunnerDeps{Repo: repo, Handlers: handlers, Metrics: metrics},
	)

	job := enqueueTestJob(t, repo, 3)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()

	waitForStatus(t, repo, job.ID, domain.StatusSucceeded, 2*time.Second)
	cancel()
	<-done

	outcomes, retries := metrics.snapshot()
	if retries != 1 {
		t.Errorf("retries = %d, want 1", retries)
	}
	if len(outcomes) != 1 || outcomes[0] != "succeeded" {
		t.Errorf("outcomes = %v, want [succeeded]", outcomes)
	}
	metrics.mu.Lock()
	gotLatencies := len(metrics.claimLatencies)
	metrics.mu.Unlock()
	if gotLatencies != 2 {
		t.Errorf("claim latency recordings = %d, want 2 (one per claim attempt)", gotLatencies)
	}
}

func enqueueTestJobWithKind(t *testing.T, repo *memoryjobs.JobRepository, kind domain.JobKind, maxAttempts int) *domain.Job {
	t.Helper()
	deps := usecases.EnqueueDeps{Repo: repo}
	job, err := usecases.Enqueue(context.Background(), deps, ports.EnqueueInput{
		TenantID:    "tenant-a",
		Kind:        kind,
		Params:      json.RawMessage(`{}`),
		MaxAttempts: maxAttempts,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	return job
}

// TestRunner_BulkBacklogDoesNotStarveLatencySensitive is the wi-261 T008
// integration test for the work item's core Motivation and the scenario
// spec/contexts/jobs.yaml "bulk laneのbacklogが滞留してもlatency_sensitiveジョブは専用実行枠でclaimされる":
// a bulk-lane Runner saturated with more long-running jobs than its
// concurrency (so a backlog persists queued behind them) must never delay a
// concurrently running latency_sensitive-lane Runner's independent claim and
// completion of a latency_sensitive Job. This is the automated counterpart
// to the work item's manual verification step (reserved capacity under a
// saturated bulk backlog).
func TestRunner_BulkBacklogDoesNotStarveLatencySensitive(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	handlers := usecases.NewHandlerRegistry()

	// No built-in JobKind is registered to LaneLatencySensitive yet in this
	// binary (backchannel_logout_delivery, ADR-129's real assignment, is
	// wi-257's still-pending handler); register a test-only kind so this
	// test exercises the same lane a real latency-sensitive JobKind would.
	const latencySensitiveTestKind domain.JobKind = "test_latency_sensitive_delivery"
	domain.RegisterKind(latencySensitiveTestKind, domain.LaneLatencySensitive)

	const bulkConcurrency = 2
	bulkRelease := make(chan struct{}) // held closed until after the assertion below
	handlers.Register(domain.KindUserImportPreview, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		<-bulkRelease
		return json.RawMessage(`{}`), nil
	})
	handlers.Register(latencySensitiveTestKind, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		return json.RawMessage(`{"echo":true}`), nil
	})

	// Saturate the bulk lane: more queued jobs than bulkConcurrency, so a
	// backlog remains queued behind the in-flight ones for the test's
	// duration (bulkRelease stays closed).
	const bulkBacklogSize = bulkConcurrency + 3
	for range bulkBacklogSize {
		enqueueTestJobWithKind(t, repo, domain.KindUserImportPreview, domain.DefaultMaxAttempts)
	}

	bulkRunner := usecases.NewRunner(
		usecases.RunnerConfig{WorkerID: "bulk-worker", Lane: domain.LaneBulk, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute, Concurrency: bulkConcurrency},
		usecases.RunnerDeps{Repo: repo, Handlers: handlers},
	)
	latencyRunner := usecases.NewRunner(
		usecases.RunnerConfig{WorkerID: "latency-worker", Lane: domain.LaneLatencySensitive, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute},
		usecases.RunnerDeps{Repo: repo, Handlers: handlers},
	)

	ctx, cancel := context.WithCancel(context.Background())
	bulkDone := make(chan error, 1)
	latencyDone := make(chan error, 1)
	go func() { bulkDone <- bulkRunner.Run(ctx) }()
	go func() { latencyDone <- latencyRunner.Run(ctx) }()

	// Give the bulk lane time to actually saturate its concurrency before
	// the latency_sensitive job is enqueued, so the backlog is real.
	time.Sleep(30 * time.Millisecond)

	latencyJob := enqueueTestJobWithKind(t, repo, latencySensitiveTestKind, domain.DefaultMaxAttempts)
	waitForStatus(t, repo, latencyJob.ID, domain.StatusSucceeded, time.Second)

	close(bulkRelease)
	cancel()
	<-bulkDone
	<-latencyDone
}

func TestRunner_SuccessPath(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	handlers := usecases.NewHandlerRegistry()
	handlers.Register(domain.KindNoopEcho, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		return json.RawMessage(`{"echo":true}`), nil
	})
	rec := &eventRecorder{}
	runner := usecases.NewRunner(
		usecases.RunnerConfig{WorkerID: "worker-1", Lane: domain.LaneDefault, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute},
		usecases.RunnerDeps{Repo: repo, Handlers: handlers, Emit: rec.record},
	)

	job := enqueueTestJob(t, repo, domain.DefaultMaxAttempts)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()

	final := waitForStatus(t, repo, job.ID, domain.StatusSucceeded, 2*time.Second)
	if string(final.Result) != `{"echo":true}` {
		t.Errorf("Result = %s, want echo payload", final.Result)
	}

	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Errorf("Run() error = %v, want context.Canceled", err)
	}

	counts := rec.typeCounts()
	if counts["JobStarted"] != 1 || counts["JobSucceeded"] != 1 {
		t.Errorf("event counts = %+v, want JobStarted=1 JobSucceeded=1", counts)
	}
}

func TestRunner_RetryThenSucceed(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	handlers := usecases.NewHandlerRegistry()
	var mu sync.Mutex
	attempts := 0
	handlers.Register(domain.KindNoopEcho, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		mu.Lock()
		attempts++
		n := attempts
		mu.Unlock()
		if n == 1 {
			return nil, errors.New("transient failure")
		}
		return json.RawMessage(`{}`), nil
	})
	rec := &eventRecorder{}
	runner := usecases.NewRunner(
		usecases.RunnerConfig{
			WorkerID: "worker-1", Lane: domain.LaneDefault, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute,
			BackoffBase: 5 * time.Millisecond, BackoffCap: 5 * time.Millisecond,
		},
		usecases.RunnerDeps{Repo: repo, Handlers: handlers, Emit: rec.record},
	)

	job := enqueueTestJob(t, repo, 3)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()

	final := waitForStatus(t, repo, job.ID, domain.StatusSucceeded, 2*time.Second)
	if final.Attempts != 2 {
		t.Errorf("Attempts = %d, want 2 (one failure, one success)", final.Attempts)
	}

	cancel()
	<-done

	counts := rec.typeCounts()
	if counts["JobFailed"] != 1 || counts["JobRetried"] != 1 || counts["JobSucceeded"] != 1 {
		t.Errorf("event counts = %+v, want JobFailed=1 JobRetried=1 JobSucceeded=1", counts)
	}
}

func TestRunner_DeadLetterAfterMaxAttempts(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	handlers := usecases.NewHandlerRegistry()
	handlers.Register(domain.KindNoopEcho, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		return nil, errors.New("permanent failure")
	})
	rec := &eventRecorder{}
	runner := usecases.NewRunner(
		usecases.RunnerConfig{
			WorkerID: "worker-1", Lane: domain.LaneDefault, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute,
			BackoffBase: 5 * time.Millisecond, BackoffCap: 5 * time.Millisecond,
		},
		usecases.RunnerDeps{Repo: repo, Handlers: handlers, Emit: rec.record},
	)

	job := enqueueTestJob(t, repo, 2)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()

	final := waitForStatus(t, repo, job.ID, domain.StatusFailed, 2*time.Second)
	if final.Attempts != 2 {
		t.Errorf("Attempts = %d, want 2 (MaxAttempts)", final.Attempts)
	}
	if final.Error == nil || *final.Error != "permanent failure" {
		t.Errorf("Error = %v, want %q", final.Error, "permanent failure")
	}
	if !domain.IsJobLifecycleTerminal(final.Status) {
		t.Error("dead-lettered Job should be terminal")
	}

	cancel()
	<-done

	counts := rec.typeCounts()
	// Both attempts fail: the first is a retry (JobFailed + JobRetried), the
	// second exhausts MaxAttempts (JobFailed only, terminal=true).
	if counts["JobFailed"] != 2 || counts["JobRetried"] != 1 || counts["JobSucceeded"] != 0 {
		t.Errorf("event counts = %+v, want JobFailed=2 JobRetried=1 JobSucceeded=0", counts)
	}
}

func TestRunner_UnregisteredHandlerDeadLetters(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	handlers := usecases.NewHandlerRegistry() // nothing registered
	runner := usecases.NewRunner(
		usecases.RunnerConfig{WorkerID: "worker-1", Lane: domain.LaneDefault, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute},
		usecases.RunnerDeps{Repo: repo, Handlers: handlers},
	)

	job := enqueueTestJob(t, repo, 1)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()

	final := waitForStatus(t, repo, job.ID, domain.StatusFailed, 2*time.Second)
	if final.Error == nil {
		t.Fatal("Error is nil, want ErrHandlerNotRegistered message")
	}

	cancel()
	<-done
}

func TestRunner_ConcurrencyLimit(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	handlers := usecases.NewHandlerRegistry()

	const concurrency = 2
	var mu sync.Mutex
	current, maxSeen := 0, 0
	handlers.Register(domain.KindNoopEcho, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		mu.Lock()
		current++
		if current > maxSeen {
			maxSeen = current
		}
		mu.Unlock()

		time.Sleep(30 * time.Millisecond)

		mu.Lock()
		current--
		mu.Unlock()
		return json.RawMessage(`{}`), nil
	})

	runner := usecases.NewRunner(
		usecases.RunnerConfig{WorkerID: "worker-1", Lane: domain.LaneDefault, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute, Concurrency: concurrency},
		usecases.RunnerDeps{Repo: repo, Handlers: handlers},
	)

	const numJobs = 6
	jobs := make([]*domain.Job, numJobs)
	for i := range numJobs {
		jobs[i] = enqueueTestJob(t, repo, domain.DefaultMaxAttempts)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()

	for _, job := range jobs {
		waitForStatus(t, repo, job.ID, domain.StatusSucceeded, 3*time.Second)
	}

	cancel()
	<-done

	mu.Lock()
	got := maxSeen
	mu.Unlock()
	if got > concurrency {
		t.Errorf("max concurrent handler executions = %d, want <= %d", got, concurrency)
	}
	if got < concurrency {
		t.Errorf("max concurrent handler executions = %d, want == %d (concurrency should be exercised)", got, concurrency)
	}
}

func TestRunner_DrainWaitsForInFlight(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	handlers := usecases.NewHandlerRegistry()
	started := make(chan struct{})
	release := make(chan struct{})
	handlers.Register(domain.KindNoopEcho, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		close(started)
		<-release
		return json.RawMessage(`{}`), nil
	})

	runner := usecases.NewRunner(
		usecases.RunnerConfig{WorkerID: "worker-1", Lane: domain.LaneDefault, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute},
		usecases.RunnerDeps{Repo: repo, Handlers: handlers},
	)
	job := enqueueTestJob(t, repo, domain.DefaultMaxAttempts)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("handler never started")
	}

	cancel() // begin drain while the handler is still in flight

	select {
	case <-done:
		t.Fatal("Run() returned before the in-flight handler finished (drain did not wait)")
	case <-time.After(50 * time.Millisecond):
	}

	close(release) // let the handler finish

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Run() error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return after the in-flight handler finished")
	}

	waitForStatus(t, repo, job.ID, domain.StatusSucceeded, time.Second)
}

// TestRunner_ReclaimsAfterWorkerCrash is the wi-126 T012 smoke test: enqueue
// a no-op/echo Job, have "worker-1" claim it and then go silent forever
// (crash, never heartbeating/completing/failing again), and confirm a second
// worker reclaims it once the lease expires and drives it to Succeeded.
//
// worker-1's claim is simulated with a direct ClaimBatch call rather than a
// full Runner, because Runner.execute deliberately runs on a context
// detached from Run's ctx (so drain doesn't abort in-flight jobs) -- its
// heartbeat goroutine is tied to the handler goroutine's own lifecycle, not
// to anything a test could cancel from outside within a single process. A
// real crash only stops heartbeating because the whole process dies; a bare
// ClaimBatch call reproduces exactly that end state (leased, never
// heartbeated again) without needing a second OS process.
func TestRunner_ReclaimsAfterWorkerCrash(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	job := enqueueTestJob(t, repo, domain.DefaultMaxAttempts)

	leaseDuration := 20 * time.Millisecond
	claimed, err := repo.ClaimBatch(context.Background(), "worker-1", domain.LaneDefault, 1, leaseDuration, time.Now().UTC())
	if err != nil || len(claimed) != 1 {
		t.Fatalf("worker-1 ClaimBatch() = %v, %v", claimed, err)
	}
	if got := claimed[0].Status; got != domain.StatusRunning {
		t.Fatalf("worker-1 claimed job Status = %q, want %q", got, domain.StatusRunning)
	}
	// worker-1 is now abandoned: no further Heartbeat/Complete/Fail calls.

	handlers := usecases.NewHandlerRegistry()
	handlers.Register(domain.KindNoopEcho, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		return json.RawMessage(`{"reclaimed":true}`), nil
	})
	runner2 := usecases.NewRunner(
		usecases.RunnerConfig{WorkerID: "worker-2", Lane: domain.LaneDefault, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute},
		usecases.RunnerDeps{Repo: repo, Handlers: handlers},
	)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner2.Run(ctx) }()

	final := waitForStatus(t, repo, job.ID, domain.StatusSucceeded, 2*time.Second)
	if final.Attempts != 2 {
		t.Errorf("Attempts = %d, want 2 (worker-1's crashed attempt + worker-2's reclaim)", final.Attempts)
	}
	if got, want := string(final.Result), `{"reclaimed":true}`; got != want {
		t.Errorf("Result = %s, want %s", got, want)
	}
	if final.LeaseOwner != nil {
		t.Errorf("LeaseOwner = %v, want nil (released on completion)", *final.LeaseOwner)
	}

	cancel()
	<-done
}
