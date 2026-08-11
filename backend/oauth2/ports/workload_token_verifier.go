package ports

import (
	"context"
	"time"

	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

// WorkloadTokenVerifier abstracts external workload attestation verification
// ([[wi-54-workload-identity-federation-spiffe]]). The token-exchange
// usecase depends on this interface only; the concrete implementation lives in
// backend/workloadidentity/usecases and is wired in at the composition root, so
// OAuth2's Go dependency on WorkloadIdentity stays limited to the published
// domain type it returns.
type WorkloadTokenVerifier interface {
	VerifyWorkloadToken(ctx context.Context, tenantID, subjectToken string, now time.Time) (*workloaddomain.WorkloadIdentityGrant, error)
}
