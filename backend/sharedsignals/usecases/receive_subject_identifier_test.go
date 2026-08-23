package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	agentmodel "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	ssusecases "github.com/ambi/idmagic/backend/sharedsignals/usecases"
)

const caepSessionRevokedURI = "https://schemas.openid.net/secevent/caep/event-type/session-revoked"

// subjectIdentifierEvent wraps an RFC 9493 Subject Identifier as the sole
// `subject` member of the sole CAEP event claim.
func subjectIdentifierEvent(jti string, subjectID map[string]any) *ssports.VerifiedSecurityEvent {
	return &ssports.VerifiedSecurityEvent{
		JTI: jti, Issuer: "https://transmitter.example", Audience: []string{"aud"}, IssuedAt: time.Now().UTC(),
		Events: map[string]any{caepSessionRevokedURI: map[string]any{"subject": subjectID}},
	}
}

// TestReceiveSecurityEvent_Rfc9493SubjectIdentifiers — RED: REQ-SHAREDSIGNALS-010
// (docs/contexts/sharedsignals/scenarios.md)。外部の transmitter が RFC 9493 の
// Subject Identifier (`iss_sub` / `opaque`) で送った SET でも主体を解決する。
// テナントは受信ストリームが属するテナントで決まり、識別子は Agent の識別子か、
// Agent に束縛済みの OAuth2Client の識別子として解決する。
func TestReceiveSecurityEvent_Rfc9493SubjectIdentifiers(t *testing.T) {
	const agentID = "agent_1"
	const boundClientID = "client_1"

	seed := func(t *testing.T, verified *ssports.VerifiedSecurityEvent) receiveTestDeps {
		t.Helper()
		d := newReceiveTestDeps(t, &fakeVerifier{result: verified})
		seedReceiveStream(t, d, ssdomain.SsfStreamStatusEnabled)
		now := time.Now().UTC()
		if err := d.agentRepo.Save(context.Background(), &agentmodel.Agent{
			ID: agentID, TenantID: receiveTestTenantID, Name: agentID,
			Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "owner_1", Status: idmdomain.AgentStatusActive,
			CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("seed agent: %v", err)
		}
		if _, err := d.agentRepo.AddBinding(context.Background(), &agentmodel.AgentCredentialBinding{
			AgentID: agentID, ClientID: boundClientID, CreatedAt: now,
		}); err != nil {
			t.Fatalf("seed binding: %v", err)
		}
		return d
	}

	assertAccepted := func(t *testing.T, d receiveTestDeps) {
		t.Helper()
		epoch, err := d.epochRepo.FindByAgent(context.Background(), receiveTestTenantID, agentID)
		if err != nil {
			t.Fatal(err)
		}
		if epoch == nil {
			t.Fatal("expected the agent's revocation epoch to advance")
		}
		if epoch.Reason != ssdomain.RevocationReasonInboundSecurityEvent {
			t.Fatalf("reason = %q, want %q", epoch.Reason, ssdomain.RevocationReasonInboundSecurityEvent)
		}
		var received bool
		for _, event := range *d.events {
			if _, ok := event.(*ssdomain.SecurityEventReceived); ok {
				received = true
			}
		}
		if !received {
			t.Fatalf("expected SecurityEventReceived, got %+v", *d.events)
		}
	}

	accepted := []struct {
		name      string
		subjectID map[string]any
	}{
		{
			name:      "iss_sub naming the agent",
			subjectID: map[string]any{"format": "iss_sub", "iss": "https://transmitter.example", "sub": agentID},
		},
		{
			name:      "iss_sub naming the bound client",
			subjectID: map[string]any{"format": "iss_sub", "iss": "https://transmitter.example", "sub": boundClientID},
		},
		{
			name:      "opaque naming the agent",
			subjectID: map[string]any{"format": "opaque", "id": agentID},
		},
	}
	for _, tc := range accepted {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now().UTC()
			d := seed(t, subjectIdentifierEvent("jti-"+tc.name, tc.subjectID))
			if err := ssusecases.ReceiveSecurityEvent(receiveTestCtx(), d.deps, receiveTestStreamID, "token", now); err != nil {
				t.Fatalf("expected the SET to be accepted: %v", err)
			}
			assertAccepted(t, d)
		})
	}

	rejected := []struct {
		name      string
		subjectID map[string]any
	}{
		{
			name:      "iss_sub whose iss is not the stream's trusted issuer",
			subjectID: map[string]any{"format": "iss_sub", "iss": "https://elsewhere.example", "sub": agentID},
		},
		{
			name:      "a format idmagic does not interpret",
			subjectID: map[string]any{"format": "email", "email": "owner@example.com"},
		},
		{
			name:      "an identifier matching no agent and no bound client",
			subjectID: map[string]any{"format": "opaque", "id": "unknown"},
		},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now().UTC()
			d := seed(t, subjectIdentifierEvent("jti-"+tc.name, tc.subjectID))
			err := ssusecases.ReceiveSecurityEvent(receiveTestCtx(), d.deps, receiveTestStreamID, "token", now)
			if !errors.Is(err, ssusecases.ErrSecurityEventRejected) {
				t.Fatalf("expected ErrSecurityEventRejected, got %v", err)
			}
			epoch, findErr := d.epochRepo.FindByAgent(context.Background(), receiveTestTenantID, agentID)
			if findErr != nil {
				t.Fatal(findErr)
			}
			if epoch != nil {
				t.Fatal("expected no revocation epoch to advance for a rejected SET")
			}
			var rejectedEvent *ssdomain.SecurityEventRejected
			for _, event := range *d.events {
				if e, ok := event.(*ssdomain.SecurityEventRejected); ok {
					rejectedEvent = e
				}
			}
			if rejectedEvent == nil || rejectedEvent.VerificationResult != ssdomain.SecurityEventVerificationRejectedSubjectUnresolved {
				t.Fatalf("expected SecurityEventRejected(rejected_subject_unresolved), got %+v", *d.events)
			}
		})
	}

	// idmagic 自身の transmitter が使う独自形式は引き続き受理する (共存)。
	t.Run("idmagic's own wire format still resolves", func(t *testing.T) {
		now := time.Now().UTC()
		d := seed(t, validAgentEvent("jti-native", receiveTestTenantID, agentID))
		if err := ssusecases.ReceiveSecurityEvent(receiveTestCtx(), d.deps, receiveTestStreamID, "token", now); err != nil {
			t.Fatalf("expected the native format to keep working: %v", err)
		}
		assertAccepted(t, d)
	})
}
