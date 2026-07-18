package ports

import "context"

// IDTokenHintClaims is the subset of ID Token claims needed to resolve an
// RP-Initiated Logout target (ADR-127). Exp is intentionally not surfaced:
// RPs commonly present an already-expired ID Token as id_token_hint at
// logout time, and ADR-127 decision 4 does not check it.
type IDTokenHintClaims struct {
	Subject  string
	Audience string
	Sid      string
}

// IDTokenHintVerifier verifies an id_token_hint presented to /end_session.
// Implementations must verify the JWT signature against the OP's own
// signing keys and the iss claim against the current issuer, fail-closed on
// any mismatch or unparsable token.
type IDTokenHintVerifier interface {
	VerifyIDTokenHint(ctx context.Context, token string) (*IDTokenHintClaims, error)
}
