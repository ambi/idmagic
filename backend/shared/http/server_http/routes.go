// Package server: Echo v5 を用いた HTTP アダプタの router。
package server_http

import (
	"context"
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/apitoken"
	"github.com/ambi/idmagic/backend/application"
	"github.com/ambi/idmagic/backend/audit"
	audithttp "github.com/ambi/idmagic/backend/audit/handlers_http"
	"github.com/ambi/idmagic/backend/authentication"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationhttp "github.com/ambi/idmagic/backend/authentication/federation/handlers_http"
	oidcprotocol "github.com/ambi/idmagic/backend/authentication/federation/protocol_oidc"
	federationusecases "github.com/ambi/idmagic/backend/authentication/federation/usecases"
	authhttp "github.com/ambi/idmagic/backend/authentication/handlers_http"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	recoveryports "github.com/ambi/idmagic/backend/authentication/recovery/ports"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	webauthnports "github.com/ambi/idmagic/backend/authentication/webauthn/ports"
	"github.com/ambi/idmagic/backend/datakeys"
	datakeyshttp "github.com/ambi/idmagic/backend/datakeys/handlers_http"
	"github.com/ambi/idmagic/backend/idgovernance"
	ighttp "github.com/ambi/idmagic/backend/idgovernance/handlers_http"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	idmhttp "github.com/ambi/idmagic/backend/idmanagement/handlers_http"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2http "github.com/ambi/idmagic/backend/oauth2/handlers_http"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/provisioning"
	"github.com/ambi/idmagic/backend/saml"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/logging"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/sharedsignals"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	sharedsignalshttp "github.com/ambi/idmagic/backend/sharedsignals/handlers_http"
	"github.com/ambi/idmagic/backend/sharedsignals/sign_jose"
	sharedsignalsusecases "github.com/ambi/idmagic/backend/sharedsignals/usecases"
	sharedsignalsverifyjose "github.com/ambi/idmagic/backend/sharedsignals/verify_jose"
	"github.com/ambi/idmagic/backend/signingkeys"
	signinghttp "github.com/ambi/idmagic/backend/signingkeys/handlers_http"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	"github.com/ambi/idmagic/backend/sourcing"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancyhttp "github.com/ambi/idmagic/backend/tenancy/handlers_http"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/ambi/idmagic/backend/workloadidentity"
	workloadidentitydomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
	workloadidentityhttp "github.com/ambi/idmagic/backend/workloadidentity/handlers_http"
	workloadidentityusecases "github.com/ambi/idmagic/backend/workloadidentity/usecases"
	workloadidentityverification "github.com/ambi/idmagic/backend/workloadidentity/verification_jose"
	"github.com/ambi/idmagic/backend/wsfederation"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

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

	Tenancy tenancy.Module
	// Deprecated: wi-179 移行中のテスト用互換入力。bootstrap は Tenancy.Module のみを設定する。
	AttrSchemaRepo tenantports.TenantUserAttributeSchemaRepository
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
	AuthEventBucketStore    authnports.AuthEventBucketStore
	PasswordHasher          passwordports.PasswordHasher
	EmailSender             sharednotification.EmailSender
	BreachedPasswordChecker passwordports.BreachedPasswordChecker
	SentinelPasswordHash    string
	SessionManager          *sessionusecases.SessionManager
	AuthnResolver           authdomain.AuthenticationContextResolver
	WebAuthnRP              *gowebauthn.WebAuthn
	WebAuthnCredentialRepo  webauthnports.WebAuthnCredentialRepository
	RecoveryCodeRepo        recoveryports.RecoveryCodeRepository
	OAuth2                  oauth2.Module
	// Deprecated: 移行中のテスト用互換入力。bootstrap は OAuth2.Module のみを設定する。
	TokenIssuer       oauthports.TokenIssuer
	TokenIntrospector oauthports.TokenIntrospector
	Authorizer        oauthports.Authorizer
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
	ApiTokens        apitoken.Module
	Jobs             jobs.Module
	Provisioning     provisioning.Module
	WorkloadIdentity workloadidentity.Module
	SharedSignals    sharedsignals.Module

	// WebAuthn / Passkey と backup recovery code (wi-26)。WebAuthnRP が nil の場合 WebAuthn は無効。
}

