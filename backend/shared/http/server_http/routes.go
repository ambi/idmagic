// Package server: Echo v5 を用いた HTTP アダプタの router。
package server_http

import (
	"context"
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/signinpolicy"

	"github.com/ambi/idmagic/backend/accesscontrol"
	policyports "github.com/ambi/idmagic/backend/accesscontrol/ports"
	"github.com/ambi/idmagic/backend/apitoken"
	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	"github.com/ambi/idmagic/backend/application"
	"github.com/ambi/idmagic/backend/applicationcatalog"
	"github.com/ambi/idmagic/backend/audit"
	"github.com/ambi/idmagic/backend/authentication"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	recoveryports "github.com/ambi/idmagic/backend/authentication/recovery/ports"
	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	trusteddeviceports "github.com/ambi/idmagic/backend/authentication/trusteddevice/ports"
	webauthnports "github.com/ambi/idmagic/backend/authentication/webauthn/ports"
	"github.com/ambi/idmagic/backend/datakeys"
	"github.com/ambi/idmagic/backend/idgovernance"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/oauth2"
	consentports "github.com/ambi/idmagic/backend/oauth2/consent/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/provisioning"
	"github.com/ambi/idmagic/backend/saml"
	"github.com/ambi/idmagic/backend/shared/http/server_http/applicationdeps"
	"github.com/ambi/idmagic/backend/shared/http/server_http/sourcingdeps"

	"github.com/ambi/idmagic/backend/shared/http/server_http/tenancydeps"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	samltoken "github.com/ambi/idmagic/backend/shared/security/tokens_saml"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/sharedsignals"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	"github.com/ambi/idmagic/backend/sourcing"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/ambi/idmagic/backend/workloadidentity"
	workloadidentitydomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/ambi/idmagic/backend/wsfederation"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/labstack/echo/v5"
)

// Deps は HTTP アダプタ全体の起動に必要な全依存関係。
type Deps struct {
	support.Deps

	// MetricsHandler serves GET /metrics (system.yaml MetricsExposition). Nil
	// leaves the route unregistered, matching the endpoint's deploy-policy
	// gated exposure.
	MetricsHandler http.Handler

	// Admission は飽和時の入場制御の上限 (REQ-SYSTEM-018)。Enabled が false の
	// ゼロ値では何も拒否しないので、入場制御を組み立てないテストはそのまま動く。
	Admission support.AdmissionBudget

	Tenancy    tenancy.Module
	TenantRepo tenantports.TenantRepository
	// Deprecated: テスト用の互換入力。bootstrap は IdManagement.Module の保存先を設定する。
	AttrSchemaRepo userports.TenantUserAttributeSchemaRepository
	IdManagement   idmanagement.Module
	IdGovernance   idgovernance.Module
	// Deprecated: wi-178 移行中のテスト用互換入力。bootstrap は IdManagement.Module のみを設定する。
	UserRepo       userports.UserRepository
	GroupRepo      groupports.GroupRepository
	AgentRepo      agentports.AgentRepository
	Authentication authentication.Module
	Notification   sharednotification.Module
	// Deprecated: wi-177 移行中のテスト用互換入力。bootstrap は Authentication.Module のみを設定する。
	MfaFactorRepo           totpports.MfaFactorRepository
	PasswordHistoryRepo     passwordports.PasswordHistoryRepository
	EmailChangeTokenStore   userports.EmailChangeTokenStore
	AuthEventBucketStore    authentication.AuthEventBucketStore
	PasswordHasher          passwordports.PasswordHasher
	EmailSender             sharednotification.EmailSender
	BreachedPasswordChecker passwordports.BreachedPasswordChecker
	SentinelPasswordHash    string
	SessionManager          *authentication.SessionManager
	AuthnResolver           authdomain.AuthenticationContextResolver
	WebAuthnRP              *gowebauthn.WebAuthn
	WebAuthnCredentialRepo  webauthnports.WebAuthnCredentialRepository
	RecoveryCodeRepo        recoveryports.RecoveryCodeRepository
	TrustedDeviceRepo       trusteddeviceports.TrustedDeviceRepository
	OAuth2                  oauth2.Module
	// Deprecated: 移行中のテスト用互換入力。bootstrap は OAuth2.Module のみを設定する。
	TokenIssuer       oauthports.TokenIssuer
	TokenIntrospector oauthports.TokenIntrospector
	Authorizer        policyports.Authorizer
	SigningKeys       signingkeys.Module
	DataKeys          datakeys.Module
	// Deprecated: tests may still provide the legacy direct field.
	KeyStore         signingports.KeyStore
	Audit            audit.Module
	JWKResolver      *tokens_jose.JWKResolver
	WsFederation     wsfederation.Module
	Saml             saml.Module
	Sourcing         sourcing.Module
	FederationSigner samltoken.SignerProvider
	Application      application.Module
	SignInPolicy     signinpolicy.Module
	ApiTokens        apitoken.Module
	Jobs             jobs.Module
	Provisioning     provisioning.Module
	WorkloadIdentity workloadidentity.Module
	SharedSignals    sharedsignals.Module
	AccessControl    accesscontrol.Module

	// WebAuthn / Passkey と backup recovery code (wi-26)。WebAuthnRP が nil の場合 WebAuthn は無効。
}

