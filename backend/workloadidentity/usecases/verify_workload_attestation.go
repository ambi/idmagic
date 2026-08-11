// Package usecases は WorkloadIdentity bounded context のアプリケーションロジックを
// 所有する。VerifyWorkloadAttestation は OAuth2 の token-exchange grant
// (subject_token_type=JwtSvid) から呼ばれ、外部 attestation token を登録済み
// WorkloadTrustBundle で検証し、AgentWorkloadBinding で Agent principal に写す。
package usecases

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/ambi/idmagic/backend/workloadidentity/ports"
)

// VerifyWorkloadAttestationDeps holds the collaborators VerifyWorkloadAttestation
// needs. FetchJWKS performs the live JWKS lookup for a bundle (network I/O is
// injected so the usecase itself stays free of transport concerns); SVIDVerifier
// is the adapters-layer port that checks signature/claims and applies the
// last-known-good JWKS fallback.
type VerifyWorkloadAttestationDeps struct {
	TrustBundleRepo ports.WorkloadTrustBundleRepository
	BindingRepo     ports.AgentWorkloadBindingRepository
	AgentRepo       agentports.AgentRepository
	SVIDVerifier    ports.WorkloadSVIDVerifier
	FetchJWKS       func(ctx context.Context, bundle *domain.WorkloadTrustBundle) ([]map[string]any, error)
	Emit            func(spec.DomainEvent)
}

type VerifyWorkloadAttestationInput struct {
	// SubjectToken is the raw external attestation token (JWT-SVID etc.).
	SubjectToken string
}

// VerifyWorkloadAttestation は fail-closed で外部 attestation を検証し、一意に決定した
// Agent principal を返す。失敗経路は必ず WorkloadAttestationRejected を emit
// してから reason 付きのエラーを返す。
func VerifyWorkloadAttestation(
	ctx context.Context,
	deps VerifyWorkloadAttestationDeps,
	tenantID string,
	in VerifyWorkloadAttestationInput,
	now time.Time,
) (*domain.WorkloadIdentityGrant, error) {
	reject := func(reason, trustBundleID string) (*domain.WorkloadIdentityGrant, error) {
		if deps.Emit != nil {
			deps.Emit(&domain.WorkloadAttestationRejected{
				At: now, TenantID: tenantID, Reason: reason, TrustBundleID: trustBundleID,
			})
		}
		return nil, errWorkloadAttestationRejected(reason)
	}

	iss, err := peekIssuer(in.SubjectToken)
	if err != nil || iss == "" {
		return reject("unregistered_issuer", "")
	}

	bundle, err := deps.TrustBundleRepo.FindByIssuer(ctx, tenantID, iss)
	if err != nil {
		return nil, err
	}
	if bundle == nil {
		return reject("unregistered_issuer", "")
	}
	if !bundle.IsEnabled() {
		return reject("trust_bundle_disabled", bundle.ID)
	}

	claims, err := deps.SVIDVerifier.Verify(
		ctx, bundle.ID, in.SubjectToken, bundle.Issuer, bundle.AcceptedAudiences,
		time.Duration(bundle.MaxSubjectTokenTTLSeconds)*time.Second,
		func(ctx context.Context) ([]map[string]any, error) { return deps.FetchJWKS(ctx, bundle) },
		now,
	)
	if err != nil {
		return reject(workloadSVIDRejectReason(err), bundle.ID)
	}

	bindings, err := deps.BindingRepo.ListByTrustBundle(ctx, tenantID, bundle.ID)
	if err != nil {
		return nil, err
	}
	bindingValues := make([]domain.AgentWorkloadBinding, 0, len(bindings))
	for _, b := range bindings {
		bindingValues = append(bindingValues, *b)
	}
	matched, err := domain.MatchAgent(bindingValues, claims.Subject)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNoBindingMatch):
			return reject("no_binding_match", bundle.ID)
		case errors.Is(err, domain.ErrAmbiguousBindingMatch):
			return reject("ambiguous_match", bundle.ID)
		default:
			return nil, err
		}
	}

	agent, err := deps.AgentRepo.FindByID(ctx, tenantID, matched.AgentID)
	if err != nil {
		return nil, err
	}
	if agent == nil || !agent.IsActive() {
		return reject("agent_not_active", bundle.ID)
	}

	agentBindings, err := deps.AgentRepo.ListBindings(ctx, tenantID, agent.ID)
	if err != nil {
		return nil, err
	}
	if len(agentBindings) == 0 {
		return reject("agent_unbound", bundle.ID)
	}

	return &domain.WorkloadIdentityGrant{
		AgentID:       agent.ID,
		ClientID:      agentBindings[0].ClientID,
		TrustBundleID: bundle.ID,
		BindingID:     matched.ID,
	}, nil
}

// peekIssuer は署名検証前に iss claim だけを読む。iss を手がかりに登録済み
// WorkloadTrustBundle (と対応する JWKS) を解決するための issuer discovery で、
// 標準的な JWT 実装がしばしば行う「まず iss/kid を見て検証鍵を選ぶ」手順と同じ。
// 署名検証自体はこの後 VerifyWorkloadSVID が行うため、ここで claim を信用した
// 判断は一切行わない。
func peekIssuer(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", errors.New("workload attestation: malformed token")
	}
	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	var payload struct {
		Issuer string `json:"iss"`
	}
	if err := json.Unmarshal(pb, &payload); err != nil {
		return "", err
	}
	return payload.Issuer, nil
}

func workloadSVIDRejectReason(err error) string {
	switch {
	case errors.Is(err, domain.ErrSVIDKeysUnavailable):
		return "jwks_unavailable"
	case errors.Is(err, domain.ErrSVIDInvalidSignature):
		return "invalid_signature"
	case errors.Is(err, domain.ErrSVIDIssuerMismatch):
		return "unregistered_issuer"
	case errors.Is(err, domain.ErrSVIDAudienceMismatch):
		return "audience_mismatch"
	case errors.Is(err, domain.ErrSVIDExpired):
		return "expired"
	case errors.Is(err, domain.ErrSVIDTTLExceeded):
		return "ttl_exceeded"
	default:
		return "invalid_signature"
	}
}

func errWorkloadAttestationRejected(reason string) error {
	return errors.New("workload attestation rejected: " + reason)
}
