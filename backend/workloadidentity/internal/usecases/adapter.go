package usecases

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/workloadidentity/domain"
)

// WorkloadTokenVerifierAdapter satisfies oauth2/ports.WorkloadTokenVerifier
// (structurally; this package does not import oauth2 to keep the dependency
// one-directional, as `mise run check-boundaries` enforces). It is the wiring point
// between OAuth2's token-exchange grant and WorkloadIdentity.
type WorkloadTokenVerifierAdapter struct {
	Deps VerifyWorkloadAttestationDeps
}

func (a WorkloadTokenVerifierAdapter) VerifyWorkloadToken(ctx context.Context, tenantID, subjectToken string, now time.Time) (*domain.WorkloadIdentityGrant, error) {
	return VerifyWorkloadAttestation(ctx, a.Deps, tenantID, VerifyWorkloadAttestationInput{SubjectToken: subjectToken}, now)
}

func (a WorkloadTokenVerifierAdapter) VerifyClientAssertion(
	ctx context.Context, tenantID, clientID, assertion, expectedAudience string, now time.Time,
) (*domain.WorkloadClientAuthentication, error) {
	return VerifyWorkloadClientAssertion(ctx, a.Deps, tenantID, VerifyWorkloadClientAssertionInput{
		ClientID: clientID, Assertion: assertion, ExpectedAudience: expectedAudience,
	}, now)
}
