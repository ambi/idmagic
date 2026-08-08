package ports

import (
	"context"
	"errors"
	"time"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// Classified verification failures a SecurityEventTokenVerifier
// implementation must return (wrapped is fine) so usecases.ReceiveSecurityEvent
// can map them to the right SecurityEventVerificationResult without
// depending on the adapter's own error types. Any other error (malformed
// token, missing claim, JWKS unavailable) is treated as a signature failure
// (fail-closed catch-all, ADR-057).
var (
	ErrSecurityEventSignatureInvalid = errors.New("security event token: signature invalid")
	ErrSecurityEventIssuerMismatch   = errors.New("security event token: issuer mismatch")
	ErrSecurityEventAudienceMismatch = errors.New("security event token: audience mismatch")
)

// VerifiedSecurityEvent is an inbound Security Event Token's verified
// standard claims plus its raw `events` claim (CAEP-event-specific; the
// caller, not this port, interprets its shape).
type VerifiedSecurityEvent struct {
	JTI      string
	Issuer   string
	Audience []string
	IssuedAt time.Time
	Events   map[string]any
}

// SecurityEventTokenVerifier verifies an inbound SET against a stream's
// SsfReceiverConfig (trusted_issuer, jwks_uri/jwks, accepted_audiences). It
// is a port so the use_cases layer (usecases.ReceiveSecurityEvent) never
// depends on the adapters-layer JWKS-fetch/JWT-verification packages
// directly (mirrors ports.SecurityEventTokenSigner, ADR-057, wi-58).
type SecurityEventTokenVerifier interface {
	Verify(ctx context.Context, config *ssdomain.SsfReceiverConfig, token string) (*VerifiedSecurityEvent, error)
}
