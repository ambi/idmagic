package usecases

import (
	"context"
	"time"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
)

// ProjectorDeps holds what ProjectAgentAccessRevoked needs to fan a local
// revocation out to registered Transmit streams.
type ProjectorDeps struct {
	StreamRepo            ssports.SsfStreamRepository
	TransmitterConfigRepo ssports.SsfTransmitterConfigRepository
	DeliveryRepo          ssports.SecurityEventDeliveryRepository
	Signer                ssports.SecurityEventTokenSigner
	// Issuer is the SET `iss` claim (idmagic's own issuer URL).
	Issuer string
}

// initiatingEntityForReason maps IdManagement/SharedSignals' RevocationReason
// to CaepEvent.InitiatingEntity. All of wi-58's current triggers originate
// from an admin console action (kill/disable/unbind, owner disable/delete);
// InboundSecurityEvent is the one case that echoes an externally-received
// signal rather than a local admin decision.
func initiatingEntityForReason(reason ssdomain.RevocationReason) ssdomain.InitiatingEntity {
	if reason == ssdomain.RevocationReasonInboundSecurityEvent {
		return ssdomain.InitiatingEntityPolicy
	}
	return ssdomain.InitiatingEntityAdmin
}

// ProjectAgentAccessRevoked implements EcosystemPropagation (ADR-057
// decision 6): it fans event out to every enabled Transmit SsfStream
// subscribed to session-revoked, builds and signs one SecurityEventToken per
// stream, and enqueues each as a pending SecurityEventDelivery for the
// retry/backoff delivery worker to pick up.
//
// This is deliberately a separate, best-effort step from LocalRevocation
// (AdvanceRevocationEpoch): ecosystem propagation must never block or delay
// the local epoch advance that already made the agent's tokens invalid.
// Callers must compose this at the wiring layer as a non-propagating
// (log-and-continue) reaction to AgentAccessRevoked, not inline inside the
// revocation call itself.
func ProjectAgentAccessRevoked(ctx context.Context, deps ProjectorDeps, event *ssdomain.AgentAccessRevoked) error {
	if deps.StreamRepo == nil {
		return nil
	}
	streams, err := deps.StreamRepo.ListAll(ctx, event.TenantID)
	if err != nil {
		return err
	}
	caepEvent := ssdomain.CaepEvent{
		EventType: ssdomain.CaepEventTypeSessionRevoked,
		Subject: ssdomain.SsfSubject{
			SubjectType: ssdomain.SsfSubjectTypeAgent, TenantID: event.TenantID, PrincipalID: event.AgentID,
		},
		Reason:           &event.Reason,
		EventTimestamp:   event.At,
		InitiatingEntity: initiatingEntityForReason(event.Reason),
	}
	for _, stream := range streams {
		if stream.Direction != ssdomain.SsfStreamDirectionTransmit || !stream.IsEnabled() || !stream.Subscribes(ssdomain.CaepEventTypeSessionRevoked) {
			continue
		}
		if err := deliverToStream(ctx, deps, stream, caepEvent, event.At); err != nil {
			return err
		}
	}
	return nil
}

func deliverToStream(ctx context.Context, deps ProjectorDeps, stream *ssdomain.SsfStream, caepEvent ssdomain.CaepEvent, now time.Time) error {
	config, err := deps.TransmitterConfigRepo.FindByStream(ctx, stream.TenantID, stream.ID)
	if err != nil {
		return err
	}
	if config == nil {
		return nil
	}
	set, err := BuildAndSignSecurityEventToken(ctx, deps.Signer, deps.Issuer, config.Audience, caepEvent)
	if err != nil {
		return err
	}
	id, err := ssdomain.NewSecurityEventDeliveryID()
	if err != nil {
		return err
	}
	delivery := ssdomain.SecurityEventDelivery{
		ID: id, TenantID: stream.TenantID, StreamID: stream.ID, SetJTI: set.JTI, Set: set,
		Status: ssdomain.SecurityEventDeliveryStatusPending, CreatedAt: now,
	}
	if err := delivery.Validate(); err != nil {
		return err
	}
	return deps.DeliveryRepo.Save(ctx, &delivery)
}
