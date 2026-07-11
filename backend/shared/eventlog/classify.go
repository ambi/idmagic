package eventlog

import (
	"encoding/json"
	"fmt"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// classification is the ADR-094 DomainEvent -> Classification catalog.
// wi-184 T004 makes this exhaustive: TestAllDomainEventTypesAreClassified
// (classify_coverage_test.go) scans every DomainEvent implementation under
// backend/ and fails if any EventType() string has no entry here, so a new
// DomainEvent added anywhere in the codebase is a hard CI failure until its
// routing is decided — not a silent gap.
//
// The mapping below preserves today's actual routing rather than inventing
// new judgment calls: an event already reaching Kafka via the outbox
// eventTopics map (backend/oauth2/adapters/persistence/postgres/outbox.go)
// is public_integration; everything else is audit_only, matching that it
// already reaches the audit trail today through the unconditional
// AuditEventRepo.Append in bootstrap/server.go's fire-and-forget emit.
// wi-184 T005+ can promote entries to public_integration deliberately, and
// a future pass can carve out true high-frequency telemetry — neither is
// decided here.
var classification = map[string]Classification{
	// public_integration: currently routed to Kafka via the outbox eventTopics
	// map (backend/oauth2/adapters/persistence/postgres/outbox.go).
	"AccessTokenIssued":                ClassificationPublicIntegration,
	"AgentCredentialBound":             ClassificationPublicIntegration,
	"AgentCredentialUnbound":           ClassificationPublicIntegration,
	"AgentDeleted":                     ClassificationPublicIntegration,
	"AgentDisabled":                    ClassificationPublicIntegration,
	"AgentEnabled":                     ClassificationPublicIntegration,
	"AgentKilled":                      ClassificationPublicIntegration,
	"AgentOwnerChanged":                ClassificationPublicIntegration,
	"AgentRegistered":                  ClassificationPublicIntegration,
	"AgentUpdated":                     ClassificationPublicIntegration,
	"AuthenticationFailed":             ClassificationPublicIntegration,
	"AuthorizationCodeIssued":          ClassificationPublicIntegration,
	"AuthorizationCodeRedeemed":        ClassificationPublicIntegration,
	"ClientRegistered":                 ClassificationPublicIntegration,
	"ConsentGranted":                   ClassificationPublicIntegration,
	"ConsentRevoked":                   ClassificationPublicIntegration,
	"DeviceAuthorizationApproved":      ClassificationPublicIntegration,
	"DeviceAuthorizationDenied":        ClassificationPublicIntegration,
	"DeviceAuthorizationRequested":     ClassificationPublicIntegration,
	"GroupCreated":                     ClassificationPublicIntegration,
	"GroupDeleted":                     ClassificationPublicIntegration,
	"GroupMemberAdded":                 ClassificationPublicIntegration,
	"GroupMemberRemoved":               ClassificationPublicIntegration,
	"GroupUpdated":                     ClassificationPublicIntegration,
	"LoginThrottled":                   ClassificationPublicIntegration,
	"PARStored":                        ClassificationPublicIntegration,
	"PasswordChanged":                  ClassificationPublicIntegration,
	"RefreshTokenIssued":               ClassificationPublicIntegration,
	"RefreshTokenReuseDetected":        ClassificationPublicIntegration,
	"RefreshTokenRotated":              ClassificationPublicIntegration,
	"SigningKeyRotated":                ClassificationPublicIntegration,
	"TenantCreated":                    ClassificationPublicIntegration,
	"TenantDisabled":                   ClassificationPublicIntegration,
	"TenantEnabled":                    ClassificationPublicIntegration,
	"TenantUpdated":                    ClassificationPublicIntegration,
	"TenantUserAttributeSchemaUpdated": ClassificationPublicIntegration,
	"TokenExchanged":                   ClassificationPublicIntegration,
	"TokenExchangeRejected":            ClassificationPublicIntegration,
	"TokenIntrospected":                ClassificationPublicIntegration,
	"TokenRevoked":                     ClassificationPublicIntegration,
	"UserAuthenticated":                ClassificationPublicIntegration,
	// audit_only: not currently routed to Kafka; still reaches the audit trail
	// today via the fire-and-forget AuditEventRepo.Append in bootstrap/server.go.
	"AdminOAuth2ClientCreated":         ClassificationAuditOnly,
	"AdminOAuth2ClientDeleted":         ClassificationAuditOnly,
	"AdminOAuth2ClientUpdated":         ClassificationAuditOnly,
	"AppAccessDeniedByPolicy":          ClassificationAuditOnly,
	"ApplicationAssigned":              ClassificationAuditOnly,
	"ApplicationCategoryCreated":       ClassificationAuditOnly,
	"ApplicationCategoryDeleted":       ClassificationAuditOnly,
	"ApplicationCategoryUpdated":       ClassificationAuditOnly,
	"ApplicationCreated":               ClassificationAuditOnly,
	"ApplicationDeleted":               ClassificationAuditOnly,
	"ApplicationIconUpdated":           ClassificationAuditOnly,
	"ApplicationUnassigned":            ClassificationAuditOnly,
	"ApplicationUpdated":               ClassificationAuditOnly,
	"AppSignInPolicyUpdated":           ClassificationAuditOnly,
	"AppStepUpRequired":                ClassificationAuditOnly,
	"AuthenticationEventAggregated":    ClassificationAuditOnly,
	"AuthenticationStepCompleted":      ClassificationAuditOnly,
	"AuthenticationStepFailed":         ClassificationAuditOnly,
	"AuthorizationDetailsConsented":    ClassificationAuditOnly,
	"AuthorizationDetailsRejected":     ClassificationAuditOnly,
	"AuthorizationDetailsRequested":    ClassificationAuditOnly,
	"BackupCodeConsumed":               ClassificationAuditOnly,
	"EmailChanged":                     ClassificationAuditOnly,
	"EmailChangeRequested":             ClassificationAuditOnly,
	"EmailSent":                        ClassificationAuditOnly,
	"EntraFederationConfigured":        ClassificationAuditOnly,
	"FederatedAuthenticated":           ClassificationAuditOnly,
	"FederationLinked":                 ClassificationAuditOnly,
	"FederationUnlinked":               ClassificationAuditOnly,
	"MfaChallengeFailed":               ClassificationAuditOnly,
	"MfaChallengeIssued":               ClassificationAuditOnly,
	"MfaChallengeSucceeded":            ClassificationAuditOnly,
	"MfaFactorEnrolled":                ClassificationAuditOnly,
	"MfaFactorRemoved":                 ClassificationAuditOnly,
	"PasswordResetRequested":           ClassificationAuditOnly,
	"ProtocolBindingAttached":          ClassificationAuditOnly,
	"ProtocolBindingDetached":          ClassificationAuditOnly,
	"RecoveryCodesGenerated":           ClassificationAuditOnly,
	"RecoveryCodesRevoked":             ClassificationAuditOnly,
	"SamlLogout":                       ClassificationAuditOnly,
	"SamlSignInIssued":                 ClassificationAuditOnly,
	"SamlSignInRejected":               ClassificationAuditOnly,
	"SessionEnded":                     ClassificationAuditOnly,
	"SessionImpersonationEnded":        ClassificationAuditOnly,
	"SessionImpersonationStarted":      ClassificationAuditOnly,
	"SessionRefreshed":                 ClassificationAuditOnly,
	"SessionStarted":                   ClassificationAuditOnly,
	"StepUpCompleted":                  ClassificationAuditOnly,
	"StepUpRequested":                  ClassificationAuditOnly,
	"TenantDefaultSignInPolicyUpdated": ClassificationAuditOnly,
	"UserCreated":                      ClassificationAuditOnly,
	"UserDeleted":                      ClassificationAuditOnly,
	"UserDisabled":                     ClassificationAuditOnly,
	"UserEnabled":                      ClassificationAuditOnly,
	"UserRequiredActionCleared":        ClassificationAuditOnly,
	"UserRequiredActionSet":            ClassificationAuditOnly,
	"UserRestored":                     ClassificationAuditOnly,
	"UserSoftDeleted":                  ClassificationAuditOnly,
	"UserUpdated":                      ClassificationAuditOnly,
	"WebAuthnCredentialRegistered":     ClassificationAuditOnly,
	"WebAuthnCredentialRemoved":        ClassificationAuditOnly,
	"WsFedSignInIssued":                ClassificationAuditOnly,
	"WsFedSignInRejected":              ClassificationAuditOnly,
	"WsFedSignOut":                     ClassificationAuditOnly,
	"WsTrustTokenIssued":               ClassificationAuditOnly,
	"WsTrustTokenRejected":             ClassificationAuditOnly,
}

// ToRecord converts event into a Record ready for Recorder.Append. eventID
// is the caller-supplied idempotency key (a fresh UUIDv4 per event);
// correlationID ties every event recorded within one HTTP request/command
// together.
//
// tenantId / actorUserId / targetUserId (or userId) are read generically
// from the event's own JSON wire form (spec.MarshalDomainEvent) using the
// same convention backend/bootstrap/audit_event_record.go already relies on
// for tenantId: every DomainEvent that carries an actor/subject tags them
// with these JSON keys, so this does not need a type switch per event.
//
// Classification is the one thing that cannot be inferred generically: it
// is an explicit routing decision, not a structural fact of the payload. An
// event type missing from the classification map fails closed with an
// error instead of guessing, so an unmigrated command is a hard failure to
// notice (and roll back) rather than a silently misrouted event.
func ToRecord(event spec.DomainEvent, eventID, correlationID string) (Record, error) {
	wire, err := spec.MarshalDomainEvent(event)
	if err != nil {
		return Record{}, err
	}
	var payload map[string]any
	if err := json.Unmarshal(wire, &payload); err != nil {
		return Record{}, err
	}
	class, ok := classification[event.EventType()]
	if !ok {
		return Record{}, fmt.Errorf(
			"eventlog: %s is not classified (wi-184 T004 tracks the full DomainEvent catalog)", event.EventType(),
		)
	}
	rec := Record{
		EventID:        eventID,
		Type:           event.EventType(),
		Classification: class,
		CorrelationID:  correlationID,
		OccurredAt:     event.OccurredAt(),
		Payload:        payload,
	}
	if tenantID, ok := payload["tenantId"].(string); ok {
		rec.TenantID = tenantID
	}
	if actor, ok := payload["actorUserId"].(string); ok {
		rec.Actor = actor
	}
	if target, ok := payload["targetUserId"].(string); ok {
		rec.Subject = target
	} else if userID, ok := payload["userId"].(string); ok {
		rec.Subject = userID
	}
	return rec, nil
}
