// Package verification_jose is the adapters-layer implementation of
// ports.WorkloadSVIDVerifier. It is the only place in
// WorkloadIdentity that imports backend/shared/security/tokens_jose,
// keeping the usecases layer free of adapter dependencies.
package verification_jose

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

type Verifier struct {
	cache *tokens_jose.WorkloadJWKSCache
}

func NewVerifier() *Verifier {
	return &Verifier{cache: tokens_jose.NewWorkloadJWKSCache()}
}

func (v *Verifier) Verify(
	ctx context.Context,
	bundleID, subjectToken, issuer string,
	acceptedAudiences []string,
	maxTTL time.Duration,
	fetchJWKS func(ctx context.Context) ([]map[string]any, error),
	now time.Time,
) (*workloaddomain.WorkloadSVIDClaims, error) {
	keys, _, err := v.cache.Get(ctx, bundleID, fetchJWKS, now)
	if err != nil {
		return nil, workloaddomain.ErrSVIDKeysUnavailable
	}
	claims, err := tokens_jose.VerifyWorkloadSVID(subjectToken, keys, issuer, acceptedAudiences, maxTTL, now)
	if err != nil {
		return nil, translateError(err)
	}
	return &workloaddomain.WorkloadSVIDClaims{
		Issuer: claims.Issuer, Subject: claims.Subject,
		ExpiresAt: claims.ExpiresAt, IssuedAt: claims.IssuedAt,
	}, nil
}

func translateError(err error) error {
	switch {
	case errors.Is(err, tokens_jose.ErrWorkloadSVIDInvalidSignature):
		return workloaddomain.ErrSVIDInvalidSignature
	case errors.Is(err, tokens_jose.ErrWorkloadSVIDIssuerMismatch):
		return workloaddomain.ErrSVIDIssuerMismatch
	case errors.Is(err, tokens_jose.ErrWorkloadSVIDAudienceMismatch):
		return workloaddomain.ErrSVIDAudienceMismatch
	case errors.Is(err, tokens_jose.ErrWorkloadSVIDExpired):
		return workloaddomain.ErrSVIDExpired
	case errors.Is(err, tokens_jose.ErrWorkloadSVIDTTLExceeded):
		return workloaddomain.ErrSVIDTTLExceeded
	default:
		return workloaddomain.ErrSVIDMalformed
	}
}
