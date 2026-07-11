package spec

import (
	"testing"
)

func TestTransitionAuthorizationCodeFlowHappyPath(t *testing.T) {
	steps := []struct {
		from  AuthorizationCodeFlowState
		event AuthorizationCodeFlowEvent
		to    AuthorizationCodeFlowState
	}{
		{AuthFlowReceived, EventStartAuthentication, AuthFlowAuthenticationPending},
		{AuthFlowAuthenticationPending, EventAuthenticateUser, AuthFlowAuthenticated},
		{AuthFlowAuthenticated, EventRequestConsent, AuthFlowConsentPending},
		{AuthFlowConsentPending, EventGrantConsent, AuthFlowConsented},
		{AuthFlowConsented, EventIssueCode, AuthFlowCodeIssued},
		{AuthFlowCodeIssued, EventRedeemCode, AuthFlowExchanged},
	}
	for _, s := range steps {
		got, err := TransitionAuthorizationCodeFlow(s.from, s.event)
		if err != nil {
			t.Fatalf("transition %q on %q failed: %v", s.from, s.event, err)
		}
		if got != s.to {
			t.Fatalf("transition %q on %q: got %q, want %q", s.from, s.event, got, s.to)
		}
	}
}

func TestTransitionAuthorizationCodeFlowRejectsInvalidEdge(t *testing.T) {
	if _, err := TransitionAuthorizationCodeFlow(AuthFlowReceived, EventRedeemCode); err == nil {
		t.Fatal("expected error: cannot redeem from Received")
	}
}

func TestTransitionAuthorizationCodeRecordRejectsDoubleRedeem(t *testing.T) {
	mid, err := TransitionAuthorizationCodeRecord(AuthCodeRecordIssued, RecordEventRedeem)
	if err != nil {
		t.Fatalf("first redeem failed: %v", err)
	}
	if _, err := TransitionAuthorizationCodeRecord(mid, RecordEventRedeem); err == nil {
		t.Fatal("expected error: cannot redeem already-redeemed code")
	}
}
