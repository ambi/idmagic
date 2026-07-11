// Package server: Echo v5 を用いた HTTP アダプタの router。
package server

import (
	"github.com/ambi/idmagic/backend/application"
	"github.com/ambi/idmagic/backend/audit"
	audithttp "github.com/ambi/idmagic/backend/audit/adapters/http"
	"github.com/ambi/idmagic/backend/authentication"
	authhttp "github.com/ambi/idmagic/backend/authentication/adapters/http"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/identitymanagement"
	idmhttp "github.com/ambi/idmagic/backend/identitymanagement/adapters/http"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2http "github.com/ambi/idmagic/backend/oauth2/adapters/http"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/saml"
	"github.com/ambi/idmagic/backend/scim"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	sharedeventlog "github.com/ambi/idmagic/backend/shared/eventlog"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/txrunner"
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

	Tenancy tenancy.Module
	// Deprecated: wi-179 移行中のテスト用互換入力。bootstrap は Tenancy.Module のみを設定する。
	AttrSchemaRepo     tenantports.TenantUserAttributeSchemaRepository
	IdentityManagement identitymanagement.Module
	// Deprecated: wi-178 移行中のテスト用互換入力。bootstrap は IdentityManagement.Module のみを設定する。
	UserRepo       idmports.UserRepository
	GroupRepo      idmports.GroupRepository
	AgentRepo      idmports.AgentRepository
	Authentication authentication.Module
	// Deprecated: wi-177 移行中のテスト用互換入力。bootstrap は Authentication.Module のみを設定する。
	MfaFactorRepo           authnports.MfaFactorRepository
	PasswordHistoryRepo     authnports.PasswordHistoryRepository
	EmailChangeTokenStore   authnports.EmailChangeTokenStore
	AuthEventBucketStore    authnports.AuthEventBucketStore
	PasswordHasher          authnports.PasswordHasher
	EmailSender             authnports.EmailSender
	BreachedPasswordChecker authnports.BreachedPasswordChecker
	SentinelPasswordHash    string
	SessionManager          *authusecases.SessionManager
	AuthnResolver           authdomain.AuthenticationContextResolver
	WebAuthnRP              *gowebauthn.WebAuthn
	WebAuthnCredentialRepo  authnports.WebAuthnCredentialRepository
	RecoveryCodeRepo        authnports.RecoveryCodeRepository
	OAuth2                  oauth2.Module
	// Deprecated: 移行中のテスト用互換入力。bootstrap は OAuth2.Module のみを設定する。
	TokenIssuer       oauthports.TokenIssuer
	TokenIntrospector oauthports.TokenIntrospector
	Authorizer        oauthports.Authorizer
	KeyStore          oauthports.KeyStore
	Audit             audit.Module
	JWKResolver       *crypto.JWKResolver
	WsFederation      wsfederation.Module
	Saml              saml.Module
	Scim              scim.Module
	FederationSigner  *samltoken.Signer
	Application       application.Module

	// WebAuthn / Passkey と backup recovery code (wi-26)。WebAuthnRP が nil の場合 WebAuthn は無効。

	// TxRunner and EventLogRecorder implement wi-184 T003's transaction-bound
	// event log (ADR-094).
	TxRunner         txrunner.Runner
	EventLogRecorder sharedeventlog.Recorder
}

func Register(e *echo.Echo, d Deps) {
	d.OAuth2 = mergeLegacyOAuth2Deps(d.OAuth2, d)
	d.Authentication = mergeLegacyAuthenticationDeps(d.Authentication, d)
	d.IdentityManagement = mergeLegacyIdentityManagementDeps(d.IdentityManagement, d)
	d.Tenancy = mergeLegacyTenancyDeps(d.Tenancy, d)
	registerTenantRoutes(e.Group("", d.ResolveDefaultTenant), d)
	registerTenantRoutes(e.Group("/realms/:tenant_id", d.ResolvePathTenant), d)

	authenticator := &support.Authenticator{
		UserRepo:          d.IdentityManagement.UserRepo,
		GroupRepo:         d.IdentityManagement.GroupRepo,
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
		UserRepo:       d.IdentityManagement.UserRepo,
	})
	tenancyhttp.RegisterControlPlaneRoutes(e.Group("", d.ResolveDefaultTenant), tenancyhttp.Deps{
		Deps:           d.Deps,
		Authenticator:  authenticator,
		TenantRepo:     d.TenantRepo,
		AttrSchemaRepo: d.Tenancy.AttrSchemaRepo,
		UserRepo:       d.IdentityManagement.UserRepo,
	})

	e.GET("/health", d.handleHealth)
	e.GET("/livez", d.handleLivez)
	e.GET("/readyz", d.handleReadyz)
	e.GET("/startupz", d.handleStartupz)
}

