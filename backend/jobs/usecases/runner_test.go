package usecases_test

// 主要ユースケース追跡: REQ-JOBS-002。

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	memoryjobs "github.com/ambi/idmagic/backend/jobs/db_memory"
	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
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

// TestRunner_OnlyClaimsConfiguredLane: RED for lane isolation
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
	durations      []string
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

func (f *fakeJobsMetrics) RecordJobDuration(_ domain.ExecutionLane, outcome string, _ time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.durations = append(f.durations, outcome)
}

func (f *fakeJobsMetrics) durationOutcomes() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.durations...)
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
	// wi-157: 実行時間は試行ごとに記録する。失敗して再試行された 1 回目も含めて
	// 測らないと、遅いのが滞留なのか処理そのものなのかを分けられない。
	durations := metrics.durationOutcomes()
	if len(durations) != 2 || durations[0] != "failed" || durations[1] != "succeeded" {
		t.Errorf("duration outcomes = %v, want [failed succeeded] (one per attempt)", durations)
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

// bulk レーンの Runner を並行数より多い長時間 Job で埋め、滞留を残したまま
// latency_sensitive レーンの Job を投入する。
//
//spec:covers EX-JOBS-009-01: bulk レーンに並行数を超える Job が滞留したままでも、latency_sensitive レーンの worker は投入された Job を取得して running にし、実行まで進める。
func TestRunner_BulkBacklogDoesNotStarveLatencySensitive(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	handlers := usecases.NewHandlerRegistry()

	// 実在の latency_sensitive の種別 (backchannel_logout_delivery) は oauth2 の Context が
	// 登録する。Jobs のテストはそれに依存せず、同じレーンへ試験用の種別を登録する。
	const latencySensitiveTestKind domain.JobKind = "test_latency_sensitive_delivery"
	domain.RegisterKind(latencySensitiveTestKind, domain.LaneLatencySensitive)

	const bulkConcurrency = 2
	bulkRelease := make(chan struct{}) // held closed until after the assertion below
	handlers.Register(domain.KindUserImportPreview, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		<-bulkRelease
		return json.RawMessage(`{}`), nil
	})
	latencyStarted := make(chan struct{})
	latencyRelease := make(chan struct{})
	handlers.Register(latencySensitiveTestKind, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		close(latencyStarted)
		<-latencyRelease
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
	select {
	case <-latencyStarted:
	case <-time.After(time.Second):
		t.Fatal("the latency_sensitive job was not claimed while the bulk lane was backlogged")
	}
	running, err := repo.Get(context.Background(), latencyJob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if running.Status != domain.StatusRunning {
		t.Fatalf("latency_sensitive job Status = %q, want %q", running.Status, domain.StatusRunning)
	}
	// 取得の時点で bulk レーンの滞留が実在していたこと。
	depths, err := repo.LaneDepths(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range depths {
		if d.Lane == domain.LaneBulk && d.Queued == 0 {
			t.Fatalf("bulk lane has no backlog (%+v); the test did not saturate it", d)
		}
	}
	close(latencyRelease)
	waitForStatus(t, repo, latencyJob.ID, domain.StatusSucceeded, time.Second)

	close(bulkRelease)
	cancel()
	<-bulkDone
	<-latencyDone
}

//spec:covers EX-JOBS-002-01: tenant-a へ投入した noop_echo の Job は queued で始まり、`worker` が取得して running になり、ハンドラーの正常終了で succeeded になって result を保持する。
func TestRunner_DrivesAQueuedJobThroughRunningToSucceeded(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	job := enqueueTestJob(t, repo, domain.DefaultMaxAttempts)
	if job.Status != domain.StatusQueued {
		t.Fatalf("enqueued Status = %q, want %q", job.Status, domain.StatusQueued)
	}
	handlers := usecases.NewHandlerRegistry()
	started := make(chan struct{})
	release := make(chan struct{})
	handlers.Register(domain.KindNoopEcho, func(context.Context, *domain.Job) (json.RawMessage, error) {
		close(started)
		<-release
		return json.RawMessage(`{"echo":true}`), nil
	})
	runner := usecases.NewRunner(
		usecases.RunnerConfig{WorkerID: "worker-1", Lane: domain.LaneDefault, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute},
		usecases.RunnerDeps{Repo: repo, Handlers: handlers},
	)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()
	defer func() { cancel(); <-done }()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("handler never started")
	}
	running, err := repo.Get(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if running.Status != domain.StatusRunning || running.LeaseOwner == nil || *running.LeaseOwner != "worker-1" {
		t.Fatalf("while the handler runs: Status = %q LeaseOwner = %v, want running held by worker-1", running.Status, running.LeaseOwner)
	}
	close(release)
	final := waitForStatus(t, repo, job.ID, domain.StatusSucceeded, 2*time.Second)
	if string(final.Result) != `{"echo":true}` {
		t.Errorf("Result = %s, want the handler's result", final.Result)
	}
}

// ハンドラーは実行コンテキストからテナントを読む。Runner が固定しなければ、
// tenancy.TenantID は既定テナントを返し、ハンドラーは別テナントの範囲で動く。
//
//spec:covers EX-JOBS-006-01: Runner はハンドラーの実行コンテキストのテナントを、取得した Job の tenant_id に固定する。
func TestRunner_BindsTheHandlerContextToTheJobsTenant(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	job := enqueueTestJob(t, repo, domain.DefaultMaxAttempts)
	handlers := usecases.NewHandlerRegistry()
	seen := make(chan string, 1)
	handlers.Register(domain.KindNoopEcho, func(ctx context.Context, _ *domain.Job) (json.RawMessage, error) {
		seen <- tenancy.TenantID(ctx)
		return json.RawMessage(`{}`), nil
	})
	runner := usecases.NewRunner(
		usecases.RunnerConfig{WorkerID: "worker-1", Lane: domain.LaneDefault, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute},
		usecases.RunnerDeps{Repo: repo, Handlers: handlers},
	)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()
	defer func() { cancel(); <-done }()

	select {
	case got := <-seen:
		if got != job.TenantID {
			t.Fatalf("handler context tenant = %q, want the job's tenant %q", got, job.TenantID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler never ran")
	}
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

// TestRunner_SuccessPath_decrementsActiveJobsQuota is a wi-160 T004.8 RED
// test: a Job reaching StatusSucceeded must free its active_jobs quota slot
// so a subsequent Enqueue at the same limit succeeds.
func TestRunner_SuccessPath_decrementsActiveJobsQuota(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	quotaRepo := tenancymemory.NewQuotaRepository()
	limit := 1
	if err := quotaRepo.SetQuota(context.Background(), "tenant-a", &tenancydomain.TenantQuota{ActiveJobs: &limit}); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	handlers := usecases.NewHandlerRegistry()
	handlers.Register(domain.KindNoopEcho, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		return json.RawMessage(`{}`), nil
	})
	runner := usecases.NewRunner(
		usecases.RunnerConfig{WorkerID: "worker-1", Lane: domain.LaneDefault, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute},
		usecases.RunnerDeps{Repo: repo, Handlers: handlers, QuotaRepo: quotaRepo},
	)

	enqueueDeps := usecases.EnqueueDeps{Repo: repo, QuotaRepo: quotaRepo}
	job, err := usecases.Enqueue(context.Background(), enqueueDeps, ports.EnqueueInput{
		TenantID: "tenant-a", Kind: domain.KindNoopEcho, Params: json.RawMessage(`{}`), MaxAttempts: domain.DefaultMaxAttempts,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if _, err := usecases.Enqueue(context.Background(), enqueueDeps, ports.EnqueueInput{
		TenantID: "tenant-a", Kind: domain.KindNoopEcho, Params: json.RawMessage(`{}`), MaxAttempts: domain.DefaultMaxAttempts,
	}, time.Now().UTC()); err == nil {
		t.Fatal("expected second Enqueue to be rejected by active_jobs quota")
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()
	waitForStatus(t, repo, job.ID, domain.StatusSucceeded, 2*time.Second)
	cancel()
	<-done

	if _, err := usecases.Enqueue(context.Background(), enqueueDeps, ports.EnqueueInput{
		TenantID: "tenant-a", Kind: domain.KindNoopEcho, Params: json.RawMessage(`{}`), MaxAttempts: domain.DefaultMaxAttempts,
	}, time.Now().UTC()); err != nil {
		t.Fatalf("expected Enqueue to succeed after completion freed quota, got %v", err)
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

//spec:covers EX-JOBS-005-01: max_attempts=3 の Job が 3 回目も失敗すると attempts が上限に達して failed になりエラーを保持し、その後も取得されずハンドラーは 3 回を超えて呼ばれない。
func TestRunner_DeadLetterAfterMaxAttempts(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	handlers := usecases.NewHandlerRegistry()
	var mu sync.Mutex
	invocations := 0
	handlers.Register(domain.KindNoopEcho, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		mu.Lock()
		invocations++
		mu.Unlock()
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

	job := enqueueTestJob(t, repo, 3)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()

	final := waitForStatus(t, repo, job.ID, domain.StatusFailed, 2*time.Second)
	if final.Attempts != 3 {
		t.Errorf("Attempts = %d, want 3 (MaxAttempts)", final.Attempts)
	}
	if final.Error == nil || *final.Error != "permanent failure" {
		t.Errorf("Error = %v, want %q", final.Error, "permanent failure")
	}
	if !domain.IsJobLifecycleTerminal(final.Status) {
		t.Error("dead-lettered Job should be terminal")
	}

	// 配信不能の後も取得を繰り返させ、Job が二度と running にならないことを見る。
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	after, err := repo.Get(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	got := invocations
	mu.Unlock()
	if after.Status != domain.StatusFailed || after.Attempts != 3 || got != 3 {
		t.Errorf("after dead-letter: Status = %q Attempts = %d handler invocations = %d, want failed, 3, 3", after.Status, after.Attempts, got)
	}

	counts := rec.typeCounts()
	// 3 回とも失敗する。最初の 2 回は再試行 (JobFailed + JobRetried)、3 回目は
	// MaxAttempts を使い切る (JobFailed だけ、terminal=true)。
	if counts["JobFailed"] != 3 || counts["JobRetried"] != 2 || counts["JobStarted"] != 3 || counts["JobSucceeded"] != 0 {
		t.Errorf("event counts = %+v, want JobStarted=3 JobFailed=3 JobRetried=2 JobSucceeded=0", counts)
	}
}

// TestRunner_DeadLetter_decrementsActiveJobsQuota is a wi-160 T004.8 RED
// test: a Job dead-lettered (StatusFailed, terminal) after exhausting
// MaxAttempts must free its active_jobs quota slot. A non-terminal retry must
// not free it prematurely.
func TestRunner_DeadLetter_decrementsActiveJobsQuota(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	quotaRepo := tenancymemory.NewQuotaRepository()
	limit := 1
	if err := quotaRepo.SetQuota(context.Background(), "tenant-a", &tenancydomain.TenantQuota{ActiveJobs: &limit}); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	handlers := usecases.NewHandlerRegistry()
	handlers.Register(domain.KindNoopEcho, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		return nil, errors.New("permanent failure")
	})
	runner := usecases.NewRunner(
		usecases.RunnerConfig{
			WorkerID: "worker-1", Lane: domain.LaneDefault, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute,
			BackoffBase: 5 * time.Millisecond, BackoffCap: 5 * time.Millisecond,
		},
		usecases.RunnerDeps{Repo: repo, Handlers: handlers, QuotaRepo: quotaRepo},
	)

	enqueueDeps := usecases.EnqueueDeps{Repo: repo, QuotaRepo: quotaRepo}
	job, err := usecases.Enqueue(context.Background(), enqueueDeps, ports.EnqueueInput{
		TenantID: "tenant-a", Kind: domain.KindNoopEcho, Params: json.RawMessage(`{}`), MaxAttempts: 2,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()
	waitForStatus(t, repo, job.ID, domain.StatusFailed, 2*time.Second)
	cancel()
	<-done

	if _, err := usecases.Enqueue(context.Background(), enqueueDeps, ports.EnqueueInput{
		TenantID: "tenant-a", Kind: domain.KindNoopEcho, Params: json.RawMessage(`{}`), MaxAttempts: domain.DefaultMaxAttempts,
	}, time.Now().UTC()); err != nil {
		t.Fatalf("expected Enqueue to succeed after dead-letter freed quota, got %v", err)
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

// TestRunner_ReclaimsAfterWorkerCrash is the wi-42 T012 smoke test: enqueue
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
