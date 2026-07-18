// Package http: oauth2 コンテキストの HTTP アダプタ。
//
// OAuth 2.0 / OIDC のプロトコルエンドポイント (authorize/token/introspect/revoke/
// userinfo/par/device/discovery/register) と、認可トランザクションのフロントエンドである
// 対話ログイン (login/totp/consent/end_session)、および client/consent/key/
// role_policy の管理 API を所有する。監査 (audit_event) の管理 API は audit context が
// 所有する (wi-146)。共有基盤 support.Deps を受け取り router から登録される。
package http

import (
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	oauthusecases "github.com/ambi/idmagic/backend/oauth2/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/labstack/echo/v5"
)

// Deps は OAuth2 HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator
	*support.ApplicationGate

	AuthzDetailTypeRepo        oauthports.AuthorizationDetailTypeRepository
	ClientRepo                 oauthports.OAuth2ClientRepository
	ConsentRepo                oauthports.ConsentRepository
	ClientDisplayNameResolver  *support.ClientDisplayNameResolver
	KeyStore                   signingports.KeyStore
	TenantRepo                 tenantports.TenantRepository
	PARStore                   oauthports.PARStore
	RequestStore               oauthports.AuthorizationRequestStore
	UserRepo                   idmports.UserRepository
	PasswordHasher             authnports.PasswordHasher
	LoginAttemptThrottle       authnports.LoginAttemptThrottle
	MfaFactorRepo              authnports.MfaFactorRepository
	MfaEnrollmentBypassRepo    authnports.MfaEnrollmentBypassRepository
	CodeStore                  oauthports.AuthorizationCodeStore
	JWKResolver                *crypto.JWKResolver
	ClientAssertionReplayStore oauthports.ClientAssertionReplayStore
	DeviceCodeStore            oauthports.DeviceCodeStore
	DpopReplayStore            oauthports.DpopReplayStore
	RefreshStore               oauthports.RefreshTokenStore
	TokenIssuer                oauthports.TokenIssuer
	AgentRepo                  idmports.AgentRepository
	TokenIntrospector          oauthports.TokenIntrospector
	IDTokenHintVerifier        oauthports.IDTokenHintVerifier
	AccessTokenDenylist        oauthports.AccessTokenDenylist
	AttrSchemaRepo             tenantports.TenantUserAttributeSchemaRepository
	AuthEventBucketStore       authnports.AuthEventBucketStore
	Authorizer                 oauthports.Authorizer
	TenantSaltStore            auditports.TenantSaltStore
	SentinelPasswordHash       string

	// WebAuthn / recovery code を第二要素 (login step) として使うための依存 (wi-26)。
	// WebAuthnRP が nil の場合 WebAuthn login は無効。
	WebAuthnRP             *gowebauthn.WebAuthn
	WebAuthnCredentialRepo authnports.WebAuthnCredentialRepository
	WebAuthnSessionStore   authnports.WebAuthnSessionStore
	RecoveryCodeRepo       authnports.RecoveryCodeRepository
}

// RegisterRoutes はテナント解決済みグループに oauth2 コンテキストのエンドポイントを
// 登録する。パス・メソッド・middleware は分割前と一致する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/authorize", d.handleAuthorize)
	g.GET("/end_session", d.handleEndSession)
	g.POST("/end_session", d.handleEndSession)
	g.GET("/api/auth/transaction", d.handleTransaction)
	g.POST("/api/auth/login", d.handleLoginAPI)
	g.POST("/api/auth/consent", d.handleConsentAPI)
	g.POST("/api/auth/totp", d.handleTOTPAPI)
	g.POST("/api/auth/webauthn/challenge", d.handleWebAuthnChallengeAPI)
	g.POST("/api/auth/webauthn", d.handleWebAuthnAPI)
	g.POST("/api/auth/recovery-code", d.handleRecoveryCodeAPI)
	g.POST("/api/auth/mfa/enrollment/totp/start", d.handleStartMfaEnrollmentAPI)
	g.POST("/api/auth/mfa/enrollment/totp/confirm", d.handleConfirmMfaEnrollmentAPI)
	g.GET("/api/auth/device", d.handleDeviceContext)
	g.POST("/api/auth/device", d.handleDeviceAPI)
	g.POST("/token", d.handleToken)
	g.POST("/revoke", d.handleRevoke)
	g.POST("/introspect", d.handleIntrospect)
	g.GET("/userinfo", d.handleUserInfo)
	g.POST("/userinfo", d.handleUserInfo)
	g.POST("/register", d.handleRegisterClient)
	g.POST("/par", d.handlePAR)
	g.POST("/device_authorization", d.handleDeviceAuthorization)
	g.GET("/.well-known/openid-configuration", d.handleDiscovery)
	g.GET("/.well-known/oauth-authorization-server", d.handleDiscovery)
	g.GET("/api/admin/clients", d.handleListAdminOAuth2Clients)
	g.GET("/api/admin/clients/:client_id", d.handleGetAdminOAuth2Client)
	g.POST("/api/admin/clients", d.handleCreateAdminOAuth2Client)
	g.PATCH("/api/admin/clients/:client_id", d.handleUpdateAdminOAuth2Client)
	g.DELETE("/api/admin/clients/:client_id", d.handleDeleteAdminOAuth2Client)
	g.GET("/api/admin/authorization-detail-types", d.handleListAuthorizationDetailTypes)
	g.GET("/api/admin/authorization-detail-types/:type", d.handleGetAuthorizationDetailType)
	g.POST("/api/admin/authorization-detail-types", d.handleCreateAuthorizationDetailType)
	g.PATCH("/api/admin/authorization-detail-types/:type", d.handleUpdateAuthorizationDetailType)
	g.DELETE("/api/admin/authorization-detail-types/:type", d.handleDeleteAuthorizationDetailType)
	g.GET("/api/admin/consents", d.handleListAdminConsents)
	g.GET("/api/admin/consents/:sub/:client_id", d.handleGetAdminConsent)
	g.DELETE("/api/admin/consents/:sub/:client_id", d.handleRevokeAdminConsent)
	g.GET("/api/admin/policy/roles", d.handleListAdminRolePolicies)
}

func (d Deps) ConsentDeps() oauthusecases.ConsentDeps {
	return oauthusecases.ConsentDeps{ConsentRepo: d.ConsentRepo, Emit: d.Emit}
}
