// Package support: HTTP アダプタの共有基盤。
//
// 複数コンテキスト (tenancy / authentication / oauth2) の adapters/http が
// 共通で使う依存集約 (Deps)・テナント解決 middleware・横断ヘルパのみを置く。
// 各コンテキストの adapters/http は support を import し、
// shared/adapters/http/server が各コンテキストの RegisterRoutes を集約する。
// support <- context adapters/http <- server の一方向。
package support

import (
	"context"
	"sync/atomic"
	"time"

	appports "github.com/ambi/idmagic/internal/application/ports"
	authdomain "github.com/ambi/idmagic/internal/authentication/domain"
	authnports "github.com/ambi/idmagic/internal/authentication/ports"
	authusecases "github.com/ambi/idmagic/internal/authentication/usecases"
	idmports "github.com/ambi/idmagic/internal/identitymanagement/ports"
	oauthports "github.com/ambi/idmagic/internal/oauth2/ports"
	samlports "github.com/ambi/idmagic/internal/saml/ports"
	scimports "github.com/ambi/idmagic/internal/scim/ports"
	"github.com/ambi/idmagic/internal/shared/adapters/crypto"
	"github.com/ambi/idmagic/internal/shared/spec"
	tenantports "github.com/ambi/idmagic/internal/tenancy/ports"
	"github.com/ambi/idmagic/internal/wsfederation/adapters/samltoken"
	wsfederationports "github.com/ambi/idmagic/internal/wsfederation/ports"
)

// Deps は全 HTTP ハンドラが共有する依存集約。bootstrap が一様に配線する。
type Deps struct {
	Issuer                      string
	SCL                         *spec.SCL
	ScimRepo                    scimports.ScimRepository
	TenantRepo                  tenantports.TenantRepository
	AttrSchemaRepo              tenantports.TenantUserAttributeSchemaRepository
	LegacyBareIssuer            bool
	ClientRepo                  oauthports.OAuth2ClientRepository
	UserRepo                    idmports.UserRepository
	ConsentRepo                 oauthports.ConsentRepository
	AuthzDetailTypeRepo         oauthports.AuthorizationDetailTypeRepository
	RequestStore                oauthports.AuthorizationRequestStore
	CodeStore                   oauthports.AuthorizationCodeStore
	PARStore                    oauthports.PARStore
	RefreshStore                oauthports.RefreshTokenStore
	DeviceCodeStore             oauthports.DeviceCodeStore
	DpopReplayStore             oauthports.DpopReplayStore
	ClientAssertionReplayStore  oauthports.ClientAssertionReplayStore
	AccessTokenDenylist         oauthports.AccessTokenDenylist
	KeyStore                    oauthports.KeyStore
	TokenIssuer                 oauthports.TokenIssuer
	TokenIntrospector           oauthports.TokenIntrospector
	AuditEventRepo              oauthports.AuditEventRepository
	AuthEventBucketStore        authnports.AuthEventBucketStore
	Authorizer                  oauthports.Authorizer
	JWKResolver                 *crypto.JWKResolver
	PasswordHasher              authnports.PasswordHasher
	GroupRepo                   idmports.GroupRepository
	AgentRepo                   idmports.AgentRepository
	MfaFactorRepo               authnports.MfaFactorRepository
	PasswordHistoryRepo         authnports.PasswordHistoryRepository
	PasswordResetTokenStore     authnports.PasswordResetTokenStore
	EmailChangeTokenStore       authnports.EmailChangeTokenStore
	EmailSender                 authnports.EmailSender
	BreachedPasswordChecker     authnports.BreachedPasswordChecker
	LoginAttemptThrottle        authnports.LoginAttemptThrottle
	TrustedForwardedHops        int
	SentinelPasswordHash        string
	SessionManager              *authusecases.SessionManager
	AuthnResolver               authdomain.AuthenticationContextResolver
	WsFedRPRepo                 wsfederationports.WsFedRelyingPartyRepository
	SamlSPRepo                  samlports.SamlServiceProviderRepository
	FederationSigner            *samltoken.Signer
	ApplicationRepo             appports.ApplicationRepository
	ApplicationIconStore        appports.ApplicationIconStore
	ApplicationAssignmentRepo   appports.AssignmentRepository
	ApplicationOrderingRepo     appports.ApplicationOrderingRepository
	ApplicationCategoryRepo     appports.ApplicationCategoryRepository
	ApplicationSignInPolicyRepo appports.SignInPolicyRepository
	DefaultSignInPolicyRepo     appports.DefaultSignInPolicyRepository
	OperationTimeout            time.Duration
	DetachedCompletionTimeout   time.Duration
	AbortMetrics                HTTPAbortMetrics
	Emit                        func(spec.DomainEvent)
	HealthInfo                  HealthInfo
	DbPing                      func(context.Context) error
	ValkeyPing                  func(context.Context) error
	ShuttingDown                *atomic.Bool
	StartupComplete             *atomic.Bool
}

// HealthInfo は bootstrap が決定した実行時構成のラベル。
// /health がそのまま JSON で返すだけの読み取り専用情報を保持する。
type HealthInfo struct {
	Persistence   string
	EventSink     string
	Observability string
	AuthZEN       string
}