func Register(e *echo.Echo, d Deps) {
	// エラーレスポンスの既定形式 (Problem Details) と型付きエラーの写像は経路の
	// 組み立てに属する。ここで入れないと、ハンドラのテストが本番と別のエラー経路を
	// 通ってしまい、500 に落ちる分類漏れがテストからは見えない。ログとメトリクスを
	// 持つ版は、起動時に cmd がこの後で差し替える。
	e.HTTPErrorHandler = support.ErrorHandler(nil, nil)
	// 入場制御は routing の後、どのハンドラーよりも前に立つ (REQ-SYSTEM-018)。
	// 経路を分類するのにルートパターンが要るので routing より後で、拒否した要求が
	// 状態を一切変えないためにハンドラーより前でなければならない。組み立ての側に
	// 置くのは、E2E がこの配線ごと通るようにするためである。
	e.Use(support.AdmissionMiddleware(d.Admission, ClassifyRoute, d.Metrics))
	e.Use(support.DeprecationHeadersMiddleware(d.Contract))
	d.OAuth2 = mergeLegacyOAuth2Deps(d.OAuth2, d)
	d.Authentication = mergeLegacyAuthenticationDeps(d.Authentication, d)
	d.IdManagement = mergeLegacyIdManagementDeps(d.IdManagement, d)
	d.Notification = mergeLegacyNotificationDeps(d.Notification, d)
	// テナントの正規ロケーションは 2 形あり、どちらか一方だけが有効になる。
	// subdomain style は origin 直下の bare path、path style は /realms/{realm} 配下。
	// 到達経路と endpoint_style の一致は resolver 側で確かめる。
	hostTenant, pathTenant := (tenancy.Module{TenantRepo: d.TenantRepo}).HTTPMiddleware(d.Issuer, d.TenantBaseDomain)
	registerTenantRoutes(e.Group("", hostTenant), d)
	tenantGroup := e.Group("/realms/:tenant_id", pathTenant)
	registerTenantRoutes(tenantGroup, d)

	// 制御面の経路は管理発行の API アクセストークンを受けない。
	authenticator := newAuthenticator(d, nil)

	// control-plane (テナント横断操作) は他の全ボンデッドコンテキストと同じ
	// tenantGroup にそのまま登録する。default テナントへの限定は
	// RequireControlPlaneUser が担う。
	// ルーティング層で別 prefix に隔離する必要はない。
	d.Tenancy.RegisterControlPlaneRoutes(tenantGroup, tenancy.RouteDeps{
		Deps:                  d.Deps,
		Authenticator:         authenticator,
		TenantRepo:            d.TenantRepo,
		TestNotificationActor: tenancydeps.TestNotificationActors(d.IdManagement.UserRepo),
	})

	e.GET("/health", d.handleHealth)
	e.GET("/livez", d.handleLivez)
	e.GET("/readyz", d.handleReadyz)
	e.GET("/startupz", d.handleStartupz)
	e.GET("/metrics", d.handleMetrics)
}

