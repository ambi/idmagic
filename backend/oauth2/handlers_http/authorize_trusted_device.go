// 信頼済みデバイス (remember this device) の login 経路への差し込み (wi-91)。
// 発行はログインで本物の第二要素が成立した直後だけ、評価はサインインポリシーが MFA を
// 要求していてセッションがまだ第二要素を持たない時だけ行う。cookie は realm scope で、
// 既存の session cookie と同じ名前解決とパスを使うため、テナントをまたいで解決されない。
package handlers_http

import (
	"net/http"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	trusteddeviceusecases "github.com/ambi/idmagic/backend/authentication/trusteddevice/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// trustedDeviceCookie は cookie 名。realm ごとの接頭辞は TenantCookieName が付ける。
const trustedDeviceCookie = "idmagic_trusted_device"

func (d Deps) trustedDeviceDeps() trusteddeviceusecases.Deps {
	return trusteddeviceusecases.Deps{Repo: d.TrustedDeviceRepo, Emit: d.Emit}
}

// trustedDeviceMaxAge はテナントが明示した有効期間を返す。0 は機能無効。テナントを
// 解決できない場合も 0 を返し、MFA を弱める側へは倒れない。
func (d Deps) trustedDeviceMaxAge(c *echo.Context) time.Duration {
	if d.TenantRepo == nil || d.TrustedDeviceRepo == nil {
		return 0
	}
	tenant, err := d.TenantRepo.FindByID(c.Request().Context(), support.RequestTenantID(c))
	if err != nil || tenant == nil {
		return 0
	}
	return tenant.EffectiveTrustedDeviceMaxAge()
}

func (d Deps) setTrustedDeviceCookie(c *echo.Context, value string, maxAge time.Duration) {
	c.SetCookie(&http.Cookie{ //nolint:gosec // Secure is selected from the configured issuer scheme.
		Name:     support.TenantCookieName(c, trustedDeviceCookie),
		Value:    value,
		Path:     support.TenantCookiePath(c),
		Secure:   d.SecureCookies() || support.TenantCookieSecure(c),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(maxAge.Seconds()),
	})
}

func (d Deps) clearTrustedDeviceCookie(c *echo.Context) {
	c.SetCookie(&http.Cookie{ //nolint:gosec // Secure is selected from the configured issuer scheme.
		Name:     support.TenantCookieName(c, trustedDeviceCookie),
		Path:     support.TenantCookiePath(c),
		Secure:   d.SecureCookies() || support.TenantCookieSecure(c),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// trustedDeviceCookieValue は提示された cookie 値を返す。realm ごとの名前を優先し、
// 見つからなければ素の名前へ落とす (session cookie と同じ解決順)。
func (d Deps) trustedDeviceCookieValue(c *echo.Context) string {
	if cookie, err := c.Request().Cookie(support.TenantCookieName(c, trustedDeviceCookie)); err == nil {
		return cookie.Value
	}
	if cookie, err := c.Request().Cookie(trustedDeviceCookie); err == nil {
		return cookie.Value
	}
	return ""
}

// rememberDeviceIfRequested は第二要素の検証に成功した直後に呼ぶ。本人が同意し、
// テナントが機能を有効にし、factor が記憶の起点として認められる場合だけ cookie を発行する。
// 発行の失敗はログインを止めない (記憶できなかっただけで認証自体は成立している)。
func (d Deps) rememberDeviceIfRequested(c *echo.Context, sub, factor string, requested bool) error {
	if !requested || !trusteddeviceusecases.Rememberable(factor) {
		return nil
	}
	maxAge := d.trustedDeviceMaxAge(c)
	if maxAge <= 0 {
		return nil
	}
	cookie, err := trusteddeviceusecases.Issue(
		c.Request().Context(), d.trustedDeviceDeps(),
		support.RequestTenantID(c), sub, factor, c.Request().UserAgent(), maxAge, time.Now().UTC(),
	)
	if err != nil {
		return err
	}
	if cookie != "" {
		d.setTrustedDeviceCookie(c, cookie, maxAge)
	}
	return nil
}

// trustedDeviceSatisfiesSecondFactor は提示された cookie で第二要素を省略できるかを判定する。
// 省略できる場合は verifier を回転させた cookie を再発行し、session の amr へ tdev を加えて、
// 呼び出し元が持つ authn をその昇格後の内容へ差し替えたうえで true を返す。省略できない
// 場合は false を返し、呼び出し側は通常どおり第二要素を要求する。
//
// allowed は実効サインインポリシーが記憶済みデバイスによる充足を認めるかどうかで、false なら
// cookie を読まずに拒否する (「毎回 MFA」)。
func (d Deps) trustedDeviceSatisfiesSecondFactor(
	c *echo.Context,
	allowed bool,
	authn *authdomain.AuthenticationContext,
) (bool, error) {
	if !allowed || authn == nil {
		return false, nil
	}
	maxAge := d.trustedDeviceMaxAge(c)
	if maxAge <= 0 {
		return false, nil
	}
	cookie := d.trustedDeviceCookieValue(c)
	if cookie == "" {
		return false, nil
	}
	result, err := trusteddeviceusecases.Evaluate(
		c.Request().Context(), d.trustedDeviceDeps(),
		support.RequestTenantID(c), authn.UserID, cookie, maxAge, time.Now().UTC(),
	)
	if err != nil {
		return false, err
	}
	if !result.Trusted {
		// 失効・期限切れ・改竄のいずれであっても、以後の提示で無駄な照合を繰り返さない
		// よう cookie を消す。判定はサーバー側の行が正本なので、消しても安全性は変わらない。
		d.clearTrustedDeviceCookie(c)
		return false, nil
	}
	completed, err := d.SessionManager.CompleteFactor(
		c.Request().Context(), authn.SessionID, []string{trusteddeviceusecases.AMRTrustedDevice},
	)
	if err != nil {
		return false, err
	}
	if completed == nil {
		return false, nil
	}
	d.setTrustedDeviceCookie(c, result.RotatedCookie, maxAge)
	*authn = *completed
	return true, nil
}
