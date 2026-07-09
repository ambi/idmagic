// Package server: Echo v5 を用いた HTTP アダプタの router。
package server

import (
	apphttp "github.com/ambi/idmagic/internal/application/adapters/http"
	appports "github.com/ambi/idmagic/internal/application/ports"
	authhttp "github.com/ambi/idmagic/internal/authentication/adapters/http"
	authdomain "github.com/ambi/idmagic/internal/authentication/domain"
	authnports "github.com/ambi/idmagic/internal/authentication/ports"
	authusecases "github.com/ambi/idmagic/internal/authentication/usecases"
	idmhttp "github.com/ambi/idmagic/internal/identitymanagement/adapters/http"
	idmports "github.com/ambi/idmagic/internal/identitymanagement/ports"
	oauth2http "github.com/ambi/idmagic/internal/oauth2/adapters/http"
	oauthports "github.com/ambi/idmagic/internal/oauth2/ports"
	samlhttp "github.com/ambi/idmagic/internal/saml/adapters/http"
	samlports "github.com/ambi/idmagic/internal/saml/ports"
	scimhttp "github.com/ambi/idmagic/internal/scim/adapters/http"
	scimports "github.com/ambi/idmagic/internal/scim/ports"
	scimusecases "github.com/ambi/idmagic/internal/scim/usecases"
	"github.com/ambi/idmagic/internal/shared/adapters/crypto"
	"github.com/ambi/idmagic/internal/shared/adapters/http/support"
	"github.com/ambi/idmagic/internal/shared/spec"
	tenancyhttp "github.com/ambi/idmagic/internal/tenancy/adapters/http"
	tenantports "github.com/ambi/idmagic/internal/tenancy/ports"
	wsfederationhttp "github.com/ambi/idmagic/internal/wsfederation/adapters/http"
	samltoken "github.com/ambi/idmagic/internal/wsfederation/adapters/samltoken"
	wsfederationports "github.com/ambi/idmagic/internal/wsfederation/ports"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/labstack/echo/v5"
)

// Deps は HTTP アダプタ全体の起動に必要な全依存関係。
type Deps struct {
	support.Deps

	ScimRepo                    scimports.ScimRepository
	AttrSchemaRepo              tenantports.TenantUserAttributeSchemaRepository
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
	TenantSaltStore             oauthports.TenantSaltStore
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

	// WebAuthn / Passkey と backup recovery code (wi-26)。WebAuthnRP が nil の場合 WebAuthn は無効。
	WebAuthnRP             *gowebauthn.WebAuthn
	WebAuthnCredentialRepo authnports.WebAuthnCredentialRepository
	WebAuthnSessionStore   authnports.WebAuthnSessionStore
	RecoveryCodeRepo       authnports.RecoveryCodeRepository
}

