// Package handlers_http は、ブラウザーでの対話のログインの HTTP アダプターである。パスワード、
// 第二要素、MFA の登録の各手順を処理し、認証を終えたら認可要求の続きか return_to の画面へ戻す。
// 認可要求の続きは OAuth2 が signin/ports の契約で実装し、組み立て地点が結ぶ。
package handlers_http

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	httpdeps "github.com/ambi/idmagic/backend/authentication/internal/deps_http"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/internal/session/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

// Deps は対話のログインの handler が使う依存である。
type Deps struct{ httpdeps.Deps }

// RegisterRoutes は対話のログインのエンドポイントを登録する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/auth/negotiate", d.handleNegotiateAvailability)
	g.POST("/api/auth/negotiate", d.handleNegotiate)
	g.POST("/api/auth/login", d.handleLoginAPI)
	g.POST("/api/auth/totp", d.handleTOTPAPI)
	g.POST("/api/auth/webauthn/challenge", d.handleWebAuthnChallengeAPI)
	g.POST("/api/auth/webauthn", d.handleWebAuthnAPI)
	g.POST("/api/auth/recovery-code", d.handleRecoveryCodeAPI)
	g.POST("/api/auth/mfa/enrollment/totp/start", d.handleStartMfaEnrollmentAPI)
	g.POST("/api/auth/mfa/enrollment/totp/confirm", d.handleConfirmMfaEnrollmentAPI)
}

type browserFlowResponse struct {
	Next       string `json:"next,omitempty"`
	RedirectTo string `json:"redirect_to,omitempty"`
}

// validReturnTo は、直接のログインの後に戻す先が同じテナントの管理コンソールか WS-Federation の
// 相対パスであるかを返す。
func validReturnTo(c *echo.Context, returnTo string) bool {
	if strings.Contains(returnTo, "\\") {
		return false
	}
	parsed, err := url.Parse(returnTo)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Fragment != "" || path.Clean(parsed.Path) != parsed.Path {
		return false
	}
	adminRoot, wsfedRoot := support.TenantRoute(c, "/admin"), support.TenantRoute(c, "/wsfed")
	return parsed.Path == adminRoot || strings.HasPrefix(parsed.Path, adminRoot+"/") || parsed.Path == wsfedRoot
}

func (d Deps) setSessionCookie(c *echo.Context, sessionID string) {
	c.SetCookie(&http.Cookie{Name: support.TenantCookieName(c, sessionusecases.SessionCookie), Value: sessionID, Path: support.TenantCookiePath(c), Secure: d.SecureCookies() || support.TenantCookieSecure(c), HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: sessionusecases.SessionTTLSeconds}) //nolint:gosec // Secure is selected from the configured issuer scheme.
}

func (d Deps) canUseTOTP(c *echo.Context, sub string) bool {
	if d.MfaFactorRepo == nil {
		return false
	}
	factor, err := d.MfaFactorRepo.Find(c.Request().Context(), sub, spec.MfaFactorTOTP)
	return err == nil && factor != nil && factor.Secret != nil && *factor.Secret != ""
}

// Steps は、OAuth2 の認可エンドポイントが既存のセッションに追加の認証を求める操作である。
// OAuth2 が要求する形を、型を import させずに満たし、組み立て地点が結ぶ。
type Steps struct{ Deps }

// EnforceTenantDefault は IdMagic 自身の画面への認可にテナントのデフォルトのポリシーを適用する。
// true は追加の認証の画面へ移すことを表す。
func (s Steps) EnforceTenantDefault(c *echo.Context, authn *authdomain.AuthenticationContext, allowChallenge bool) (bool, error) {
	return s.enforceDefaultSignInPolicy(c, authn, allowChallenge)
}

// PendingPath は保留中のセッションが進む画面の経路を返す。
func (s Steps) PendingPath(c *echo.Context, authn *authdomain.AuthenticationContext) string {
	return s.pendingAuthPath(c, authn)
}

// CanUseTOTP は sub が TOTP を登録しているかを返す。
func (s Steps) CanUseTOTP(c *echo.Context, sub string) bool {
	return s.canUseTOTP(c, sub)
}

// SecondFactorMethods は sub が使える第二要素を返す。
func (s Steps) SecondFactorMethods(c *echo.Context, sub string) []string {
	return s.secondFactorMethods(c, sub)
}

// RequireSecondFactor はセッションを第二要素の照合の待ちにして、その Cookie を発行する。
// セッションが失効していれば nil を返す。
func (s Steps) RequireSecondFactor(ctx context.Context, c *echo.Context, sessionID string) (*authdomain.AuthenticationContext, error) {
	pending, err := s.requireSecondFactor(ctx, c, sessionID)
	if err != nil || pending == nil {
		return pending, err
	}
	s.setSessionCookie(c, pending.SessionID) //nolint:contextcheck // Cookie path is derived from the Echo request.
	return pending, nil
}

// PendingStep は保留中のセッションが進む画面の種類と、第二要素の画面に出す選択肢を返す。
func (s Steps) PendingStep(c *echo.Context, authn *authdomain.AuthenticationContext) (kind string, methods []string, canRememberDevice bool) {
	return s.pendingStep(c, authn)
}

// ValidReturnTo は直接のログインの後に戻す先として returnTo を受け付けるかを返す。
func (s Steps) ValidReturnTo(c *echo.Context, returnTo string) bool {
	return validReturnTo(c, returnTo)
}

// CompleteSignIn は、ほかの経路で成立した認証を UserAuthenticated として記録し、最終ログインと
// 必須操作を記録する。空でない戻り値は、認可を続けずに進ませる画面である。
func (s Steps) CompleteSignIn(
	c *echo.Context, user *userdomain.User, authn *authdomain.AuthenticationContext, clientID string, at time.Time,
) (string, error) {
	s.emitAuthenticationSuccess(c, at, user, authn, clientID)
	return s.recordLoginAndRequiredAction(c, user, at)
}
