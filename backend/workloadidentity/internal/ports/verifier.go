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
	// fallback, keyed by bundleID), then checks signature/iss/aud/exp/TTL.
	Verify(
		ctx context.Context,
		bundleID, subjectToken, issuer string,
		acceptedAudiences []string,
		maxTTL time.Duration,
		fetchJWKS func(ctx context.Context) ([]map[string]any, error),
		now time.Time,
	) (*workloaddomain.WorkloadSVIDClaims, error)
}