func mergeLegacyAuthenticationDeps(module authentication.Module, d Deps) authentication.Module {
	if module.MfaFactorRepo == nil {
		module.MfaFactorRepo = d.MfaFactorRepo
	}
	if module.PasswordHistoryRepo == nil {
		module.PasswordHistoryRepo = d.PasswordHistoryRepo
	}
	if module.AuthEventBucketStore == nil {
		module.AuthEventBucketStore = d.AuthEventBucketStore
	}
	if module.PasswordHasher == nil {
		module.PasswordHasher = d.PasswordHasher
	}
	if module.BreachedPasswordChecker == nil {
		module.BreachedPasswordChecker = d.BreachedPasswordChecker
	}
	if module.SentinelPasswordHash == "" {
		module.SentinelPasswordHash = d.SentinelPasswordHash
	}
	if module.SessionManager == nil {
		module.SessionManager = d.SessionManager
	}
	if module.AuthnResolver == nil {
		module.AuthnResolver = d.AuthnResolver
	}
	if module.WebAuthnRP == nil {
		module.WebAuthnRP = d.WebAuthnRP
	}
	if module.WebAuthnCredentialRepo == nil {
		module.WebAuthnCredentialRepo = d.WebAuthnCredentialRepo
	}
	if module.RecoveryCodeRepo == nil {
		module.RecoveryCodeRepo = d.RecoveryCodeRepo
	}
	if module.TrustedDeviceRepo == nil {
		module.TrustedDeviceRepo = d.TrustedDeviceRepo
	}
	if module.AuthnResolver == nil {
		module.AuthnResolver = module.SessionManager
	}
	return module
}

func mergeLegacyOAuth2Deps(module oauth2.Module, d Deps) oauth2.Module {
	if module.TokenIssuer == nil {
		module.TokenIssuer = d.TokenIssuer
	}
	if module.TokenIntrospector == nil {
		module.TokenIntrospector = d.TokenIntrospector
	}
	if module.Authorizer == nil {
		module.Authorizer = d.Authorizer
	}
	return module
}

func mergeLegacyIdManagementDeps(module idmanagement.Module, d Deps) idmanagement.Module {
	if module.AttrSchemaRepo == nil {
		module.AttrSchemaRepo = d.AttrSchemaRepo
	}
	if module.UserRepo == nil {
		module.UserRepo = d.UserRepo
	}
	if module.GroupRepo == nil {
		module.GroupRepo = d.GroupRepo
	}
	if module.AgentRepo == nil {
		module.AgentRepo = d.AgentRepo
	}
	if module.EmailChangeTokenStore == nil {
		module.EmailChangeTokenStore = d.EmailChangeTokenStore
	}
	return module
}

func mergeLegacyNotificationDeps(module sharednotification.Module, d Deps) sharednotification.Module {
	if module.EmailSender == nil {
		module.EmailSender = d.EmailSender
	}
	// 旧来の互換入力 (EmailSender だけを渡すテスト) からも通知テンプレートカタログを
	// 通した送信になるよう、Notifier を既定構成で組み立てる。テナント上書きは
	// Tenancy 由来なので、この経路では組込み既定だけを使う。
	if module.Notifier == nil && module.EmailSender != nil {
		module.Notifier = &template.Notifier{Sender: module.EmailSender}
	}
	return module
}

