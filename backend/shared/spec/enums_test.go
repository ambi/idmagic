package spec

import (
	"testing"
	"time"
)

func TestZogErrorWrapsIssues(t *testing.T) {
	if err := ZogError(nil); err != nil {
		t.Fatalf("ZogError(nil) = %v, want nil", err)
	}
	// authorizationRequestSchema.Validate rejects a struct missing every
	// required field, giving zogError-shaped issues to exercise the wrapper
	// with real content.
	issues := authorizationRequestSchema.Validate(&struct {
		ID                  string
		State               AuthorizationCodeFlowState
		ClientID            string
		RedirectURI         string
		ResponseType        ResponseType
		CodeChallenge       string
		CodeChallengeMethod CodeChallengeMethod
		MaxAge              *int
		CreatedAt           time.Time
		ExpiresAt           time.Time
	}{})
	if wrapped := ZogError(issues); wrapped == nil {
		t.Fatal("expected ZogError to wrap validation issues into a non-nil error")
	}
}

func TestEnumValidMethods(t *testing.T) {
	t.Run("ClientType", func(t *testing.T) {
		if !ClientPublic.Valid() || !ClientConfidential.Valid() {
			t.Fatal("expected known ClientType values to be valid")
		}
		if ClientType("bogus").Valid() {
			t.Fatal("expected unknown ClientType to be invalid")
		}
	})

	t.Run("GrantType", func(t *testing.T) {
		for _, g := range []GrantType{
			GrantAuthorizationCode, GrantRefreshToken, GrantClientCredentials,
			GrantDeviceCode, GrantTokenExchange, GrantCiba,
		} {
			if !g.Valid() {
				t.Fatalf("expected %q to be valid", g)
			}
		}
		if GrantType("bogus").Valid() {
			t.Fatal("expected unknown GrantType to be invalid")
		}
	})

	t.Run("ResponseType", func(t *testing.T) {
		if !ResponseTypeCode.Valid() {
			t.Fatal("expected code to be valid")
		}
		if ResponseType("token").Valid() {
			t.Fatal("expected unknown ResponseType to be invalid")
		}
	})

	t.Run("CodeChallengeMethod", func(t *testing.T) {
		if !CodeChallengeMethodS256.Valid() {
			t.Fatal("expected S256 to be valid")
		}
		if CodeChallengeMethod("plain").Valid() {
			t.Fatal("expected plain to be invalid")
		}
	})

	t.Run("MfaFactorType", func(t *testing.T) {
		for _, m := range []MfaFactorType{MfaFactorTOTP, MfaFactorWebAuthn, MfaFactorHWK, MfaFactorSWK} {
			if !m.Valid() {
				t.Fatalf("expected %q to be valid", m)
			}
		}
		if MfaFactorType("bogus").Valid() {
			t.Fatal("expected unknown MfaFactorType to be invalid")
		}
	})

	t.Run("AuthenticatorResetTarget", func(t *testing.T) {
		for _, a := range []AuthenticatorResetTarget{
			AuthenticatorResetTotp, AuthenticatorResetWebauthn, AuthenticatorResetRecoveryCode,
		} {
			if !a.Valid() {
				t.Fatalf("expected %q to be valid", a)
			}
		}
		if AuthenticatorResetTarget("bogus").Valid() {
			t.Fatal("expected unknown AuthenticatorResetTarget to be invalid")
		}
	})

	t.Run("WebAuthnTransport", func(t *testing.T) {
		for _, tr := range []WebAuthnTransport{
			WebAuthnTransportUSB, WebAuthnTransportNFC, WebAuthnTransportBLE,
			WebAuthnTransportInternal, WebAuthnTransportHybrid,
		} {
			if !tr.Valid() {
				t.Fatalf("expected %q to be valid", tr)
			}
		}
		if WebAuthnTransport("bogus").Valid() {
			t.Fatal("expected unknown WebAuthnTransport to be invalid")
		}
	})

	t.Run("AuthorizationCodeFlowState", func(t *testing.T) {
		for _, s := range allAuthCodeFlowStates {
			if !s.Valid() {
				t.Fatalf("expected %q to be valid", s)
			}
		}
		if AuthorizationCodeFlowState("bogus").Valid() {
			t.Fatal("expected unknown state to be invalid")
		}
	})

	t.Run("AuthorizationCodeRecordState", func(t *testing.T) {
		for _, s := range []AuthorizationCodeRecordState{
			AuthCodeRecordIssued, AuthCodeRecordRedeemed, AuthCodeRecordExpired,
		} {
			if !s.Valid() {
				t.Fatalf("expected %q to be valid", s)
			}
		}
		if AuthorizationCodeRecordState("bogus").Valid() {
			t.Fatal("expected unknown state to be invalid")
		}
	})

	t.Run("SessionEndReason", func(t *testing.T) {
		for _, r := range []SessionEndReason{
			SessionEndLogout, SessionEndIdle, SessionEndAbsolute, SessionEndSelfRevoke,
			SessionEndAdminRevoke, SessionEndPasswordChange, SessionEndMfaChange, SessionEndOther,
		} {
			if !r.Valid() {
				t.Fatalf("expected %q to be valid", r)
			}
		}
		if SessionEndReason("bogus").Valid() {
			t.Fatal("expected unknown reason to be invalid")
		}
	})

	t.Run("TrustedDeviceRevokeReason", func(t *testing.T) {
		for _, r := range []TrustedDeviceRevokeReason{
			TrustedDeviceSelfRevoke, TrustedDevicePasswordChange, TrustedDeviceMfaChange,
			TrustedDeviceAdminRevoke, TrustedDeviceAccountDisabled, TrustedDeviceSessionRevoke,
		} {
			if !r.Valid() {
				t.Fatalf("expected %q to be valid", r)
			}
		}
		if TrustedDeviceRevokeReason("bogus").Valid() {
			t.Fatal("expected unknown reason to be invalid")
		}
	})

	t.Run("DeviceCodeFlowState", func(t *testing.T) {
		for _, s := range []DeviceCodeFlowState{
			DeviceFlowIssued, DeviceFlowUserCodeEntered, DeviceFlowApproved,
			DeviceFlowDenied, DeviceFlowExchanged, DeviceFlowExpired,
		} {
			if !s.Valid() {
				t.Fatalf("expected %q to be valid", s)
			}
		}
		if DeviceCodeFlowState("bogus").Valid() {
			t.Fatal("expected unknown state to be invalid")
		}
	})

	t.Run("ApprovalRequestState", func(t *testing.T) {
		for _, s := range []ApprovalRequestState{
			ApprovalPending, ApprovalApproved, ApprovalDenied, ApprovalExpired, ApprovalConsumed,
		} {
			if !s.Valid() {
				t.Fatalf("expected %q to be valid", s)
			}
		}
		if ApprovalRequestState("bogus").Valid() {
			t.Fatal("expected unknown state to be invalid")
		}
	})

	t.Run("ResponseMode", func(t *testing.T) {
		if !ResponseModeQuery.Valid() || !ResponseModeFormPost.Valid() {
			t.Fatal("expected query and form_post to be valid")
		}
		if ResponseMode("fragment").Valid() {
			t.Fatal("expected unknown ResponseMode to be invalid")
		}
	})
}

