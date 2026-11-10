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
	// VerifyClientAssertion は spiffe_jwt のクライアントアサーションの JWT-SVID を検証し、
	// clientID が SPIFFE ID の関連付けの Agent に束縛されていることを確かめる
	// (REQ-WORKLOADIDENTITY-017)。expectedAudience はテナントの発行者識別子である。
	// 拒否は WorkloadIdentity が監査に残すので、呼び出し側は invalid_client にだけ写す。
	VerifyClientAssertion(
		ctx context.Context, tenantID, clientID, assertion, expectedAudience string, now time.Time,
	) (*workloaddomain.WorkloadClientAuthentication, error)
}
