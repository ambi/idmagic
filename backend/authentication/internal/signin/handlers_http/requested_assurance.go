package handlers_http

import (
	"net/http"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	signinports "github.com/ambi/idmagic/backend/authentication/signin/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// enforceRequestedAssurance は、認可の要求が acr_values で MFA の水準を求めたとき、パスワードで
// 認証したログインを終える前に、その水準を満たさせる (REQ-OAUTH2-005)。サインインポリシーの判定の
// 後に呼び、ポリシーが MFA を求めないアプリと IdMagic 自身の画面の両方に掛ける。
//
// 再認証を求めた要求では、記憶済みの端末を MFA の充足と認めない (REQ-AUTHENTICATION-027)。
// 再認証は直前に要素を提示したことを求めるので、記憶済みの端末では満たさない。
//
// 応答を書いたときは handled=true を返す。
func (d Deps) enforceRequestedAssurance(
	c *echo.Context,
	authorization signinports.Authorization,
	requested signinports.RequestedAssurance,
	authn *authdomain.AuthenticationContext,
) (handled bool, err error) {
	if !requested.MFA || authn.SecondFactorAuthenticated() {
		return false, nil
	}
	if !requested.ExcludesTrustedDevice() {
		if authn.MFAAuthenticated() {
			return false, nil
		}
		trusted, err := d.trustedDeviceSatisfiesSecondFactor(c, true, authn)
		if err != nil || trusted {
			return false, err
		}
	}
	if len(d.secondFactorMethods(c, authn.UserID)) == 0 {
		return true, support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{
			RedirectTo: authorization.DeniedRedirect(c, "unmet_authentication_requirements",
				"The requested authentication level cannot be met because no second factor is enrolled."),
		})
	}
	pending, err := d.requireSecondFactor(c.Request().Context(), c, authn.SessionID)
	if err != nil {
		return true, err
	}
	if pending == nil {
		return true, support.WriteProblem(c, http.StatusUnauthorized, "authentication_required", "The session has expired.")
	}
	d.setSessionCookie(c, pending.SessionID)
	return true, support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{Next: d.pendingAuthPath(c, authn)})
}
