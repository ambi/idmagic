package ports

import "context"

// SecurityEventTokenSigner signs RFC 8417 claims into a compact PS256 JWT
// using the given tenant's active signing key. It is a port so the
// use_cases layer (usecases.BuildAndSignSecurityEventToken) never depends
// on the adapters-layer JWT signing package directly (Clean Architecture
// layering, enforced by `mise run check-boundaries`); the composition root
// injects an implementation that wraps SigningKeys' KeyStore + the shared
// PS256 signer (decisions 3/7: reuse existing key management).
type SecurityEventTokenSigner interface {
	Sign(ctx context.Context, tenantID string, claims map[string]any) (compact string, err error)
}
