package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerOpensAfterThresholdAndRecovers(t *testing.T) {
	cb := NewCircuitBreaker(Settings{
		Name: "test", FailureThreshold: 0.5, MinRequests: 2, Cooldown: 20 * time.Millisecond,
	})

	boom := errors.New("boom")
	if err := cb.Execute(func() error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("err=%v", err)
	}
	if err := cb.Execute(func() error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("err=%v", err)
	}
	// Two of two requests failed (>= 50% threshold, MinRequests reached) -> open.
	if err := cb.Execute(func() error { return nil }); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("err=%v, want ErrCircuitOpen", err)
	}

	time.Sleep(30 * time.Millisecond) // let the cooldown elapse -> half-open

	// A successful probe in half-open closes the breaker again.
	if err := cb.Execute(func() error { return nil }); err != nil {
		t.Fatalf("half-open probe rejected: %v", err)
	}
	if err := cb.Execute(func() error { return nil }); err != nil {
		t.Fatalf("closed breaker rejected a call: %v", err)
	}
}

func TestCircuitBreakerHalfOpenFailureReopens(t *testing.T) {
	cb := NewCircuitBreaker(Settings{
		Name: "test", FailureThreshold: 0.5, MinRequests: 1, Cooldown: 20 * time.Millisecond,
	})
	boom := errors.New("boom")
	if err := cb.Execute(func() error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("err=%v", err)
	}
	if err := cb.Execute(func() error { return nil }); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("err=%v, want ErrCircuitOpen", err)
	}
	time.Sleep(30 * time.Millisecond)
	// A failing probe in half-open reopens the breaker rather than closing it.
	if err := cb.Execute(func() error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("err=%v", err)
	}
	if err := cb.Execute(func() error { return nil }); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("err=%v, want ErrCircuitOpen after half-open failure", err)
	}
}

func TestCircuitBreakerDefaultsAppliedForZeroSettings(t *testing.T) {
	cb := NewCircuitBreaker(Settings{Name: "defaults"})
	if cb.minRequests != 5 || cb.cooldown != 15*time.Second || cb.failureThreshold != 0.5 {
		t.Fatalf("cb=%+v", cb)
	}
}

func TestCircuitBreakerDefaultsForOutOfRangeThreshold(t *testing.T) {
	cb := NewCircuitBreaker(Settings{Name: "bad-threshold", FailureThreshold: 1.5})
	if cb.failureThreshold != 0.5 {
		t.Fatalf("failureThreshold=%v, want default 0.5", cb.failureThreshold)
	}
}

func TestCircuitBreakerExecuteRecoversAndRepanics(t *testing.T) {
	cb := NewCircuitBreaker(Settings{Name: "panic", MinRequests: 100})
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected the panic to propagate")
		}
	}()
	_ = cb.Execute(func() error { panic("kaboom") })
	t.Fatal("unreachable: Execute must not swallow the panic")
}

func TestCircuitBreakerBeginRequestCompleteIsIdempotent(t *testing.T) {
	// A high MinRequests keeps the breaker Closed across this single completion,
	// so counts.Requests isn't reset by a state transition mid-test.
	cb := NewCircuitBreaker(Settings{Name: "idempotent", MinRequests: 100, FailureThreshold: 0.5})
	complete, err := cb.BeginRequest()
	if err != nil {
		t.Fatal(err)
	}
	complete(errors.New("first"))
	complete(nil) // must be a no-op: sync.Once guards afterRequest
	cb.mu.Lock()
	requests, failures := cb.counts.Requests, cb.counts.Failures
	cb.mu.Unlock()
	if requests != 1 || failures != 1 {
		t.Fatalf("Requests=%d Failures=%d, want 1/1 (second complete() call must be ignored)", requests, failures)
	}
}

func TestRetryWithBackoffSucceedsOnFirstAttempt(t *testing.T) {
	calls := 0
	err := RetryWithBackoff(context.Background(), func() error {
		calls++
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestRetryWithBackoffStopsAfterFirstFailureWhenContextAlreadyDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	boom := errors.New("boom")
	err := RetryWithBackoff(ctx, func() error {
		calls++
		return boom
	})
	if calls != 1 {
		t.Fatalf("calls=%d, want exactly 1 (context already done must stop retrying)", calls)
	}
	if !errors.Is(err, boom) {
		t.Fatalf("err=%v, want the last operation error", err)
	}
}

func TestRetryWithBackoffReturnsContextErrorWhenCanceledDuringWait(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	boom := errors.New("boom")
	err := RetryWithBackoff(ctx, func() error { return boom })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err=%v, want context.DeadlineExceeded", err)
	}
}
