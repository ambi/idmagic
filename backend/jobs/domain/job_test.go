package domain

import (
	"testing"
	"time"
)

// allStatuses and allEvents enumerate the JobLifecycle alphabet so the invariant
// tests below can exhaustively check every (status, event) pair, mirroring
// backend/shared/spec/authorization_code_machine_invariants_test.go.
var (
	allStatuses = []JobStatus{StatusQueued, StatusRunning, StatusSucceeded, StatusFailed, StatusCanceled}
	allEvents   = []JobLifecycleEvent{EventClaim, EventComplete, EventFail, EventRetry, EventCancel}
)

func TestTransitionJobLifecycle_DeclaredTransitions(t *testing.T) {
	tests := []struct {
		from  JobStatus
		event JobLifecycleEvent
		want  JobStatus
	}{
		{StatusQueued, EventClaim, StatusRunning},
		{StatusRunning, EventComplete, StatusSucceeded},
		{StatusRunning, EventFail, StatusFailed},
		{StatusRunning, EventRetry, StatusQueued},
		{StatusQueued, EventCancel, StatusCanceled},
		{StatusRunning, EventCancel, StatusCanceled},
	}
	for _, tt := range tests {
		got, err := TransitionJobLifecycle(tt.from, tt.event)
		if err != nil {
			t.Errorf("TransitionJobLifecycle(%q, %q) unexpected error: %v", tt.from, tt.event, err)
			continue
		}
		if got != tt.want {
			t.Errorf("TransitionJobLifecycle(%q, %q) = %q, want %q", tt.from, tt.event, got, tt.want)
		}
	}
}

func TestTransitionJobLifecycle_InvariantOnlyDeclaredTransitionsSucceed(t *testing.T) {
	declared := map[[2]string]bool{}
	for _, tr := range jobTransitions {
		declared[[2]string{string(tr.From), string(tr.Event)}] = true
	}
	for _, from := range allStatuses {
		for _, event := range allEvents {
			_, err := TransitionJobLifecycle(from, event)
			ok := declared[[2]string{string(from), string(event)}]
			if ok && err != nil {
				t.Errorf("TransitionJobLifecycle(%q, %q) should succeed (declared) but got error: %v", from, event, err)
			}
			if !ok && err == nil {
				t.Errorf("TransitionJobLifecycle(%q, %q) should fail (undeclared) but succeeded", from, event)
			}
		}
	}
}

func TestTransitionJobLifecycle_InvariantTerminalStatesHaveNoOutgoingTransitions(t *testing.T) {
	for _, tr := range jobTransitions {
		if IsJobLifecycleTerminal(tr.From) {
			t.Errorf("terminal status %q has outgoing transition on event %q", tr.From, tr.Event)
		}
	}
}

func TestTransitionJobLifecycle_InvariantDeterministic(t *testing.T) {
	seen := map[[2]string]JobStatus{}
	for _, tr := range jobTransitions {
		key := [2]string{string(tr.From), string(tr.Event)}
		if prev, ok := seen[key]; ok && prev != tr.To {
			t.Errorf("transition (%q, %q) is non-deterministic: %q and %q", tr.From, tr.Event, prev, tr.To)
		}
		seen[key] = tr.To
	}
}

func TestJobStatus_Valid(t *testing.T) {
	for _, s := range allStatuses {
		if !s.Valid() {
			t.Errorf("JobStatus(%q).Valid() = false, want true", s)
		}
	}
	if JobStatus("bogus").Valid() {
		t.Error(`JobStatus("bogus").Valid() = true, want false`)
	}
}

func TestJobKind_Valid(t *testing.T) {
	if !KindNoopEcho.Valid() {
		t.Error("KindNoopEcho.Valid() = false, want true")
	}
	if JobKind("unregistered_kind").Valid() {
		t.Error(`JobKind("unregistered_kind").Valid() = true, want false`)
	}
}

func TestIsJobLifecycleTerminal(t *testing.T) {
	terminal := map[JobStatus]bool{StatusSucceeded: true, StatusFailed: true, StatusCanceled: true}
	for _, s := range allStatuses {
		if got, want := IsJobLifecycleTerminal(s), terminal[s]; got != want {
			t.Errorf("IsJobLifecycleTerminal(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestNextRetryRunAt(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	base := 30 * time.Second
	maxBackoff := 30 * time.Minute
	tests := []struct {
		attempts int
		want     time.Duration
	}{
		{attempts: 0, want: 30 * time.Second},  // clamped to attempts=1
		{attempts: 1, want: 30 * time.Second},  // base * 2^0
		{attempts: 2, want: 1 * time.Minute},   // base * 2^1
		{attempts: 3, want: 2 * time.Minute},   // base * 2^2
		{attempts: 10, want: 30 * time.Minute}, // capped
	}
	for _, tt := range tests {
		got := NextRetryRunAt(now, tt.attempts, base, maxBackoff)
		want := now.Add(tt.want)
		if !got.Equal(want) {
			t.Errorf("NextRetryRunAt(now, %d, ...) = %v, want %v", tt.attempts, got, want)
		}
	}
}

func TestNextRetryRunAt_NeverExceedsCap(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	base := 30 * time.Second
	maxBackoff := 30 * time.Minute
	for attempts := 1; attempts <= 50; attempts++ {
		got := NextRetryRunAt(now, attempts, base, maxBackoff)
		if got.After(now.Add(maxBackoff)) {
			t.Errorf("NextRetryRunAt(now, %d, ...) = %v exceeds cap %v", attempts, got, now.Add(maxBackoff))
		}
	}
}

func TestJobEvents_ImplementDomainEvent(t *testing.T) {
	// Structural check: each event exposes EventType()/OccurredAt() with the
	// signature backend/shared/spec.DomainEvent requires, without importing that
	// package from domain.
	type domainEvent interface {
		EventType() string
		OccurredAt() time.Time
	}
	now := time.Now()
	events := []domainEvent{
		&JobEnqueued{At: now, JobID: "job-1", TenantID: "tenant-a", Kind: KindNoopEcho},
		&JobStarted{At: now, JobID: "job-1", TenantID: "tenant-a", WorkerID: "worker-1", Attempt: 1},
		&JobSucceeded{At: now, JobID: "job-1", TenantID: "tenant-a"},
		&JobFailed{At: now, JobID: "job-1", TenantID: "tenant-a", Attempt: 1, Terminal: false, Error: "boom"},
		&JobRetried{At: now, JobID: "job-1", TenantID: "tenant-a", Attempt: 2, NextRunAt: now},
		&JobCanceled{At: now, JobID: "job-1", TenantID: "tenant-a"},
	}
	wantTypes := []string{"JobEnqueued", "JobStarted", "JobSucceeded", "JobFailed", "JobRetried", "JobCanceled"}
	for i, e := range events {
		if got := e.EventType(); got != wantTypes[i] {
			t.Errorf("events[%d].EventType() = %q, want %q", i, got, wantTypes[i])
		}
		if !e.OccurredAt().Equal(now) {
			t.Errorf("events[%d].OccurredAt() = %v, want %v", i, e.OccurredAt(), now)
		}
	}
}
