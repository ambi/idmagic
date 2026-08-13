package domain_test

import (
	"testing"
	"time"

	approvaldomain "github.com/ambi/idmagic/backend/oauth2/approval/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func newRequest(t *testing.T, state spec.ApprovalRequestState, now time.Time) *approvaldomain.ApprovalRequest {
	t.Helper()
	id, err := approvaldomain.NewApprovalRequestID()
	if err != nil {
		t.Fatalf("NewApprovalRequestID: %v", err)
	}
	return &approvaldomain.ApprovalRequest{
		ID: id, TenantID: "default", ClientID: "agent-app", UserID: "alice",
		Scopes: []string{"openid"}, State: state,
		AuthReqIDHash: approvaldomain.HashAuthReqID("AR1"), IntervalSeconds: 5,
		RequestedAt: now, ExpiresAt: now.Add(approvaldomain.DefaultTTL),
	}
}

// REQ-OAUTH2-042: transitions are one-way and only Approved can reach Consumed.
func TestApprovalRequestTransitionsAreOneWay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		from    spec.ApprovalRequestState
		event   spec.ApprovalRequestEvent
		want    spec.ApprovalRequestState
		wantErr bool
	}{
		{"pending approve", spec.ApprovalPending, spec.ApprovalEventApprove, spec.ApprovalApproved, false},
		{"pending deny", spec.ApprovalPending, spec.ApprovalEventDeny, spec.ApprovalDenied, false},
		{"pending expire", spec.ApprovalPending, spec.ApprovalEventExpire, spec.ApprovalExpired, false},
		{"approved consume", spec.ApprovalApproved, spec.ApprovalEventConsume, spec.ApprovalConsumed, false},
		{"approved expire", spec.ApprovalApproved, spec.ApprovalEventExpire, spec.ApprovalExpired, false},
		{"pending cannot be consumed", spec.ApprovalPending, spec.ApprovalEventConsume, "", true},
		{"denied cannot be approved", spec.ApprovalDenied, spec.ApprovalEventApprove, "", true},
		{"expired cannot be approved", spec.ApprovalExpired, spec.ApprovalEventApprove, "", true},
		{"consumed cannot be consumed twice", spec.ApprovalConsumed, spec.ApprovalEventConsume, "", true},
		{"approved cannot be denied", spec.ApprovalApproved, spec.ApprovalEventDeny, "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := spec.TransitionApprovalRequest(tc.from, tc.event)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("TransitionApprovalRequest(%q, %q) = %q, want error", tc.from, tc.event, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("TransitionApprovalRequest(%q, %q): %v", tc.from, tc.event, err)
			}
			if got != tc.want {
				t.Errorf("TransitionApprovalRequest(%q, %q) = %q, want %q", tc.from, tc.event, got, tc.want)
			}
		})
	}
}

func TestApprovalRequestTerminalStates(t *testing.T) {
	t.Parallel()

	terminal := []spec.ApprovalRequestState{spec.ApprovalDenied, spec.ApprovalExpired, spec.ApprovalConsumed}
	for _, state := range terminal {
		if !spec.IsApprovalRequestTerminal(state) {
			t.Errorf("IsApprovalRequestTerminal(%q) = false, want true", state)
		}
	}
	for _, state := range []spec.ApprovalRequestState{spec.ApprovalPending, spec.ApprovalApproved} {
		if spec.IsApprovalRequestTerminal(state) {
			t.Errorf("IsApprovalRequestTerminal(%q) = true, want false", state)
		}
	}
}

// REQ-OAUTH2-042: expiry includes both the exact deadline and the Expired state.
func TestIsExpired(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rec := newRequest(t, spec.ApprovalPending, now)

	if approvaldomain.IsExpired(rec, now.Add(4*time.Minute)) {
		t.Error("IsExpired before expires_at = true, want false")
	}
	if !approvaldomain.IsExpired(rec, rec.ExpiresAt) {
		t.Error("IsExpired at expires_at = false, want true")
	}
	if !approvaldomain.IsExpired(rec, now.Add(6*time.Minute)) {
		t.Error("IsExpired after expires_at = false, want true")
	}

	expired := newRequest(t, spec.ApprovalExpired, now)
	if !approvaldomain.IsExpired(expired, now) {
		t.Error("IsExpired for an already Expired record = false, want true")
	}
}

// REQ-OAUTH2-041: requested_expiry uses the default only when omitted.
func TestResolveTTL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		requested *int
		want      time.Duration
	}{
		{"unspecified falls back to the default", nil, approvaldomain.DefaultTTL},
		{"shorter is honored", new(60), 60 * time.Second},
		{"maximum is honored", new(600), approvaldomain.MaxTTL},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := approvaldomain.ResolveTTL(tc.requested); got != tc.want {
				t.Errorf("ResolveTTL(%v) = %v, want %v", tc.requested, got, tc.want)
			}
		})
	}
}

// REQ-OAUTH2-041: only polling faster than the interval triggers slow_down.
func TestIsPollTooFast(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rec := newRequest(t, spec.ApprovalPending, now)

	if approvaldomain.IsPollTooFast(rec, now) {
		t.Error("IsPollTooFast on the first poll = true, want false")
	}
	last := now
	rec.LastPolledAt = &last
	if !approvaldomain.IsPollTooFast(rec, now.Add(2*time.Second)) {
		t.Error("IsPollTooFast after 2s with interval 5s = false, want true")
	}
	if approvaldomain.IsPollTooFast(rec, now.Add(5*time.Second)) {
		t.Error("IsPollTooFast after exactly the interval = true, want false")
	}
}

// auth_req_id is a bearer secret and only its digest is retained.
func TestAuthReqIDIsRandomAndHashed(t *testing.T) {
	t.Parallel()

	first, err := approvaldomain.GenerateAuthReqID()
	if err != nil {
		t.Fatalf("GenerateAuthReqID: %v", err)
	}
	second, err := approvaldomain.GenerateAuthReqID()
	if err != nil {
		t.Fatalf("GenerateAuthReqID: %v", err)
	}
	if first == second {
		t.Error("GenerateAuthReqID returned the same value twice")
	}
	hash := approvaldomain.HashAuthReqID(first)
	if hash == first {
		t.Error("HashAuthReqID returned the plaintext auth_req_id")
	}
	if hash != approvaldomain.HashAuthReqID(first) {
		t.Error("HashAuthReqID is not deterministic")
	}
}

func TestApprovalRequestValidate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := newRequest(t, spec.ApprovalPending, now).Validate(); err != nil {
		t.Fatalf("Validate on a well-formed request: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(rec *approvaldomain.ApprovalRequest)
	}{
		{"missing user binding", func(rec *approvaldomain.ApprovalRequest) { rec.UserID = "" }},
		{"missing client", func(rec *approvaldomain.ApprovalRequest) { rec.ClientID = "" }},
		{"missing auth_req_id hash", func(rec *approvaldomain.ApprovalRequest) { rec.AuthReqIDHash = "" }},
		{"state outside the enum", func(rec *approvaldomain.ApprovalRequest) { rec.State = "approved-ish" }},
		{"non-positive poll interval", func(rec *approvaldomain.ApprovalRequest) { rec.IntervalSeconds = 0 }},
		{"id is not a UUID", func(rec *approvaldomain.ApprovalRequest) { rec.ID = "AR1" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rec := newRequest(t, spec.ApprovalPending, now)
			tc.mutate(rec)
			if err := rec.Validate(); err == nil {
				t.Errorf("Validate accepted %s", tc.name)
			}
		})
	}
}