func Register(e *echo.Echo, d Deps) {
	registerTenantRoutes(e.Group("", d.ResolveDefaultTenant), d)
	registerTenantRoutes(e.Group("/realms/:tenant_id", d.ResolvePathTenant), d)

	authenticator := &support.Authenticator{
		UserRepo:          d.UserRepo,
		GroupRepo:         d.GroupRepo,
		SessionManager:    d.SessionManager,
		TokenIntrospector: d.TokenIntrospector,
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

func registerTenantRoutes(g *echo.Group, d Deps) {
	authenticator := &support.Authenticator{
		UserRepo:          d.UserRepo,
		GroupRepo:         d.GroupRepo,
		SessionManager:    d.SessionManager,
		TokenIntrospector: d.TokenIntrospector,
		AuthnResolver:     d.AuthnResolver,
	}

	appGate := &support.ApplicationGate{
		ApplicationRepo:             d.ApplicationRepo,
		ApplicationAssignmentRepo:   d.ApplicationAssignmentRepo,
		GroupRepo:                   d.GroupRepo,
		ApplicationSignInPolicyRepo: d.ApplicationSignInPolicyRepo,
		DefaultSignInPolicyRepo:     d.DefaultSignInPolicyRepo,
		GateTrustedForwardedHops:    d.TrustedForwardedHops,
	}

	clientDisplayNames := &support.ClientDisplayNameResolver{
		ClientRepo:      d.ClientRepo,
		ApplicationRepo: d.ApplicationRepo,
	}

	oauth2http.RegisterRoutes(g, oauth2http.Deps{
		Deps:                       d.Deps,
		Authenticator:              authenticator,
		ApplicationGate:            appGate,
		AuditEventRepo:             d.AuditEventRepo,
		AuthzDetailTypeRepo:        d.AuthzDetailTypeRepo,
		ClientRepo:                 d.ClientRepo,
		ConsentRepo:                d.ConsentRepo,
		ClientDisplayNameResolver:  clientDisplayNames,
		KeyStore:                   d.KeyStore,
		TenantSaltStore:            d.TenantSaltStore,
		TenantRepo:                 d.TenantRepo,
		PARStore:                   d.PARStore,
		RequestStore:               d.RequestStore,
		UserRepo:                   d.UserRepo,
		PasswordHasher:             d.PasswordHasher,
		LoginAttemptThrottle:       d.LoginAttemptThrottle,
		MfaFactorRepo:              d.MfaFactorRepo,
		CodeStore:                  d.CodeStore,
		JWKResolver:                d.JWKResolver,
		ClientAssertionReplayStore: d.ClientAssertionReplayStore,
		DeviceCodeStore:            d.DeviceCodeStore,
		DpopReplayStore:            d.DpopReplayStore,
		RefreshStore:               d.RefreshStore,
		TokenIssuer:                d.TokenIssuer,
		AgentRepo:                  d.AgentRepo,
		TokenIntrospector:          d.TokenIntrospector,
		AccessTokenDenylist:        d.AccessTokenDenylist,
		AttrSchemaRepo:             d.AttrSchemaRepo,
		AuthEventBucketStore:       d.AuthEventBucketStore,
		Authorizer:                 d.Authorizer,
		SentinelPasswordHash:       d.SentinelPasswordHash,
		WebAuthnRP:                 d.WebAuthnRP,
		WebAuthnCredentialRepo:     d.WebAuthnCredentialRepo,
		WebAuthnSessionStore:       d.WebAuthnSessionStore,
		RecoveryCodeRepo:           d.RecoveryCodeRepo,
	})

	authhttp.RegisterRoutes(g, authhttp.Deps{
		Deps:                      d.Deps,
		Authenticator:             authenticator,
		AuditEventRepo:            d.AuditEventRepo,
		UserRepo:                  d.UserRepo,
		PasswordHasher:            d.PasswordHasher,
		PasswordHistoryRepo:       d.PasswordHistoryRepo,
		ConsentRepo:               d.ConsentRepo,
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
		ClientRepo:            d.ClientRepo,
		ScimRepo:              d.ScimRepo,
		AttrSchemaRepo:        d.AttrSchemaRepo,
		ConsentRepo:           d.ConsentRepo,
		RefreshStore:          d.RefreshStore,
		DeviceCodeStore:       d.DeviceCodeStore,
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

	wsfederationhttp.RegisterRoutes(g, wsfederationhttp.Deps{
		Deps:                       d.Deps,
		Authenticator:              authenticator,
		ApplicationGate:            appGate,
		WsFedRPRepo:                d.WsFedRPRepo,
		UserRepo:                   d.UserRepo,
		FederationSigner:           d.FederationSigner,
		ClientAssertionReplayStore: d.ClientAssertionReplayStore,
		LoginAttemptThrottle:       d.LoginAttemptThrottle,
		PasswordHasher:             d.PasswordHasher,
		SentinelPasswordHash:       d.SentinelPasswordHash,
	})

	samlhttp.RegisterRoutes(g, samlhttp.Deps{
		Deps:             d.Deps,
		Authenticator:    authenticator,
		ApplicationGate:  appGate,
		SamlSPRepo:       d.SamlSPRepo,
		FederationSigner: d.FederationSigner,
		UserRepo:         d.UserRepo,
	})

	apphttp.RegisterRoutes(g, apphttp.Deps{
		Deps:                        d.Deps,
		Authenticator:               authenticator,
		ApplicationRepo:             d.ApplicationRepo,
		ApplicationIconStore:        d.ApplicationIconStore,
		ApplicationAssignmentRepo:   d.ApplicationAssignmentRepo,
		ApplicationOrderingRepo:     d.ApplicationOrderingRepo,
		ApplicationCategoryRepo:     d.ApplicationCategoryRepo,
		ApplicationSignInPolicyRepo: d.ApplicationSignInPolicyRepo,
		DefaultSignInPolicyRepo:     d.DefaultSignInPolicyRepo,
		GroupRepo:                   d.GroupRepo,
		UserRepo:                    d.UserRepo,
		ClientRepo:                  d.ClientRepo,
		WsFedRPRepo:                 d.WsFedRPRepo,
		SamlSPRepo:                  d.SamlSPRepo,
	})

	scimUsecasesInst := scimusecases.NewUsecases(d.ScimRepo, d.UserRepo, d.GroupRepo, d.Emit)
	scimhttp.RegisterRoutes(g, scimhttp.Deps{
		Deps:          d.Deps,
		Authenticator: authenticator,
		Usecases:      scimUsecasesInst,
	})
}
