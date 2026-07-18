package http

import (
	"net/http"
	"net/url"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/oauth2/usecases"

	"github.com/labstack/echo/v5"
)

// handleEndSession は RP-Initiated Logout 1.0 endpoint (ADR-127)。id_token_hint が
// あれば署名・iss・aud・sub・sid を検証して対象 sid を解決し (client_id パラメータと
// 矛盾する hint は拒否)、無ければ browser cookie による既存の解決方法にフォールバック
// する。ローカル revoke (LoginSession + 同じ sid を持つ RefreshTokenRecord) を
// 確定させてから post_logout_redirect_uri へリダイレクトする。
func (d Deps) handleEndSession(c *echo.Context) error {
	ctx := c.Request().Context()

	idTokenHint := c.QueryParam("id_token_hint")
	if idTokenHint == "" {
		idTokenHint = c.Request().PostFormValue("id_token_hint")
	}
	clientID := c.QueryParam("client_id")
	if clientID == "" {
		clientID = c.Request().PostFormValue("client_id")
	}
	post := c.QueryParam("post_logout_redirect_uri")
	if post == "" {
		post = c.Request().PostFormValue("post_logout_redirect_uri")
	}

	target, err := usecases.ResolveEndSession(ctx, usecases.EndSessionDeps{
		ClientRepo: d.ClientRepo, HintVerifier: d.IDTokenHintVerifier,
	}, usecases.EndSessionInput{ClientID: clientID, PostLogoutRedirectURI: post, IDTokenHint: idTokenHint})
	if err != nil {
		return writeOAuthError(c, err)
	}

	d.endLocalSession(c, target.Sid)

	if target.Client == nil {
		return c.Redirect(http.StatusSeeOther, "/status?state=signed-out")
	}
	// Redirect only to a URI from the client's registered allowlist (resolved by
	// ResolveEndSession). Selecting the stored value avoids open-redirect via
	// user input.
	u, err := url.Parse(target.RedirectURI)
	if err != nil {
		return writeOAuthError(c, usecases.NewOAuthError("invalid_request", "post_logout_redirect_uri が不正"))
	}
	query := u.Query()
	if state := c.QueryParam("state"); state != "" {
		query.Set("state", state)
	}
	u.RawQuery = query.Encode()
	return c.Redirect(http.StatusFound, u.String())
}

// endLocalSession は id_token_hint (優先) または browser cookie から解決した sid を
// もとに、LoginSession と同じ sid を共有する RefreshTokenRecord を失効させる
// (ADR-127)。RP への通知 (front/back-channel logout) は T006 のスコープであり、
// ここではローカル revoke のみを行う。
func (d Deps) endLocalSession(c *echo.Context, sid string) {
	ctx := c.Request().Context()
	now := time.Now().UTC()
	if d.SessionManager == nil {
		return
	}
	if sid == "" {
		sid = d.SessionManager.SessionIDFromCookie(c.Request().Header.Get("Cookie"))
	}
	if sid != "" {
		_ = authusecases.EndSession(ctx, authusecases.SessionDeps{Store: d.SessionManager.Store, Emit: d.Emit}, sid, now)
		if d.RefreshStore != nil {
			_ = usecases.RevokeTokensBySid(ctx, usecases.RevokeDeps{RefreshStore: d.RefreshStore}, sid, now)
		}
	}
	d.clearSessionCookie(c)
}
