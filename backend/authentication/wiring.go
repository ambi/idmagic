package authentication

import (
	"context"
	"net/http"
	"time"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	appports "github.com/ambi/idmagic/backend/application/ports"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	authnmemory "github.com/ambi/idmagic/backend/authentication/internal/db_memory"
	authnpostgres "github.com/ambi/idmagic/backend/authentication/internal/db_postgres"
	authhttpdeps "github.com/ambi/idmagic/backend/authentication/internal/deps_http"
	federationmemory "github.com/ambi/idmagic/backend/authentication/internal/federation/db_memory"
	federationpostgres "github.com/ambi/idmagic/backend/authentication/internal/federation/db_postgres"
	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
	federationhttp "github.com/ambi/idmagic/backend/authentication/internal/federation/handlers_http"
	oidcprotocol "github.com/ambi/idmagic/backend/authentication/internal/federation/protocol_oidc"
	federationsecrets "github.com/ambi/idmagic/backend/authentication/internal/federation/secrets_env"
	federationusecases "github.com/ambi/idmagic/backend/authentication/internal/federation/usecases"
	authhttp "github.com/ambi/idmagic/backend/authentication/internal/handlers_http"
	mfamemory "github.com/ambi/idmagic/backend/authentication/internal/mfa/db_memory"
	mfapostgres "github.com/ambi/idmagic/backend/authentication/internal/mfa/db_postgres"
	passwordmemory "github.com/ambi/idmagic/backend/authentication/internal/password/db_memory"
	passwordpostgres "github.com/ambi/idmagic/backend/authentication/internal/password/db_postgres"
	authnports "github.com/ambi/idmagic/backend/authentication/internal/ports"
	recoverymemory "github.com/ambi/idmagic/backend/authentication/internal/recovery/db_memory"
	recoverypostgres "github.com/ambi/idmagic/backend/authentication/internal/recovery/db_postgres"
	securitynotificationmemory "github.com/ambi/idmagic/backend/authentication/internal/securitynotification/db_memory"
	securitynotificationpostgres "github.com/ambi/idmagic/backend/authentication/internal/securitynotification/db_postgres"
	securitynotificationusecases "github.com/ambi/idmagic/backend/authentication/internal/securitynotification/usecases"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/internal/session/db_memory"
	sessionpostgres "github.com/ambi/idmagic/backend/authentication/internal/session/db_postgres"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/internal/session/usecases"
	signinhttp "github.com/ambi/idmagic/backend/authentication/internal/signin/handlers_http"
	totpmemory "github.com/ambi/idmagic/backend/authentication/internal/totp/db_memory"
	totppostgres "github.com/ambi/idmagic/backend/authentication/internal/totp/db_postgres"
	totpusecases "github.com/ambi/idmagic/backend/authentication/internal/totp/usecases"
	trusteddevicememory "github.com/ambi/idmagic/backend/authentication/internal/trusteddevice/db_memory"
	trusteddevicepostgres "github.com/ambi/idmagic/backend/authentication/internal/trusteddevice/db_postgres"
	authusecases "github.com/ambi/idmagic/backend/authentication/internal/usecases"
	webauthnmemory "github.com/ambi/idmagic/backend/authentication/internal/webauthn/db_memory"
	webauthnpostgres "github.com/ambi/idmagic/backend/authentication/internal/webauthn/db_postgres"
	webauthnusecases "github.com/ambi/idmagic/backend/authentication/internal/webauthn/usecases"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	signinports "github.com/ambi/idmagic/backend/authentication/signin/ports"
	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	consentports "github.com/ambi/idmagic/backend/oauth2/consent/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// SessionManager は対話のログインセッションを管理する。組み立て地点が起動の後半で発行先と使用量を結ぶ。
type SessionManager = sessionusecases.SessionManager

// AuthEventBucketStore は認証イベントの集計の保存先である。
type AuthEventBucketStore = authnports.AuthEventBucketStore

// FederatedClaims は、外部の IdP の主張を正規化したものである。
type FederatedClaims = federationdomain.NormalizedClaims

// NewMemoryModule は Authentication が所有する保存先をメモリで作る。users はパスワードの再設定が
// 確定するときに書き込む IdManagement の User の保存先である。
func NewMemoryModule(users userports.UserRepository) Module {
	federation := federationmemory.NewRepositories()
	history := passwordmemory.NewPasswordHistoryRepository()
	return Module{
		FederationConnectionRepo: federation.Connections,
		FederationIdentityRepo:   federation.Identities,
		FederationAttemptStore:   federation.Attempts,
		FederationReplayStore:    federation.Replay,
		FederationSecretResolver: federationsecrets.Resolver{},
		MfaFactorRepo:            totpmemory.NewMfaFactorRepository(),
		MfaEnrollmentBypassRepo:  mfamemory.NewMfaEnrollmentBypassRepository(),
		PasswordHistoryRepo:      history,
		PasswordResetTokenStore:  passwordmemory.NewPasswordResetTokenStore(users, history),
		SessionStore:             sessionmemory.NewSessionStore(),
		WebAuthnCredentialRepo:   webauthnmemory.NewWebAuthnCredentialRepository(),
		WebAuthnSessionStore:     webauthnmemory.NewWebAuthnSessionStore(),
		RecoveryCodeRepo:         recoverymemory.NewRecoveryCodeRepository(),
		TrustedDeviceRepo:        trusteddevicememory.NewTrustedDeviceRepository(),

		NotificationPreferenceRepo: securitynotificationmemory.NewPreferenceRepository(),
		KnownSignInDeviceRepo:      securitynotificationmemory.NewKnownDeviceRepository(),
		NewLoginAttemptThrottle: func(configs sessionports.LoginThrottleConfigs) sessionports.LoginAttemptThrottle {
			return sessionmemory.NewLoginAttemptThrottle(configs)
		},
		AuthEventBucketStore: authnmemory.NewAuthEventBucketStore(),
	}
}

// NewPostgresModule は Authentication が所有する保存先を PostgreSQL で作る。cipher は MFA と外部 IdP の
// 秘密情報を暗号化し、usersInTx はパスワードの再設定と同じトランザクションで User を保存する。
func NewPostgresModule(pool postgres.DB, cipher totpports.SecretCipher, usersInTx func(pgx.Tx) userports.UserWriter) Module {
	return Module{
		FederationConnectionRepo: &federationpostgres.ConnectionRepository{Pool: pool, Cipher: cipher},
		FederationIdentityRepo:   &federationpostgres.IdentityRepository{Pool: pool},
		FederationAttemptStore:   &federationpostgres.AttemptStore{Pool: pool},
		FederationReplayStore:    &federationpostgres.ReplayStore{Pool: pool},
		FederationSecretResolver: federationsecrets.Resolver{},
		MfaFactorRepo:            &totppostgres.MfaFactorRepository{Pool: pool, Cipher: cipher},
		MfaEnrollmentBypassRepo:  &mfapostgres.MfaEnrollmentBypassRepository{Pool: pool},
		PasswordHistoryRepo:      &passwordpostgres.PasswordHistoryRepository{Pool: pool},
		PasswordResetTokenStore:  passwordpostgres.NewPasswordResetTokenStore(pool, usersInTx),
		SessionStore:             &sessionpostgres.SessionRepository{Pool: pool},
		WebAuthnCredentialRepo:   &webauthnpostgres.WebAuthnCredentialRepository{Pool: pool},
		WebAuthnSessionStore:     &webauthnpostgres.WebAuthnSessionStore{Pool: pool},
		RecoveryCodeRepo:         &recoverypostgres.RecoveryCodeRepository{Pool: pool},
		TrustedDeviceRepo:        &trusteddevicepostgres.TrustedDeviceRepository{Pool: pool},

		NotificationPreferenceRepo: &securitynotificationpostgres.PreferenceRepository{Pool: pool},
		KnownSignInDeviceRepo:      &securitynotificationpostgres.KnownDeviceRepository{Pool: pool},
		NewLoginAttemptThrottle: func(configs sessionports.LoginThrottleConfigs) sessionports.LoginAttemptThrottle {
			return &sessionpostgres.LoginAttemptThrottle{Pool: pool, Configs: configs}
		},
		AuthEventBucketStore: &authnpostgres.AuthEventBucketStore{Pool: pool},
	}
}

// FieldMigrator は、暗号化した列を有効な鍵の版へ再暗号化する。DataKeys の移行器と同じ操作を持つ。
type FieldMigrator interface {
	ReencryptBatch(ctx context.Context, tenantID string, activeVersion, batchSize int) (migrated int, err error)
	PendingCount(ctx context.Context, tenantID string, activeVersion int) (int, error)
}

// RegisterFieldMigrators は、PostgreSQL の保存先が暗号化する MFA の秘密と外部 IdP の秘密の移行器を
// 名前とともに register へ渡す。メモリの保存先は暗号化しないので何も渡さない。
func (m Module) RegisterFieldMigrators(register func(name string, migrator FieldMigrator)) {
	if factors, ok := m.MfaFactorRepo.(*totppostgres.MfaFactorRepository); ok {
		register(totppostgres.MfaFactorMigratorName, &totppostgres.MfaFactorReencryptor{Repo: factors})
	}
	if connections, ok := m.FederationConnectionRepo.(*federationpostgres.ConnectionRepository); ok {
		register(federationpostgres.IdentityProviderSecretMigratorName, &federationpostgres.ConnectionSecretReencryptor{
			Repo: connections, EnvResolver: federationsecrets.Resolver{},
		})
	}
}

// NewWebAuthn は WebAuthn の Relying Party を設定から作る。
func NewWebAuthn(rpID, rpDisplayName string, rpOrigins []string) (*gowebauthn.WebAuthn, error) {
	return webauthnusecases.NewWebAuthn(webauthnusecases.WebAuthnConfig{
		RPID: rpID, RPDisplayName: rpDisplayName, RPOrigins: rpOrigins,
	})
}

// NewSessionManager は、この Module のセッションの保存先でログインセッションを管理する SessionManager を作る。
func (m Module) NewSessionManager() *SessionManager {
	return sessionusecases.NewSessionManager(m.SessionStore)
}

// SessionAuthentications は、管理 API と account API の認証がログインセッションを読む先を返す。
func (m Module) SessionAuthentications() support.SessionAuthenticator {
	return authhttpdeps.SessionAuthentications{Resolver: m.AuthnResolver, Sessions: m.SessionManager}
}

// SecurityNotificationDeps は、アカウントのセキュリティ通知がほかのモジュールと共通基盤から受け取る依存である。
type SecurityNotificationDeps struct {
	UserRepo            userports.UserRepository
	Notifier            sharednotification.Notifier
	TenantSource        sharednotification.TenantNotificationSource
	SaltStore           auditports.TenantSaltStore
	IssuerResolver      func(ctx context.Context, tenantID string) string
	SystemDefaultLocale string
	Emit                func(spec.DomainEvent)
}

// DispatchSecurityNotification は、イベント 1 件に対応するアカウントのセキュリティ通知を送る。
// カタログに載らないイベント種別では何もしない。
func (m Module) DispatchSecurityNotification(ctx context.Context, d SecurityNotificationDeps, event spec.DomainEvent) error {
	return securitynotificationusecases.Dispatch(ctx, securitynotificationusecases.DispatchDeps{
		UserRepo:            d.UserRepo,
		Preferences:         m.NotificationPreferenceRepo,
		KnownDevices:        m.KnownSignInDeviceRepo,
		Notifier:            d.Notifier,
		TenantSource:        d.TenantSource,
		SaltStore:           d.SaltStore,
		IssuerResolver:      d.IssuerResolver,
		SystemDefaultLocale: d.SystemDefaultLocale,
		Emit:                d.Emit,
	}, event)
}

// RunRetentionSweep は、監査イベント、認証イベントの集計、期限切れのログインセッション、使われなくなった
// 既知の端末を保持期間に従って削除する。期限切れの削除を持つ保存先が一つもなければ ran を false にする。
func (m Module) RunRetentionSweep(
	ctx context.Context, auditEvents auditports.AuditEventRepository, now time.Time,
) (result authusecases.RetentionSweepResult, ran bool, err error) {
	audit, _ := auditEvents.(authusecases.AuditEventPurger)
	buckets, _ := m.AuthEventBucketStore.(authusecases.AuthEventBucketPurger)
	sessions, _ := m.SessionStore.(authusecases.SessionPurger)
	knownDevices, _ := m.KnownSignInDeviceRepo.(authusecases.KnownDevicePurger)
	stores := authusecases.RetentionStores{
		Audit: audit, Buckets: buckets, Sessions: sessions, KnownDevices: knownDevices,
	}
	if stores.Empty() {
		return result, false, nil
	}
	result, err = authusecases.RunRetentionSweep(ctx, stores, authusecases.DefaultRetentionPolicy(), now)
	return result, true, err
}

// RouteDeps は Authentication の経路が共通基盤とほかのモジュールから受け取る依存である。
type RouteDeps struct {
	support.Deps
	*support.Authenticator
	AuditEventRepo            auditports.AuditEventRepository
	UserRepo                  userports.UserRepository
	Consents                  consentports.AccountConsents
	ClientDisplayNameResolver authhttpdeps.ClientDisplayNameResolver
	TenantRepo                tenantports.TenantRepository
	EmailSender               sharednotification.EmailSender
	Notifier                  sharednotification.Notifier
	TenantSaltStore           auditports.TenantSaltStore
	ClientIP                  func(*http.Request) string
	DefaultSignInPolicyRepo   appports.DefaultSignInPolicyRepository
}

// RegisterRoutes は、対話のログイン、アカウントの資格情報、MFA、セッションの経路を登録する。
// セッションの失効は OAuth のトークンの失効を伴い、対話のログインは認証を終えたら認可要求を続けるので、
// sessionTokens と authorizations を必須とする。
func (m Module) RegisterRoutes(
	g *echo.Group, d RouteDeps, sessionTokens authhttpdeps.SessionTokenRevoker, authorizations signinports.PendingAuthorizations,
) {
	authhttp.RegisterRoutes(g, m.httpDeps(d), sessionTokens, authorizations)
}

// SignInSteps は、既存のセッションに追加の認証を求める操作を返す。OAuth2 の認可が使う。
func (m Module) SignInSteps(d RouteDeps) signinhttp.Steps {
	return signinhttp.Steps{Deps: signinhttp.Deps{Deps: m.httpDeps(d)}}
}

// RegisterFederationRoutes は外部の IdP によるログインの経路を登録する。provisionUser は、
// 初めてログインした利用者を IdManagement の User として作る。
func (m Module) RegisterFederationRoutes(
	g *echo.Group, d RouteDeps,
	provisionUser func(context.Context, FederatedClaims, time.Time) (*userdomain.User, error),
) {
	oidcClient := oidcprotocol.Client{SecretResolver: m.FederationSecretResolver}
	broker := federationusecases.BrokerDeps{
		Connections: m.FederationConnectionRepo,
		Identities:  m.FederationIdentityRepo,
		Attempts:    m.FederationAttemptStore,
		Users:       d.UserRepo,
		Sessions:    m.SessionManager,
		Drivers: map[federationdomain.Protocol]federationusecases.ProtocolDriver{
			federationdomain.ProtocolOIDC: federationhttp.OIDCDriver{Client: oidcClient},
			federationdomain.ProtocolSAML: federationhttp.SAMLDriver{Replay: m.FederationReplayStore},
		},
		ProvisionUser: provisionUser,
		Emit:          d.Emit,
	}
	federationhttp.RegisterRoutes(g, federationhttp.Deps{Broker: broker, Auth: m.httpDeps(d), OIDC: &oidcClient})
}

func (m Module) httpDeps(d RouteDeps) authhttp.Deps {
	return authhttp.Deps{
		Deps:                      d.Deps,
		Authenticator:             d.Authenticator,
		SessionManager:            m.SessionManager,
		AuditEventRepo:            d.AuditEventRepo,
		UserRepo:                  d.UserRepo,
		PasswordHasher:            m.PasswordHasher,
		PasswordHistoryRepo:       m.PasswordHistoryRepo,
		Consents:                  d.Consents,
		ClientDisplayNameResolver: d.ClientDisplayNameResolver,
		MfaFactorRepo:             m.MfaFactorRepo,
		MfaEnrollmentBypassRepo:   m.MfaEnrollmentBypassRepo,
		AuthEventBucketStore:      m.AuthEventBucketStore,
		TenantRepo:                d.TenantRepo,
		PasswordResetTokenStore:   m.PasswordResetTokenStore,
		EmailSender:               d.EmailSender,
		Notifier:                  d.Notifier,
		BreachedPasswordChecker:   m.BreachedPasswordChecker,
		WebAuthnRP:                m.WebAuthnRP,
		WebAuthnCredentialRepo:    m.WebAuthnCredentialRepo,
		WebAuthnSessionStore:      m.WebAuthnSessionStore,
		RecoveryCodeRepo:          m.RecoveryCodeRepo,
		TrustedDeviceRepo:         m.TrustedDeviceRepo,
		LoginAttemptThrottle:      m.LoginAttemptThrottle,
		SentinelPasswordHash:      m.SentinelPasswordHash,
		TenantSaltStore:           d.TenantSaltStore,
		ClientIP:                  d.ClientIP,

		NotificationPreferenceRepo: m.NotificationPreferenceRepo,
		DefaultSignInPolicyRepo:    d.DefaultSignInPolicyRepo,
	}
}

// SessionCookie はログインセッションの Cookie の名前である。
const SessionCookie = sessionusecases.SessionCookie

// DemoHeaderResolver は、X-Demo-Sub ヘッダーから認証文脈を作る解決器である。IdMagic の外の結合テストが
// ログインを経ずに主体を与えるときに使う。
type DemoHeaderResolver = authusecases.DemoHeaderResolver

// GenerateTOTP は、Base32 の秘密と時刻から TOTP のコードを計算する。結合テストが MFA の入力を作るときに使う。
func GenerateTOTP(secretBase32 string, unixSeconds int64) (string, error) {
	return totpusecases.GenerateTOTP(secretBase32, unixSeconds)
}