func TestApprovalRequestStateMachine(t *testing.T) {
	allApprovalStates := []ApprovalRequestState{
		ApprovalPending, ApprovalApproved, ApprovalDenied, ApprovalExpired, ApprovalConsumed,
	}
	allApprovalEvents := []ApprovalRequestEvent{
		ApprovalEventApprove, ApprovalEventDeny, ApprovalEventConsume, ApprovalEventExpire,
	}

	t.Run("valid transitions succeed", func(t *testing.T) {
		cases := []struct {
			from  ApprovalRequestState
			event ApprovalRequestEvent
			want  ApprovalRequestState
		}{
			{ApprovalPending, ApprovalEventApprove, ApprovalApproved},
			{ApprovalPending, ApprovalEventDeny, ApprovalDenied},
			{ApprovalPending, ApprovalEventExpire, ApprovalExpired},
			{ApprovalApproved, ApprovalEventConsume, ApprovalConsumed},
			{ApprovalApproved, ApprovalEventExpire, ApprovalExpired},
		}
		for _, tc := range cases {
			got, err := TransitionApprovalRequest(tc.from, tc.event)
			if err != nil {
				t.Fatalf("TransitionApprovalRequest(%q, %q) error: %v", tc.from, tc.event, err)
			}
			if got != tc.want {
				t.Fatalf("TransitionApprovalRequest(%q, %q) = %q, want %q", tc.from, tc.event, got, tc.want)
			}
		}
	})

	t.Run("terminal states accept no events", func(t *testing.T) {
		for _, s := range allApprovalStates {
			if !IsApprovalRequestTerminal(s) {
				continue
			}
			for _, e := range allApprovalEvents {
				if _, err := TransitionApprovalRequest(s, e); err == nil {
					t.Fatalf("terminal state %q accepted event %q", s, e)
				}
			}
		}
	})

	t.Run("non-terminal states are pending only", func(t *testing.T) {
		for _, s := range allApprovalStates {
			want := s == ApprovalDenied || s == ApprovalExpired || s == ApprovalConsumed
			if got := IsApprovalRequestTerminal(s); got != want {
				t.Fatalf("IsApprovalRequestTerminal(%q) = %v, want %v", s, got, want)
			}
		}
	})

	t.Run("unknown transition is rejected", func(t *testing.T) {
		if _, err := TransitionApprovalRequest(ApprovalDenied, ApprovalEventApprove); err == nil {
			t.Fatal("expected error transitioning from a terminal state")
		}
	})
}
