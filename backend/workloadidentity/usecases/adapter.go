package usecases

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/workloadidentity/domain"
)

// WorkloadTokenVerifierAdapter satisfies oauth2/ports.WorkloadTokenVerifier
// (structurally; this package does not import oauth2 to keep the dependency
// one-directional per spec/scl.yaml). It is the composition-root wiring point
// between OAuth2's token-exchange grant and WorkloadIdentity (ADR-053).
type WorkloadTokenVerifierAdapter struct {
	Deps VerifyWorkloadAttestationDeps
}

func (a WorkloadTokenVerifierAdapter) VerifyWorkloadToken(ctx context.Context, tenantID, subjectToken string, now time.Time) (*domain.WorkloadIdentityGrant, error) {
	return VerifyWorkloadAttestation(ctx, a.Deps, tenantID, VerifyWorkloadAttestationInput{SubjectToken: subjectToken}, now)
}
