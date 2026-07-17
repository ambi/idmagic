// Package http: authentication コンテキストの HTTP アダプタ。
//
// アカウント自己管理のうち認証・MFA・セッション・consent・signin activity、
// パスワード変更・リセット、認証イベントバケットの閲覧を所有する。
// 共有基盤 support.Deps を受け取り router から登録される。
package http

import (
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	oauthusecases "github.com/ambi/idmagic/backend/oauth2/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/labstack/echo/v5"
)

// Deps は authentication HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator

	AuditEventRepo            auditports.AuditEventRepository
	UserRepo                  idmports.UserRepository
	PasswordHasher            authnports.PasswordHasher
	PasswordHistoryRepo       authnports.PasswordHistoryRepository
	ConsentRepo               oauthports.ConsentRepository
	ClientDisplayNameResolver *support.ClientDisplayNameResolver
	AttrSchemaRepo            tenantports.TenantUserAttributeSchemaRepository
	MfaFactorRepo             authnports.MfaFactorRepository
	MfaEnrollmentBypassRepo   authnports.MfaEnrollmentBypassRepository
	AuthEventBucketStore      authnports.AuthEventBucketStore
	TenantRepo                tenantports.TenantRepository
	PasswordResetTokenStore   authnports.PasswordResetTokenStore
	EmailSender               authnports.EmailSender
	BreachedPasswordChecker   authnports.BreachedPasswordChecker

	// WebAuthn / Passkey と backup recovery code の self-service 管理 (wi-26)。
	// WebAuthnRP が nil の場合 WebAuthn 登録は無効。
	WebAuthnRP             *gowebauthn.WebAuthn
	WebAuthnCredentialRepo authnports.WebAuthnCredentialRepository
	WebAuthnSessionStore   authnports.WebAuthnSessionStore
	RecoveryCodeRepo       authnports.RecoveryCodeRepository
}

// RegisterRoutes はテナント解決済みグループに authentication コンテキストの
// エンドポイントを登録する。パス・メソッド・middleware は分割前と一致する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/auth/account", d.handleAccountContext)
	g.GET("/api/account/consents", d.handleListAccountConsents)
	g.POST("/api/account/consents/:client_id/revoke", d.handleRevokeAccountConsent)
	g.POST("/api/account/step_up/start", d.handleStartStepUp)
	g.POST("/api/account/step_up/complete", d.handleCompleteStepUp)
	g.POST("/api/account/step_up/webauthn/challenge", d.handleStepUpWebAuthnChallenge)
	g.GET("/api/account/security", d.handleGetAccountSecurity)
	g.POST("/api/account/mfa/totp/enroll/start", d.handleStartTotpEnrollment)
	g.POST("/api/account/mfa/totp/enroll/confirm", d.handleConfirmTotpEnrollment)
	g.POST("/api/account/mfa/totp/remove", d.handleRemoveTotpFactor)
	g.POST("/api/account/mfa/webauthn/register/start", d.handleStartWebAuthnRegistration)
	g.POST("/api/account/mfa/webauthn/register/finish", d.handleFinishWebAuthnRegistration)
	g.POST("/api/account/mfa/webauthn/remove", d.handleRemoveWebAuthnCredential)
	g.POST("/api/account/mfa/recovery-codes/generate", d.handleGenerateRecoveryCodes)
	g.POST("/api/account/mfa/recovery-codes/revoke", d.handleRevokeRecoveryCodes)
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
	g.POST("/api/admin/users/:sub/mfa-enrollment-bypass", d.handleIssueMfaEnrollmentBypass)
	g.DELETE("/api/admin/users/:sub/mfa-enrollment-bypass", d.handleRevokeMfaEnrollmentBypass)
}

func (d Deps) ConsentDeps() oauthusecases.ConsentDeps {
	return oauthusecases.ConsentDeps{ConsentRepo: d.ConsentRepo, Emit: d.Emit}
}

// legacyEmit adapts the fire-and-forget support.Deps.Emit to the
// error-returning signature ChangePasswordDeps requires (wi-184 T003). It is
// the default for handlers not yet migrated to the transaction runner.
func (d Deps) legacyEmit() func(spec.DomainEvent) error {
	return func(event spec.DomainEvent) error {
		if d.Emit != nil {
			d.Emit(event)
		}
		return nil
	}
}
