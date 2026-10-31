package ports

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// ErrCrossAppAccessPolicyExists は同じ組の許可がすでにあることを表す。
var ErrCrossAppAccessPolicyExists = errors.New("cross-app access policy already exists")

// CrossAppAccessPolicyRepository は ID-JAG の許可の永続境界である。すべてテナントに閉じる。
type CrossAppAccessPolicyRepository interface {
	ListIssuePolicies(ctx context.Context, tenantID string) ([]*domain.CrossAppAccessIssuePolicy, error)
	FindIssuePolicy(ctx context.Context, tenantID, id string) (*domain.CrossAppAccessIssuePolicy, error)
	// CreateIssuePolicy は同じクライアントと audience の組があれば ErrCrossAppAccessPolicyExists を返す。
	CreateIssuePolicy(ctx context.Context, policy *domain.CrossAppAccessIssuePolicy) error
	DeleteIssuePolicy(ctx context.Context, tenantID, id string) error

	ListRedeemPolicies(ctx context.Context, tenantID string) ([]*domain.CrossAppAccessRedeemPolicy, error)
	FindRedeemPolicy(ctx context.Context, tenantID, id string) (*domain.CrossAppAccessRedeemPolicy, error)
	FindRedeemPolicyFor(ctx context.Context, tenantID, providerID, clientID string) (*domain.CrossAppAccessRedeemPolicy, error)
	// CreateRedeemPolicy は同じ接続とクライアントの組があれば ErrCrossAppAccessPolicyExists を返す。
	CreateRedeemPolicy(ctx context.Context, policy *domain.CrossAppAccessRedeemPolicy) error
	DeleteRedeemPolicy(ctx context.Context, tenantID, id string) error
}

// IDJAGTrustedIssuer はテナントが SSO で信頼する発行者である。
type IDJAGTrustedIssuer struct {
	ProviderID string
	JWKS       []map[string]any
}

// IDJAGIssuerTrust は ID-JAG の発行者と主体を、Authentication が所有する接続と連携から解決する。
// OAuth2 はこの契約だけに依存し、組み立て地点で結ぶ。
type IDJAGIssuerTrust interface {
	// ResolveIssuer は有効な OIDC の接続のうち issuer と一致するものがちょうど一つのときだけ返し、
	// それ以外は nil を返す。
	ResolveIssuer(ctx context.Context, tenantID, issuer string) (*IDJAGTrustedIssuer, error)
	// ProviderExists は接続がテナントにあるかを返す。
	ProviderExists(ctx context.Context, tenantID, providerID string) (bool, error)
	// ResolveSubject は連携した利用者を返す。連携がなければ空を返す。
	ResolveSubject(ctx context.Context, tenantID, providerID, externalSubject string) (string, error)
}

// IDTokenAssertion は検証を通った、自身が発行した ID Token のクレームである。
type IDTokenAssertion struct {
	Subject   string
	Audiences []string
	IssuedAt  time.Time
	AuthTime  int64
	AMR       []string
	ACR       string
}

// IDTokenAssertionVerifier は subject_token を ID Token として検証する。ID Token ではないか、
// 署名、発行者、期限が正しくなければ error を返す (draft-05 §4.3.3)。
type IDTokenAssertionVerifier interface {
	VerifyIDTokenAssertion(ctx context.Context, token string) (*IDTokenAssertion, error)
}

// IDJAGInput は ID-JAG に載せる値である。
type IDJAGInput struct {
	Subject              string
	Audience             string
	ClientID             string
	Resources            []string
	Scopes               []string
	AuthorizationDetails []spec.AuthorizationDetail
	AuthTime             int64
	AMR                  []string
	ACR                  string
	Email                string
	SenderConstraint     *domain.SenderConstraint
}

// IDJAGSigner は ID-JAG に専用の鍵で署名する。
type IDJAGSigner interface {
	SignIDJAG(ctx context.Context, in IDJAGInput) (token, jti string, err error)
	IDJAGTTLSeconds() int
}
