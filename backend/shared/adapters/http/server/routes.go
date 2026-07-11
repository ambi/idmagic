// Package server: Echo v5 を用いた HTTP アダプタの router。
package server

import (
	"github.com/ambi/idmagic/backend/application"
	authhttp "github.com/ambi/idmagic/backend/authentication/adapters/http"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	idmhttp "github.com/ambi/idmagic/backend/identitymanagement/adapters/http"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2http "github.com/ambi/idmagic/backend/oauth2/adapters/http"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	samlhttp "github.com/ambi/idmagic/backend/saml/adapters/http"
	samlports "github.com/ambi/idmagic/backend/saml/ports"
	scimhttp "github.com/ambi/idmagic/backend/scim/adapters/http"
	scimports "github.com/ambi/idmagic/backend/scim/ports"
	scimusecases "github.com/ambi/idmagic/backend/scim/usecases"
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

	ScimRepo       scimports.ScimRepository
	AttrSchemaRepo tenantports.TenantUserAttributeSchemaRepository
	UserRepo       idmports.UserRepository
	OAuth2         oauth2.Module
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
	AuthEventBucketStore       authnports.AuthEventBucketStore
	JWKResolver                *crypto.JWKResolver
	PasswordHasher             authnports.PasswordHasher
	GroupRepo                  idmports.GroupRepository
	AgentRepo                  idmports.AgentRepository
	MfaFactorRepo              authnports.MfaFactorRepository
	PasswordHistoryRepo        authnports.PasswordHistoryRepository
	PasswordResetTokenStore    authnports.PasswordResetTokenStore
	EmailChangeTokenStore      authnports.EmailChangeTokenStore
	EmailSender                authnports.EmailSender
	BreachedPasswordChecker    authnports.BreachedPasswordChecker
	LoginAttemptThrottle       authnports.LoginAttemptThrottle
	SentinelPasswordHash       string
	SessionManager             *authusecases.SessionManager
	AuthnResolver              authdomain.AuthenticationContextResolver
	WsFederation               wsfederation.Module
	SamlSPRepo                 samlports.SamlServiceProviderRepository
	FederationSigner           *samltoken.Signer
	Application                application.Module

	// WebAuthn / Passkey と backup recovery code (wi-26)。WebAuthnRP が nil の場合 WebAuthn は無効。
	WebAuthnRP             *gowebauthn.WebAuthn
	WebAuthnCredentialRepo authnports.WebAuthnCredentialRepository
	WebAuthnSessionStore   authnports.WebAuthnSessionStore
	RecoveryCodeRepo       authnports.RecoveryCodeRepository
}