func registerTenantRoutes(g *echo.Group, d Deps) {
	if d.SigningKeys.KeyStore == nil {
		d.SigningKeys.KeyStore = d.KeyStore
	}
	if d.ApiTokens.TokenIssuer == nil {
		d.ApiTokens.TokenIssuer = d.OAuth2.TokenIssuer
	}
	if d.ApiTokens.TokenIntrospector == nil {
		d.ApiTokens.TokenIntrospector = d.OAuth2.TokenIntrospector
	}
	apiTokenService := d.ApiTokens.Service()
	authenticator := newAuthenticator(d, apiTokenService)

	appGate := d.Application.Gate(d.IdManagement.GroupRepo, d.TrustedForwardedHops, d.SignInPolicy.ApplicationRepo, d.SignInPolicy.DefaultRepo)
	clientDisplayNames := d.Application.ClientDisplayNames(applicationdeps.NewClientNames(d.OAuth2.ClientRepo))
	jobEnqueuer := d.Jobs.Enqueuer

	// 管理 API の User と Agent の操作は、IdManagement の経路の ReactiveEmit が発行の後にこの反応器を呼ぶ。
	// 管理 API の外から User を止める経路は、組み立ての地点の UserLifecycleCommands が同じ反応器を使う。
	revocationReactor := d.SharedSignals.AgentRevocationReactor(d.IdManagement.AgentRepo, d.SigningKeys.KeyStore, d.Issuer, d.Emit)
	// 以降の経路が発行する Authentication と OAuth2 のイベントは、記録の後に User の失効へ反応させる。
	d.Emit = userRevocationEmit(d.Emit, revocationReactor)

	// fetchWorkloadJWKS resolves a WorkloadTrustBundle's signing keys (inline
	// jwks or jwks_uri) via the shared, SSRF-safe JWKResolver (基盤の
	// 再利用). Shared by the admin JWKS-refresh action and
	// VerifyWorkloadAttestation's verification path.
	fetchWorkloadJWKS := func(ctx context.Context, bundle *workloadidentitydomain.WorkloadTrustBundle) ([]map[string]any, error) {
		return d.JWKResolver.ResolveJWKSSource(ctx, bundle.JWKSURI, bundle.JWKS)
	}
	workloadVerifier := d.OAuth2.WorkloadVerifier
	if workloadVerifier == nil {
		workloadVerifier = d.WorkloadIdentity.TokenVerifier(d.IdManagement.AgentRepo, fetchWorkloadJWKS, d.Emit)
	}

	// 対話のログインは Authentication が処理し、認証を終えたら OAuth2 の認可要求を続ける。
	// OAuth2 は既存のセッションに追加の認証を求めるときに Authentication の Steps を使う。
	authRoutes := authentication.RouteDeps{
		Deps:                    d.Deps,
		Authenticator:           authenticator,
		AuditEventRepo:          d.Audit.AuditEventRepo,
		UserRepo:                d.IdManagement.UserRepo,
		TenantRepo:              d.TenantRepo,
		EmailSender:             d.Notification.EmailSender,
		Notifier:                d.Notification.Notifier,
		TenantSaltStore:         d.Audit.TenantSaltStore,
		ClientIP:                appGate.ClientIP,
		DefaultSignInPolicyRepo: d.SignInPolicy.DefaultRepo,
	}
	oauthRoutes := oauth2.RouteDeps{
		WorkloadVerifier:          workloadVerifier,
		Deps:                      d.Deps,
		Authenticator:             authenticator,
		ApplicationGate:           appGate,
		ClientDisplayNameResolver: clientDisplayNames,
		KeyStore:                  d.SigningKeys.KeyStore,
		TenantRepo:                d.TenantRepo,
		UserRepo:                  d.IdManagement.UserRepo,
		JWKResolver:               d.JWKResolver,
		JobEnqueuer:               jobEnqueuer,
		Notifier:                  d.Notification.Notifier,
		AgentRepo:                 d.IdManagement.AgentRepo,
		RevocationEpochRepo:       d.SharedSignals.RevocationEpochRepo,
		UserRevocationEpochRepo:   d.SharedSignals.UserRevocationEpochRepo,
		ManagedTokenRevoker:       apiTokenService,
		BuiltinResource: oauth2.BuiltinResource{
			Scopes:             apitokendomain.AllScopes(),
			RevocationClientID: apitokendomain.BuiltinClientID,
		},
		AttrSchemaRepo: d.IdManagement.AttrSchemaRepo,
		QuotaRepo:      d.Tenancy.QuotaRepo,

		AuthnResolver: d.Authentication.AuthnResolver,
		SignIn:        d.Authentication.SignInSteps(authRoutes),

		ApplicationSignInPolicyRepo: d.SignInPolicy.ApplicationRepo,
		DefaultSignInPolicyRepo:     d.SignInPolicy.DefaultRepo,

		RelationChecker: d.AccessControl.RelationChecker(accesscontrol.RelationCheckerDeps{
			Agents: d.IdManagement.AgentRepo, Users: d.IdManagement.UserRepo,
			Authorizer: d.OAuth2.Authorizer, Emit: d.Emit,
		}),
	}
	// nil の *SessionManager を包んだ値はインターフェースの nil の確認をすり抜けるので、
	// ないときはポートごと渡さない。
	if d.Authentication.SessionManager != nil {
		oauthRoutes.LoginSessions = d.Authentication.LogoutSessions(d.Emit)
	}
	d.OAuth2.RegisterRoutes(g, oauthRoutes)

	d.WorkloadIdentity.RegisterRoutes(g, workloadidentity.RouteDeps{
		Deps: d.Deps, Authenticator: authenticator,
		AgentRepo: d.IdManagement.AgentRepo, FetchJWKS: fetchWorkloadJWKS,
	})

	d.AccessControl.RegisterRoutes(g, accesscontrol.RouteDeps{
		Deps: d.Deps, Authenticator: authenticator,
		Agents: d.IdManagement.AgentRepo, Users: d.IdManagement.UserRepo,
		Authorizer: d.OAuth2.Authorizer,
	})

	d.SharedSignals.RegisterRoutes(g, sharedsignals.RouteDeps{
		Deps: d.Deps, Authenticator: authenticator,
		AgentRepo: d.IdManagement.AgentRepo, JWKResolver: d.JWKResolver,
		Users: d.IdManagement.UserRepo, EndUserSessions: endUserSessions(d.Authentication.SessionStore),
		QuotaRepo: d.Tenancy.QuotaRepo,
		Emit: func(event spec.DomainEvent) error {
			if d.Emit != nil {
				d.Emit(event)
			}
			return nil
		},
	})

	d.SigningKeys.RegisterRoutes(g, signingkeys.RouteDeps{
		Deps:          d.Deps,
		Authenticator: authenticator,
		TenantRepo:    d.TenantRepo,
	})

	d.Jobs.RegisterRoutes(g, d.Deps, authenticator, d.Emit)

	d.DataKeys.RegisterRoutes(g, datakeys.RouteDeps{
		Deps:          d.Deps,
		Authenticator: authenticator,
		TenantRepo:    d.TenantRepo,
		Jobs:          d.Jobs.Enqueuer,
	})

	d.Audit.RegisterRoutes(g, audit.RouteDeps{
		Deps:          d.Deps,
		Authenticator: authenticator,
		// d.UserRepo (トップレベル) は wi-178 移行中の非推奨テスト互換フィールドで、実運用の
		// bootstrap では設定されない。username -> user_id 解決 (wi-147) には
		// d.IdManagement.UserRepo を使う。
		LookupUsername: auditUsernameLookup(d.IdManagement.UserRepo),
		Jobs:           d.Jobs.Repo, Enqueuer: d.Jobs.Enqueuer, Quotas: d.Tenancy.QuotaRepo,
	})

	d.Authentication.RegisterRoutes(g, authRoutes, d.OAuth2.SessionTokenRevoker(), d.OAuth2.SignInContinuation(oauthRoutes))

	idmRoutes := idmanagement.RouteDeps{
		TenantRepo:              d.TenantRepo,
		AttrSchemaRepo:          d.IdManagement.AttrSchemaRepo,
		QuotaRepo:               d.Tenancy.QuotaRepo,
		ConsentRepo:             d.OAuth2.ConsentRepo,
		RefreshStore:            d.OAuth2.RefreshStore,
		DeviceCodeStore:         d.OAuth2.DeviceCodeStore,
		ApprovalRequestStore:    d.OAuth2.ApprovalRequestStore,
		UserCredentials:         d.Authentication.UserCredentials(),
		TrustedDevices:          d.Authentication.UserDeviceRevoker(),
		ValidateInitialPassword: authentication.ValidateInitialPassword,
		AccountIdentity:         authentication.AccountIdentity(authenticator),
		PasswordHasher:          d.Authentication.PasswordHasher,
		PasswordHistoryRepo:     d.Authentication.PasswordHistoryRepo,
		Deps:                    d.Deps,
		Authenticator:           authenticator,
		Reactor:                 revocationReactor,
		ClientRepo:              d.OAuth2.ClientRepo,
		AgentActivity:           newAuditAgentActivity(d.Audit.AuditEventRepo),
		SCIMUsers:               d.Sourcing.SCIMUserLookup(),
		SCIMGroups:              d.Sourcing.SCIMGroupLookup(),
		GroupAttrSchemaRepo:     d.IdManagement.GroupAttrSchemaRepo,
		Consents:                accountConsentExport(d.OAuth2.AccountConsents(d.Emit, d.Tenancy.QuotaRepo)),
		EmailSender:             d.Notification.EmailSender,
		Notifier:                d.Notification.Notifier,
		JobRepo:                 d.Jobs.Repo,
		JobEnqueuer:             jobEnqueuer,
	}
	if d.Authentication.SessionManager != nil {
		idmRoutes.SessionStore = d.Authentication.SessionManager.Store
	}
	provisionFederatedUser := d.IdManagement.FederatedUserProvisioner(idmRoutes)

	d.Authentication.RegisterFederationRoutes(g, authRoutes,
		func(ctx context.Context, claims authentication.FederatedClaims, now time.Time) (*userdomain.User, error) {
			var email, name *string
			if claims.Email != "" {
				email = &claims.Email
			}
			if claims.Name != "" {
				name = &claims.Name
			}
			return provisionFederatedUser(ctx, idmanagement.FederatedUserInput{
				PreferredUsername: claims.Username, Name: name, Email: email,
				EmailVerified: claims.EmailVerified, Now: now,
			})
		})

	d.IdManagement.RegisterRoutes(g, idmRoutes)

	d.IdGovernance.RegisterRoutes(g, idgovernance.RouteDeps{
		Deps: d.Deps, Authenticator: authenticator,
		JobEnqueuer: jobEnqueuer,
		UserRepo:    d.IdManagement.UserRepo, GroupRepo: d.IdManagement.GroupRepo,
		ApplicationRepo: d.Application.Repo, AssignmentRepo: d.Application.AssignmentRepo,
		Notifier:       d.Notification.Notifier,
		AttrSchemaRepo: d.IdManagement.AttrSchemaRepo,
	})

	d.Tenancy.RegisterRoutes(g, tenancy.RouteDeps{
		Deps:                  d.Deps,
		Authenticator:         authenticator,
		TenantRepo:            d.TenantRepo,
		Notifier:              d.Notification.Notifier,
		TestNotificationActor: tenancydeps.TestNotificationActors(d.IdManagement.UserRepo),
		FederationSigner:      d.FederationSigner,
	})

	// nil の *SessionManager をインターフェースへ入れると nil でなくなり、受け取り側の nil の確認を
	// すり抜けるので、ないときはポートごと渡さない。
	var wsfedSessionRevoker wsfederation.SessionRevoker
	var samlSessionRevoker saml.SessionRevoker
	if d.Authentication.SessionManager != nil {
		wsfedSessionRevoker = d.Authentication.SessionManager
		samlSessionRevoker = d.Authentication.SessionManager
	}
	// 取り込み元の所有は SCIM の保存先を引くので、Sourcing を配線していない構成では渡さない。
	var sourceOwnership userports.UserSourceOwnershipGuard
	if d.Sourcing.ScimRepo != nil {
		sourceOwnership = d.Sourcing.UserOwnershipGuard()
	}
	d.WsFederation.Register(g, d.Deps, authenticator, appGate,
		wsfederation.Sessions{
			AuthnResolver: d.Authentication.AuthnResolver, SessionManager: wsfedSessionRevoker,
			Negotiate: d.Authentication.NegotiateSignIn(authRoutes),
		},
		d.Authentication.SignInSteps(authRoutes), d.IdManagement.UserRepo, d.FederationSigner,
		d.OAuth2.ClientAssertionReplayStore, d.Authentication.PasswordAuthentication(authRoutes),
		d.IdManagement.AttrSchemaRepo, sourceOwnership)

	d.Saml.Register(g, d.Deps, authenticator, appGate,
		saml.Sessions{AuthnResolver: d.Authentication.AuthnResolver, SessionManager: samlSessionRevoker},
		d.IdManagement.UserRepo, d.FederationSigner, d.IdManagement.AttrSchemaRepo)

	applicationcatalog.Register(g, applicationcatalog.Dependencies{
		Deps: d.Deps, Authenticator: authenticator,
		ApplicationRepo:             d.Application.Repo,
		ApplicationIconStore:        d.Application.IconStore,
		ApplicationAssignmentRepo:   d.Application.AssignmentRepo,
		ApplicationOrderingRepo:     d.Application.OrderingRepo,
		ApplicationCategoryRepo:     d.Application.CategoryRepo,
		PolicyApplications:          d.Application,
		ApplicationSignInPolicyRepo: d.SignInPolicy.ApplicationRepo,
		DefaultSignInPolicyRepo:     d.SignInPolicy.DefaultRepo,
		AssignmentSubjectDirectory:  application.SubjectDirectory(d.IdManagement.UserRepo, d.IdManagement.GroupRepo),
		GroupRepo:                   d.IdManagement.GroupRepo, UserRepo: d.IdManagement.UserRepo,
		Clients:              d.OAuth2.ClientAdministration(d.Emit, d.Tenancy.QuotaRepo),
		ClientRepo:           d.OAuth2.ClientRepo,
		RelyingParties:       d.WsFederation.RelyingPartyAdministration(),
		ServiceProviders:     d.Saml.ServiceProviderAdministration(),
		ProvisioningNotifier: d.Application.ProvisioningNotifier,
		QuotaRepo:            d.Tenancy.QuotaRepo,
		AttrSchemaRepo:       d.IdManagement.AttrSchemaRepo,
	})

	d.ApiTokens.Register(g, d.Deps, authenticator)

	sourceLifecycle := idmanagement.LifecycleDeps{AttrSchemaRepo: d.IdManagement.AttrSchemaRepo, QuotaRepo: d.Tenancy.QuotaRepo}
	if d.Sourcing.ScimRepo != nil && d.Sourcing.Writes == nil {
		d.Sourcing.Writes = sourcingdeps.NewWrites(d.IdManagement, sourceLifecycle, d.Sourcing, d.Emit)
	}
	if d.Sourcing.ScimRepo != nil && d.Sourcing.DirectoryWrites == nil {
		d.Sourcing.DirectoryWrites = sourcingdeps.NewDirectoryWrites(d.IdManagement, sourceLifecycle, d.Sourcing, d.Emit)
	}
	if d.Sourcing.ScimRepo != nil && d.Sourcing.DirectoryUserLifecycle == nil {
		lifecycle := sourceLifecycle
		lifecycle.TrustedDevices = d.Authentication.UserDeviceRevoker()
		d.Sourcing.DirectoryUserLifecycle = d.IdManagement.UserLifecycleCommands(lifecycle, func(event spec.DomainEvent) error {
			d.Emit(event)
			return revocationReactor.React(context.Background(), event)
		}, "directory_connector")
	}
	d.Sourcing.Register(g, d.Deps, authenticator, d.IdManagement.UserRepo, d.IdManagement.GroupRepo, d.Emit, apiTokenService,
		d.OAuth2.ClientAssertionReplayStore)

	d.Provisioning.Register(g, d.Deps, authenticator, d.Application.AssignmentRepo, d.IdManagement.UserRepo, d.IdManagement.GroupRepo)
}

// accountConsentExport は、同意の保存型を本人データに含める表示値へ写す。
// 本人の絞り込みと Granted の判定は OAuth2 に残す。
func accountConsentExport(consents consentports.AccountConsents) func(context.Context, string) ([]userports.AccountConsent, error) {
	return func(ctx context.Context, sub string) ([]userports.AccountConsent, error) {
		granted, err := consents.ListGranted(ctx, sub)
		if err != nil {
			return nil, err
		}
		result := make([]userports.AccountConsent, len(granted))
		for i, consent := range granted {
			result[i] = userports.AccountConsent{
				ClientID: consent.ClientID, Scopes: append([]string(nil), consent.Scopes...),
				State: string(consent.State), GrantedAt: consent.GrantedAt, ExpiresAt: consent.ExpiresAt,
			}
		}
		return result, nil
	}
}
