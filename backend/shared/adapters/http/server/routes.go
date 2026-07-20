// Package server: Echo v5 を用いた HTTP アダプタの router。
package server

import (
	"net/http"

	"github.com/ambi/idmagic/backend/application"
	"github.com/ambi/idmagic/backend/audit"
	audithttp "github.com/ambi/idmagic/backend/audit/adapters/http"
	"github.com/ambi/idmagic/backend/authentication"
	authhttp "github.com/ambi/idmagic/backend/authentication/adapters/http"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	recoveryports "github.com/ambi/idmagic/backend/authentication/recovery/ports"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	webauthnports "github.com/ambi/idmagic/backend/authentication/webauthn/ports"
	"github.com/ambi/idmagic/backend/idgovernance"
	ighttp "github.com/ambi/idmagic/backend/idgovernance/adapters/http"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmhttp "github.com/ambi/idmagic/backend/idmanagement/adapters/http"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2http "github.com/ambi/idmagic/backend/oauth2/adapters/http"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/provisioning"
	"github.com/ambi/idmagic/backend/saml"
	"github.com/ambi/idmagic/backend/scim"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification"
	"github.com/ambi/idmagic/backend/signingkeys"
	signinghttp "github.com/ambi/idmagic/backend/signingkeys/adapters/http"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancyhttp "github.com/ambi/idmagic/backend/tenancy/adapters/http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/ambi/idmagic/backend/wsfederation"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/adapters/samltoken"

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
	// Deprecated: tests may still provide the legacy direct field.
	KeyStore         signingports.KeyStore
	Audit            audit.Module
	JWKResolver      *crypto.JWKResolver
	WsFederation     wsfederation.Module
	Saml             saml.Module
	Scim             scim.Module
	FederationSigner *samltoken.Signer
	Application      application.Module
	Jobs             jobs.Module
	Provisioning     provisioning.Module

	// WebAuthn / Passkey と backup recovery code (wi-26)。WebAuthnRP が nil の場合 WebAuthn は無効。
}

