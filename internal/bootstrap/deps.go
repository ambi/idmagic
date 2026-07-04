package bootstrap

import (
	"context"
	"errors"

	appports "github.com/ambi/idmagic/internal/application/ports"
	authnports "github.com/ambi/idmagic/internal/authentication/ports"
	idmports "github.com/ambi/idmagic/internal/identitymanagement/ports"
	oauthports "github.com/ambi/idmagic/internal/oauth2/ports"
	samlports "github.com/ambi/idmagic/internal/saml/ports"
	scimports "github.com/ambi/idmagic/internal/scim/ports"
	tenantports "github.com/ambi/idmagic/internal/tenancy/ports"
	wsfederationports "github.com/ambi/idmagic/internal/wsfederation/ports"
)

// Dependencies は HTTP 層に渡す全境界をまとめた DI コンテナ。
// 永続層 (memory/postgres_valkey) や event sink の差分を本構造体で吸収する。
type Dependencies struct {
	ScimRepo                scimports.ScimRepository
	ClientRepo              oauthports.OAuth2ClientRepository
	TenantRepo              tenantports.TenantRepository
	AttrSchemaRepo          tenantports.TenantUserAttributeSchemaRepository
	UserRepo                idmports.UserRepository
	GroupRepo               idmports.GroupRepository
	AgentRepo               idmports.AgentRepository
	MfaFactorRepo           authnports.MfaFactorRepository
	PasswordHistoryRepo     authnports.PasswordHistoryRepository
	PasswordResetTokenStore authnports.PasswordResetTokenStore
	EmailChangeTokenStore   authnports.EmailChangeTokenStore
	ConsentRepo             oauthports.ConsentRepository
	AuthzDetailTypeRepo     oauthports.AuthorizationDetailTypeRepository
	RequestStore            oauthports.AuthorizationRequestStore
	CodeStore               oauthports.AuthorizationCodeStore
	PARStore                oauthports.PARStore
	RefreshStore            oauthports.RefreshTokenStore
	DeviceCodeStore         oauthports.DeviceCodeStore
	DpopReplay              oauthports.DpopReplayStore
	ClientAssertionReplay   oauthports.ClientAssertionReplayStore
	AccessTokenDenylist     oauthports.AccessTokenDenylist
	SessionStore            authnports.SessionStore
	// NewLoginAttemptThrottle は SCL 由来のしきい値から throttle adapter を生成する。
	// memory ランタイムはプロセスメモリ版、postgres_valkey ランタイムは Valkey 共有版を返す
	// (ADR-077: 複数レプリカで閾値がクラスタ全体で一つになるよう共有ストア化する)。
	NewLoginAttemptThrottle     func(authnports.LoginThrottleConfigs) authnports.LoginAttemptThrottle
	KeyStore                    oauthports.KeyStore
	EventSink                   oauthports.EventSink
	AuditEventRepo              oauthports.AuditEventRepository
	AuthEventBucketStore        authnports.AuthEventBucketStore
	WsFedRPRepo                 wsfederationports.WsFedRelyingPartyRepository
	SamlSPRepo                  samlports.SamlServiceProviderRepository
	ApplicationRepo             appports.ApplicationRepository
	ApplicationIconStore        appports.ApplicationIconStore
	ApplicationAssignmentRepo   appports.AssignmentRepository
	ApplicationOrderingRepo     appports.ApplicationOrderingRepository
	ApplicationCategoryRepo     appports.ApplicationCategoryRepository
	ApplicationSignInPolicyRepo appports.SignInPolicyRepository
	Close                       func()
	DbPing                      func(context.Context) error
	ValkeyPing                  func(context.Context) error
}

// RuntimeConfig は /health などで露出するための実行時構成ラベルを集約する。
type RuntimeConfig struct {
	Persistence   string
	EventSink     string
	Observability string
	AuthZEN       string
}

func loadRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		Persistence:   envDefault("PERSISTENCE", "memory"),
		EventSink:     envDefault("EVENT_SINK", "console"),
		Observability: envDefault("OBSERVABILITY", "noop"),
		AuthZEN:       envDefault("AUTHZEN", "local"),
	}
}

// assemble は PERSISTENCE 環境変数に応じて memory/postgres_valkey いずれかの構成を組み立てる。
func assemble(ctx context.Context) (*Dependencies, error) {
	switch envDefault("PERSISTENCE", "memory") {
	case "memory":
		return assembleMemory()
	case "postgres_valkey":
		return assemblePostgresValkey(ctx)
	default:
		return nil, errors.New("PERSISTENCE must be memory or postgres_valkey")
	}
}
