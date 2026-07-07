// Package http: authentication コンテキストの HTTP アダプタ。
//
// アカウント自己管理のうち認証・MFA・セッション・consent・signin activity、
// パスワード変更・リセット、認証イベントバケットの閲覧を所有する。
// 共有基盤 support.Deps を受け取り router から登録される。
package http

import (
	authnports "github.com/ambi/idmagic/internal/authentication/ports"
	idmports "github.com/ambi/idmagic/internal/identitymanagement/ports"
	oauthports "github.com/ambi/idmagic/internal/oauth2/ports"
	oauthusecases "github.com/ambi/idmagic/internal/oauth2/usecases"
	"github.com/ambi/idmagic/internal/shared/adapters/http/support"
	tenantports "github.com/ambi/idmagic/internal/tenancy/ports"

	"github.com/labstack/echo/v5"
)

// Deps は authentication HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator

	AuditEventRepo            oauthports.AuditEventRepository
	UserRepo                  idmports.UserRepository
	PasswordHasher            authnports.PasswordHasher
	PasswordHistoryRepo       authnports.PasswordHistoryRepository
	ConsentRepo               oauthports.ConsentRepository
	ClientDisplayNameResolver *support.ClientDisplayNameResolver
	AttrSchemaRepo            tenantports.TenantUserAttributeSchemaRepository
	MfaFactorRepo             authnports.MfaFactorRepository
	AuthEventBucketStore      authnports.AuthEventBucketStore
	TenantRepo                tenantports.TenantRepository
	PasswordResetTokenStore   authnports.PasswordResetTokenStore
	EmailSender               authnports.EmailSender
	BreachedPasswordChecker   authnports.BreachedPasswordChecker
}

// RegisterRoutes はテナント解決済みグループに authentication コンテキストの
// エンドポイントを登録する。パス・メソッド・middleware は分割前と一致する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/auth/account", d.handleAccountContext)
	g.GET("/api/account/consents", d.handleListAccountConsents)
	g.POST("/api/account/consents/:client_id/revoke", d.handleRevokeAccountConsent)
	g.POST("/api/account/step_up/start", d.handleStartStepUp)
	g.POST("/api/account/step_up/complete", d.handleCompleteStepUp)
	g.GET("/api/account/security", d.handleGetAccountSecurity)
	g.POST("/api/account/mfa/totp/enroll/start", d.handleStartTotpEnrollment)
	g.POST("/api/account/mfa/totp/enroll/confirm", d.handleConfirmTotpEnrollment)
	g.POST("/api/account/mfa/totp/remove", d.handleRemoveTotpFactor)
	g.GET("/api/account/signin_activity", d.handleListSignInActivity)
	g.GET("/api/account/sessions", d.handleListAccountSessions)
	g.POST("/api/account/sessions/:id/revoke", d.handleRevokeAccountSession)
	g.POST("/api/account/sessions/revoke_others", d.handleRevokeOtherAccountSessions)
	g.POST("/api/auth/change_password", d.handleChangePasswordAPI)
	g.GET("/api/auth/password_reset_context", d.handlePasswordResetContext)
	g.POST("/api/auth/forgot_password", d.handleForgotPasswordAPI)
	g.POST("/api/auth/reset_password", d.handleResetPasswordAPI)
	g.GET("/api/admin/users/:sub/signin_activity", d.handleGetUserSignInActivity)
	g.GET("/api/admin/authentication_event_buckets", d.handleListAuthEventBuckets)
}

func (d Deps) ConsentDeps() oauthusecases.ConsentDeps {
	return oauthusecases.ConsentDeps{ConsentRepo: d.ConsentRepo, Emit: d.Emit}
}