func mergeLegacyAuthenticationDeps(module authentication.Module, d Deps) authentication.Module {
	if module.MfaFactorRepo == nil {
		module.MfaFactorRepo = d.MfaFactorRepo
	}
	if module.PasswordHistoryRepo == nil {
		module.PasswordHistoryRepo = d.PasswordHistoryRepo
	}
	if module.EmailChangeTokenStore == nil {
		module.EmailChangeTokenStore = d.EmailChangeTokenStore
	}
	if module.AuthEventBucketStore == nil {
		module.AuthEventBucketStore = d.AuthEventBucketStore
	}
	if module.PasswordHasher == nil {
		module.PasswordHasher = d.PasswordHasher
	}
	if module.EmailSender == nil {
		module.EmailSender = d.EmailSender
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

func mergeLegacyIdentityManagementDeps(module identitymanagement.Module, d Deps) identitymanagement.Module {
	if module.UserRepo == nil {
		module.UserRepo = d.UserRepo
	}
	if module.GroupRepo == nil {
		module.GroupRepo = d.GroupRepo
	}
	if module.AgentRepo == nil {
		module.AgentRepo = d.AgentRepo
	}
	return module
}

func registerTenantRoutes(g *echo.Group, d Deps) {
	authenticator := &support.Authenticator{
		UserRepo:          d.IdentityManagement.UserRepo,
		GroupRepo:         d.IdentityManagement.GroupRepo,
		SessionManager:    d.Authentication.SessionManager,
		TokenIntrospector: d.OAuth2.TokenIntrospector,
		AuthnResolver:     d.Authentication.AuthnResolver,
	}

	appGate := d.Application.Gate(d.IdentityManagement.GroupRepo, d.TrustedForwardedHops)
	clientDisplayNames := d.Application.ClientDisplayNames(d.OAuth2.ClientRepo)

	oauth2http.RegisterRoutes(g, oauth2http.Deps{
		Deps:                       d.Deps,
		Authenticator:              authenticator,
		ApplicationGate:            appGate,
		AuthzDetailTypeRepo:        d.OAuth2.AuthzDetailTypeRepo,
		ClientRepo:                 d.OAuth2.ClientRepo,
		ConsentRepo:                d.OAuth2.ConsentRepo,
		ClientDisplayNameResolver:  clientDisplayNames,
		KeyStore:                   d.KeyStore,
		TenantSaltStore:            d.Audit.TenantSaltStore,
		TenantRepo:                 d.TenantRepo,
		PARStore:                   d.OAuth2.PARStore,
		RequestStore:               d.OAuth2.RequestStore,
		UserRepo:                   d.IdentityManagement.UserRepo,
		PasswordHasher:             d.Authentication.PasswordHasher,
		LoginAttemptThrottle:       d.Authentication.LoginAttemptThrottle,
		MfaFactorRepo:              d.Authentication.MfaFactorRepo,
		CodeStore:                  d.OAuth2.CodeStore,
		JWKResolver:                d.JWKResolver,
		ClientAssertionReplayStore: d.OAuth2.ClientAssertionReplayStore,
		DeviceCodeStore:            d.OAuth2.DeviceCodeStore,
		DpopReplayStore:            d.OAuth2.DpopReplayStore,
		RefreshStore:               d.OAuth2.RefreshStore,
		TokenIssuer:                d.OAuth2.TokenIssuer,
		AgentRepo:                  d.IdentityManagement.AgentRepo,
		TokenIntrospector:          d.OAuth2.TokenIntrospector,
		AccessTokenDenylist:        d.OAuth2.AccessTokenDenylist,
		AttrSchemaRepo:             d.Tenancy.AttrSchemaRepo,
		AuthEventBucketStore:       d.Authentication.AuthEventBucketStore,
		Authorizer:                 d.OAuth2.Authorizer,
		SentinelPasswordHash:       d.Authentication.SentinelPasswordHash,
		WebAuthnRP:                 d.Authentication.WebAuthnRP,
		WebAuthnCredentialRepo:     d.Authentication.WebAuthnCredentialRepo,
		WebAuthnSessionStore:       d.Authentication.WebAuthnSessionStore,
		RecoveryCodeRepo:           d.Authentication.RecoveryCodeRepo,
		CommandRunner: sharedeventlog.CommandRunner{
			Transactions:  d.TxRunner,
			Recorder:      d.EventLogRecorder,
			LegacyEmit:    d.Emit,
			CorrelationID: logging.RequestIDFromContext,
		},
	})

	audithttp.RegisterRoutes(g, audithttp.Deps{
		Deps:            d.Deps,
		Authenticator:   authenticator,
		AuditEventRepo:  d.Audit.AuditEventRepo,
		TenantSaltStore: d.Audit.TenantSaltStore,
	})

	authhttp.RegisterRoutes(g, authhttp.Deps{
		Deps:                      d.Deps,
		Authenticator:             authenticator,
		AuditEventRepo:            d.Audit.AuditEventRepo,
		UserRepo:                  d.IdentityManagement.UserRepo,
		PasswordHasher:            d.Authentication.PasswordHasher,
		PasswordHistoryRepo:       d.Authentication.PasswordHistoryRepo,
		ConsentRepo:               d.OAuth2.ConsentRepo,
		ClientDisplayNameResolver: clientDisplayNames,
		AttrSchemaRepo:            d.Tenancy.AttrSchemaRepo,
		MfaFactorRepo:             d.Authentication.MfaFactorRepo,
		AuthEventBucketStore:      d.Authentication.AuthEventBucketStore,
		TenantRepo:                d.TenantRepo,
		PasswordResetTokenStore:   d.Authentication.PasswordResetTokenStore,
		EmailSender:               d.Authentication.EmailSender,
		BreachedPasswordChecker:   d.Authentication.BreachedPasswordChecker,
		WebAuthnRP:                d.Authentication.WebAuthnRP,
		WebAuthnCredentialRepo:    d.Authentication.WebAuthnCredentialRepo,
		WebAuthnSessionStore:      d.Authentication.WebAuthnSessionStore,
		RecoveryCodeRepo:          d.Authentication.RecoveryCodeRepo,
		TxRunner:                  d.TxRunner,
		EventLogRecorder:          d.EventLogRecorder,
	})

	idmhttp.RegisterRoutes(g, idmhttp.Deps{
		Deps:                  d.Deps,
		Authenticator:         authenticator,
		UserRepo:              d.IdentityManagement.UserRepo,
		GroupRepo:             d.IdentityManagement.GroupRepo,
		AgentRepo:             d.IdentityManagement.AgentRepo,
		ClientRepo:            d.OAuth2.ClientRepo,
		ScimRepo:              d.Scim.Repo,
		AttrSchemaRepo:        d.Tenancy.AttrSchemaRepo,
		ConsentRepo:           d.OAuth2.ConsentRepo,
		RefreshStore:          d.OAuth2.RefreshStore,
		DeviceCodeStore:       d.OAuth2.DeviceCodeStore,
		MfaFactorRepo:         d.Authentication.MfaFactorRepo,
		PasswordHasher:        d.Authentication.PasswordHasher,
		PasswordHistoryRepo:   d.Authentication.PasswordHistoryRepo,
		EmailChangeTokenStore: d.Authentication.EmailChangeTokenStore,
		EmailSender:           d.Authentication.EmailSender,
		TxRunner:              d.TxRunner,
		EventLogRecorder:      d.EventLogRecorder,
	})

	tenancyhttp.RegisterRoutes(g, tenancyhttp.Deps{
		Deps:           d.Deps,
		Authenticator:  authenticator,
		TenantRepo:     d.TenantRepo,
		AttrSchemaRepo: d.Tenancy.AttrSchemaRepo,
		UserRepo:       d.IdentityManagement.UserRepo,
	})

	d.WsFederation.Register(g, d.Deps, authenticator, appGate, d.IdentityManagement.UserRepo, d.FederationSigner,
		d.OAuth2.ClientAssertionReplayStore, d.Authentication.LoginAttemptThrottle, d.Authentication.PasswordHasher, d.Authentication.SentinelPasswordHash)

	d.Saml.Register(g, d.Deps, authenticator, appGate, d.IdentityManagement.UserRepo, d.FederationSigner)

	d.Application.Register(g, d.Deps, authenticator, d.IdentityManagement.GroupRepo, d.IdentityManagement.UserRepo, d.OAuth2.ClientRepo, d.WsFederation.RPRepo, d.Saml.SPRepo, sharedeventlog.CommandRunner{
		Transactions:  d.TxRunner,
		Recorder:      d.EventLogRecorder,
		LegacyEmit:    d.Emit,
		CorrelationID: logging.RequestIDFromContext,
	})

	d.Scim.Register(g, d.Deps, authenticator, d.IdentityManagement.UserRepo, d.IdentityManagement.GroupRepo, d.Emit)
}
