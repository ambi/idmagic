package ports

import (
	"context"
	"time"

	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

// WorkloadSVIDVerifier verifies an external workload attestation token
// against a resolved JWKS source. Concrete implementation lives in
// the adapters layer (backend/workloadidentity/verification_jose) so the
// usecases layer never imports the shared JOSE/crypto adapter directly.
type WorkloadSVIDVerifier interface {
	// Verify resolves signing keys via fetchJWKS (cached with last-known-good
	// fallback, keyed by the bundle ID), then checks the JOSE header, signature,
	// iss, aud, exp and TTL by the bundle's profile, issuer, audiences and max TTL.
	// The SPIFFE ID of sub is left to the caller, which owns the value type.
	Verify(
		ctx context.Context,
		bundle *workloaddomain.WorkloadTrustBundle,
		subjectToken string,
		fetchJWKS func(ctx context.Context) ([]map[string]any, error),
		now time.Time,
	) (*workloaddomain.WorkloadSVIDClaims, error)
}
