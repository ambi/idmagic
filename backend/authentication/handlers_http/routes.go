// Package http: authentication コンテキストの HTTP アダプタ。
//
// Deps 型定義・route 登録・feature 横断ハンドラ (account context, consents, signin
// activity, auth event buckets, security 集計) を所有する。feature 固有のハンドラは
// password/totp/webauthn/mfa/session/recovery の handlers_http パッケージへ分割されている。
// 共有基盤 support.Deps を受け取り router から登録される。
package handlers_http

import (
	httpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	mfahttp "github.com/ambi/idmagic/backend/authentication/mfa/handlers_http"
	passwordhttp "github.com/ambi/idmagic/backend/authentication/password/handlers_http"
	recoveryhttp "github.com/ambi/idmagic/backend/authentication/recovery/handlers_http"
	sessionhttp "github.com/ambi/idmagic/backend/authentication/session/handlers_http"
	trusteddevicehttp "github.com/ambi/idmagic/backend/authentication/trusteddevice/handlers_http"
	webauthnhttp "github.com/ambi/idmagic/backend/authentication/webauthn/handlers_http"

	"github.com/labstack/echo/v5"
)

// Deps は authentication HTTP ハンドラが必要とする依存。型定義自体は leaf package
// httpdeps にあり、この alias で従来通り http.Deps として参照できる。
type Deps = httpdeps.Deps

// RegisterRoutes はテナント解決済みグループに authentication コンテキストの
// エンドポイントを登録する。パス・メソッド・middleware は分割前と一致する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/auth/account", func(c *echo.Context) error { return handleAccountContext(d, c) })
	g.GET("/api/account/v1/consents", func(c *echo.Context) error { return handleListAccountConsents(d, c) })
	g.POST("/api/account/v1/consents/:client_id/revoke", func(c *echo.Context) error { return handleRevokeAccountConsent(d, c) })
	g.POST("/api/account/v1/step_up/start", func(c *echo.Context) error { return mfahttp.HandleStartStepUp(d, c) })
	g.POST("/api/account/v1/step_up/complete", func(c *echo.Context) error { return mfahttp.HandleCompleteStepUp(d, c) })
	g.POST("/api/account/v1/step_up/webauthn/challenge", func(c *echo.Context) error { return mfahttp.HandleStepUpWebAuthnChallenge(d, c) })
	g.GET("/api/account/v1/security", func(c *echo.Context) error { return handleGetAccountSecurity(d, c) })
	g.POST("/api/account/v1/mfa/totp/enroll/start", func(c *echo.Context) error { return mfahttp.HandleStartTotpEnrollment(d, c) })
	g.POST("/api/account/v1/mfa/totp/enroll/confirm", func(c *echo.Context) error { return mfahttp.HandleConfirmTotpEnrollment(d, c) })
	g.POST("/api/account/v1/mfa/totp/remove", func(c *echo.Context) error { return mfahttp.HandleRemoveTotpFactor(d, c) })
	g.POST("/api/account/v1/mfa/webauthn/register/start", func(c *echo.Context) error { return webauthnhttp.HandleStartWebAuthnRegistration(d, c) })
	g.POST("/api/account/v1/mfa/webauthn/register/finish", func(c *echo.Context) error { return webauthnhttp.HandleFinishWebAuthnRegistration(d, c) })
	g.POST("/api/account/v1/mfa/webauthn/remove", func(c *echo.Context) error { return webauthnhttp.HandleRemoveWebAuthnCredential(d, c) })
	g.POST("/api/account/v1/mfa/recovery-codes/generate", func(c *echo.Context) error { return recoveryhttp.HandleGenerateRecoveryCodes(d, c) })
	g.POST("/api/account/v1/mfa/recovery-codes/revoke", func(c *echo.Context) error { return recoveryhttp.HandleRevokeRecoveryCodes(d, c) })
	g.GET("/api/account/v1/signin_activity", func(c *echo.Context) error { return handleListSignInActivity(d, c) })
	g.GET("/api/account/v1/sessions", func(c *echo.Context) error { return sessionhttp.HandleListAccountSessions(d, c) })
	g.POST("/api/account/v1/sessions/:id/revoke", func(c *echo.Context) error { return sessionhttp.HandleRevokeAccountSession(d, c) })
	g.POST("/api/account/v1/sessions/revoke_others", func(c *echo.Context) error { return sessionhttp.HandleRevokeOtherAccountSessions(d, c) })
	g.GET("/api/account/v1/trusted_devices", func(c *echo.Context) error { return trusteddevicehttp.HandleListTrustedDevices(d, c) })
	g.POST("/api/account/v1/trusted_devices/:id/revoke", func(c *echo.Context) error { return trusteddevicehttp.HandleRevokeTrustedDevice(d, c) })
	g.POST("/api/account/v1/trusted_devices/revoke_all", func(c *echo.Context) error { return trusteddevicehttp.HandleRevokeAllTrustedDevices(d, c) })
	g.POST("/api/auth/change_password", func(c *echo.Context) error { return passwordhttp.HandleChangePasswordAPI(d, c) })
	g.GET("/api/auth/password_reset_context", func(c *echo.Context) error { return passwordhttp.HandlePasswordResetContext(d, c) })
	g.POST("/api/auth/forgot_password", func(c *echo.Context) error { return passwordhttp.HandleForgotPasswordAPI(d, c) })
	g.POST("/api/auth/reset_password", func(c *echo.Context) error { return passwordhttp.HandleResetPasswordAPI(d, c) })
	g.GET("/api/admin/v1/users/:sub/signin_activity", func(c *echo.Context) error { return handleGetUserSignInActivity(d, c) })
	g.GET("/api/admin/v1/users/:sub/sessions", func(c *echo.Context) error { return sessionhttp.HandleAdminListSessions(d, c) })
	g.POST("/api/admin/v1/users/:sub/sessions/:id/revoke", func(c *echo.Context) error { return sessionhttp.HandleAdminRevokeSession(d, c) })
	g.POST("/api/admin/v1/users/:sub/sessions/revoke_all", func(c *echo.Context) error { return sessionhttp.HandleAdminRevokeAllSessions(d, c) })
	g.GET("/api/admin/v1/authentication_event_buckets", func(c *echo.Context) error { return handleListAuthEventBuckets(d, c) })
	g.POST("/api/admin/v1/users/:sub/mfa-enrollment-bypass", func(c *echo.Context) error { return mfahttp.HandleIssueMfaEnrollmentBypass(d, c) })
	g.DELETE("/api/admin/v1/users/:sub/mfa-enrollment-bypass", func(c *echo.Context) error { return mfahttp.HandleRevokeMfaEnrollmentBypass(d, c) })
	g.POST("/api/admin/v1/users/:sub/authenticator-reset", func(c *echo.Context) error { return mfahttp.HandleResetUserAuthenticators(d, c) })
}
