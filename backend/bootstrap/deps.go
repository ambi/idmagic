package bootstrap

import (
	"context"
	"errors"
	"strings"

	"github.com/ambi/idmagic/backend/application"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	samlports "github.com/ambi/idmagic/backend/saml/ports"
	scimports "github.com/ambi/idmagic/backend/scim/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	wsfederationports "github.com/ambi/idmagic/backend/wsfederation/ports"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
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
	// WebAuthn / Passkey と backup recovery code (wi-26 / ADR-087)。WebAuthnRP は env config
	// 由来で、未設定なら nil (WebAuthn 無効)。session store / repo は永続層に応じて差し替える。
	WebAuthnRP             *gowebauthn.WebAuthn
	WebAuthnCredentialRepo authnports.WebAuthnCredentialRepository
	WebAuthnSessionStore   authnports.WebAuthnSessionStore
	RecoveryCodeRepo       authnports.RecoveryCodeRepository
	// NewLoginAttemptThrottle は SCL 由来のしきい値から throttle adapter を生成する。
	// memory ランタイムはプロセスメモリ版、postgres_valkey ランタイムは Valkey 共有版を返す
	// (ADR-077: 複数レプリカで閾値がクラスタ全体で一つになるよう共有ストア化する)。
	NewLoginAttemptThrottle func(authnports.LoginThrottleConfigs) authnports.LoginAttemptThrottle
	KeyStore                oauthports.KeyStore
	TenantSaltStore         oauthports.TenantSaltStore
	EventSink               oauthports.EventSink
	AuditEventRepo          oauthports.AuditEventRepository
	AuthEventBucketStore    authnports.AuthEventBucketStore
	WsFedRPRepo             wsfederationports.WsFedRelyingPartyRepository
	SamlSPRepo              samlports.SamlServiceProviderRepository
	Application             application.Module
	Close                   func()
	DbPing                  func(context.Context) error
	ValkeyPing              func(context.Context) error
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
	var deps *Dependencies
	var err error
	switch envDefault("PERSISTENCE", "memory") {
	case "memory":
		deps, err = assembleMemory()
	case "postgres_valkey":
		deps, err = assemblePostgresValkey(ctx)
	default:
		return nil, errors.New("PERSISTENCE must be memory or postgres_valkey")
	}
	if err != nil {
		return nil, err
	}
	// WebAuthn RP は永続層に依らず env config から構築する (wi-26 / ADR-087)。
	rp, err := loadWebAuthnRP()
	if err != nil {
		return nil, err
	}
	deps.WebAuthnRP = rp
	return deps, nil
}

// loadWebAuthnRP は WEBAUTHN_RP_ID / WEBAUTHN_RP_ORIGINS / WEBAUTHN_RP_DISPLAY_NAME から RP を
// 構築する。RP_ID 未設定なら WebAuthn は無効 (nil) とし、RP_ID 設定時に origin が無ければ
// 起動を失敗させる (誤設定の silent 無効化を防ぐ起動時検証)。
func loadWebAuthnRP() (*gowebauthn.WebAuthn, error) {
	rpID := strings.TrimSpace(envDefault("WEBAUTHN_RP_ID", ""))
	if rpID == "" {
		return nil, nil //nolint:nilnil // RP_ID 未設定は WebAuthn 無効を表す正当な状態 (エラーではない)。
	}
	origins := splitAndTrim(envDefault("WEBAUTHN_RP_ORIGINS", ""))
	if len(origins) == 0 {
		return nil, errors.New("WEBAUTHN_RP_ORIGINS must be set when WEBAUTHN_RP_ID is set")
	}
	return authusecases.NewWebAuthn(authusecases.WebAuthnConfig{
		RPID:          rpID,
		RPDisplayName: envDefault("WEBAUTHN_RP_DISPLAY_NAME", "idmagic"),
		RPOrigins:     origins,
	})
}

// splitAndTrim はカンマ区切り文字列を空要素を除いてトリムして分割する。
func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
