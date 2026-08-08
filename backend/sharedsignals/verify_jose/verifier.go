// Package verify_jose implements ports.SecurityEventTokenVerifier: inbound
// Security Event Token verification (ADR-057 §ReceiveSecurityEvent). It is
// the only place in SharedSignals that imports backend/shared/security/tokens_jose,
// keeping the usecases layer free of adapter dependencies (mirrors
// workloadidentity/verification_jose and sharedsignals/sign_jose).
package verify_jose

import (
	"context"
	"errors"

	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
)

type Verifier struct {
	JWKResolver *tokens_jose.JWKResolver
}

var _ ssports.SecurityEventTokenVerifier = (*Verifier)(nil)

func (v *Verifier) Verify(ctx context.Context, config *ssdomain.SsfReceiverConfig, token string) (*ssports.VerifiedSecurityEvent, error) {
	keys, err := v.JWKResolver.ResolveJWKSSource(ctx, config.JWKSURI, config.JWKS)
	if err != nil {
		return nil, err
	}
	claims, err := tokens_jose.VerifySecurityEventToken(token, keys, config.TrustedIssuer, config.AcceptedAudiences)
	if err != nil {
		return nil, translateError(err)
	}
	return &ssports.VerifiedSecurityEvent{
		JTI: claims.JTI, Issuer: claims.Issuer, Audience: claims.Audience, IssuedAt: claims.IssuedAt, Events: eventsClaim(claims.Payload),
	}, nil
}

func eventsClaim(payload map[string]any) map[string]any {
	events, _ := payload["events"].(map[string]any)
	return events
}

func translateError(err error) error {
	switch {
	case errors.Is(err, tokens_jose.ErrSecurityEventTokenIssuerMismatch):
		return ssports.ErrSecurityEventIssuerMismatch
	case errors.Is(err, tokens_jose.ErrSecurityEventTokenAudienceMismatch):
		return ssports.ErrSecurityEventAudienceMismatch
	default:
		// Malformed token, invalid signature, missing claim: all fail-closed
		// as a signature failure (ADR-057).
		return ssports.ErrSecurityEventSignatureInvalid
	}
}
