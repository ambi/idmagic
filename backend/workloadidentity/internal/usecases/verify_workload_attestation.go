// Package usecases は WorkloadIdentity モジュールのアプリケーションロジックを
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
	"github.com/ambi/idmagic/backend/workloadidentity/internal/ports"
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
	// ClientID はトークンエンドポイントで認証した要求のクライアントである。OAuth2 が認証で
	// 登録から解決した値を渡すので、束縛の client_id と完全一致で比べる。
	ClientID string
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
		emitRejected(deps, tenantID, reason, trustBundleID, domain.AttestationUsageTokenExchange, now)
		return nil, errWorkloadAttestationRejected(reason)
	}

	bundle, err := findTrustBundle(ctx, deps.TrustBundleRepo, tenantID, in.SubjectToken)
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
		ctx, bundle, in.SubjectToken,
		func(ctx context.Context) ([]map[string]any, error) { return deps.FetchJWKS(ctx, bundle) },
		now,
	)
	if err != nil {
		return reject(workloadSVIDRejectReason(err), bundle.ID)
	}
	if bundle.IsSPIFFE() {
		id, err := domain.ParseSpiffeID(claims.Subject)
		if err != nil {
			return reject("invalid_spiffe_id", bundle.ID)
		}
		if id.TrustDomain() != bundle.TrustDomain {
			return reject("trust_domain_mismatch", bundle.ID)
		}
	}

	agentID, bindingID, reason, err := resolveBoundAgent(ctx, deps, tenantID, bundle.ID, claims.Subject)
	if err != nil {
		return nil, err
	}
	if reason != "" {
		return reject(reason, bundle.ID)
	}

	agentBindings, err := deps.AgentRepo.ListBindings(ctx, tenantID, agentID)
	if err != nil {
		return nil, err
	}
	if len(agentBindings) == 0 {
		return reject("agent_unbound", bundle.ID)
	}
	// JWT-SVID は Bearer なので、提示しただけで Agent の資格情報にしない。要求のクライアントが
	// 束縛先であることを求め、発行するトークンの sub と client_id を同じ主体にする。
	bound := false
	for _, b := range agentBindings {
		bound = bound || b.ClientID == in.ClientID
	}
	if !bound {
		return reject("client_not_bound", bundle.ID)
	}

	return &domain.WorkloadIdentityGrant{
		AgentID:       agentID,
		ClientID:      in.ClientID,
		TrustBundleID: bundle.ID,
		BindingID:     bindingID,
	}, nil
}

// findTrustBundle は署名の検証の前に読んだ iss と sub で、テナントの信頼設定を選ぶ
// (docs/modules/workloadidentity/features/attestation-exchange.md の実現方式)。
// iss に一致する信頼設定を先に探し、なければ sub を SPIFFE ID として読み、その
// トラストドメインの spiffe の信頼設定を探す。読んだ値は信頼設定と JWKS の選択にだけ
// 使い、主体の特定とトラストドメインの一致の判定は署名の検証の後に行う。トークンとして
// 読めなければ nil を返し、呼び出し側が unregistered_issuer で拒否する。
func findTrustBundle(
	ctx context.Context, repo ports.WorkloadTrustBundleRepository, tenantID, token string,
) (*domain.WorkloadTrustBundle, error) {
	keys := trustBundleLookupKeys(token)
	var bundle *domain.WorkloadTrustBundle
	var err error
	if keys.issuer != "" {
		bundle, err = repo.FindByIssuer(ctx, tenantID, keys.issuer)
	}
	if err == nil && bundle == nil && keys.trustDomain != "" {
		bundle, err = repo.FindSPIFFEByTrustDomain(ctx, tenantID, keys.trustDomain)
	}
	return bundle, err
}

type lookupKeys struct {
	issuer      string
	trustDomain string
}

// trustBundleLookupKeys は署名を検証せずに読んだ iss と、SPIFFE ID として読めた sub の
// トラストドメインを返す。読めない値は空にする。
func trustBundleLookupKeys(token string) lookupKeys {
	unverified, ok := peekClaims(token)
	if !ok {
		return lookupKeys{}
	}
	keys := lookupKeys{issuer: unverified.Issuer}
	if id, err := domain.ParseSpiffeID(unverified.Subject); err == nil {
		keys.trustDomain = id.TrustDomain()
	}
	return keys
}

type unverifiedClaims struct {
	Issuer  string `json:"iss"`
	Subject string `json:"sub"`
}

// peekClaims は署名を検証せずに iss と sub だけを読む。
func peekClaims(token string) (unverifiedClaims, bool) {
	var claims unverifiedClaims
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return claims, false
	}
	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, false
	}
	if err := json.Unmarshal(pb, &claims); err != nil {
		return claims, false
	}
	return claims, true
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
