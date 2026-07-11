// Package server: Echo v5 を用いた HTTP アダプタの router。
package server

import (
	"github.com/ambi/idmagic/backend/application"
	"github.com/ambi/idmagic/backend/authentication"
	authhttp "github.com/ambi/idmagic/backend/authentication/adapters/http"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	idmhttp "github.com/ambi/idmagic/backend/identitymanagement/adapters/http"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2http "github.com/ambi/idmagic/backend/oauth2/adapters/http"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/saml"
	"github.com/ambi/idmagic/backend/scim"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancyhttp "github.com/ambi/idmagic/backend/tenancy/adapters/http"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/ambi/idmagic/backend/wsfederation"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/adapters/samltoken"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/labstack/echo/v5"
)

// Deps は HTTP アダプタ全体の起動に必要な全依存関係。
type Deps struct {
	support.Deps

	AttrSchemaRepo tenantports.TenantUserAttributeSchemaRepository
	UserRepo       idmports.UserRepository
	Authentication authentication.Module
	// Deprecated: wi-177 移行中のテスト用互換入力。bootstrap は Authentication.Module のみを設定する。
	MfaFactorRepo           authnports.MfaFactorRepository
	PasswordHistoryRepo     authnports.PasswordHistoryRepository
	PasswordResetTokenStore authnports.PasswordResetTokenStore
	EmailChangeTokenStore   authnports.EmailChangeTokenStore
	AuthEventBucketStore    authnports.AuthEventBucketStore
	PasswordHasher          authnports.PasswordHasher
	EmailSender             authnports.EmailSender
	BreachedPasswordChecker authnports.BreachedPasswordChecker
	LoginAttemptThrottle    authnports.LoginAttemptThrottle
	SentinelPasswordHash    string
	SessionManager          *authusecases.SessionManager
	AuthnResolver           authdomain.AuthenticationContextResolver
	WebAuthnRP              *gowebauthn.WebAuthn
	WebAuthnCredentialRepo  authnports.WebAuthnCredentialRepository
	WebAuthnSessionStore    authnports.WebAuthnSessionStore
	RecoveryCodeRepo        authnports.RecoveryCodeRepository
	OAuth2                  oauth2.Module
	// Deprecated: 移行中のテスト用互換入力。bootstrap は OAuth2.Module のみを設定する。
	RequestStore               oauthports.AuthorizationRequestStore
	CodeStore                  oauthports.AuthorizationCodeStore
	PARStore                   oauthports.PARStore
	RefreshStore               oauthports.RefreshTokenStore
	DeviceCodeStore            oauthports.DeviceCodeStore
	DpopReplayStore            oauthports.DpopReplayStore
	ClientAssertionReplayStore oauthports.ClientAssertionReplayStore
	AccessTokenDenylist        oauthports.AccessTokenDenylist
	TokenIssuer                oauthports.TokenIssuer
	TokenIntrospector          oauthports.TokenIntrospector
	Authorizer                 oauthports.Authorizer
	KeyStore                   oauthports.KeyStore
	TenantSaltStore            oauthports.TenantSaltStore
	JWKResolver                *crypto.JWKResolver
	GroupRepo                  idmports.GroupRepository
	AgentRepo                  idmports.AgentRepository
	WsFederation               wsfederation.Module
	Saml                       saml.Module
	Scim                       scim.Module
	FederationSigner           *samltoken.Signer
	Application                application.Module

	// WebAuthn / Passkey と backup recovery code (wi-26)。WebAuthnRP が nil の場合 WebAuthn は無効。
}

