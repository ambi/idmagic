// Package tokens_jose: 外部 SSF transmitter からの Security Event Token (SET,
// RFC 8417) の検証 (§ReceiveSecurityEvent, [[wi-58-continuous-access-evaluation-agent-revocation]])。
package tokens_jose

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrSecurityEventTokenMalformed        = errors.New("security_event_token: malformed JWT")
	ErrSecurityEventTokenInvalidSignature = errors.New("security_event_token: signature invalid")
	ErrSecurityEventTokenIssuerMismatch   = errors.New("security_event_token: iss does not match the registered trusted_issuer")
	ErrSecurityEventTokenAudienceMismatch = errors.New("security_event_token: aud does not match any accepted audience")
	ErrSecurityEventTokenMissingClaim     = errors.New("security_event_token: required claim missing")
)

// SecurityEventTokenClaims is a verified inbound SET's standard claims plus
// its full decoded payload (the caller extracts the `events` claim itself —
// its shape is CAEP-event-specific, not something this package interprets).
type SecurityEventTokenClaims struct {
	Issuer   string
	Audience []string
	JTI      string
	IssuedAt time.Time
	Payload  map[string]any
}

// VerifySecurityEventToken verifies an inbound Security Event Token (
// fail-closed): alg must be PS256/ES256/RS256 (external transmitters have a
// wider algorithm choice than this app's own self-issued JWTs, same
// allowance as VerifyWorkloadSVID), iss must equal expectedIssuer exactly,
// aud must contain at least one accepted audience, and jti/iat must be
// present (RFC 8417 requires jti for replay detection; SETs are point-in-time
// notifications and do not carry exp, unlike VerifyWorkloadSVID's access
// tokens).
func VerifySecurityEventToken(
	token string,
	jwks []map[string]any,
	expectedIssuer string,
	acceptedAudiences []string,
) (*SecurityEventTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrSecurityEventTokenMalformed
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSecurityEventTokenMalformed, err)
	}
	var header map[string]any
	if err := json.Unmarshal(hb, &header); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSecurityEventTokenMalformed, err)
	}
	alg, _ := header["alg"].(string)
	if alg != "PS256" && alg != "ES256" && alg != "RS256" {
		return nil, fmt.Errorf("%w: alg %q not allowed", ErrSecurityEventTokenInvalidSignature, alg)
	}
	kid, _ := header["kid"].(string)

	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSecurityEventTokenMalformed, err)
	}
	var payload map[string]any
	if err := json.Unmarshal(pb, &payload); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSecurityEventTokenMalformed, err)
	}

	pub, err := pickJWK(jwks, kid)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSecurityEventTokenInvalidSignature, err)
	}
	if err := verifyJWTSignature(parts, alg, pub); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSecurityEventTokenInvalidSignature, err)
	}

	iss, _ := payload["iss"].(string)
	if iss == "" || iss != expectedIssuer {
		return nil, ErrSecurityEventTokenIssuerMismatch
	}
	if !verifyAudience(payload["aud"], acceptedAudiences) {
		return nil, ErrSecurityEventTokenAudienceMismatch
	}
	jti, _ := payload["jti"].(string)
	if jti == "" {
		return nil, fmt.Errorf("%w: jti", ErrSecurityEventTokenMissingClaim)
	}
	iatF, ok := payload["iat"].(float64)
	if !ok {
		return nil, fmt.Errorf("%w: iat", ErrSecurityEventTokenMissingClaim)
	}

	return &SecurityEventTokenClaims{
		Issuer: iss, Audience: audienceStrings(payload["aud"]), JTI: jti,
		IssuedAt: time.Unix(int64(iatF), 0).UTC(), Payload: payload,
	}, nil
}
