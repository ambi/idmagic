package usecases

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
)

// caepEventTypeURIs maps CaepEventType to the CAEP event-type URI used as
// the key of the RFC 8417 `events` claim.
var caepEventTypeURIs = map[ssdomain.CaepEventType]string{
	ssdomain.CaepEventTypeSessionRevoked:       "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
	ssdomain.CaepEventTypeTokenClaimsChange:    "https://schemas.openid.net/secevent/caep/event-type/token-claims-change",
	ssdomain.CaepEventTypeCredentialChange:     "https://schemas.openid.net/secevent/caep/event-type/credential-change",
	ssdomain.CaepEventTypeAssuranceLevelChange: "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change",
}

func caepEventTypeURI(t ssdomain.CaepEventType) string {
	if uri, ok := caepEventTypeURIs[t]; ok {
		return uri
	}
	return string(t)
}

// BuildAndSignSecurityEventToken builds a RFC 8417 Security Event Token for
// event and signs it via signer, which reuses SigningKeys' existing
// rotation/JWKS distribution instead of introducing separate SET key
// material (ADR-057 decisions 3/7). signer is a port
// (ports.SecurityEventTokenSigner): the actual JWT signing package lives in
// the adapters layer (sign_jose), so this use_cases-layer function never
// imports it directly.
func BuildAndSignSecurityEventToken(ctx context.Context, signer ssports.SecurityEventTokenSigner, issuer, audience string, event ssdomain.CaepEvent) (ssdomain.SecurityEventToken, error) {
	jti, err := spec.NewUUIDv4()
	if err != nil {
		return ssdomain.SecurityEventToken{}, err
	}
	now := time.Now().UTC()

	eventClaims := map[string]any{
		"subject": map[string]any{
			"subject_type": event.Subject.SubjectType,
			"tenant_id":    event.Subject.TenantID,
			"principal_id": event.Subject.PrincipalID,
		},
		"event_timestamp":   event.EventTimestamp.Unix(),
		"initiating_entity": string(event.InitiatingEntity),
	}
	if event.Reason != nil {
		eventClaims["reason"] = string(*event.Reason)
	}
	claims := map[string]any{
		"iss":    issuer,
		"jti":    jti,
		"iat":    now.Unix(),
		"aud":    audience,
		"events": map[string]any{caepEventTypeURI(event.EventType): eventClaims},
	}
	compact, err := signer.Sign(ctx, event.Subject.TenantID, claims)
	if err != nil {
		return ssdomain.SecurityEventToken{}, err
	}

	set := ssdomain.SecurityEventToken{
		JTI: jti, Issuer: issuer, Audience: audience, IssuedAt: now, Event: event, Compact: compact,
	}
	if err := set.Validate(); err != nil {
		return ssdomain.SecurityEventToken{}, err
	}
	return set, nil
}
