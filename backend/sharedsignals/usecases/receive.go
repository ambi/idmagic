package usecases

import (
	"context"
	"errors"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
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
// own kill-switch (decision 5). Every requires clause is
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
	eventType, subject, ok := extractCaepEventAndSubject(verified.Events, config, tenantID)
	if !ok || subject.SubjectType != ssdomain.SsfSubjectTypeAgent || subject.TenantID != tenantID {
		return rejectEvent(ctx, deps, tenantID, streamID, verified.JTI, now, ssdomain.SecurityEventVerificationRejectedSubjectUnresolved)
	}
	agent, err := resolveAgentPrincipal(ctx, deps, tenantID, subject.PrincipalID)
	if err != nil {
		return err
	}
	if agent == nil {
		return rejectEvent(ctx, deps, tenantID, streamID, verified.JTI, now, ssdomain.SecurityEventVerificationRejectedSubjectUnresolved)
	}
	// 解決できた Agent の識別子で記録する。RFC 9493 形式は束縛先クライアントの
	// 識別子を名乗ることがあり、その場合の principal_id は Agent のものに正規化する。
	subject.PrincipalID = agent.ID

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

// extractCaepEventAndSubject reads the sole `events` claim entry: one CAEP
// event-type URI mapped to a claims object carrying a `subject`. Two shapes
// are accepted, told apart by the presence of a `format` member
// (RFC9493-SUBID-FORMAT):
//
//   - idmagic's own transmitter (BuildAndSignSecurityEventToken) writes
//     subject_type/tenant_id/principal_id and names its own tenant.
//   - an external transmitter writes an RFC 9493 Subject Identifier, which
//     carries no tenant of its own; the receiving stream's tenant supplies it.
func extractCaepEventAndSubject(events map[string]any, config *ssdomain.SsfReceiverConfig, tenantID string) (ssdomain.CaepEventType, ssdomain.SsfSubject, bool) {
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
		if format, isSubjectIdentifier := subjectClaims["format"].(string); isSubjectIdentifier {
			subject, resolved := subjectFromIdentifier(format, subjectClaims, config, tenantID)
			return eventType, subject, resolved
		}
		subjectType, _ := subjectClaims["subject_type"].(string)
		subjectTenantID, _ := subjectClaims["tenant_id"].(string)
		principalID, _ := subjectClaims["principal_id"].(string)
		return eventType, ssdomain.SsfSubject{SubjectType: subjectType, TenantID: subjectTenantID, PrincipalID: principalID}, true
	}
	return "", ssdomain.SsfSubject{}, false
}

// subjectFromIdentifier maps an RFC 9493 Subject Identifier onto an Agent
// subject in the receiving stream's tenant. Only `iss_sub` and `opaque` are
// interpreted; every other format (`email`, `phone_number`, `did`, `uri`,
// `account`, `aliases`) names something this context cannot revoke — Agent
// revocation is what SharedSignals owns today — so it is fail-closed rejected
// rather than guessed at.
//
// `iss_sub` additionally pins `iss` to the stream's registered trusted_issuer
// (RFC9493-SUBID-ISS-SUB): a correctly signed SET may still name a subject in
// some other issuer's namespace, and resolving that against local identifiers
// would let one trusted transmitter revoke on another's behalf. `opaque`
// carries no issuer of its own and takes the stream's, which is the same
// namespace by construction.
func subjectFromIdentifier(format string, claims map[string]any, config *ssdomain.SsfReceiverConfig, tenantID string) (ssdomain.SsfSubject, bool) {
	var identifier string
	switch format {
	case "iss_sub":
		issuer, _ := claims["iss"].(string)
		if config == nil || issuer != config.TrustedIssuer {
			return ssdomain.SsfSubject{}, false
		}
		identifier, _ = claims["sub"].(string)
	case "opaque":
		identifier, _ = claims["id"].(string)
	default:
		return ssdomain.SsfSubject{}, false
	}
	if identifier == "" {
		return ssdomain.SsfSubject{}, false
	}
	return ssdomain.SsfSubject{
		SubjectType: ssdomain.SsfSubjectTypeAgent, TenantID: tenantID, PrincipalID: identifier,
	}, true
}

// resolveAgentPrincipal resolves an inbound subject's identifier to an Agent:
// first as the Agent's own id, then as the id of an OAuth2Client bound to one.
// External transmitters see idmagic's agents through the tokens they present,
// whose `sub` for client_credentials is the client_id, so both identifiers
// legitimately name the same principal from outside.
func resolveAgentPrincipal(ctx context.Context, deps ReceiveDeps, tenantID, identifier string) (*agentdomain.Agent, error) {
	agent, err := deps.AgentRepo.FindByID(ctx, tenantID, identifier)
	if err != nil || agent != nil {
		return agent, err
	}
	return deps.AgentRepo.FindByClientID(ctx, tenantID, identifier)
}