func Register(e *echo.Echo, d Deps) {
	e.Use(support.DeprecationHeadersMiddleware(d.SCL))
	d.OAuth2 = mergeLegacyOAuth2Deps(d.OAuth2, d)
	d.Authentication = mergeLegacyAuthenticationDeps(d.Authentication, d)
	d.IdManagement = mergeLegacyIdManagementDeps(d.IdManagement, d)
	d.Notification = mergeLegacyNotificationDeps(d.Notification, d)
	d.Tenancy = mergeLegacyTenancyDeps(d.Tenancy, d)
	// テナントの正規ロケーションは 2 形あり、どちらか一方だけが有効になる (ADR-144)。
	// subdomain style は origin 直下の bare path、path style は /realms/{realm} 配下。
	// 到達経路と endpoint_style の一致は resolver 側で確かめる。
	registerTenantRoutes(e.Group("", d.ResolveHostTenant), d)
	tenantGroup := e.Group("/realms/:tenant_id", d.ResolvePathTenant)
	registerTenantRoutes(tenantGroup, d)

	authenticator := &support.Authenticator{
		UserRepo:          d.IdManagement.UserRepo,
		GroupRepo:         d.IdManagement.GroupRepo,
		SessionManager:    d.Authentication.SessionManager,
		TokenIntrospector: d.OAuth2.TokenIntrospector,
		DpopReplayStore:   d.OAuth2.DpopReplayStore,
		AuthnResolver:     d.Authentication.AuthnResolver,
	}

	// control-plane (テナント横断操作) は他の全ボンデッドコンテキストと同じ
	// tenantGroup にそのまま登録する。default テナントへの限定は
	// requireSystemAdmin (user.TenantID == DefaultTenantID) が担う (ADR-032) —
	// ルーティング層で別 prefix に隔離する必要はない。
	tenancyhttp.RegisterControlPlaneRoutes(tenantGroup, tenancyhttp.Deps{
		Deps:           d.Deps,
		Authenticator:  authenticator,
		TenantRepo:     d.TenantRepo,
		AttrSchemaRepo: d.Tenancy.AttrSchemaRepo,
		UserRepo:       d.IdManagement.UserRepo,
		QuotaRepo:      d.Tenancy.QuotaRepo,
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

func mergeLegacyTenancyDeps(module tenancy.Module, d Deps) tenancy.Module {
	if module.AttrSchemaRepo == nil {
		module.AttrSchemaRepo = d.AttrSchemaRepo
	}
	return module
}

func mergeLegacyIdManagementDeps(module idmanagement.Module, d Deps) idmanagement.Module {
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
	// Tenancy 由来なので、この経路では組込み既定だけを使う (ADR-142)。
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
	authenticator := &support.Authenticator{
		UserRepo: d.IdManagement.UserRepo, GroupRepo: d.IdManagement.GroupRepo,
		SessionManager: d.Authentication.SessionManager, TokenIntrospector: d.OAuth2.TokenIntrospector,
		ApiTokenAuthenticator: apiTokenService, DpopReplayStore: d.OAuth2.DpopReplayStore, AuthnResolver: d.Authentication.AuthnResolver,
	}

	appGate := d.Application.Gate(d.IdManagement.GroupRepo, d.TrustedForwardedHops)
	clientDisplayNames := d.Application.ClientDisplayNames(d.OAuth2.ClientRepo)

	// revocationReactor fail-closed reacts to already-emitted IdManagement
	// events (AgentKilled/AgentDisabled/AgentCredentialUnbound/UserDisabled/
	// UserSoftDeleted/UserDeleted) by advancing SharedSignals' Agent
	// revocation epoch (ADR-057, wi-58). Composed into idmhttp.Deps.Reactor,
	// which ReactiveEmit calls after every Emit.
	//
	// Its own Emit records the derived RevocationEpochAdvanced/
	// AgentAccessRevoked events, and best-effort fans AgentAccessRevoked out
	// to registered SSF Transmit streams (EcosystemPropagation, wi-58 T004).
	// This projection step is deliberately non-propagating: ecosystem
	// propagation must never block or delay the local revocation that just
	// succeeded (ADR-057 decision 6), so a projection failure is logged, not
	// returned.
	projectorDeps := sharedsignalsusecases.ProjectorDeps{
		StreamRepo: d.SharedSignals.StreamRepo, TransmitterConfigRepo: d.SharedSignals.TransmitterConfigRepo,
		DeliveryRepo: d.SharedSignals.DeliveryRepo, Signer: &sign_jose.Signer{KeyStore: d.SigningKeys.KeyStore}, Issuer: d.Issuer,
	}
	revocationReactor := &sharedsignalsusecases.AgentRevocationReactor{
		EpochRepo: d.SharedSignals.RevocationEpochRepo,
		AgentRepo: d.IdManagement.AgentRepo,
		Emit: func(event spec.DomainEvent) error {
			if d.Emit != nil {
				d.Emit(event)
			}
			if revoked, ok := event.(*ssdomain.AgentAccessRevoked); ok {
				projectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := sharedsignalsusecases.ProjectAgentAccessRevoked(projectCtx, projectorDeps, revoked); err != nil {
					logging.Error(projectCtx, "sharedsignals: SET projection failed", "error", err, "agent_id", revoked.AgentID, "tenant_id", revoked.TenantID)
				}
			}
			return nil
		},
	}

	// fetchWorkloadJWKS resolves a WorkloadTrustBundle's signing keys (inline
	// jwks or jwks_uri) via the shared, SSRF-safe JWKResolver (ADR-023 基盤の
	// 再利用、ADR-053). Shared by the admin JWKS-refresh action and
	// VerifyWorkloadAttestation's verification path.
	fetchWorkloadJWKS := func(ctx context.Context, bundle *workloadidentitydomain.WorkloadTrustBundle) ([]map[string]any, error) {
		return d.JWKResolver.ResolveJWKSSource(ctx, bundle.JWKSURI, bundle.JWKS)
	}
	workloadVerifier := workloadidentityusecases.WorkloadTokenVerifierAdapter{
		Deps: workloadidentityusecases.VerifyWorkloadAttestationDeps{
			TrustBundleRepo: d.WorkloadIdentity.TrustBundleRepo,
			BindingRepo:     d.WorkloadIdentity.BindingRepo,
			AgentRepo:       d.IdManagement.AgentRepo,
			SVIDVerifier:    workloadidentityverification.NewVerifier(),
			FetchJWKS:       fetchWorkloadJWKS,
			Emit:            d.Emit,
		},
	}

	oauth2http.RegisterRoutes(g, oauth2http.Deps{
		WorkloadVerifier:           workloadVerifier,
		Deps:                       d.Deps,
		Authenticator:              authenticator,
		ApplicationGate:            appGate,
		AuthzDetailTypeRepo:        d.OAuth2.AuthzDetailTypeRepo,
		McpResourceServerRepo:      d.OAuth2.McpResourceServerRepo,
		ClientRepo:                 d.OAuth2.ClientRepo,
		ConsentRepo:                d.OAuth2.ConsentRepo,
		ClientDisplayNameResolver:  clientDisplayNames,
		KeyStore:                   d.SigningKeys.KeyStore,
		TenantSaltStore:            d.Audit.TenantSaltStore,
		TenantRepo:                 d.TenantRepo,
		PARStore:                   d.OAuth2.PARStore,
		RequestStore:               d.OAuth2.RequestStore,
		UserRepo:                   d.IdManagement.UserRepo,
		PasswordHasher:             d.Authentication.PasswordHasher,
		LoginAttemptThrottle:       d.Authentication.LoginAttemptThrottle,
		MfaFactorRepo:              d.Authentication.MfaFactorRepo,
		MfaEnrollmentBypassRepo:    d.Authentication.MfaEnrollmentBypassRepo,
		CodeStore:                  d.OAuth2.CodeStore,
		JWKResolver:                d.JWKResolver,
		ClientAssertionReplayStore: d.OAuth2.ClientAssertionReplayStore,
		DeviceCodeStore:            d.OAuth2.DeviceCodeStore,
		DpopReplayStore:            d.OAuth2.DpopReplayStore,
		RefreshStore:               d.OAuth2.RefreshStore,
		TokenIssuer:                d.OAuth2.TokenIssuer,
		AgentRepo:                  d.IdManagement.AgentRepo,
		RevocationEpochRepo:        d.SharedSignals.RevocationEpochRepo,
		TokenIntrospector:          d.OAuth2.TokenIntrospector,
		IDTokenHintVerifier:        d.OAuth2.IDTokenHintVerifier,
		AccessTokenDenylist:        d.OAuth2.AccessTokenDenylist,
		ManagedTokenRevoker:        apiTokenService,
		AttrSchemaRepo:             d.Tenancy.AttrSchemaRepo,
		AuthEventBucketStore:       d.Authentication.AuthEventBucketStore,
		Authorizer:                 d.OAuth2.Authorizer,
		SentinelPasswordHash:       d.Authentication.SentinelPasswordHash,
		WebAuthnRP:                 d.Authentication.WebAuthnRP,
		WebAuthnCredentialRepo:     d.Authentication.WebAuthnCredentialRepo,
		WebAuthnSessionStore:       d.Authentication.WebAuthnSessionStore,
		RecoveryCodeRepo:           d.Authentication.RecoveryCodeRepo,
		QuotaRepo:                  d.Tenancy.QuotaRepo,
	})

	workloadidentityhttp.RegisterRoutes(g, workloadidentityhttp.Deps{
		Deps: d.Deps, Authenticator: authenticator,
		TrustBundleRepo: d.WorkloadIdentity.TrustBundleRepo, BindingRepo: d.WorkloadIdentity.BindingRepo,
		AgentRepo: d.IdManagement.AgentRepo, FetchJWKS: fetchWorkloadJWKS,
	})

	sharedsignalshttp.RegisterRoutes(g, sharedsignalshttp.Deps{
		Deps: d.Deps, Authenticator: authenticator,
		StreamRepo: d.SharedSignals.StreamRepo, TransmitterConfigRepo: d.SharedSignals.TransmitterConfigRepo,
		ReceiverConfigRepo: d.SharedSignals.ReceiverConfigRepo, DeliveryRepo: d.SharedSignals.DeliveryRepo,
		ReceivedEventRepo: d.SharedSignals.ReceivedEventRepo, EpochRepo: d.SharedSignals.RevocationEpochRepo,
		AgentRepo: d.IdManagement.AgentRepo, Verifier: &sharedsignalsverifyjose.Verifier{JWKResolver: d.JWKResolver},
		Emit: func(event spec.DomainEvent) error {
			if d.Emit != nil {
				d.Emit(event)
			}
			return nil
		},
	})

	signinghttp.RegisterRoutes(g, signinghttp.Deps{
		Deps:          d.Deps,
		Authenticator: authenticator,
		KeyStore:      d.SigningKeys.KeyStore,
		TenantRepo:    d.TenantRepo,
	})

	datakeyshttp.RegisterRoutes(g, datakeyshttp.Deps{
		Deps:          d.Deps,
		Authenticator: authenticator,
		Repository:    d.DataKeys.Repository,
		Crypto:        d.DataKeys.Crypto,
		TenantRepo:    d.TenantRepo,
	})

	audithttp.RegisterRoutes(g, audithttp.Deps{
		Deps:            d.Deps,
		Authenticator:   authenticator,
		AuditEventRepo:  d.Audit.AuditEventRepo,
		TenantSaltStore: d.Audit.TenantSaltStore,
		// d.UserRepo (トップレベル) は wi-178 移行中の非推奨テスト互換フィールドで、実運用の
		// bootstrap では設定されない。username -> user_id 解決 (wi-147) には
		// d.IdManagement.UserRepo を使う。
		UserRepo: d.IdManagement.UserRepo,
	})

	authDeps := authhttp.Deps{
		Deps:                      d.Deps,
		Authenticator:             authenticator,
		AuditEventRepo:            d.Audit.AuditEventRepo,
		UserRepo:                  d.IdManagement.UserRepo,
		PasswordHasher:            d.Authentication.PasswordHasher,
		PasswordHistoryRepo:       d.Authentication.PasswordHistoryRepo,
		ConsentRepo:               d.OAuth2.ConsentRepo,
		RefreshStore:              d.OAuth2.RefreshStore,
		ClientDisplayNameResolver: clientDisplayNames,
		AttrSchemaRepo:            d.Tenancy.AttrSchemaRepo,
		MfaFactorRepo:             d.Authentication.MfaFactorRepo,
		MfaEnrollmentBypassRepo:   d.Authentication.MfaEnrollmentBypassRepo,
		AuthEventBucketStore:      d.Authentication.AuthEventBucketStore,
		TenantRepo:                d.TenantRepo,
		PasswordResetTokenStore:   d.Authentication.PasswordResetTokenStore,
		EmailSender:               d.Notification.EmailSender,
		Notifier:                  d.Notification.Notifier,
		BreachedPasswordChecker:   d.Authentication.BreachedPasswordChecker,
		WebAuthnRP:                d.Authentication.WebAuthnRP,
		WebAuthnCredentialRepo:    d.Authentication.WebAuthnCredentialRepo,
		WebAuthnSessionStore:      d.Authentication.WebAuthnSessionStore,
		RecoveryCodeRepo:          d.Authentication.RecoveryCodeRepo,
	}
	authhttp.RegisterRoutes(g, authDeps)

	oidcClient := oidcprotocol.Client{SecretResolver: d.Authentication.FederationSecretResolver}
	brokerDeps := federationusecases.BrokerDeps{
		Connections: d.Authentication.FederationConnectionRepo,
		Identities:  d.Authentication.FederationIdentityRepo,
		Attempts:    d.Authentication.FederationAttemptStore,
		Users:       d.IdManagement.UserRepo,
		Sessions:    d.Authentication.SessionManager,
		Drivers: map[federationdomain.Protocol]federationusecases.ProtocolDriver{
			federationdomain.ProtocolOIDC: federationhttp.OIDCDriver{Client: oidcClient},
			federationdomain.ProtocolSAML: federationhttp.SAMLDriver{Replay: d.Authentication.FederationReplayStore},
		},
		ProvisionUser: func(ctx context.Context, claims federationdomain.NormalizedClaims, now time.Time) (*userdomain.User, error) {
			var email, name *string
			if claims.Email != "" {
				email = &claims.Email
			}
			if claims.Name != "" {
				name = &claims.Name
			}
			return userusecases.ProvisionFederatedUser(ctx, userusecases.AdminUserDeps{
				UserRepo: d.IdManagement.UserRepo, AttrSchemaRepo: d.Tenancy.AttrSchemaRepo,
				UserMutationCommitter: d.IdManagement.UserMutationCommitter,
				ProvisioningNotifier:  d.IdManagement.ProvisioningNotifier,
				QuotaRepo:             d.Tenancy.QuotaRepo,
				Emit: func(event spec.DomainEvent) error {
					if d.Emit != nil {
						d.Emit(event)
					}
					return nil
				},
			}, userusecases.ProvisionFederatedUserInput{
				PreferredUsername: claims.Username, Name: name, Email: email,
				EmailVerified: claims.EmailVerified, Now: now,
			})
		},
		Emit: d.Emit,
	}
	federationhttp.RegisterRoutes(g, federationhttp.Deps{Broker: brokerDeps, Auth: authDeps, OIDC: &oidcClient})

	idmhttp.RegisterRoutes(g, idmhttp.Deps{
		Deps:                  d.Deps,
		Authenticator:         authenticator,
		UserRepo:              d.IdManagement.UserRepo,
		GroupRepo:             d.IdManagement.GroupRepo,
		AgentRepo:             d.IdManagement.AgentRepo,
		UserMutationCommitter: d.IdManagement.UserMutationCommitter,
		ProvisioningNotifier:  d.IdManagement.ProvisioningNotifier,
		Reactor:               revocationReactor,
		ClientRepo:            d.OAuth2.ClientRepo,
		ScimRepo:              d.Sourcing.ScimRepo,
		AttrSchemaRepo:        d.Tenancy.AttrSchemaRepo,
		ConsentRepo:           d.OAuth2.ConsentRepo,
		RefreshStore:          d.OAuth2.RefreshStore,
		DeviceCodeStore:       d.OAuth2.DeviceCodeStore,
		MfaFactorRepo:         d.Authentication.MfaFactorRepo,
		PasswordHasher:        d.Authentication.PasswordHasher,
		PasswordHistoryRepo:   d.Authentication.PasswordHistoryRepo,
		EmailChangeTokenStore: d.IdManagement.EmailChangeTokenStore,
		EmailSender:           d.Notification.EmailSender,
		Notifier:              d.Notification.Notifier,
		JobRepo:               d.Jobs.Repo,
		QuotaRepo:             d.Tenancy.QuotaRepo,
	})

	ighttp.RegisterRoutes(g, ighttp.Deps{
		Deps: d.Deps, Authenticator: authenticator,
		LifecycleWorkflowRepo:    d.IdGovernance.LifecycleWorkflowRepo,
		LifecycleWorkflowRunRepo: d.IdGovernance.LifecycleWorkflowRunRepo,
		JobRepo:                  d.Jobs.Repo,
		UserRepo:                 d.IdManagement.UserRepo, GroupRepo: d.IdManagement.GroupRepo,
		ApplicationRepo: d.Application.Repo, AssignmentRepo: d.Application.AssignmentRepo,
		Notifier:  d.Notification.Notifier,
		QuotaRepo: d.Tenancy.QuotaRepo,
	})

	tenancyhttp.RegisterRoutes(g, tenancyhttp.Deps{
		Deps:               d.Deps,
		Authenticator:      authenticator,
		TenantRepo:         d.TenantRepo,
		AttrSchemaRepo:     d.Tenancy.AttrSchemaRepo,
		BrandingRepo:       d.Tenancy.BrandingRepo,
		BrandingAssetStore: d.Tenancy.BrandingAssetStore,

		NotificationTemplateRepo: d.Tenancy.NotificationTemplates,
		Notifier:                 d.Notification.Notifier,
		UserRepo:                 d.IdManagement.UserRepo,
		GroupRepo:                d.IdManagement.GroupRepo,
		QuotaRepo:                d.Tenancy.QuotaRepo,
		FederationSigner:         d.FederationSigner,
	})

	d.WsFederation.Register(g, d.Deps, authenticator, appGate, d.IdManagement.UserRepo, d.FederationSigner,
		d.OAuth2.ClientAssertionReplayStore, d.Authentication.LoginAttemptThrottle, d.Authentication.PasswordHasher, d.Authentication.SentinelPasswordHash,
		d.Tenancy.AttrSchemaRepo)

	d.Saml.Register(g, d.Deps, authenticator, appGate, d.IdManagement.UserRepo, d.FederationSigner, d.Tenancy.AttrSchemaRepo)

	d.Application.Register(g, d.Deps, authenticator, d.IdManagement.GroupRepo, d.IdManagement.UserRepo, d.OAuth2.ClientRepo, d.WsFederation.RPRepo, d.Saml.SPRepo, d.Tenancy.QuotaRepo, d.Tenancy.AttrSchemaRepo)

	d.ApiTokens.Register(g, d.Deps, authenticator)

	d.Sourcing.Register(g, d.Deps, authenticator, d.IdManagement.UserRepo, d.IdManagement.GroupRepo, d.Emit, apiTokenService)

	d.Provisioning.Register(g, d.Deps, authenticator, d.Application.AssignmentRepo, d.IdManagement.UserRepo)
}