func Register(e *echo.Echo, d Deps) {
	d.OAuth2 = mergeLegacyOAuth2Deps(d.OAuth2, d)
	d.Authentication = mergeLegacyAuthenticationDeps(d.Authentication, d)
	d.IdManagement = mergeLegacyIdManagementDeps(d.IdManagement, d)
	d.Notification = mergeLegacyNotificationDeps(d.Notification, d)
	d.Tenancy = mergeLegacyTenancyDeps(d.Tenancy, d)
	registerTenantRoutes(e.Group("", d.ResolveDefaultTenant), d)
	registerTenantRoutes(e.Group("/realms/:tenant_id", d.ResolvePathTenant), d)

	authenticator := &support.Authenticator{
		UserRepo:          d.IdManagement.UserRepo,
		GroupRepo:         d.IdManagement.GroupRepo,
		SessionManager:    d.Authentication.SessionManager,
		TokenIntrospector: d.OAuth2.TokenIntrospector,
		AuthnResolver:     d.Authentication.AuthnResolver,
	}

	controlPlane := e.Group("/realms/"+tenancydomain.DefaultRealm, d.ResolveControlPlaneTenant)
	tenancyhttp.RegisterControlPlaneRoutes(controlPlane, tenancyhttp.Deps{
		Deps:           d.Deps,
		Authenticator:  authenticator,
		TenantRepo:     d.TenantRepo,
		AttrSchemaRepo: d.Tenancy.AttrSchemaRepo,
		UserRepo:       d.IdManagement.UserRepo,
	})
	tenancyhttp.RegisterControlPlaneRoutes(e.Group("", d.ResolveDefaultTenant), tenancyhttp.Deps{
		Deps:           d.Deps,
		Authenticator:  authenticator,
		TenantRepo:     d.TenantRepo,
		AttrSchemaRepo: d.Tenancy.AttrSchemaRepo,
		UserRepo:       d.IdManagement.UserRepo,
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
	return module
}

func registerTenantRoutes(g *echo.Group, d Deps) {
	if d.SigningKeys.KeyStore == nil {
		d.SigningKeys.KeyStore = d.KeyStore
	}
	authenticator := &support.Authenticator{
		UserRepo:          d.IdManagement.UserRepo,
		GroupRepo:         d.IdManagement.GroupRepo,
		SessionManager:    d.Authentication.SessionManager,
		TokenIntrospector: d.OAuth2.TokenIntrospector,
		AuthnResolver:     d.Authentication.AuthnResolver,
	}

	appGate := d.Application.Gate(d.IdManagement.GroupRepo, d.TrustedForwardedHops)
	clientDisplayNames := d.Application.ClientDisplayNames(d.OAuth2.ClientRepo)

	oauth2http.RegisterRoutes(g, oauth2http.Deps{
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
		TokenIntrospector:          d.OAuth2.TokenIntrospector,
		IDTokenHintVerifier:        d.OAuth2.IDTokenHintVerifier,
		AccessTokenDenylist:        d.OAuth2.AccessTokenDenylist,
		AttrSchemaRepo:             d.Tenancy.AttrSchemaRepo,
		AuthEventBucketStore:       d.Authentication.AuthEventBucketStore,
		Authorizer:                 d.OAuth2.Authorizer,
		SentinelPasswordHash:       d.Authentication.SentinelPasswordHash,
		WebAuthnRP:                 d.Authentication.WebAuthnRP,
		WebAuthnCredentialRepo:     d.Authentication.WebAuthnCredentialRepo,
		WebAuthnSessionStore:       d.Authentication.WebAuthnSessionStore,
		RecoveryCodeRepo:           d.Authentication.RecoveryCodeRepo,
	})

	signinghttp.RegisterRoutes(g, signinghttp.Deps{
		Deps:          d.Deps,
		Authenticator: authenticator,
		KeyStore:      d.SigningKeys.KeyStore,
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

	authhttp.RegisterRoutes(g, authhttp.Deps{
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
		BreachedPasswordChecker:   d.Authentication.BreachedPasswordChecker,
		WebAuthnRP:                d.Authentication.WebAuthnRP,
		WebAuthnCredentialRepo:    d.Authentication.WebAuthnCredentialRepo,
		WebAuthnSessionStore:      d.Authentication.WebAuthnSessionStore,
		RecoveryCodeRepo:          d.Authentication.RecoveryCodeRepo,
	})

	idmhttp.RegisterRoutes(g, idmhttp.Deps{
		Deps:                  d.Deps,
		Authenticator:         authenticator,
		UserRepo:              d.IdManagement.UserRepo,
		GroupRepo:             d.IdManagement.GroupRepo,
		AgentRepo:             d.IdManagement.AgentRepo,
		UserMutationCommitter: d.IdManagement.UserMutationCommitter,
		ProvisioningNotifier:  d.IdManagement.ProvisioningNotifier,
		ClientRepo:            d.OAuth2.ClientRepo,
		ScimRepo:              d.Scim.Repo,
		AttrSchemaRepo:        d.Tenancy.AttrSchemaRepo,
		ConsentRepo:           d.OAuth2.ConsentRepo,
		RefreshStore:          d.OAuth2.RefreshStore,
		DeviceCodeStore:       d.OAuth2.DeviceCodeStore,
		MfaFactorRepo:         d.Authentication.MfaFactorRepo,
		PasswordHasher:        d.Authentication.PasswordHasher,
		PasswordHistoryRepo:   d.Authentication.PasswordHistoryRepo,
		EmailChangeTokenStore: d.IdManagement.EmailChangeTokenStore,
		EmailSender:           d.Notification.EmailSender,
		JobRepo:               d.Jobs.Repo,
	})

	ighttp.RegisterRoutes(g, ighttp.Deps{
		Deps: d.Deps, Authenticator: authenticator,
		LifecycleWorkflowRepo:    d.IdGovernance.LifecycleWorkflowRepo,
		LifecycleWorkflowRunRepo: d.IdGovernance.LifecycleWorkflowRunRepo,
		JobRepo:                  d.Jobs.Repo,
		UserRepo:                 d.IdManagement.UserRepo, GroupRepo: d.IdManagement.GroupRepo,
		ApplicationRepo: d.Application.Repo, AssignmentRepo: d.Application.AssignmentRepo,
		EmailSender: d.Notification.EmailSender,
	})

	tenancyhttp.RegisterRoutes(g, tenancyhttp.Deps{
		Deps:               d.Deps,
		Authenticator:      authenticator,
		TenantRepo:         d.TenantRepo,
		AttrSchemaRepo:     d.Tenancy.AttrSchemaRepo,
		BrandingRepo:       d.Tenancy.BrandingRepo,
		BrandingAssetStore: d.Tenancy.BrandingAssetStore,
		UserRepo:           d.IdManagement.UserRepo,
		GroupRepo:          d.IdManagement.GroupRepo,
	})

	d.WsFederation.Register(g, d.Deps, authenticator, appGate, d.IdManagement.UserRepo, d.FederationSigner,
		d.OAuth2.ClientAssertionReplayStore, d.Authentication.LoginAttemptThrottle, d.Authentication.PasswordHasher, d.Authentication.SentinelPasswordHash)

	d.Saml.Register(g, d.Deps, authenticator, appGate, d.IdManagement.UserRepo, d.FederationSigner)

	d.Application.Register(g, d.Deps, authenticator, d.IdManagement.GroupRepo, d.IdManagement.UserRepo, d.OAuth2.ClientRepo, d.WsFederation.RPRepo, d.Saml.SPRepo)

	d.Scim.Register(g, d.Deps, authenticator, d.IdManagement.UserRepo, d.IdManagement.GroupRepo, d.Emit)

	d.Provisioning.Register(g, d.Deps, authenticator, d.Application.AssignmentRepo, d.IdManagement.UserRepo)
}
