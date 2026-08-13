package ports

import "context"

// IDTokenHintClaims is the verified subset needed by logout and CIBA. Logout
// deliberately ignores ExpiresAt, while CIBA requires a currently valid hint.
type IDTokenHintClaims struct {
	Subject   string
	Audience  string
	Audiences []string
	Sid       string
	ExpiresAt int64
}

// IDTokenHintVerifier verifies an id_token_hint presented to protocol endpoints.
// Implementations must verify the JWT signature against the OP's own
// signing keys and the iss claim against the current issuer, fail-closed on
// any mismatch or unparsable token.
type IDTokenHintVerifier interface {
	VerifyIDTokenHint(ctx context.Context, token string) (*IDTokenHintClaims, error)
}
