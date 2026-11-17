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
	// onFallback は、取得に失敗して最後に取得できた鍵を使ったとき、その出所
	// ("memory" または "stored") を受け取る。nil なら何もしない。
	onFallback func(source string)
}

func NewVerifier(onFallback func(source string)) *Verifier {
	return &Verifier{cache: tokens_jose.NewWorkloadJWKSCache(), onFallback: onFallback}
}

func (v *Verifier) Verify(
	ctx context.Context,
	bundle *workloaddomain.WorkloadTrustBundle,
	subjectToken string,
	fetchJWKS func(ctx context.Context) ([]map[string]any, error),
	now time.Time,
) (*workloaddomain.WorkloadSVIDClaims, error) {
	keys, source, err := v.cache.Get(ctx, jwksCacheKey(bundle), fetchJWKS, storedJWKS(bundle), now)
	if err != nil {
		return nil, workloaddomain.ErrSVIDKeysUnavailable
	}
	if source != tokens_jose.WorkloadJWKSLive && v.onFallback != nil {
		v.onFallback(source.String())
	}
	profile := tokens_jose.WorkloadSVIDProfileOIDC
	if bundle.IsSPIFFE() {
		profile = tokens_jose.WorkloadSVIDProfileSPIFFE
	}
	claims, err := tokens_jose.VerifyWorkloadSVID(subjectToken, keys, tokens_jose.WorkloadSVIDExpectation{
		Profile:           profile,
		Issuer:            bundle.Issuer,
		AcceptedAudiences: bundle.AcceptedAudiences,
		MaxTTL:            time.Duration(bundle.MaxSubjectTokenTTLSeconds) * time.Second,
	}, now)
	if err != nil {
		return nil, translateError(err)
	}
	return &workloaddomain.WorkloadSVIDClaims{
		Issuer: claims.Issuer, Subject: claims.Subject, Audience: claims.Audience, JTI: claims.JTI,
		ExpiresAt: claims.ExpiresAt, IssuedAt: claims.IssuedAt,
	}, nil
}

// jwksCacheKey は、メモリの鍵を信頼設定と取得元の組ごとに分ける。jwks_uri を変えた後に、
// 古い取得元の鍵を最後に取得できた鍵として使わない。
func jwksCacheKey(bundle *workloaddomain.WorkloadTrustBundle) string {
	if bundle.JWKSURI == nil {
		return bundle.ID
	}
	return bundle.ID + "\x00" + *bundle.JWKSURI
}

// storedJWKS は、jwks_uri から取得する信頼設定に保存した鍵を返す。
func storedJWKS(bundle *workloaddomain.WorkloadTrustBundle) *tokens_jose.WorkloadJWKSSnapshot {
	if !bundle.FetchesJWKSURI() {
		return nil
	}
	stored := bundle.StoredJWKS()
	if stored == nil {
		return nil
	}
	return &tokens_jose.WorkloadJWKSSnapshot{Keys: stored.Keys, FetchedAt: stored.FetchedAt}
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