func Register(e *echo.Echo, d Deps) {
	d.OAuth2 = mergeLegacyOAuth2Deps(d.OAuth2, d)
	registerTenantRoutes(e.Group("", d.ResolveDefaultTenant), d)
	registerTenantRoutes(e.Group("/realms/:tenant_id", d.ResolvePathTenant), d)

	authenticator := &support.Authenticator{
		UserRepo:          d.UserRepo,
		GroupRepo:         d.GroupRepo,
		SessionManager:    d.SessionManager,
		TokenIntrospector: d.OAuth2.TokenIntrospector,
		AuthnResolver:     d.AuthnResolver,
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
		SessionManager:    d.SessionManager,
		TokenIntrospector: d.OAuth2.TokenIntrospector,
		AuthnResolver:     d.AuthnResolver,
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
		PasswordHasher:             d.PasswordHasher,
		LoginAttemptThrottle:       d.LoginAttemptThrottle,
		MfaFactorRepo:              d.MfaFactorRepo,
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
		AuthEventBucketStore:       d.AuthEventBucketStore,
		Authorizer:                 d.OAuth2.Authorizer,
		SentinelPasswordHash:       d.SentinelPasswordHash,
		WebAuthnRP:                 d.WebAuthnRP,
		WebAuthnCredentialRepo:     d.WebAuthnCredentialRepo,
		WebAuthnSessionStore:       d.WebAuthnSessionStore,
		RecoveryCodeRepo:           d.RecoveryCodeRepo,
	})

	authhttp.RegisterRoutes(g, authhttp.Deps{
		Deps:                      d.Deps,
		Authenticator:             authenticator,
		AuditEventRepo:            d.OAuth2.AuditEventRepo,
		UserRepo:                  d.UserRepo,
		PasswordHasher:            d.PasswordHasher,
		PasswordHistoryRepo:       d.PasswordHistoryRepo,
		ConsentRepo:               d.OAuth2.ConsentRepo,
		ClientDisplayNameResolver: clientDisplayNames,
		AttrSchemaRepo:            d.AttrSchemaRepo,
		MfaFactorRepo:             d.MfaFactorRepo,
		AuthEventBucketStore:      d.AuthEventBucketStore,
		TenantRepo:                d.TenantRepo,
		PasswordResetTokenStore:   d.PasswordResetTokenStore,
		EmailSender:               d.EmailSender,
		BreachedPasswordChecker:   d.BreachedPasswordChecker,
		WebAuthnRP:                d.WebAuthnRP,
		WebAuthnCredentialRepo:    d.WebAuthnCredentialRepo,
		WebAuthnSessionStore:      d.WebAuthnSessionStore,
		RecoveryCodeRepo:          d.RecoveryCodeRepo,
	})

	idmhttp.RegisterRoutes(g, idmhttp.Deps{
		Deps:                  d.Deps,
		Authenticator:         authenticator,
		UserRepo:              d.UserRepo,
		GroupRepo:             d.GroupRepo,
		AgentRepo:             d.AgentRepo,
		ClientRepo:            d.OAuth2.ClientRepo,
		ScimRepo:              d.ScimRepo,
		AttrSchemaRepo:        d.AttrSchemaRepo,
		ConsentRepo:           d.OAuth2.ConsentRepo,
		RefreshStore:          d.OAuth2.RefreshStore,
		DeviceCodeStore:       d.OAuth2.DeviceCodeStore,
		MfaFactorRepo:         d.MfaFactorRepo,
		PasswordHasher:        d.PasswordHasher,
		PasswordHistoryRepo:   d.PasswordHistoryRepo,
		EmailChangeTokenStore: d.EmailChangeTokenStore,
		EmailSender:           d.EmailSender,
	})

	tenancyhttp.RegisterRoutes(g, tenancyhttp.Deps{
		Deps:           d.Deps,
		Authenticator:  authenticator,
		TenantRepo:     d.TenantRepo,
		AttrSchemaRepo: d.AttrSchemaRepo,
		UserRepo:       d.UserRepo,
	})

	d.WsFederation.Register(g, d.Deps, authenticator, appGate, d.UserRepo, d.FederationSigner,
		d.OAuth2.ClientAssertionReplayStore, d.LoginAttemptThrottle, d.PasswordHasher, d.SentinelPasswordHash)

	samlhttp.RegisterRoutes(g, samlhttp.Deps{
		Deps:             d.Deps,
		Authenticator:    authenticator,
		ApplicationGate:  appGate,
		SamlSPRepo:       d.SamlSPRepo,
		FederationSigner: d.FederationSigner,
		UserRepo:         d.UserRepo,
	})

	d.Application.Register(g, d.Deps, authenticator, d.GroupRepo, d.UserRepo, d.OAuth2.ClientRepo, d.WsFederation.RPRepo, d.SamlSPRepo)

	scimUsecasesInst := scimusecases.NewUsecases(d.ScimRepo, d.UserRepo, d.GroupRepo, d.Emit)
	scimhttp.RegisterRoutes(g, scimhttp.Deps{
		Deps:          d.Deps,
		Authenticator: authenticator,
		Usecases:      scimUsecasesInst,
	})
}
