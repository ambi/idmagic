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

func TestRunner_SuccessPath(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	handlers := usecases.NewHandlerRegistry()
	handlers.Register(domain.KindNoopEcho, func(_ context.Context, job *domain.Job) (json.RawMessage, error) {
		return json.RawMessage(`{"echo":true}`), nil
	})
	rec := &eventRecorder{}
	runner := usecases.NewRunner(
		usecases.RunnerConfig{WorkerID: "worker-1", PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute},
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
			WorkerID: "worker-1", PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute,
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
			WorkerID: "worker-1", PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute,
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
		usecases.RunnerConfig{WorkerID: "worker-1", PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute},
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
		usecases.RunnerConfig{WorkerID: "worker-1", PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute, Concurrency: concurrency},
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
		usecases.RunnerConfig{WorkerID: "worker-1", PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute},
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
