package support_http

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v5"
)

const (
	CSRFCookie = "idmagic_csrf"
	CSRFHeader = "X-Csrf-Token"
)

// ErrBrowserVerificationFailed は VerifyBrowserRequest が Origin または CSRF トークンの
// 検証に失敗し、403 の応答を書き終えたことを表す。
//
// 検証の失敗を戻り値で表さなければ、呼び出し元の `if err != nil { return err }` は
// 素通りし、403 を書いた後もハンドラーが副作用まで進んでしまう。応答だけが拒否に見えて
// 実際には要求が通る状態は、検証が無いより危険である。
var ErrBrowserVerificationFailed = errors.New("support_http: browser request verification failed")

// VerifyBrowserRequest は Origin 一致と double-submit CSRF トークンを検証する。
// 認証必須のブラウザ向け POST/PATCH 系ハンドラが冒頭で呼ぶ。
//
// 失敗した場合は 403 の Problem Details を書いたうえで ErrBrowserVerificationFailed を
// 返す。呼び出し元はそれをそのまま返し、以降の処理へ進んではならない。応答は書き終えて
// いるので、エラーハンドラーは二重に書かない。
func (d Deps) VerifyBrowserRequest(c *echo.Context) error {
	authorization := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authorization), "bearer ") || strings.HasPrefix(strings.ToLower(authorization), "dpop ") {
		// Authorization header credentials are not ambient browser credentials and are therefore
		// outside the cookie CSRF threat model. Authentication/scope validation remains mandatory.
		return nil
	}
	origin := c.Request().Header.Get("Origin")
	issuer, err := url.Parse(d.Issuer)
	if err != nil || origin == "" || origin != issuer.Scheme+"://"+issuer.Host {
		_ = WriteProblem(c, http.StatusForbidden, "invalid_origin", "The request origin does not match.")
		return ErrBrowserVerificationFailed
	}
	cookie, err := c.Cookie(TenantCookieName(c, CSRFCookie))
	header := c.Request().Header.Get(CSRFHeader)
	if err != nil || cookie.Value == "" || header == "" ||
		len(cookie.Value) != len(header) ||
		subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
		_ = WriteProblem(c, http.StatusForbidden, "csrf_failed", "CSRF validation failed.")
		return ErrBrowserVerificationFailed
	}
	return nil
}

// EnsureCSRFCookie は CSRF cookie が無ければ発行し、そのトークン値を返す。
func (d Deps) EnsureCSRFCookie(c *echo.Context) (string, error) {
	if cookie, err := c.Cookie(TenantCookieName(c, CSRFCookie)); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}
	value, err := randomToken(32)
	if err != nil {
		return "", err
	}
	c.SetCookie(&http.Cookie{ //nolint:gosec // Secure is enabled for HTTPS issuers; local HTTP development intentionally disables it.
		Name: TenantCookieName(c, CSRFCookie), Value: value, Path: TenantCookiePath(c),
		Secure: d.SecureCookies() || TenantCookieSecure(c), HttpOnly: false, SameSite: http.SameSiteStrictMode,
		MaxAge: 600,
	})
	return value, nil
}

// SecureCookies は issuer が HTTPS のときだけ cookie に Secure を付けるべきか返す。
func (d Deps) SecureCookies() bool {
	return strings.HasPrefix(d.Issuer, "https://")
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
