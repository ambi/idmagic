package usecases

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/ambi/idmagic/backend/workloadidentity/domain"
)

// VerifyWorkloadClientAssertionInput は spiffe_jwt のクライアント認証の入力である。
type VerifyWorkloadClientAssertionInput struct {
	// ClientID は認証するクライアントである。要求が client_id を省いたときは、OAuth2 が
	// JWT-SVID の sub を入れる。
	ClientID string
	// Assertion は client_assertion として送られた JWT-SVID である。
	Assertion string
	// ExpectedAudience はテナントの発行者識別子である。aud はこの値だけを持たなければならない。
	ExpectedAudience string
}

// VerifyWorkloadClientAssertion は、クライアントアサーションとして提示された JWT-SVID を
// fail-closed で検証し、要求のクライアントが関連付けの Agent に束縛されていることを確かめる
// (REQ-WORKLOADIDENTITY-017)。失敗経路は必ず用途 client_authentication の
// WorkloadAttestationRejected を emit してから reason 付きのエラーを返す。
//
// Token Exchange の検証 (VerifyWorkloadAttestation) との違いは三つある。信頼設定は sub の
// トラストドメインの spiffe の信頼設定だけから選び、iss では選ばない。aud は信頼設定の受理する
// audience ではなく発行者識別子だけを求める。結果の client_id は束縛の先頭ではなく要求の
// クライアントであり、束縛されていなければ拒否する。
func VerifyWorkloadClientAssertion(
	ctx context.Context,
	deps VerifyWorkloadAttestationDeps,
	tenantID string,
	in VerifyWorkloadClientAssertionInput,
	now time.Time,
) (*domain.WorkloadClientAuthentication, error) {
	reject := func(reason, trustBundleID string) (*domain.WorkloadClientAuthentication, error) {
		emitRejected(deps, tenantID, reason, trustBundleID, domain.AttestationUsageClientAuthentication, now)
		return nil, errWorkloadAttestationRejected(reason)
	}

	unverified, ok := peekClaims(in.Assertion)
	if !ok {
		return reject("unregistered_issuer", "")
	}
	id, err := domain.ParseSpiffeID(unverified.Subject)
	if err != nil {
		return reject("invalid_spiffe_id", "")
	}
	bundle, err := deps.TrustBundleRepo.FindSPIFFEByTrustDomain(ctx, tenantID, id.TrustDomain())
	if err != nil {
		return nil, err
	}
	if bundle == nil {
		return reject("unregistered_issuer", "")
	}
	if !bundle.IsEnabled() {
		return reject("trust_bundle_disabled", bundle.ID)
	}

	// 信頼設定の受理する audience は Token Exchange の subject_token のためのものなので、
	// 署名と有効期間の検証には発行者識別子だけを audience として渡す。
	expectation := *bundle
	expectation.AcceptedAudiences = []string{in.ExpectedAudience}
	claims, err := deps.SVIDVerifier.Verify(
		ctx, &expectation, in.Assertion,
		func(ctx context.Context) ([]map[string]any, error) { return deps.FetchJWKS(ctx, bundle) },
		now,
	)
	if err != nil {
		return reject(workloadSVIDRejectReason(err), bundle.ID)
	}
	// 受理する audience に含むかの検証は、ほかの値と並んでいても通す。ドラフトの 3.2 節は
	// 発行者識別子を唯一の値として求める。
	if !slices.Equal(claims.Audience, []string{in.ExpectedAudience}) {
		return reject("audience_mismatch", bundle.ID)
	}
	// 署名の検証の前に読んだ sub は検証後の sub と同じトークンの同じクレームなので、
	// トラストドメインは選んだ信頼設定と一致している。
	agent, binding, reason, err := resolveBoundAgent(ctx, deps, tenantID, bundle.ID, claims.Subject)
	if err != nil {
		return nil, err
	}
	if reason != "" {
		return reject(reason, bundle.ID)
	}
	agentBindings, err := deps.AgentRepo.ListBindings(ctx, tenantID, agent)
	if err != nil {
		return nil, err
	}
	bound := false
	for _, b := range agentBindings {
		bound = bound || b.ClientID == in.ClientID
	}
	if !bound {
		return reject("client_not_bound", bundle.ID)
	}

	return &domain.WorkloadClientAuthentication{
		AgentID:       agent,
		ClientID:      in.ClientID,
		TrustBundleID: bundle.ID,
		BindingID:     binding,
		SpiffeID:      claims.Subject,
		JTI:           claims.JTI,
		ExpiresAt:     claims.ExpiresAt,
	}, nil
}

// resolveBoundAgent は、検証を通った主体をちょうど一つの enabled の関連付けで Agent に写し、
// その Agent が Active であることを確かめる。拒否なら理由を返す。
func resolveBoundAgent(
	ctx context.Context, deps VerifyWorkloadAttestationDeps, tenantID, trustBundleID, subject string,
) (agentID, bindingID, reason string, err error) {
	bindings, err := deps.BindingRepo.ListByTrustBundle(ctx, tenantID, trustBundleID)
	if err != nil {
		return "", "", "", err
	}
	bindingValues := make([]domain.AgentWorkloadBinding, 0, len(bindings))
	for _, b := range bindings {
		bindingValues = append(bindingValues, *b)
	}
	matched, err := domain.MatchAgent(bindingValues, subject)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNoBindingMatch):
			return "", "", "no_binding_match", nil
		case errors.Is(err, domain.ErrAmbiguousBindingMatch):
			return "", "", "ambiguous_match", nil
		default:
			return "", "", "", err
		}
	}
	agent, err := deps.AgentRepo.FindByID(ctx, tenantID, matched.AgentID)
	if err != nil {
		return "", "", "", err
	}
	if agent == nil || !agent.IsActive() {
		return "", "", "agent_not_active", nil
	}
	return agent.ID, matched.ID, "", nil
}

func emitRejected(
	deps VerifyWorkloadAttestationDeps, tenantID, reason, trustBundleID string, usage domain.AttestationUsage, now time.Time,
) {
	if deps.Emit != nil {
		deps.Emit(&domain.WorkloadAttestationRejected{
			At: now, TenantID: tenantID, Reason: reason, TrustBundleID: trustBundleID, Usage: usage,
		})
	}
}
