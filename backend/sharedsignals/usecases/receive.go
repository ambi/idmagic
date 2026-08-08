package usecases

import (
	"context"
	"errors"
	"time"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	"github.com/ambi/idmagic/backend/tenancy"
)

// ErrSecurityEventRejected is returned for any ReceiveSecurityEvent rejection
// (SCL SecurityEventRejectedError): stream not enabled for receiving,
// signature/claims invalid, replayed, or an unresolvable subject. The
// specific reason is recorded in a ReceivedSecurityEvent row and a
// SecurityEventRejected event, not distinguished at this error's call site
// (matches the SCL interface's single declared error).
var ErrSecurityEventRejected = errors.New("security event token rejected")

// ReceiveDeps holds what ReceiveSecurityEvent needs.
type ReceiveDeps struct {
	StreamRepo         ssports.SsfStreamRepository
	ReceiverConfigRepo ssports.SsfReceiverConfigRepository
	ReceivedEventRepo  ssports.ReceivedSecurityEventRepository
	EpochRepo          ssports.AgentRevocationEpochRepository
	AgentRepo          agentports.AgentRepository
	Verifier           ssports.SecurityEventTokenVerifier
	Emit               func(spec.DomainEvent) error
}

// ReceiveSecurityEvent implements the SCL interface ReceiveSecurityEvent
// (spec/contexts/sharedsignals.yaml): it verifies an inbound SET against the
// stream's SsfReceiverConfig, and on success reflects it as LocalRevocation
// (AdvanceRevocationEpoch, reason=InboundSecurityEvent) — external-origin
// signals converge onto the same fail-closed revocation path as idmagic's
// own kill-switch (ADR-057 decision 5). Every requires clause is
// fail-closed: any failure rejects without reflecting the event.
func ReceiveSecurityEvent(ctx context.Context, deps ReceiveDeps, streamID, token string, now time.Time) error {
	tenantID := tenancy.TenantID(ctx)

	// requires: ssf_receiver_stream_enabled(tenant_id, stream_id)
	stream, err := deps.StreamRepo.FindByID(ctx, tenantID, streamID)
	if err != nil {
		return err
	}
	if stream == nil || stream.Direction != ssdomain.SsfStreamDirectionReceive || !stream.IsEnabled() {
		return ErrSecurityEventRejected
	}
	config, err := deps.ReceiverConfigRepo.FindByStream(ctx, tenantID, streamID)
	if err != nil {
		return err
	}
	if config == nil {
		return ErrSecurityEventRejected
	}

	// requires: security_event_signature_and_claims_valid(...)
	verified, err := deps.Verifier.Verify(ctx, config, token)
	if err != nil {
		return rejectEvent(ctx, deps, tenantID, streamID, "", now, verificationResultFor(err))
	}

	// requires: !security_event_replayed(...)
	replayed, err := deps.ReceivedEventRepo.ExistsByJTI(ctx, tenantID, streamID, verified.JTI)
	if err != nil {
		return err
	}
	if replayed {
		return rejectEvent(ctx, deps, tenantID, streamID, verified.JTI, now, ssdomain.SecurityEventVerificationRejectedReplay)
	}

	// requires: security_event_subject_resolves_to_tenant_local_principal(...)
	eventType, subject, ok := extractCaepEventAndSubject(verified.Events)
	if !ok || subject.SubjectType != ssdomain.SsfSubjectTypeAgent || subject.TenantID != tenantID {
		return rejectEvent(ctx, deps, tenantID, streamID, verified.JTI, now, ssdomain.SecurityEventVerificationRejectedSubjectUnresolved)
	}
	agent, err := deps.AgentRepo.FindByID(ctx, tenantID, subject.PrincipalID)
	if err != nil {
		return err
	}
	if agent == nil {
		return rejectEvent(ctx, deps, tenantID, streamID, verified.JTI, now, ssdomain.SecurityEventVerificationRejectedSubjectUnresolved)
	}

	// Accepted: record, reflect as LocalRevocation, emit.
	recordID, err := ssdomain.NewReceivedSecurityEventID()
	if err != nil {
		return err
	}
	record := ssdomain.ReceivedSecurityEvent{
		ID: recordID, TenantID: tenantID, StreamID: streamID, SetJTI: verified.JTI, EventType: eventType,
		Subject: subject, VerificationResult: ssdomain.SecurityEventVerificationAccepted, ReceivedAt: now, ReflectedAt: &now,
	}
	if err := record.Validate(); err != nil {
		return err
	}
	if err := deps.ReceivedEventRepo.Save(ctx, &record); err != nil {
		return err
	}
	if err := AdvanceRevocationEpoch(ctx, RevocationDeps{EpochRepo: deps.EpochRepo, Emit: deps.Emit}, tenantID, []string{agent.ID}, ssdomain.RevocationReasonInboundSecurityEvent, &recordID, now); err != nil {
		return err
	}
	return emit(deps.Emit, &ssdomain.SecurityEventReceived{At: now, TenantID: tenantID, StreamID: streamID, SetJTI: verified.JTI, CaepType: eventType})
}

