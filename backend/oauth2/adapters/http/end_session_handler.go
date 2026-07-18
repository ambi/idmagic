package http

import (
	"net/http"
	"net/url"

	"github.com/ambi/idmagic/backend/oauth2/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

func (d Deps) handleEndSession(c *echo.Context) error {
	if d.SessionManager != nil {
		_ = d.SessionManager.Revoke(c.Request().Context(), c.Request().Header.Get("Cookie"))
		d.clearSessionCookie(c)
	}
	post := c.QueryParam("post_logout_redirect_uri")
	if post == "" {
		post = c.Request().PostFormValue("post_logout_redirect_uri")
	}
	if post == "" {
		return c.Redirect(http.StatusSeeOther, "/status?state=signed-out")
	}
	clientID := c.QueryParam("client_id")
	if clientID == "" {
		clientID = c.Request().PostFormValue("client_id")
	}
	if clientID == "" {
		return writeOAuthError(c, usecases.NewOAuthError("invalid_request", "client_id が必要"))
	}
	client, err := d.ClientRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), clientID)
	if err != nil {
		return writeOAuthError(c, err)
	}
	if client == nil {
		return writeOAuthError(c, usecases.NewOAuthError("invalid_request", "post_logout_redirect_uri が未登録"))
	}
	// Redirect only to a URI from the client's registered allowlist. Selecting the
	// stored value (rather than reusing the request parameter) keeps the redirect
	// target server-controlled and avoids open-redirect via user input.
	registered := ""
	for _, uri := range client.RedirectURIs {
		if uri == post {
			registered = uri
			break
		}
	}
	if registered == "" {
		return writeOAuthError(c, usecases.NewOAuthError("invalid_request", "post_logout_redirect_uri が未登録"))
	}
	u, err := url.Parse(registered)
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