func Register(e *echo.Echo, d Deps) {
	d.OAuth2 = mergeLegacyOAuth2Deps(d.OAuth2, d)
	d.Authentication = mergeLegacyAuthenticationDeps(d.Authentication, d)
	registerTenantRoutes(e.Group("", d.ResolveDefaultTenant), d)
	registerTenantRoutes(e.Group("/realms/:tenant_id", d.ResolvePathTenant), d)

	authenticator := &support.Authenticator{
		UserRepo:          d.UserRepo,
		GroupRepo:         d.GroupRepo,
		SessionManager:    d.Authentication.SessionManager,
		TokenIntrospector: d.OAuth2.TokenIntrospector,
		AuthnResolver:     d.Authentication.AuthnResolver,
	}

	controlPlane := e.Group("/realms/"+spec.DefaultRealm, d.ResolveControlPlaneTenant)
	tenancyhttp.RegisterControlPlaneRoutes(controlPlane, tenancyhttp.Deps{
		Deps:           d.Deps,
		Authenticator:  authenticator,
		TenantRepo:     d.TenantRepo,
		AttrSchemaRepo: d.AttrSchemaRepo,
		UserRepo:       d.UserRepo,
	})
	tenancyhttp.RegisterControlPlaneRoutes(e.Group("", d.ResolveDefaultTenant), tenancyhttp.Deps{
		Deps:           d.Deps,
		Authenticator:  authenticator,
		TenantRepo:     d.TenantRepo,
		AttrSchemaRepo: d.AttrSchemaRepo,
		UserRepo:       d.UserRepo,
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
	if module.PasswordResetTokenStore == nil {
		module.PasswordResetTokenStore = d.PasswordResetTokenStore
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
	if module.LoginAttemptThrottle == nil {
		module.LoginAttemptThrottle = d.LoginAttemptThrottle
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
	if module.WebAuthnSessionStore == nil {
		module.WebAuthnSessionStore = d.WebAuthnSessionStore
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
	if module.RequestStore == nil {
		module.RequestStore = d.RequestStore
	}
	if module.CodeStore == nil {
		module.CodeStore = d.CodeStore
	}
	if module.PARStore == nil {
		module.PARStore = d.PARStore
	}
	if module.RefreshStore == nil {
		module.RefreshStore = d.RefreshStore
	}
	if module.DeviceCodeStore == nil {
		module.DeviceCodeStore = d.DeviceCodeStore
	}
	if module.DpopReplayStore == nil {
		module.DpopReplayStore = d.DpopReplayStore
	}
	if module.ClientAssertionReplayStore == nil {
		module.ClientAssertionReplayStore = d.ClientAssertionReplayStore
	}
	if module.AccessTokenDenylist == nil {
		module.AccessTokenDenylist = d.AccessTokenDenylist
	}
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

func registerTenantRoutes(g *echo.Group, d Deps) {
	authenticator := &support.Authenticator{
		UserRepo:          d.UserRepo,
		GroupRepo:         d.GroupRepo,
		SessionManager:    d.Authentication.SessionManager,
		TokenIntrospector: d.OAuth2.TokenIntrospector,
		AuthnResolver:     d.Authentication.AuthnResolver,
	}

	appGate := d.Application.Gate(d.GroupRepo, d.TrustedForwardedHops)
	clientDisplayNames := d.Application.ClientDisplayNames(d.OAuth2.ClientRepo)

	oauth2http.RegisterRoutes(g, oauth2http.Deps{
		Deps:                       d.Deps,
		Authenticator:              authenticator,
		ApplicationGate:            appGate,
		AuditEventRepo:             d.OAuth2.AuditEventRepo,
		AuthzDetailTypeRepo:        d.OAuth2.AuthzDetailTypeRepo,
		ClientRepo:                 d.OAuth2.ClientRepo,
		ConsentRepo:                d.OAuth2.ConsentRepo,
		ClientDisplayNameResolver:  clientDisplayNames,
		KeyStore:                   d.KeyStore,
		TenantSaltStore:            d.TenantSaltStore,
		TenantRepo:                 d.TenantRepo,
		PARStore:                   d.OAuth2.PARStore,
		RequestStore:               d.OAuth2.RequestStore,
		UserRepo:                   d.UserRepo,
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
		AgentRepo:                  d.AgentRepo,
		TokenIntrospector:          d.OAuth2.TokenIntrospector,
		AccessTokenDenylist:        d.OAuth2.AccessTokenDenylist,
		AttrSchemaRepo:             d.AttrSchemaRepo,
		AuthEventBucketStore:       d.Authentication.AuthEventBucketStore,
		Authorizer:                 d.OAuth2.Authorizer,
		SentinelPasswordHash:       d.Authentication.SentinelPasswordHash,
		WebAuthnRP:                 d.Authentication.WebAuthnRP,
		WebAuthnCredentialRepo:     d.Authentication.WebAuthnCredentialRepo,
		WebAuthnSessionStore:       d.Authentication.WebAuthnSessionStore,
		RecoveryCodeRepo:           d.Authentication.RecoveryCodeRepo,
	})

	authhttp.RegisterRoutes(g, authhttp.Deps{
		Deps:                      d.Deps,
		Authenticator:             authenticator,
		AuditEventRepo:            d.OAuth2.AuditEventRepo,
		UserRepo:                  d.UserRepo,
		PasswordHasher:            d.Authentication.PasswordHasher,
		PasswordHistoryRepo:       d.Authentication.PasswordHistoryRepo,
		ConsentRepo:               d.OAuth2.ConsentRepo,
		ClientDisplayNameResolver: clientDisplayNames,
		AttrSchemaRepo:            d.AttrSchemaRepo,
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
	})

	idmhttp.RegisterRoutes(g, idmhttp.Deps{
		Deps:                  d.Deps,
		Authenticator:         authenticator,
		UserRepo:              d.UserRepo,
		GroupRepo:             d.GroupRepo,
		AgentRepo:             d.AgentRepo,
		ClientRepo:            d.OAuth2.ClientRepo,
		ScimRepo:              d.Scim.Repo,
		AttrSchemaRepo:        d.AttrSchemaRepo,
		ConsentRepo:           d.OAuth2.ConsentRepo,
		RefreshStore:          d.OAuth2.RefreshStore,
		DeviceCodeStore:       d.OAuth2.DeviceCodeStore,
		MfaFactorRepo:         d.Authentication.MfaFactorRepo,
		PasswordHasher:        d.Authentication.PasswordHasher,
		PasswordHistoryRepo:   d.Authentication.PasswordHistoryRepo,
		EmailChangeTokenStore: d.Authentication.EmailChangeTokenStore,
		EmailSender:           d.Authentication.EmailSender,
	})

	tenancyhttp.RegisterRoutes(g, tenancyhttp.Deps{
		Deps:           d.Deps,
		Authenticator:  authenticator,
		TenantRepo:     d.TenantRepo,
		AttrSchemaRepo: d.AttrSchemaRepo,
		UserRepo:       d.UserRepo,
	})

	d.WsFederation.Register(g, d.Deps, authenticator, appGate, d.UserRepo, d.FederationSigner,
		d.OAuth2.ClientAssertionReplayStore, d.Authentication.LoginAttemptThrottle, d.Authentication.PasswordHasher, d.Authentication.SentinelPasswordHash)

	d.Saml.Register(g, d.Deps, authenticator, appGate, d.UserRepo, d.FederationSigner)

	d.Application.Register(g, d.Deps, authenticator, d.GroupRepo, d.UserRepo, d.OAuth2.ClientRepo, d.WsFederation.RPRepo, d.Saml.SPRepo)

	d.Scim.Register(g, d.Deps, authenticator, d.UserRepo, d.GroupRepo, d.Emit)
}