func verificationResultFor(err error) ssdomain.SecurityEventVerificationResult {
	switch {
	case errors.Is(err, ssports.ErrSecurityEventIssuerMismatch):
		return ssdomain.SecurityEventVerificationRejectedUnknownIssuer
	case errors.Is(err, ssports.ErrSecurityEventAudienceMismatch):
		return ssdomain.SecurityEventVerificationRejectedAudience
	default:
		return ssdomain.SecurityEventVerificationRejectedSignature
	}
}

// rejectEvent records a ReceivedSecurityEvent (best-effort audit trail; a
// recording failure must not mask the rejection itself) and emits
// SecurityEventRejected, then returns ErrSecurityEventRejected.
func rejectEvent(ctx context.Context, deps ReceiveDeps, tenantID, streamID, setJTI string, now time.Time, result ssdomain.SecurityEventVerificationResult) error {
	if setJTI != "" {
		id, err := ssdomain.NewReceivedSecurityEventID()
		if err == nil {
			_ = deps.ReceivedEventRepo.Save(ctx, &ssdomain.ReceivedSecurityEvent{
				ID: id, TenantID: tenantID, StreamID: streamID, SetJTI: setJTI, EventType: ssdomain.CaepEventTypeSessionRevoked,
				Subject:            ssdomain.SsfSubject{SubjectType: ssdomain.SsfSubjectTypeAgent, TenantID: tenantID, PrincipalID: "unresolved"},
				VerificationResult: result, ReceivedAt: now,
			})
		}
	}
	streamIDCopy := streamID
	_ = emit(deps.Emit, &ssdomain.SecurityEventRejected{At: now, TenantID: tenantID, StreamID: &streamIDCopy, VerificationResult: result})
	return ErrSecurityEventRejected
}

// extractCaepEventAndSubject reads the sole `events` claim entry idmagic's
// own transmitter produces (BuildAndSignSecurityEventToken): one CAEP
// event-type URI mapped to a claims object carrying a `subject`
// (subject_type/tenant_id/principal_id). Full RFC 9493 Subject Identifiers
// interop is out of scope (ADR-057 keeps the wire format idmagic-defined
// until a concrete external transmitter requires otherwise).
func extractCaepEventAndSubject(events map[string]any) (ssdomain.CaepEventType, ssdomain.SsfSubject, bool) {
	for uri, raw := range events {
		eventType := caepEventTypeFromURI(uri)
		claims, ok := raw.(map[string]any)
		if !ok {
			return "", ssdomain.SsfSubject{}, false
		}
		subjectClaims, ok := claims["subject"].(map[string]any)
		if !ok {
			return "", ssdomain.SsfSubject{}, false
		}
		subjectType, _ := subjectClaims["subject_type"].(string)
		subjectTenantID, _ := subjectClaims["tenant_id"].(string)
		principalID, _ := subjectClaims["principal_id"].(string)
		return eventType, ssdomain.SsfSubject{SubjectType: subjectType, TenantID: subjectTenantID, PrincipalID: principalID}, true
	}
	return "", ssdomain.SsfSubject{}, false
}
